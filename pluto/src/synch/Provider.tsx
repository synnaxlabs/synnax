// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { type PropsWithChildren, useCallback, useRef } from "react";

import { useAsyncEffect } from "@/hooks";
import { Status } from "@/status";
import { Context } from "@/synch/Context";
import { type FrameHandler, type ListenerAdder } from "@/synch/types";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {}

const LOGGING = false;

export const Provider: React.FC<ProviderProps> = (props) => {
  const client = Synnax.use();
  const handlersRef = useRef(new Map<channel.Key, Set<FrameHandler>>());
  const streamerRef = useRef<framer.Streamer>(null);
  const logProviderState = useCallback(
    (name: string) => {
      if (!LOGGING) return;
      console.group(name);

      // Log client state
      if (client == null) console.log("Client: null");
      else console.log(`Client Key: ${client.key}`);

      // Log handlers state
      const handlersRecord: Record<channel.Key, number> = {};
      handlersRef.current.forEach((handlers, key) => {
        handlersRecord[key] = handlers.size;
      });
      console.log("Handlers:", deep.copy(handlersRecord));

      // Log streamer state
      if (streamerRef.current == null) console.log("Streamer: null");
      else console.log("Streamer keys:", deep.copy(streamerRef.current.keys));

      console.groupEnd();
    },
    [client],
  );
  const handleError = Status.useErrorHandler();
  useAsyncEffect(async () => {
    logProviderState("Synch.Provider.tsx useAsyncEffect Start");
    if (client == null) {
      streamerRef.current?.close();
      streamerRef.current = null;
      return;
    }
    streamerRef.current = await client.openStreamer([...handlersRef.current.keys()]);
    const observableStreamer = new framer.ObservableStreamer(streamerRef.current);
    observableStreamer.onChange((frame) => {
      logProviderState("Observable streamer changed");
      const calledHandlers = new Set<FrameHandler>();
      frame.uniqueKeys.forEach((key) => {
        const handlers = handlersRef.current.get(key);
        handlers?.forEach((handler) => {
          if (calledHandlers.has(handler)) return;
          handleError(handler(frame), "Error calling Synch Frame Handler");
          calledHandlers.add(handler);
        });
      });
      console.log(`${calledHandlers.size} handlers called for frame:`, frame);
    });
    logProviderState("Synch.Provider.tsx useAsyncEffect end");
    return async () => {
      console.log("Closing streamer");
      logProviderState("Synch.Provider.tsx useAsyncEffect return start");
      await observableStreamer.close();
      logProviderState("Synch.Provider.tsx useAsyncEffect return end");
      console.log("Streamer closed");
    };
  }, [client?.key]);

  const addListener: ListenerAdder = useCallback(
    ({ channels, handler }) => {
      logProviderState("Synch.Provider.tsx addListener start");
      let isListeningToNewChannels = false;
      channels.forEach((channel) => {
        if (handlersRef.current.has(channel))
          handlersRef.current.get(channel)?.add(handler);
        else {
          isListeningToNewChannels = true;
          handlersRef.current.set(channel, new Set([handler]));
        }
      });
      if (isListeningToNewChannels)
        streamerRef.current?.updateKeys([...handlersRef.current.keys()]);
      logProviderState("Synch.Provider.tsx addListener end");
      return () => {
        logProviderState("Synch.Provider.tsx destructing listener start");
        let isDeletingChannels = false;
        channels.forEach((channel) => {
          const handlerSet = handlersRef.current.get(channel);
          handlerSet?.delete(handler);
          if (handlerSet?.size === 0) {
            isDeletingChannels = true;
            handlersRef.current.delete(channel);
          }
        });
        if (isDeletingChannels)
          streamerRef.current?.updateKeys([...handlersRef.current.keys()]);
        logProviderState("Synch.Provider.tsx destructing listener end");
      };
    },
    [client],
  );
  return <Context {...props} value={addListener} />;
};

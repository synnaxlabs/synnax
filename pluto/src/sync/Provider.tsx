// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer } from "@synnaxlabs/client";
import { StreamClosed } from "@synnaxlabs/freighter";
import { strings, toArray } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useCallback, useRef } from "react";

import { useAsyncEffect } from "@/hooks";
import { Status } from "@/status";
import { Context } from "@/sync/Context";
import { type FrameHandler, type ListenerAdder } from "@/sync/types";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {}

export const Provider = (props: PropsWithChildren): ReactElement => {
  const client = Synnax.use();
  const handlersRef = useRef(new Map<channel.Name, Set<FrameHandler>>());
  const streamerRef = useRef<framer.Streamer>(null);
  const handleError = Status.useErrorHandler();
  useAsyncEffect(async () => {
    if (client == null) {
      streamerRef.current?.close();
      streamerRef.current = null;
      return;
    }
    try {
      streamerRef.current = await client.openStreamer([...handlersRef.current.keys()]);
    } catch (e) {
      handleError(e, "Failed to open Sync.Provider streamer");
      return;
    }
    const observableStreamer = new framer.ObservableStreamer(streamerRef.current);
    observableStreamer.onChange((frame) => {
      const calledHandlers = new Set<FrameHandler>();
      frame.uniqueNames.forEach((name) => {
        handlersRef.current.get(name)?.forEach((handler) => {
          if (calledHandlers.has(handler)) return;
          try {
            handler(frame);
          } catch (e) {
            handleError(e, `Error calling Sync Frame Handler on channel ${name}`);
          }
          calledHandlers.add(handler);
        });
      });
    });
    return async () => await observableStreamer.close();
  }, [client, handleError]);

  const updateStreamer = useCallback(async () => {
    try {
      await streamerRef.current?.update([...handlersRef.current.keys()]);
    } catch (e) {
      if (StreamClosed.matches(e)) return;
      throw e;
    }
  }, []);

  const addListener: ListenerAdder = useCallback(
    ({ channels, handler }) => {
      const addedChannels: channel.Names = [];
      const channelNames = toArray(channels);
      channelNames.forEach((ch) => {
        if (handlersRef.current.has(ch)) handlersRef.current.get(ch)?.add(handler);
        else {
          addedChannels.push(ch);
          handlersRef.current.set(ch, new Set([handler]));
        }
      });
      if (addedChannels.length > 0)
        handleError(
          updateStreamer,
          `Failed to add ${strings.naturalLanguageJoin(addedChannels)} to the Sync.Provider streamer`,
        );
      return () => {
        const removedChannels: channel.Names = [];
        channelNames.forEach((ch) => {
          const handlerSet = handlersRef.current.get(ch);
          handlerSet?.delete(handler);
          if (handlerSet?.size === 0) {
            removedChannels.push(ch);
            handlersRef.current.delete(ch);
          }
        });
        if (removedChannels.length > 0)
          handleError(
            updateStreamer,
            `Failed to remove ${strings.naturalLanguageJoin(removedChannels)} from the Sync.Provider streamer`,
          );
      };
    },
    [handleError, updateStreamer],
  );

  return <Context {...props} value={addListener} />;
};

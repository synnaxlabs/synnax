// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer } from "@synnaxlabs/client";
import { sync } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { AddListenerContext } from "@/flux/sync/Context";
import { type FrameHandler, type ListenerAdder } from "@/flux/sync/types";
import { useAsyncEffect } from "@/hooks";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {
  openStreamer?: framer.StreamOpener;
}

interface MutexValue {
  streamer: framer.ObservableStreamer | null;
}

export const Provider = ({
  children,
  openStreamer: propsOpenStreamer,
}: ProviderProps): ReactElement => {
  const client = Synnax.use();
  const handlersRef = useRef(new Map<FrameHandler, channel.Name>());
  const streamerRef = useRef<sync.Mutex<MutexValue>>(sync.newMutex({ streamer: null }));
  const handleError = Status.useErrorHandler();

  useAsyncEffect(async () => {
    await updateStreamer();
    return async () =>
      await streamerRef.current.runExclusive(async () => {
        const { streamer } = streamerRef.current;
        if (streamer == null) return;
        await streamer.close();
        streamerRef.current.streamer = null;
      });
  }, [client]);

  const handleChange = useCallback(
    (frame: framer.Frame) => {
      const namesInFrame = new Set([...frame.uniqueNames]);
      handlersRef.current.forEach((channel, handler) => {
        if (!namesInFrame.has(channel)) return;
        try {
          handler(frame);
        } catch (e) {
          handleError(e, `Error calling Sync Frame Handler on channel(s): ${channel}`);
        }
      });
    },
    [handleError],
  );

  const openStreamer = useMemo(
    () => propsOpenStreamer ?? client?.openStreamer.bind(client),
    [client, propsOpenStreamer],
  );

  const updateStreamer = useCallback(async () => {
    await streamerRef.current.runExclusive(async () => {
      if (openStreamer == null) return;
      const { streamer } = streamerRef.current;
      const names = Array.from(handlersRef.current.values());
      if (streamer != null) {
        if (names.length === 0) {
          streamerRef.current.streamer = null;
          return await streamer.close();
        }
        return await streamer.update(names);
      }
      if (names.length === 0) return;
      const hardenedStreamer = await framer.HardenedStreamer.open(openStreamer, names);
      streamerRef.current.streamer = new framer.ObservableStreamer(hardenedStreamer);
      streamerRef.current.streamer.onChange(handleChange);
    });
  }, [client, handleChange, openStreamer]);

  const addListener: ListenerAdder = useCallback(
    ({ channel, handler, onOpen }) => {
      const prevNames = new Set(handlersRef.current.values());
      handlersRef.current.set(handler, channel);
      if (!prevNames.has(channel))
        handleError(async () => {
          await updateStreamer();
          onOpen?.();
        }, `Failed to add ${channel} to the Sync.Provider streamer`);
      return () => {
        handlersRef.current.delete(handler);
        handleError(
          updateStreamer,
          `Failed to remove ${channel} from the Sync.Provider streamer`,
        );
      };
    },
    [handleError, updateStreamer],
  );

  return <AddListenerContext value={addListener}>{children}</AddListenerContext>;
};

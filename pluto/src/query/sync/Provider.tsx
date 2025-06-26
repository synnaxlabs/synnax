// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, framer } from "@synnaxlabs/client";
import { array, strings, sync, unique } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useCallback, useRef } from "react";

import { useAsyncEffect } from "@/hooks";
import { AddListenerContext } from "@/query/sync/Context";
import { type FrameHandler, type ListenerAdder } from "@/query/sync/types";
import { Status } from "@/status";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {}

const uniqueNamesInMap = (map: Map<FrameHandler, Set<channel.Name>>): channel.Names =>
  unique.unique([...map.values()].flatMap((names) => [...names]));

interface Client {
  openStreamer: framer.StreamOpener;
}

export interface ProviderProps extends PropsWithChildren {
  useClient?: () => Client | null;
}

interface MutexValue {
  streamer: framer.ObservableStreamer | null;
}

export const Provider = ({
  useClient = () => Synnax.use(),
  children,
}: ProviderProps): ReactElement => {
  const client = useClient();
  const handlersRef = useRef(new Map<FrameHandler, Set<channel.Name>>());
  const streamerRef = useRef<sync.Mutex<MutexValue>>(sync.newMutex({ streamer: null }));
  const handleError = Status.useErrorHandler();

  useAsyncEffect(
    async () => async () => {
      await updateStreamer();
      return async () =>
        await streamerRef.current.runExclusive(async () => {
          const { streamer } = streamerRef.current;
          if (streamer == null) return;
          await streamer.close();
          streamerRef.current.streamer = null;
        });
    },
    [client],
  );

  const handleChange = useCallback(
    (frame: framer.Frame) => {
      const namesInFrame = new Set([...frame.uniqueNames]);
      handlersRef.current.forEach((channels, handler) => {
        if (namesInFrame.isDisjointFrom(channels)) return;
        try {
          handler(frame);
        } catch (e) {
          handleError(
            e,
            `Error calling Sync Frame Handler on channel(s): ${strings.naturalLanguageJoin([...channels])}`,
          );
        }
      });
    },
    [handleError],
  );

  const updateStreamer = useCallback(async () => {
    await streamerRef.current.runExclusive(async () => {
      if (client == null) return;
      const { streamer } = streamerRef.current;
      const names = uniqueNamesInMap(handlersRef.current);
      if (streamer != null) {
        if (names.length === 0) {
          await streamer.close();
          streamerRef.current.streamer = null;
          return;
        }
        await streamer.update(names);
        return;
      }
      const hardenedStreamer = await framer.HardenedStreamer.open(
        client.openStreamer.bind(client),
        names,
      );
      streamerRef.current.streamer = new framer.ObservableStreamer(hardenedStreamer);
      streamerRef.current.streamer.onChange(handleChange);
    });
  }, [client, handleChange]);

  const addListener: ListenerAdder = useCallback(
    ({ channels, handler, onOpen }) => {
      const channelNames = array.toArray(channels);
      if (channelNames.length === 0)
        throw new Error("No channels provided to Sync.Provider listener");
      const newNames = new Set(channelNames);
      const prevNames = new Set(uniqueNamesInMap(handlersRef.current));
      handlersRef.current.set(handler, newNames);
      if (!newNames.isSubsetOf(prevNames))
        handleError(
          async () => {
            await updateStreamer();
            onOpen?.();
          },
          `Failed to add ${strings.naturalLanguageJoin(channelNames)} to the Sync.Provider streamer`,
        );
      return () => {
        handlersRef.current.delete(handler);
        handleError(
          updateStreamer,
          `Failed to remove ${strings.naturalLanguageJoin(channelNames)} from the Sync.Provider streamer`,
        );
      };
    },
    [handleError, updateStreamer],
  );

  return <AddListenerContext value={addListener}>{children}</AddListenerContext>;
};

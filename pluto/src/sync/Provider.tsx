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
import { strings, toArray, unique } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useCallback, useRef } from "react";

import { useAsyncEffect } from "@/hooks";
import { Status } from "@/status";
import { Context } from "@/sync/Context";
import { type FrameHandler, type ListenerAdder } from "@/sync/types";
import { Synnax } from "@/synnax";

export interface ProviderProps extends PropsWithChildren {}

const uniqueNamesInMap = (map: Map<FrameHandler, Set<channel.Name>>): channel.Names =>
  unique.unique([...map.values()].flatMap((names) => [...names]));

export const Provider = (props: PropsWithChildren): ReactElement => {
  const client = Synnax.use();
  const handlersRef = useRef(new Map<FrameHandler, Set<channel.Name>>());
  const streamerRef = useRef<framer.Streamer>(null);
  const handleError = Status.useErrorHandler();

  useAsyncEffect(async () => {
    if (client == null) {
      streamerRef.current?.close();
      streamerRef.current = null;
      return;
    }
    try {
      streamerRef.current = await client.openStreamer(
        uniqueNamesInMap(handlersRef.current),
      );
    } catch (e) {
      handleError(e, "Failed to open streamer in Sync.Provider");
      return;
    }
    const observableStreamer = new framer.ObservableStreamer(streamerRef.current);
    observableStreamer.onChange((frame) => {
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
    });
    return async () => await observableStreamer.close();
  }, [client, handleError]);

  const updateStreamer = useCallback(async () => {
    try {
      await streamerRef.current?.update(uniqueNamesInMap(handlersRef.current));
    } catch (e) {
      if (StreamClosed.matches(e)) return;
      throw e;
    }
  }, []);

  const addListener: ListenerAdder = useCallback(
    ({ channels, handler }) => {
      const channelNames = toArray(channels);
      if (channelNames.length === 0)
        throw new Error("No channels provided to Sync.Provider listener");
      handlersRef.current.set(handler, new Set(channelNames));
      handleError(
        updateStreamer,
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

  return <Context {...props} value={addListener} />;
};

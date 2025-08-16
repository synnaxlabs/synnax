// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { framer, type Synnax } from "@synnaxlabs/client";
import { type AsyncDestructor, DataType, unique } from "@synnaxlabs/x";
import type z from "zod";

import { type Store, type StoreConfig } from "@/flux/aether/store";
import { type Status } from "@/status";

// This is a hack to ensure that deletions are processed before other changes, which
// ensures that modifications to things like relationships, which are a delete
// followed by a create, are processed in the correct order.
const channelNameSort = (a: string, b: string) => {
  const aHasDelete = a.includes("delete");
  const bHasDelete = b.includes("delete");
  if (aHasDelete && !bHasDelete) return -1;
  if (!aHasDelete && bHasDelete) return 1;
  return 0;
};

export interface StreamerArgs<ScopedStore extends Store> {
  handleError: Status.ErrorHandler;
  storeConfig: StoreConfig<ScopedStore>;
  client: Synnax;
  openStreamer: framer.StreamOpener;
  store: ScopedStore;
}

export const openStreamer = async <ScopedStore extends Store>({
  openStreamer: streamOpener,
  storeConfig,
  handleError,
  client,
  store,
}: StreamerArgs<ScopedStore>): Promise<AsyncDestructor> => {
  const configValues = Object.values(storeConfig);
  const channels = unique.unique(
    configValues.flatMap(({ listeners }) => listeners.map(({ channel }) => channel)),
  );
  const listenersForChannel = (name: string) =>
    configValues.flatMap(({ listeners }) =>
      listeners.filter(({ channel }) => channel === name),
    );
  const hardenedStreamer = await framer.HardenedStreamer.open(streamOpener, channels);
  const observableStreamer = new framer.ObservableStreamer(hardenedStreamer);
  const handleChange = (frame: framer.Frame) => {
    const namesInFrame = [...frame.uniqueNames];
    namesInFrame.sort(channelNameSort);
    namesInFrame.forEach((name) => {
      const series = frame.get(name);
      listenersForChannel(name).forEach(({ onChange, schema }) => {
        handleError(async () => {
          let parsed: z.output<typeof schema>[];
          if (!series.dataType.equals(DataType.JSON))
            parsed = Array.from(series).map((s) => schema.parse(s));
          else parsed = series.parseJSON(schema);
          if (series == null || client == null) return;
          for (const changed of parsed) await onChange({ changed, client, store });
        }, "Failed to handle streamer change");
      });
    });
  };
  observableStreamer.onChange(handleChange);
  return observableStreamer.close.bind(observableStreamer);
};

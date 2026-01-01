// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, ranger } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import { type flux } from "@/flux/aether";

export const FLUX_STORE_KEY = "ranges";

export interface FluxStore extends flux.UnaryStore<ranger.Key, ranger.Range> {}

interface FluxSubStore extends flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_LISTENER: flux.ChannelListener<FluxSubStore, typeof ranger.payloadZ> = {
  channel: ranger.SET_CHANNEL_NAME,
  schema: ranger.payloadZ,
  onChange: async ({ store, changed, client }) => {
    const range = client.ranges.sugarOne(changed);
    const prev = store.ranges.get(changed.key);
    let labels: label.Label[] | undefined;
    if (prev?.labels == null) labels = await range.retrieveLabels();
    let parent: ranger.Range | null = null;
    if (prev?.parent == null) parent = await range.retrieveParent();
    store.ranges.set(changed.key, (p) =>
      client.ranges.sugarOne({
        ...range.payload,
        labels: p?.labels ?? labels,
        parent: p?.parent ?? parent,
      }),
    );
  },
};

const DELETE_LISTENER: flux.ChannelListener<FluxSubStore, typeof ranger.keyZ> = {
  channel: ranger.DELETE_CHANNEL_NAME,
  schema: ranger.keyZ,
  onChange: ({ store, changed }) => store.ranges.delete(changed),
};

export const FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  ranger.Key,
  ranger.Range
> = {
  equal: (a, b) => deep.equal(a.payload, b.payload),
  listeners: [SET_LISTENER, DELETE_LISTENER],
};

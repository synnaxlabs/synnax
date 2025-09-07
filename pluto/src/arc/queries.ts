// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";

import { Flux } from "@/flux";
import { type List } from "@/list";

export interface FluxStore extends Flux.UnaryStore<arc.Key, arc.Arc> {}

export const FLUX_STORE_KEY = "arcs";

export interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

export interface ListParams extends List.PagerParams {
  keys?: arc.Key[];
}

export const useList = Flux.createList<ListParams, arc.Key, arc.Arc, SubStore>({
  name: "Arcs",
  retrieveCached: ({ store, params }) =>
    store.arcs.get((a) => {
      if (primitive.isNonZero(params.keys)) return params.keys.includes(a.key);
      return true;
    }),
  retrieve: async ({ client, params }) => await client.arcs.retrieve(params),
  retrieveByKey: async ({ client, key, store }) => {
    const cached = store.arcs.get(key);
    if (cached != null) return cached;
    const arc = await client.arcs.retrieve({ key });
    store.arcs.set(key, arc);
    return arc;
  },
  mountListeners: ({ store, onChange, onDelete }) => [
    store.arcs.onSet((arc) => onChange(arc.key, arc)),
    store.arcs.onDelete(onDelete),
  ],
});

export interface DeleteParams {
  keys: arc.Key[];
}

export const useDelete = Flux.createUpdate<undefined, DeleteParams, SubStore>({
  name: "Arcs",
  update: async ({ client, value }) => await client.arcs.delete(value.keys),
});

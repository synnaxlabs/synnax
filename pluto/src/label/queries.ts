// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology } from "@synnaxlabs/client";
import { primitive } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";
import { type Ontology } from "@/ontology";
import { state } from "@/state";

export const FLUX_STORE_KEY = "labels";
export const RESOURCE_NAME = "Label";
export const PLURAL_RESOURCE_NAME = "Labels";

export interface FluxStore extends Flux.UnaryStore<label.Key, label.Label> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
}

const SET_LABEL_LISTENER: Flux.ChannelListener<FluxSubStore, typeof label.labelZ> = {
  channel: label.SET_CHANNEL_NAME,
  schema: label.labelZ,
  onChange: ({ store, changed }) => store.labels.set(changed.key, changed),
};

const DELETE_LABEL_LISTENER: Flux.ChannelListener<FluxSubStore, typeof label.keyZ> = {
  channel: label.DELETE_CHANNEL_NAME,
  schema: label.keyZ,
  onChange: ({ store, changed }) => store.labels.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_LABEL_LISTENER, DELETE_LABEL_LISTENER],
};

export const matchRelationship = (rel: ontology.Relationship, id: ontology.ID) =>
  ontology.matchRelationship(rel, {
    from: id,
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  });

interface LabelsOfQuery {
  id: ontology.ID;
}

export const retrieveCachedLabelsOf = (store: FluxSubStore, id: ontology.ID) => {
  const keys = store.relationships
    .get((rel) => matchRelationship(rel, id))
    .map((rel) => rel.to.key);
  return store.labels.get(keys);
};

export const { useRetrieve: useRetrieveLabelsOf } = Flux.createRetrieve<
  LabelsOfQuery,
  label.Label[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieve: async ({ client, query: { id } }) =>
    await client.labels.retrieve({ for: id }),
  mountListeners: ({ client, store, query: { id }, onChange }) => [
    store.labels.onSet((label) => {
      onChange(
        state.skipNull((prev) => {
          const filtered = prev.filter((l) => l.key !== label.key);
          if (filtered.length === prev.length) return prev;
          return [...filtered, label];
        }),
      );
    }),
    store.labels.onDelete((key) =>
      onChange(state.skipNull((prev) => prev.filter((l) => l.key !== key))),
    ),
    store.relationships.onSet(async (rel) => {
      if (!matchRelationship(rel, id)) return;
      const { key } = rel.to;
      const l = await client.labels.retrieve({ key });
      store.labels.set(key, l);
      onChange(state.skipNull((prev) => [...prev.filter((l) => l.key !== key), l]));
    }),
    store.relationships.onDelete((relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      if (!matchRelationship(rel, id)) return;
      onChange(state.skipNull((prev) => prev.filter((l) => l.key !== rel.to.key)));
    }),
  ],
});

export interface ListQuery extends label.RetrieveMultipleParams {}

export const useList = Flux.createList<ListQuery, label.Key, label.Label, FluxSubStore>(
  {
    name: PLURAL_RESOURCE_NAME,
    retrieveCached: ({ query, store }) => {
      if (primitive.isNonZero(query.for) || primitive.isNonZero(query.searchTerm))
        return [];
      let labels = store.labels.list();
      if (query.keys != null && query.keys.length > 0) {
        const { keys } = query;
        labels = labels.filter((l) => keys.includes(l.key));
      }
      return labels;
    },
    retrieve: async ({ client, query }) => await client.labels.retrieve(query),
    retrieveByKey: async ({ client, key }) => await client.labels.retrieve({ key }),
    mountListeners: ({ store, onChange, onDelete, query: { keys } }) => {
      const keysSet = keys ? new Set(keys) : undefined;
      return [
        store.labels.onSet(async (label) => {
          if (keysSet != null && !keysSet.has(label.key)) return;
          onChange(label.key, label, { mode: "prepend" });
        }),
        store.labels.onDelete(async (key) => onDelete(key)),
      ];
    },
  },
);

interface FormQuery {
  key?: label.Key;
}

export const formSchema = label.labelZ.partial({ key: true });

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
  color: "#000000",
};

export const useForm = Flux.createForm<FormQuery, typeof formSchema, FluxSubStore>({
  name: RESOURCE_NAME,
  initialValues: INITIAL_VALUES,
  schema: formSchema,
  retrieve: async ({ client, query: { key }, reset }) => {
    if (key == null) return;
    reset(await client.labels.retrieve({ key }));
  },
  update: async ({ client, value, reset }) => {
    const updated = await client.labels.create(value());
    reset(updated);
  },
  mountListeners: ({ store, query: { key }, reset }) => [
    store.labels.onSet(async (label) => {
      if (key == null || label.key !== key) return;
      reset(label);
    }, key),
  ],
});

export type DeleteParams = label.Key | label.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data }) => {
    await client.labels.delete(data);
    return data;
  },
});

export interface RetrieveMultipleParams {
  keys: label.Key[];
}

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleParams,
  label.Label[],
  FluxSubStore
>({
  name: "Labels",
  retrieve: async ({ client, query: { keys }, store }) => {
    const cached = store.labels.get(keys);
    const missing = keys.filter((k) => !store.labels.has(k));
    if (missing.length === 0) return cached;
    const retrieved = await client.labels.retrieve({ keys: missing });
    store.labels.set(retrieved);
    return [...cached, ...retrieved];
  },
  mountListeners: ({ store, query: { keys }, onChange }) => {
    const keysSet = new Set(keys);
    return [
      store.labels.onSet(async (label) => {
        if (!keysSet.has(label.key)) return;
        onChange((prev) => {
          if (prev == null) return [label];
          return [...prev.filter((l) => l.key !== label.key), label];
        });
      }),
      store.labels.onDelete(async (key) => {
        keysSet.delete(key);
        onChange(state.skipNull((prev) => prev.filter((l) => l.key !== key)));
      }),
    ];
  },
});

export interface RetrieveMultipleParams {
  keys: label.Key[];
}

export const { useRetrieve: useRetrieveMultiple } = Flux.createRetrieve<
  RetrieveMultipleParams,
  label.Label[],
  SubStore
>({
  name: "Labels",
  retrieve: async ({ client, params: { keys }, store }) => {
    const cached = store.labels.get(keys);
    const missing = keys.filter((k) => !store.labels.has(k));
    if (missing.length === 0) return cached;
    const retrieved = await client.labels.retrieve({ keys: missing });
    store.labels.set(retrieved);
    return [...cached, ...retrieved];
  },
  mountListeners: ({ store, params: { keys }, onChange }) => {
    const keysSet = new Set(keys);
    return [
      store.labels.onSet(async (label) => {
        if (!keysSet.has(label.key)) return;
        onChange((prev) => [...prev.filter((l) => l.key !== label.key), label]);
      }),
      store.labels.onDelete(async (key) => {
        keysSet.delete(key);
        onChange((prev) => prev.filter((l) => l.key !== key));
      }),
    ];
  },
});

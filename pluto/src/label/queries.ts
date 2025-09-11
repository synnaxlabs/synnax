// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "labels";

export interface FluxStore extends Flux.UnaryStore<label.Key, label.Label> {}

interface SubStore extends Flux.Store {
  labels: FluxStore;
}

const SET_LABEL_LISTENER: Flux.ChannelListener<SubStore, typeof label.labelZ> = {
  channel: label.SET_CHANNEL_NAME,
  schema: label.labelZ,
  onChange: ({ store, changed }) => store.labels.set(changed.key, changed),
};

const DELETE_LABEL_LISTENER: Flux.ChannelListener<SubStore, typeof label.keyZ> = {
  channel: label.DELETE_CHANNEL_NAME,
  schema: label.keyZ,
  onChange: ({ store, changed }) => store.labels.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_LABEL_LISTENER, DELETE_LABEL_LISTENER],
};

export const matchRelationship = (rel: ontology.Relationship, id: ontology.ID) =>
  ontology.matchRelationship(rel, {
    from: id,
    type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  });

interface UseLabelsOfQueryParams {
  id: ontology.ID;
}

interface SubStore extends Flux.Store {
  labels: FluxStore;
  relationships: Flux.UnaryStore<string, ontology.Relationship>;
}

export const retrieveCachedLabelsOf = (store: SubStore, id: ontology.ID) => {
  const keys = store.relationships
    .get((rel) => matchRelationship(rel, id))
    .map((rel) => rel.to.key);
  return store.labels.get(keys);
};

export const { useRetrieve: useRetrieveLabelsOf } = Flux.createRetrieve<
  UseLabelsOfQueryParams,
  label.Label[],
  SubStore
>({
  name: "Labels",
  retrieve: async ({ client, params: { id } }) =>
    await client.labels.retrieve({ for: id }),
  mountListeners: ({ client, store, params: { id }, onChange }) => [
    store.labels.onSet((label) => {
      onChange((prev) => {
        const filtered = prev.filter((l) => l.key !== label.key);
        if (filtered.length === prev.length) return prev;
        return [...filtered, label];
      });
    }),
    store.labels.onDelete((key) =>
      onChange((prev) => prev.filter((l) => l.key !== key)),
    ),
    store.relationships.onSet(async (rel) => {
      if (!matchRelationship(rel, id)) return;
      const { key } = rel.to;
      const l = await client.labels.retrieve({ key });
      store.labels.set(key, l);
      onChange((prev) => [...prev.filter((l) => l.key !== key), l]);
    }),
    store.relationships.onDelete((relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      if (!matchRelationship(rel, id)) return;
      onChange((prev) => prev.filter((l) => l.key !== rel.to.key));
    }),
  ],
});

export interface ListParams extends label.MultiRetrieveArgs {}

export const useList = Flux.createList<ListParams, label.Key, label.Label, SubStore>({
  name: "Labels",
  retrieve: async ({ client, params }) => await client.labels.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.labels.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete, params: { keys } }) => {
    const keysSet = keys ? new Set(keys) : undefined;
    return [
      store.labels.onSet(async (label) => {
        if (keysSet != null && !keysSet.has(label.key)) return;
        onChange(label.key, label, { mode: "prepend" });
      }),
      store.labels.onDelete(async (key) => onDelete(key)),
    ];
  },
});

interface FormParams {
  key?: label.Key;
}

export const formSchema = label.labelZ.partial({ key: true });

export const useForm = Flux.createForm<FormParams, typeof formSchema, SubStore>({
  name: "Label",
  initialValues: {
    name: "",
    color: "#000000",
  },
  schema: formSchema,
  retrieve: async ({ client, params: { key }, reset }) => {
    if (key == null) return;
    reset(await client.labels.retrieve({ key }));
  },
  update: async ({ client, value, reset }) => {
    const updated = await client.labels.create(value());
    reset(updated);
  },
  mountListeners: ({ store, params: { key }, reset }) => [
    store.labels.onSet(async (label) => {
      if (key == null || label.key !== key) return;
      reset(label);
    }, key),
  ],
});

export interface UseDeleteArgs {
  key: label.Key;
}

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs>({
  name: "Label",
  update: async ({ client, value }) => await client.labels.delete(value.key),
});

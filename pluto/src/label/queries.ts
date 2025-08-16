// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology } from "@synnaxlabs/client";
import { z } from "zod";

import { Flux } from "@/flux";

export interface FluxStore extends Flux.UnaryStore<label.Key, label.Label> {}

interface SubStore extends Flux.Store {
  labels: FluxStore;
}

const SET_LABEL_LISTENER: Flux.ChannelListener<SubStore, typeof label.labelZ> = {
  channel: label.SET_CHANNEL_NAME,
  schema: label.labelZ,
  onChange: async ({ store, changed }) => store.labels.set(changed.key, changed),
};

const DELETE_LABEL_LISTENER: Flux.ChannelListener<SubStore, typeof label.keyZ> = {
  channel: label.DELETE_CHANNEL_NAME,
  schema: label.keyZ,
  onChange: async ({ store, changed }) => store.labels.delete(changed),
};

export const STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
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

export const retrieveLabelsOf = Flux.createRetrieve<
  UseLabelsOfQueryParams,
  label.Label[],
  SubStore
>({
  name: "Labels",
  retrieve: async ({ client, params: { id } }) =>
    await client.labels.retrieve({ for: id }),
  mountListeners: ({ client, store, params: { id }, onChange }) => [
    store.labels.onSet(async (label) => {
      onChange((prev) => {
        const filtered = prev.filter((l) => l.key !== label.key);
        if (filtered.length === prev.length) return prev;
        return [...filtered, label];
      });
    }),
    store.labels.onDelete(async (key) => {
      onChange((prev) => prev.filter((l) => l.key !== key));
    }),
    store.relationships.onSet(async (rel) => {
      if (!matchRelationship(rel, id)) return;
      const { key } = rel.to;
      const l = await client.labels.retrieve({ key });
      store.labels.set(key, l);
      onChange((prev) => [...prev.filter((l) => l.key !== key), l]);
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      if (!matchRelationship(rel, id)) return;
      onChange((prev) => prev.filter((l) => l.key !== rel.to.key));
    }),
  ],
});

export const labelsOfFormSchema = z.object({ labels: z.array(label.keyZ) });

export const useLabelsOfForm = Flux.createForm<
  UseLabelsOfQueryParams,
  typeof labelsOfFormSchema,
  SubStore
>({
  name: "Labels",
  schema: labelsOfFormSchema,
  initialValues: { labels: [] },
  retrieve: async ({ client, params: { id } }) => {
    if (id == null) return null;
    const labels = await client.labels.retrieve({ for: id });
    return { labels: labels.map((l) => l.key) };
  },
  update: async ({ client, value, params: { id } }) => {
    await client.labels.label(id, value.labels, { replace: true });
  },
  mountListeners: ({ client, store, params: { id }, onChange }) => [
    store.relationships.onSet(async (rel) => {
      if (!matchRelationship(rel, id)) return;
      const { key } = rel.to;
      const l = await client.labels.retrieve({ key });
      store.labels.set(key, l);
      onChange((prev) => {
        if (prev == null) return { labels: [l.key] };
        return { labels: [...prev.labels.filter((l) => l !== key), l.key] };
      });
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      if (!matchRelationship(rel, id)) return;
      onChange((prev) => {
        if (prev == null) return { labels: [] };
        return { labels: prev.labels.filter((l) => l !== rel.to.key) };
      });
    }),
  ],
});

export interface ListParams extends label.MultiRetrieveArgs {}

export const useList = Flux.createList<ListParams, label.Key, label.Label, SubStore>({
  name: "Labels",
  retrieve: async ({ client, params }) => await client.labels.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.labels.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.labels.onSet(async (label) => {
      onChange(label.key, label, { mode: "prepend" });
    }),
    store.labels.onDelete(async (key) => onDelete(key)),
  ],
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
  retrieve: async ({ client, params: { key } }) => {
    if (key == null) return null;
    const label = await client.labels.retrieve({ key });
    return label;
  },
  update: async ({ client, value, onChange }) =>
    onChange(await client.labels.create(value)),
  mountListeners: ({ store, params, onChange }) => [
    store.labels.onSet(async (label) => {
      if (params.key == null || label.key !== params.key) return;
      onChange(label);
    }, params.key),
  ],
});

export interface DeleteParams {
  key: label.Key;
}

export const useDelete = Flux.createUpdate<DeleteParams, void>({
  name: "Label",
  update: async ({ client, params: { key } }) => await client.labels.delete(key),
}).useDirect;

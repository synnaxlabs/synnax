// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { label, ontology, status, TimeStamp } from "@synnaxlabs/client";
import { array, primitive, uuid } from "@synnaxlabs/x";
import { useEffect } from "react";
import type z from "zod";

import { type Flux } from "@/flux";
import { DELETE_VERBS, SET_VERBS } from "@/flux/external";
import { createForm } from "@/flux/form";
import { createList } from "@/flux/list";
import { useStore } from "@/flux/Provider";
import { createRetrieve, type RetrieveParams } from "@/flux/retrieve";
import { createUpdate } from "@/flux/update";
import { Label } from "@/label";
import { state } from "@/state";

export const FLUX_STORE_KEY = "statuses";
const RESOURCE_NAME = "Status";
const PLURAL_RESOURCE_NAME = "Statuses";

export interface FluxStore extends Flux.UnaryStore<status.Key, status.Status> {}

interface FluxSubStore extends Label.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_STATUS_LISTENER: Flux.ChannelListener<FluxSubStore, typeof status.statusZ> = {
  channel: status.SET_CHANNEL_NAME,
  schema: status.statusZ,
  onChange: ({ store, changed }) => store.statuses.set(changed.key, changed),
};

const DELETE_STATUS_LISTENER: Flux.ChannelListener<FluxSubStore, typeof status.keyZ> = {
  channel: status.DELETE_CHANNEL_NAME,
  schema: status.keyZ,
  onChange: ({ store, changed }) => store.statuses.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_STATUS_LISTENER, DELETE_STATUS_LISTENER],
};

export interface ListParams extends status.MultiRetrieveArgs {}

export const useList = createList<ListParams, status.Key, status.Status, FluxSubStore>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.statuses.list(),
  retrieve: async ({ client, query }) => await client.statuses.retrieve(query),
  retrieveByKey: async ({ client, key }) => await client.statuses.retrieve({ key }),
  mountListeners: ({ store, onChange, onDelete, query: { keys } }) => {
    const keysSet = keys ? new Set(keys) : undefined;
    return [
      store.statuses.onSet(async (status) => {
        if (keysSet != null && !keysSet.has(status.key)) return;
        onChange(status.key, status, { mode: "prepend" });
      }),
      store.statuses.onDelete(async (key) => onDelete(key)),
    ];
  },
});

export const { useUpdate: useDelete } = createUpdate<
  status.Key | status.Key[],
  FluxSubStore
>({
  name: RESOURCE_NAME,
  verbs: DELETE_VERBS,
  update: async ({ client, data }) => {
    await client.statuses.delete(data);
    return data;
  },
});

export interface SetParams {
  statuses: status.New | status.New[];
  parent?: ontology.ID;
}

export const { useUpdate: useSet } = createUpdate<SetParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: SET_VERBS,
  update: async ({ client, data, data: { statuses, parent } }) => {
    if (Array.isArray(statuses)) await client.statuses.set(statuses, { parent });
    else await client.statuses.set(statuses, { parent });
    return data;
  },
});

export interface RetrieveQuery extends status.SingleRetrieveArgs {}

const BASE_QUERY: Pick<RetrieveQuery, "includeLabels"> = {
  includeLabels: true,
};

const retrieveSingle = async ({
  store,
  client,
  query,
}: RetrieveParams<status.SingleRetrieveArgs, FluxSubStore>) => {
  const cached = store.statuses.get(query.key);
  if (cached != null) {
    cached.labels = Label.retrieveCachedLabelsOf(store, status.ontologyID(query.key));
    return cached;
  }
  const res = await client.statuses.retrieve({ ...BASE_QUERY, ...query });
  store.labels.set(res.labels);
  store.statuses.set(query.key, res);
  return res;
};

export const { useRetrieve } = createRetrieve<
  RetrieveQuery,
  status.Status,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, client, onChange }) => [
    store.statuses.onSet(onChange, key),
    store.relationships.onSet(async (rel) => {
      const isLabelChange = Label.matchRelationship(rel, status.ontologyID(key));
      if (!isLabelChange) return;
      const l = await Label.retrieveSingle({
        store,
        query: { key: rel.to.key },
        client,
      });
      onChange(
        state.skipNull((p) => ({
          ...p,
          labels: array.upsertKeyed(p.labels, l),
        })),
      );
    }),
    store.relationships.onDelete(async (relKey) => {
      const rel = ontology.relationshipZ.parse(relKey);
      const otgID = status.ontologyID(key);
      const isLabelChange = Label.matchRelationship(rel, otgID);
      if (!isLabelChange) return;
      onChange(
        state.skipNull((p) => ({
          ...p,
          labels: array.removeKeyed(p.labels, rel.to.key),
        })),
      );
    }),
  ],
});

export const useSetSynchronizer = (onSet: (status: status.Status) => void): void => {
  const store = useStore<FluxSubStore>();
  useEffect(() => store.statuses.onSet(onSet), [store]);
};

export const formSchema = status.statusZ.omit({ labels: true }).safeExtend({
  labels: label.keyZ.array().optional(),
});

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  key: "",
  variant: "success",
  message: "",
  time: TimeStamp.now(),
  name: "",
  description: "",
  labels: [],
};

export const useForm = createForm<
  Partial<RetrieveQuery>,
  typeof formSchema,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ reset, ...args }) => {
    const {
      query: { key },
      client,
    } = args;
    if (primitive.isZero(key)) return;
    const stat = await retrieveSingle({ ...args, query: { key } });
    const labels = await client.labels.retrieve({ for: status.ontologyID(stat.key) });
    reset({ ...stat, labels: labels.map((l) => l.key) });
  },
  update: async ({ client, value, store, rollbacks, set }) => {
    set("time", TimeStamp.now());
    const v = value();
    if (primitive.isZero(v.key)) v.key = uuid.create();
    const { labels: labelKeys, ...rest } = v;
    const res = await client.statuses.set(rest);
    if (labelKeys != null) {
      const labels = await Label.setLabelsFor({
        store,
        client,
        rollbacks,
        data: { id: status.ontologyID(res.key), labels: labelKeys },
      });
      res.labels = labels;
    }
    store.statuses.set(res);
    set("key", res.key);
  },
  mountListeners: ({ store, query: { key }, set, get }) => {
    const getKey = () => get<label.Key>("key").value;
    return [
      store.statuses.onSet((v) => {
        if (getKey() != v.key) return;
        set("key", v.key);
        set("message", v.message);
        set("time", v.time);
        set("name", v.name);
        set("description", v.description);
      }, key),
      store.relationships.onSet(async (rel) => {
        const key = getKey();
        const isLabelChange = Label.matchRelationship(rel, status.ontologyID(key));
        if (!isLabelChange) return;
        set("labels", array.upsert(get<string[]>("labels").value, rel.to.key));
      }),
      store.relationships.onDelete(async (relKey) => {
        const key = getKey();
        const rel = ontology.relationshipZ.parse(relKey);
        const isLabelChange = Label.matchRelationship(rel, status.ontologyID(key));
        if (!isLabelChange) return;
        return set("labels", array.remove(get<string[]>("labels").value, rel.to.key));
      }),
    ];
  },
});

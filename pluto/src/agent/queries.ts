// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { agent } from "@synnaxlabs/client";
import { array, primitive } from "@synnaxlabs/x";
import type z from "zod";

import { Flux } from "@/flux";
import { type List } from "@/list";
import { Ontology } from "@/ontology";
import { type Status } from "@/status";

export interface FluxStore extends Flux.UnaryStore<agent.Key, agent.Agent> {}

export const FLUX_STORE_KEY = "agents";
const RESOURCE_NAME = "Agent";
const PLURAL_RESOURCE_NAME = "Agents";

export interface FluxSubStore extends Status.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_LISTENER: Flux.ChannelListener<FluxSubStore, typeof agent.agentZ> = {
  channel: agent.SET_CHANNEL_NAME,
  schema: agent.agentZ,
  onChange: ({ store, changed }) => store.agents.set(changed.key, changed),
};

const DELETE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof agent.keyZ> = {
  channel: agent.DELETE_CHANNEL_NAME,
  schema: agent.keyZ,
  onChange: ({ store, changed }) => store.agents.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  agent.Key,
  agent.Agent
> = { listeners: [SET_LISTENER, DELETE_LISTENER] };

export interface RetrieveQuery {
  key: agent.Key;
}

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.agents.get(key);
  if (cached != null) return cached;
  const a = await client.agents.retrieve(key);
  store.agents.set(key, a);
  return a;
};

export interface ListQuery extends List.PagerParams {
  keys?: agent.Key[];
}

export const useList = Flux.createList<ListQuery, agent.Key, agent.Agent, FluxSubStore>(
  {
    name: PLURAL_RESOURCE_NAME,
    retrieveCached: ({ store, query }) =>
      store.agents.get((a) => {
        if (primitive.isNonZero(query.keys)) return query.keys.includes(a.key);
        return true;
      }),
    retrieve: async ({ client, query }) => await client.agents.retrieve(query),
    retrieveByKey: async ({ client, key, store }) => {
      const cached = store.agents.get(key);
      if (cached != null) return cached;
      const a = await client.agents.retrieve(key);
      store.agents.set(key, a);
      return a;
    },
    mountListeners: ({ store, onChange, onDelete }) => [
      store.agents.onSet((a) => onChange(a.key, a)),
      store.agents.onDelete(onDelete),
    ],
  },
);

export const { useUpdate: useDelete } = Flux.createUpdate<
  agent.Key | agent.Key[],
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = agent.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.agents.delete(keys));
    await client.agents.delete(keys);
    return data;
  },
});

export const { useUpdate: useCreate } = Flux.createUpdate<
  agent.New,
  FluxSubStore,
  agent.Agent
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const created = await client.agents.create(data);
    rollbacks.push(store.agents.set(created.key, created));
    return created;
  },
});

export interface SendParams {
  key: agent.Key;
  content: string;
}

export const { useUpdate: useSend } = Flux.createUpdate<
  SendParams,
  FluxSubStore,
  agent.Agent
>({
  name: RESOURCE_NAME,
  verbs: { present: "send to", past: "sent to", participle: "sending to" },
  update: async ({ client, data, store }) => {
    const updated = await client.agents.send(data.key, data.content);
    store.agents.set(data.key, updated);
    return updated;
  },
});

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  agent.Agent,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query, onChange }) => {
    if (!("key" in query) || primitive.isZero(query.key)) return [];
    return [store.agents.onSet(onChange, query.key)];
  },
});

export const formSchema = agent.agentZ.partial({ key: true });

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
  messages: [],
  arcKey: "",
  state: "stopped",
};

export const useForm = Flux.createForm<
  Partial<RetrieveQuery>,
  typeof formSchema,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  schema: formSchema,
  initialValues: INITIAL_VALUES,
  retrieve: async ({ client, store, query: { key }, reset }) => {
    if (key == null) return;
    const res = await retrieveSingle({ client, store, query: { key } });
    reset(res);
  },
  update: async ({ client, value, set }) => {
    const res = await client.agents.create(value());
    set("key", res.key);
  },
});

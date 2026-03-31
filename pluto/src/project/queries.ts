// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, project } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import type z from "zod";

import { type policy } from "@/access/policy/aether";
import { type role } from "@/access/role/aether";
import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "projects";
const RESOURCE_NAME = "project";
const PLURAL_RESOURCE_NAME = "projects";

export interface FluxStore extends Flux.UnaryStore<project.Key, project.Project> {}

interface FluxSubStore extends Flux.Store, role.FluxSubStore, policy.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_PROJECT_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof project.projectZ
> = {
  channel: project.SET_CHANNEL_NAME,
  schema: project.projectZ,
  onChange: ({ store, changed }) => store.projects.set(changed.key, changed),
};

const DELETE_PROJECT_LISTENER: Flux.ChannelListener<FluxSubStore, typeof project.keyZ> =
  {
    channel: project.DELETE_CHANNEL_NAME,
    schema: project.keyZ,
    onChange: ({ store, changed }) => store.projects.delete(changed),
  };

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_PROJECT_LISTENER, DELETE_PROJECT_LISTENER],
};

export interface RetrieveQuery {
  key: project.Key;
}

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.projects.get(key);
  if (cached != null) return cached;
  const p = await client.projects.retrieve(key);
  store.projects.set(p.key, p);
  return p;
};

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  project.Project,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.projects.onSet(onChange, key),
  ],
});

export interface ListParams {
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<
  ListParams,
  project.Key,
  project.Project,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.projects.list(),
  retrieve: async ({ client, query }) => await client.projects.retrieve(query),
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.projects.onSet((p) => onChange(p.key, p)),
    store.projects.onDelete(onDelete),
  ],
});

export type DeleteParams = project.Key | project.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = project.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.projects.delete(keys));
    await client.projects.delete(keys);
    return data;
  },
});

export interface RenameParams {
  key: project.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    await client.projects.rename(key, name);
    rollbacks.push(Flux.partialUpdate(store.projects, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, project.ontologyID(key), name));
    return data;
  },
});

export interface RetrieveGroupQuery {}

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  RetrieveGroupQuery,
  ontology.ID | undefined,
  FluxSubStore
>({
  name: "Project Group",
  retrieve: async ({ client, store }) => {
    const rels = store.relationships.get((rel) =>
      ontology.matchRelationship(rel, {
        from: ontology.ROOT_ID,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      }),
    );
    const groups = store.resources.get(rels.map((rel) => ontology.idToString(rel.to)));
    const cachedRes = groups.find((group) => group.name === "Projects");
    if (cachedRes != null) return cachedRes.id;
    const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    store.resources.set(res);
    return res.find((r) => r.name === "Projects")?.id;
  },
});

export const formSchema = project.projectZ.partial({ key: true });

const INITIAL_VALUES: z.infer<typeof formSchema> = {
  name: "",
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
    const res = await client.projects.create(value());
    set("key", res.key);
  },
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, project } from "@synnaxlabs/client";
import { array, type record } from "@synnaxlabs/x";
import type z from "zod";

import { type policy } from "@/access/policy/aether";
import { type role } from "@/access/role/aether";
import { Flux } from "@/flux";
import { Ontology } from "@/ontology";
import { state } from "@/state";

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

export type RetrieveQuery = {
  key: project.Key;
};

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.projects.get(key);
  if (cached != null) return cached;
  const project = await client.projects.retrieve(key);
  store.projects.set(project.key, project);
  return project;
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

export type ListParams = Pick<
  project.RetrieveRequest,
  "keys" | "offset" | "limit"
>;

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
    store.projects.onSet((project) => onChange(project.key, project)),
    store.projects.onDelete(onDelete),
  ],
});

export type DeleteParams = project.Key | project.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks, onOptimisticComplete }) => {
    const keys = array.toArray(data);
    const ids = project.ontologyID(keys);
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.projects.delete(keys));
    await onOptimisticComplete(data);
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

export type RetrieveGroupQuery = Record<string, never>;

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
  layout: {},
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

export interface SaveLayoutParams extends project.SetLayoutArgs {}

const LAYOUT_RESOURCE_NAME = "project layout";

export const { useUpdate: useSaveLayout } = Flux.createUpdate<
  SaveLayoutParams,
  FluxSubStore
>({
  name: LAYOUT_RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks, onOptimisticComplete }) => {
    const { key, layout } = data;
    rollbacks.push(
      store.projects.set(
        key,
        state.skipUndefined((p) => ({ ...p, layout })),
      ),
    );
    await onOptimisticComplete(data);
    await client.projects.setLayout(key, layout);
    return data;
  },
});

export type RetrieveChildrenQuery = {
  resourceID?: ontology.ID;
  types: ontology.ResourceType[];
};

const collectChildren = async (
  client: Flux.RetrieveParams<RetrieveChildrenQuery, FluxSubStore>["client"],
  parentID: ontology.ID,
  types: ontology.ResourceType[],
  exclude?: string,
): Promise<record.KeyedNamed[]> => {
  const children = await client.ontology.retrieveChildren(parentID, {
    types: [...types, "group"],
  });
  const results: record.KeyedNamed[] = [];
  for (const child of children)
    if (types.includes(child.id.type) && child.id.key !== exclude)
      results.push({ key: child.id.key, name: child.name });
    else if (child.id.type === "group")
      results.push(...(await collectChildren(client, child.id, types, exclude)));
  return results;
};

const findProjectAncestor = async (
  client: Flux.RetrieveParams<RetrieveChildrenQuery, FluxSubStore>["client"],
  resourceID: ontology.ID,
): Promise<ontology.ID | null> => {
  const parents = await client.ontology.retrieveParents(resourceID);
  for (const parent of parents) {
    if (parent.id.type === "project") return parent.id;
    if (parent.id.type === "group") return await findProjectAncestor(client, parent.id);
  }
  return null;
};

const retrieveChildrenImpl = async ({
  client,
  query: { resourceID, types },
}: Flux.RetrieveParams<RetrieveChildrenQuery, FluxSubStore>): Promise<
  record.KeyedNamed[]
> => {
  if (resourceID == null) return [];
  const projectID = await findProjectAncestor(client, resourceID);
  if (projectID == null) return [];
  return await collectChildren(client, projectID, types, resourceID.key);
};

export const { useRetrieve: useRetrieveChildren } = Flux.createRetrieve<
  RetrieveChildrenQuery,
  record.KeyedNamed[],
  FluxSubStore
>({
  name: "project children",
  retrieve: retrieveChildrenImpl,
  mountListeners: ({ store, onChange }) => [
    store.relationships.onSet(() => onChange(undefined)),
    store.resources.onSet(() => onChange(undefined)),
  ],
});

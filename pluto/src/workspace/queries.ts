// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology, workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";
import type z from "zod";

import { Access } from "@/access";
import { type policy } from "@/access/policy/aether";
import { type role } from "@/access/role/aether";
import { Flux } from "@/flux";
import { Ontology } from "@/ontology";
import { state } from "@/state";

export const FLUX_STORE_KEY = "workspaces";
const RESOURCE_NAME = "Workspace";
const PLURAL_RESOURCE_NAME = "Workspaces";

export interface FluxStore
  extends Flux.UnaryStore<workspace.Key, workspace.Workspace> {}

interface FluxSubStore extends Flux.Store, role.FluxSubStore, policy.FluxSubStore {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_WORKSPACE_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof workspace.workspaceZ
> = {
  channel: workspace.SET_CHANNEL_NAME,
  schema: workspace.workspaceZ,
  onChange: ({ store, changed }) => {
    store.workspaces.set(changed.key, changed);
  },
};

const DELETE_WORKSPACE_LISTENER: Flux.ChannelListener<
  FluxSubStore,
  typeof workspace.keyZ
> = {
  channel: workspace.DELETE_CHANNEL_NAME,
  schema: workspace.keyZ,
  onChange: ({ store, changed }) => store.workspaces.delete(changed),
};

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore> = {
  listeners: [SET_WORKSPACE_LISTENER, DELETE_WORKSPACE_LISTENER],
};

export interface RetrieveQuery {
  key: workspace.Key;
}

const retrieveSingle = async ({
  client,
  query: { key },
  store,
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.workspaces.get(key);
  if (cached != null) return cached;
  const workspace = await client.workspaces.retrieve(key);
  store.workspaces.set(workspace.key, workspace);
  return workspace;
};

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveQuery,
  workspace.Workspace,
  FluxSubStore
>({
  name: "Workspace",
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.workspaces.onSet(onChange, key),
  ],
});

export interface ListParams {
  offset?: number;
  limit?: number;
}

export const useList = Flux.createList<
  ListParams,
  workspace.Key,
  workspace.Workspace,
  FluxSubStore
>({
  name: PLURAL_RESOURCE_NAME,
  retrieveCached: ({ store }) => store.workspaces.list(),
  retrieve: async ({ client, query }) => await client.workspaces.retrieve(query),
  retrieveByKey: async ({ key, ...rest }) =>
    await retrieveSingle({ ...rest, query: { key } }),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.workspaces.onSet((workspace) => onChange(workspace.key, workspace)),
    store.workspaces.onDelete(onDelete),
  ],
});

export type DeleteParams = workspace.Key | workspace.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<DeleteParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const keys = array.toArray(data);
    const ids = keys.map((key) => workspace.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    rollbacks.push(store.resources.delete(keys));
    rollbacks.push(store.workspaces.delete(keys));
    await client.workspaces.delete(keys);
    return data;
  },
});

export interface RenameParams {
  key: workspace.Key;
  name: string;
}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    await client.workspaces.rename(key, name);
    rollbacks.push(Flux.partialUpdate(store.workspaces, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, workspace.ontologyID(key), name));
    return data;
  },
});

export interface RetrieveGroupQuery {}

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  RetrieveGroupQuery,
  ontology.ID | undefined,
  FluxSubStore
>({
  name: "Workspace Group",
  retrieve: async ({ client, store }) => {
    const rels = store.relationships.get((rel) =>
      ontology.matchRelationship(rel, {
        from: ontology.ROOT_ID,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      }),
    );
    const groups = store.resources.get(rels.map((rel) => ontology.idToString(rel.to)));
    const cachedRes = groups.find((group) => group.name === "Workspaces");
    if (cachedRes != null) return cachedRes.id;
    const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    store.resources.set(res);
    return res.find((r) => r.name === "Workspaces")?.id;
  },
});

export const formSchema = workspace.workspaceZ.partial({ key: true });

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
    const res = await client.workspaces.create(value());
    set("key", res.key);
  },
});

export interface SaveLayoutParams extends workspace.SetLayoutArgs {}

const LAYOUT_RESOURCE_NAME = "workspace layout";

export const { useUpdate: useSaveLayout } = Flux.createUpdate<
  SaveLayoutParams,
  FluxSubStore
>({
  name: LAYOUT_RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store, rollbacks }) => {
    const { key, layout } = data;
    if (!editAccessGranted({ key, store, client })) return data;
    rollbacks.push(
      store.workspaces.set(
        key,
        state.skipNull((p) => ({ ...p, layout })),
      ),
    );
    await client.workspaces.setLayout(key, layout);
    return data;
  },
});

const editAccessQuery = (
  key: workspace.Key | workspace.Key[] = "",
): Access.PermissionsQuery => ({
  objects: workspace.ontologyID(key),
  actions: ["retrieve", "create", "update"],
});

export const useEditAccessGranted = (key: workspace.Key | workspace.Key[]) =>
  Access.useGranted(editAccessQuery(key));

export const editAccessGranted = ({
  key,
  ...rest
}: Access.IsGrantedExtensionParams & { key?: workspace.Key | workspace.Key[] }) =>
  Access.isGranted({ ...rest, query: editAccessQuery(key) });

const viewAccessQuery = (
  key: workspace.Key | workspace.Key[] = "",
): Access.PermissionsQuery => ({
  objects: workspace.ontologyID(key),
  actions: ["retrieve"],
});

export const viewAccessGranted = ({
  key,
  ...rest
}: Access.IsGrantedExtensionParams & { key?: workspace.Key | workspace.Key[] }) =>
  Access.isGranted({ ...rest, query: viewAccessQuery(key) });

export const useViewAccessGranted = (key: workspace.Key | workspace.Key[]) =>
  Access.useGranted(viewAccessQuery(key ?? ""));

const deleteAccessQuery = (
  key: workspace.Key | workspace.Key[] = "",
): Access.PermissionsQuery => ({
  objects: workspace.ontologyID(key),
  actions: ["retrieve", "create", "update", "delete"],
});

export const useDeleteAccessGranted = (key: workspace.Key | workspace.Key[]) =>
  Access.useGranted(deleteAccessQuery(key));

export const deleteAccessGranted = ({
  key,
  ...rest
}: Access.IsGrantedExtensionParams & { key?: workspace.Key }) =>
  Access.isGranted({ ...rest, query: deleteAccessQuery(key) });

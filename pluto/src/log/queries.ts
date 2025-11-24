// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type workspace } from "@synnaxlabs/client";
import { array } from "@synnaxlabs/x";

import { Access } from "@/access";
import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<FluxSubStore, log.Key, log.Log> =
  {
    listeners: [],
  };

export const FLUX_STORE_KEY = "logs";
const RESOURCE_NAME = "Log";

export interface FluxStore extends Flux.UnaryStore<log.Key, log.Log> {}

export type UseDeleteArgs = log.Params;

interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

export type RetrieveQuery = log.RetrieveSingleParams;

export const retrieveSingle = async ({
  store,
  client,
  query: { key },
}: Flux.RetrieveParams<RetrieveQuery, FluxSubStore>) => {
  const cached = store.logs.get(key);
  if (cached != null) return cached;
  const l = await client.workspaces.logs.retrieve({ key });
  store.logs.set(l);
  return l;
};

export const { useRetrieve, useRetrieveObservable } = Flux.createRetrieve<
  RetrieveQuery,
  log.Log,
  FluxSubStore
>({
  name: RESOURCE_NAME,
  retrieve: retrieveSingle,
  mountListeners: ({ store, query: { key }, onChange }) => [
    store.logs.onSet(onChange, key),
  ],
});

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.DELETE_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const keys = array.toArray(data);
    const ids = keys.map((key) => log.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.push(store.relationships.delete(relFilter));
    await client.workspaces.logs.delete(data);
    return data;
  },
});

export interface CreateParams extends log.New {
  workspace: workspace.Key;
}

export interface CreateOutput extends log.Log {
  workspace: workspace.Key;
}

export const { useUpdate: useCreate } = Flux.createUpdate<
  CreateParams,
  FluxSubStore,
  CreateOutput
>({
  name: RESOURCE_NAME,
  verbs: Flux.CREATE_VERBS,
  update: async ({ client, data, store }) => {
    const { workspace, ...rest } = data;
    const l = await client.workspaces.logs.create(workspace, rest);
    store.logs.set(l.key, l);
    return { ...l, workspace };
  },
});

export interface RenameParams extends Pick<log.Log, "key" | "name"> {}

export const { useUpdate: useRename } = Flux.createUpdate<RenameParams, FluxSubStore>({
  name: RESOURCE_NAME,
  verbs: Flux.RENAME_VERBS,
  update: async ({ client, data, rollbacks, store }) => {
    const { key, name } = data;
    rollbacks.push(Flux.partialUpdate(store.logs, key, { name }));
    rollbacks.push(Ontology.renameFluxResource(store, log.ontologyID(key), name));
    await client.workspaces.logs.rename(key, data.name);
    return data;
  },
});

const editAccessQuery = (key: log.Key | log.Key[] = ""): Access.PermissionsQuery => ({
  objects: log.ontologyID(key),
  actions: ["retrieve", "create", "update"],
});

export const useEditAccessGranted = (key?: log.Key | log.Key[]) =>
  Access.useGranted(editAccessQuery(key));

export const editAccessGranted = ({
  key,
  ...rest
}: Access.IsGrantedExtensionParams & { key?: log.Key | log.Key[] }) =>
  Access.isGranted({ ...rest, query: editAccessQuery(key) });

const viewAccessQuery = (key: log.Key | log.Key[] = ""): Access.PermissionsQuery => ({
  objects: log.ontologyID(key),
  actions: ["retrieve"],
});

export const viewAccessGranted = ({
  key,
  ...rest
}: Access.IsGrantedExtensionParams & { key?: log.Key | log.Key[] }) =>
  Access.isGranted({ ...rest, query: viewAccessQuery(key) });

export const useViewAccessGranted = (key?: log.Key | log.Key[]) =>
  Access.useGranted(viewAccessQuery(key ?? ""));

const deleteAccessQuery = (key: log.Key | log.Key[] = ""): Access.PermissionsQuery => ({
  objects: log.ontologyID(key),
  actions: ["retrieve", "create", "update", "delete"],
});

export const useDeleteAccessGranted = (key?: log.Key | log.Key[]) =>
  Access.useGranted(deleteAccessQuery(key));

export const deleteAccessGranted = ({
  key,
  ...rest
}: Access.IsGrantedExtensionParams & { key?: log.Key }) =>
  Access.isGranted({ ...rest, query: deleteAccessQuery(key) });

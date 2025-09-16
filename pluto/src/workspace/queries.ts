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

import { Flux } from "@/flux";
import { Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "workspaces";

export interface FluxStore
  extends Flux.UnaryStore<workspace.Key, workspace.Workspace> {}

interface FluxSubStore extends Flux.Store {
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

export interface RetrieveParams {
  key: workspace.Key;
}

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveParams,
  workspace.Workspace,
  FluxSubStore
>({
  name: "Workspace",
  retrieve: ({ params, client }) => client.workspaces.retrieve(params.key),
  mountListeners: ({ store, params, onChange }) => [
    store.workspaces.onSet(onChange, params.key),
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
  name: "Workspace",
  retrieve: async ({ client, params }) => await client.workspaces.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.workspaces.retrieve(key),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.workspaces.onSet((workspace) => onChange(workspace.key, workspace)),
    store.workspaces.onDelete(onDelete),
  ],
});

export type UseDeleteArgs = workspace.Key | workspace.Key[];

export const { useUpdate: useDelete } = Flux.createUpdate<UseDeleteArgs, FluxSubStore>({
  name: "Workspace",
  update: async ({ client, value, store, rollbacks }) => {
    const keys = array.toArray(value);
    const ids = keys.map((key) => workspace.ontologyID(key));
    const relFilter = Ontology.filterRelationshipsThatHaveIDs(ids);
    rollbacks.add(store.relationships.delete(relFilter));
    rollbacks.add(store.resources.delete(keys));
    rollbacks.add(store.workspaces.delete(keys));
    await client.workspaces.delete(keys);
    return value;
  },
});

export interface UseRetrieveGroupArgs {}

export const { useRetrieve: useRetrieveGroupID } = Flux.createRetrieve<
  UseRetrieveGroupArgs,
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

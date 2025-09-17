// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { workspace } from "@synnaxlabs/client";

import { Flux } from "@/flux";

export const FLUX_STORE_KEY = "workspaces";

export interface FluxStore
  extends Flux.UnaryStore<workspace.Key, workspace.Workspace> {}

interface SubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
}

const SET_WORKSPACE_LISTENER: Flux.ChannelListener<
  SubStore,
  typeof workspace.workspaceZ
> = {
  channel: workspace.SET_CHANNEL_NAME,
  schema: workspace.workspaceZ,
  onChange: ({ store, changed }) => store.workspaces.set(changed.key, changed),
};

const DELETE_WORKSPACE_LISTENER: Flux.ChannelListener<SubStore, typeof workspace.keyZ> =
  {
    channel: workspace.DELETE_CHANNEL_NAME,
    schema: workspace.keyZ,
    onChange: ({ store, changed }) => store.workspaces.delete(changed),
  };

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<SubStore> = {
  listeners: [SET_WORKSPACE_LISTENER, DELETE_WORKSPACE_LISTENER],
};

export interface RetrieveParams {
  key: workspace.Key;
}

export const { useRetrieve } = Flux.createRetrieve<
  RetrieveParams,
  workspace.Workspace,
  SubStore
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
  SubStore
>({
  name: "Workspace",
  retrieve: async ({ client, params }) => await client.workspaces.retrieve(params),
  retrieveByKey: async ({ client, key }) => await client.workspaces.retrieve(key),
  mountListeners: ({ store, onChange, onDelete }) => [
    store.workspaces.onSet((workspace) => onChange(workspace.key, workspace)),
    store.workspaces.onDelete(onDelete),
  ],
});

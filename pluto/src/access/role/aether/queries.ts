// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access } from "@synnaxlabs/client";

import { type Flux } from "@/flux";
import { type flux } from "@/flux/aether";
import { type Ontology } from "@/ontology";

export const FLUX_STORE_KEY = "roles";

export interface FluxStore extends flux.UnaryStore<access.role.Key, access.role.Role> {}

export interface FluxSubStore extends Flux.Store {
  [FLUX_STORE_KEY]: FluxStore;
  [Ontology.RELATIONSHIPS_FLUX_STORE_KEY]: Ontology.RelationshipFluxStore;
  [Ontology.RESOURCES_FLUX_STORE_KEY]: Ontology.ResourceFluxStore;
}

const SET_LISTENER: Flux.ChannelListener<FluxSubStore, typeof access.role.roleZ> = {
  channel: access.role.SET_CHANNEL_NAME,
  schema: access.role.roleZ,
  onChange: ({ store, changed }) => {
    store.roles.set(changed.key, changed);
  },
};

const DELETE_LISTENER: Flux.ChannelListener<FluxSubStore, typeof access.role.keyZ> = {
  channel: access.role.DELETE_CHANNEL_NAME,
  schema: access.role.keyZ,
  onChange: ({ store, changed }) => store.roles.delete(changed),
};

export const FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  access.role.Key,
  access.role.Role
> = {
  listeners: [SET_LISTENER, DELETE_LISTENER],
};

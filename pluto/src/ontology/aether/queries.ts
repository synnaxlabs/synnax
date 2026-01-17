// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";

import { type flux } from "@/flux/aether";

export interface RelationshipFluxStore extends flux.UnaryStore<
  string,
  ontology.Relationship
> {}

export interface ResourceFluxStore extends flux.UnaryStore<string, ontology.Resource> {}

export const RELATIONSHIPS_FLUX_STORE_KEY = "relationships";
export const RESOURCES_FLUX_STORE_KEY = "resources";

export interface FluxSubStore extends flux.Store {
  [RELATIONSHIPS_FLUX_STORE_KEY]: RelationshipFluxStore;
  [RESOURCES_FLUX_STORE_KEY]: ResourceFluxStore;
}

const RELATIONSHIP_SET_LISTENER: flux.ChannelListener<
  FluxSubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_SET_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: ({ store, changed }) =>
    store.relationships.set(ontology.relationshipToString(changed), changed),
};

const RELATIONSHIP_DELETE_LISTENER: flux.ChannelListener<
  FluxSubStore,
  typeof ontology.relationshipZ
> = {
  channel: ontology.RELATIONSHIP_DELETE_CHANNEL_NAME,
  schema: ontology.relationshipZ,
  onChange: ({ store, changed }) => {
    store.relationships.delete(ontology.relationshipToString(changed));
  },
};

export const RELATIONSHIP_FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  string,
  ontology.Relationship
> = {
  equal: (a, b) =>
    ontology.idsEqual(a.from, b.from) &&
    ontology.idsEqual(a.to, b.to) &&
    a.type === b.type,
  listeners: [RELATIONSHIP_SET_LISTENER, RELATIONSHIP_DELETE_LISTENER],
};

const RESOURCE_SET_LISTENER: flux.ChannelListener<
  FluxSubStore,
  typeof ontology.resourceZ
> = {
  channel: ontology.RESOURCE_SET_CHANNEL_NAME,
  schema: ontology.resourceZ,
  onChange: async ({ store, changed }) => {
    store.resources.set(changed.key, (p) =>
      p == null ? changed : { ...p, ...changed },
    );
  },
};

const RESOURCE_DELETE_LISTENER: flux.ChannelListener<
  FluxSubStore,
  typeof ontology.idZ
> = {
  channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: ({ store, changed }) => store.resources.delete(changed.key),
};

export const RESOURCE_FLUX_STORE_CONFIG: flux.UnaryStoreConfig<FluxSubStore> = {
  equal: (a, b) => deep.equal(a, b),
  listeners: [RESOURCE_SET_LISTENER, RESOURCE_DELETE_LISTENER],
};

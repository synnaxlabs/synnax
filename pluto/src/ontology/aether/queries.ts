// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { z } from "zod";

import { type flux } from "@/flux/aether";

export interface RelationshipFluxStore extends flux.UnaryStore<
  string,
  ontology.Relationship
> {}

// A resource in the ontology is just its ID; the store keys each ID by its string form.
export interface ResourceFluxStore extends flux.UnaryStore<string, ontology.ID> {}

// resourceSetZ parses a resource-set channel payload (a full resource) down to its ID,
// which is all the store tracks.
const resourceSetZ = z.object({ id: ontology.idZ }).transform(({ id }) => id);

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

const RESOURCE_SET_LISTENER: flux.ChannelListener<FluxSubStore, typeof resourceSetZ> = {
  channel: ontology.RESOURCE_SET_CHANNEL_NAME,
  schema: resourceSetZ,
  onChange: ({ store, changed }) =>
    store.resources.set(ontology.idToString(changed), changed),
};

const RESOURCE_DELETE_LISTENER: flux.ChannelListener<
  FluxSubStore,
  typeof ontology.idZ
> = {
  channel: ontology.RESOURCE_DELETE_CHANNEL_NAME,
  schema: ontology.idZ,
  onChange: ({ store, changed }) =>
    store.resources.delete(ontology.idToString(changed)),
};

export const RESOURCE_FLUX_STORE_CONFIG: flux.UnaryStoreConfig<
  FluxSubStore,
  string,
  ontology.ID
> = {
  equal: (a, b) => ontology.idsEqual(a, b),
  listeners: [RESOURCE_SET_LISTENER, RESOURCE_DELETE_LISTENER],
};

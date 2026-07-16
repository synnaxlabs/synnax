// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";

import { type cache } from "@/cache";
import {
  type ID,
  idsEqual,
  idToString,
  idZ,
  parseIDs,
  type Relationship,
  relationshipToString,
  relationshipZ,
  type Resource,
  resourceZ,
} from "@/ontology/payload";

export const RESOURCE_SET_CHANNEL_NAME = "sy_ontology_resource_set";
export const RESOURCE_DELETE_CHANNEL_NAME = "sy_ontology_resource_delete";
export const RELATIONSHIP_SET_CHANNEL_NAME = "sy_ontology_relationship_set";
export const RELATIONSHIP_DELETE_CHANNEL_NAME = "sy_ontology_relationship_delete";

export const RELATIONSHIPS_STORE_KEY = "relationships";
export const RESOURCES_STORE_KEY = "resources";

/** Registers the relationship and resource stores on the given engine. */
export const bindStores = (
  engine: cache.Engine,
  retrieveResources: (ids: ID[]) => Promise<Resource[]>,
): void => {
  const relationships = () =>
    engine.store<string, Relationship>(RELATIONSHIPS_STORE_KEY);
  const relationshipSet: cache.ChannelListener<{}, typeof relationshipZ> = {
    channel: RELATIONSHIP_SET_CHANNEL_NAME,
    schema: relationshipZ,
    onChange: ({ changed }) =>
      relationships().set(relationshipToString(changed), changed),
  };
  const relationshipDelete: cache.ChannelListener<{}, typeof relationshipZ> = {
    channel: RELATIONSHIP_DELETE_CHANNEL_NAME,
    schema: relationshipZ,
    onChange: ({ changed }) => relationships().delete(relationshipToString(changed)),
  };
  engine.registerStore<string, Relationship>(RELATIONSHIPS_STORE_KEY, {
    equal: (a, b) =>
      idsEqual(a.from, b.from) && idsEqual(a.to, b.to) && a.type === b.type,
    listeners: [relationshipSet, relationshipDelete],
  });

  const resources = () => engine.store<string, Resource>(RESOURCES_STORE_KEY);
  const resourceSet: cache.ChannelListener<{}, typeof resourceZ> = {
    channel: RESOURCE_SET_CHANNEL_NAME,
    schema: resourceZ,
    onChange: ({ changed }) =>
      resources().set(changed.key, (p) => (p == null ? changed : { ...p, ...changed })),
  };
  const resourceDelete: cache.ChannelListener<{}, typeof idZ> = {
    channel: RESOURCE_DELETE_CHANNEL_NAME,
    schema: idZ,
    // The store is keyed by the full "type:key" string, not the bare key.
    onChange: ({ changed }) => resources().delete(idToString(changed)),
  };
  engine.registerStore<string, Resource>(RESOURCES_STORE_KEY, {
    equal: (a, b) => deep.equal(a, b),
    listeners: [resourceSet, resourceDelete],
    refetch: async (keys) => await retrieveResources(parseIDs(keys)),
  });
};

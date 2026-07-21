// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type destructor } from "@synnaxlabs/x";

import { cache } from "@/cache";
import {
  type ID,
  idsEqual,
  idToString,
  idZ,
  matchRelationship,
  PARENT_OF_RELATIONSHIP_TYPE,
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

/** Returns the cached parent ID of the given ontology ID, or null if unknown. */
export const cachedParentID = (
  relationships: cache.Table<string, Relationship>,
  id: ID,
): ID | null => {
  const res = relationships.get((r) =>
    matchRelationship(r, { type: PARENT_OF_RELATIONSHIP_TYPE, to: id }),
  );
  if (res.length === 0) return null;
  return res[0].from;
};

/**
 * Optimistically renames the cached resource for the given ID. Returns a
 * rollback restoring the prior name. A no-op when the resource isn't cached.
 */
export const renameCachedResource = (
  engine: cache.Cache,
  id: ID,
  name: string,
): destructor.Destructor =>
  cache.partialUpdate(
    engine.table<string, Resource>(RESOURCES_STORE_KEY),
    idToString(id),
    { name },
  );

/**
 * Optimistically drops every cached relationship touching the given IDs.
 * Returns a rollback restoring them.
 */
export const deleteCachedRelationships = (
  engine: cache.Cache,
  ids: ID | ID[],
): destructor.Destructor => {
  const idsArr = Array.isArray(ids) ? ids : [ids];
  const relationships = engine.table<string, Relationship>(RELATIONSHIPS_STORE_KEY);
  return relationships.delete((rel) =>
    idsArr.some((id) => idsEqual(rel.to, id) || idsEqual(rel.from, id)),
  );
};

/**
 * Optimistically drops the cached resources for the given IDs along with every
 * relationship touching them. Returns a rollback restoring both.
 */
export const deleteCachedResources = (
  engine: cache.Cache,
  ids: ID | ID[],
): destructor.Destructor => {
  const idsArr = Array.isArray(ids) ? ids : [ids];
  const resources = engine.table<string, Resource>(RESOURCES_STORE_KEY);
  const undoRels = deleteCachedRelationships(engine, idsArr);
  const undoResources = resources.delete(idToString(idsArr));
  return () => {
    undoResources();
    undoRels();
  };
};

/** Registers the relationship and resource stores on the given cache. */
export const bindStores = (
  engine: cache.Cache,
  retrieveResources: (ids: ID[]) => Promise<Resource[]>,
): void => {
  const relationships = () =>
    engine.table<string, Relationship>(RELATIONSHIPS_STORE_KEY);
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
  engine.registerTable<string, Relationship>(RELATIONSHIPS_STORE_KEY, {
    equal: (a, b) =>
      idsEqual(a.from, b.from) && idsEqual(a.to, b.to) && a.type === b.type,
    listeners: [relationshipSet, relationshipDelete],
  });

  const resources = () => engine.table<string, Resource>(RESOURCES_STORE_KEY);
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
  engine.registerTable<string, Resource>(RESOURCES_STORE_KEY, {
    equal: (a, b) => deep.equal(a, b),
    listeners: [resourceSet, resourceDelete],
    refetch: async (keys) => await retrieveResources(parseIDs(keys)),
  });
};

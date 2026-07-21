// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import { cache } from "@/cache";
import {
  type ID,
  idsEqual,
  idToString,
  matchRelationship,
  PARENT_OF_RELATIONSHIP_TYPE,
  type Relationship,
  type Resource,
} from "@/ontology/payload";

export const RESOURCE_SET_CHANNEL_NAME = "sy_ontology_resource_set";
export const RESOURCE_DELETE_CHANNEL_NAME = "sy_ontology_resource_delete";
export const RELATIONSHIP_SET_CHANNEL_NAME = "sy_ontology_relationship_set";
export const RELATIONSHIP_DELETE_CHANNEL_NAME = "sy_ontology_relationship_delete";

/** The ontology tables shared across domain clients. */
export interface Stores {
  relationships: cache.Table<string, Relationship>;
  resources: cache.Table<string, Resource>;
}

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
  { resources }: Stores,
  id: ID,
  name: string,
): destructor.Destructor => cache.partialUpdate(resources, idToString(id), { name });

/**
 * Optimistically drops every cached relationship touching the given IDs.
 * Returns a rollback restoring them.
 */
export const deleteCachedRelationships = (
  { relationships }: Stores,
  ids: ID | ID[],
): destructor.Destructor => {
  const idsArr = Array.isArray(ids) ? ids : [ids];
  return relationships.delete((rel) =>
    idsArr.some((id) => idsEqual(rel.to, id) || idsEqual(rel.from, id)),
  );
};

/**
 * Optimistically drops the cached resources for the given IDs along with every
 * relationship touching them. Returns a rollback restoring both.
 */
export const deleteCachedResources = (
  stores: Stores,
  ids: ID | ID[],
): destructor.Destructor => {
  const idsArr = Array.isArray(ids) ? ids : [ids];
  const undoRels = deleteCachedRelationships(stores, idsArr);
  const undoResources = stores.resources.delete(idToString(idsArr));
  return () => {
    undoResources();
    undoRels();
  };
};

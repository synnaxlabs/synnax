// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type destructor } from "@synnaxlabs/x";

import {
  PARENT_OF_RELATIONSHIP_TYPE,
  type Relationship,
  relationshipToString,
  type Resource,
  ROOT_ID,
} from "@/ontology/payload";
import { query } from "@/query";
import { type ID, idToString } from "@/resource/payload";

/** Channels the Core writes ontology changes to. */
export const RESOURCE_SET_CHANNEL_NAME = "sy_ontology_resource_set";
export const RESOURCE_DELETE_CHANNEL_NAME = "sy_ontology_resource_delete";
export const RELATIONSHIP_SET_CHANNEL_NAME = "sy_ontology_relationship_set";
export const RELATIONSHIP_DELETE_CHANNEL_NAME = "sy_ontology_relationship_delete";

/**
 * The secondary indexes registered on the relationships table, one per endpoint.
 * Type filtering runs at the call site over the matched handful.
 */
export interface RelationshipIndexes {
  /** Keyed by the stringified `to` end. */
  byTo: query.LookupIndex<string, Relationship>;
  /** Keyed by the stringified `from` end. */
  byFrom: query.LookupIndex<string, Relationship>;
}

/** The ontology record tables: resources and the relationships between them. */
export class Cache {
  readonly relationships: query.Table<string, Relationship>;
  readonly resources: query.Table<string, Resource>;
  private readonly indexes: RelationshipIndexes;

  constructor(
    relationships: query.Table<string, Relationship>,
    resources: query.Table<string, Resource>,
    indexes: RelationshipIndexes,
  ) {
    this.relationships = relationships;
    this.resources = resources;
    this.indexes = indexes;
  }

  /** Returns the cached relationships pointing at the given ID. */
  relationshipsTo(id: ID): Relationship[] {
    return this.indexes.byTo.get(idToString(id));
  }

  /** Returns the cached relationships originating at the given ID. */
  relationshipsFrom(id: ID): Relationship[] {
    return this.indexes.byFrom.get(idToString(id));
  }

  /** Returns the cached parent ID of the given resource ID, or null if unknown. */
  parentID(id: ID): ID | null {
    return this.parentsOf(id)[0]?.from ?? null;
  }

  /**
   * Returns the string form of every ID that cached parent relationships lead to from
   * ids, walking up to root or, with no root, up to the ontology root. Leaves out the
   * root and the IDs themselves.
   */
  ancestorKeys(ids: ID[], root: ID = ROOT_ID): Set<string> {
    const rootKey = idToString(root);
    const out = new Set<string>();
    const visit = (id: ID): void => {
      for (const { from } of this.parentsOf(id)) {
        const key = idToString(from);
        if (key === rootKey || out.has(key)) continue;
        out.add(key);
        visit(from);
      }
    };
    ids.forEach(visit);
    return out;
  }

  /**
   * Returns each path from root down to the parent of each ID, read from cached parent
   * relationships and keyed by the string form of the ID. A path lists IDs from the top
   * down and leaves out root, so a direct child of root has one empty path. A path that
   * does not reach root is left out. With no root, paths run up to the ontology root.
   */
  paths(ids: ID[], root: ID = ROOT_ID): Record<string, ID[][]> {
    const rootKey = idToString(root);
    const memo = new Map<string, ID[][]>();
    const pathsTo = (id: ID): ID[][] => {
      const key = idToString(id);
      const memoized = memo.get(key);
      if (memoized != null) return memoized;
      // Marks the walk in progress, so a cycle ends with no path.
      memo.set(key, []);
      const out: ID[][] = [];
      for (const { from } of this.parentsOf(id))
        if (idToString(from) === rootKey) out.push([]);
        else for (const path of pathsTo(from)) out.push([...path, from]);
      memo.set(key, out);
      return out;
    };
    return Object.fromEntries(ids.map((id) => [idToString(id), pathsTo(id)]));
  }

  private parentsOf(id: ID): Relationship[] {
    return this.relationshipsTo(id).filter(
      (r) => r.type === PARENT_OF_RELATIONSHIP_TYPE,
    );
  }

  /**
   * Optimistically renames the cached resource for the given ID. Returns a
   * rollback restoring the prior name. A no-op when the resource isn't cached.
   */
  renameResource(id: ID, name: string): destructor.Destructor {
    return query.partialUpdate(this.resources, idToString(id), { name });
  }

  /**
   * Optimistically drops every cached relationship touching the given IDs.
   * Returns a rollback restoring them.
   */
  deleteRelationships(ids: ID | ID[]): destructor.Destructor {
    const idsArr = array.toArray(ids);
    const keys = new Set<string>();
    for (const id of idsArr) {
      for (const rel of this.relationshipsTo(id)) keys.add(relationshipToString(rel));
      for (const rel of this.relationshipsFrom(id)) keys.add(relationshipToString(rel));
    }
    return this.relationships.delete(Array.from(keys));
  }

  /**
   * Optimistically drops the cached resources for the given IDs along with
   * every relationship touching them. Returns a rollback restoring both.
   */
  deleteResources(ids: ID | ID[]): destructor.Destructor {
    const idsArr = array.toArray(ids);
    const undoRels = this.deleteRelationships(idsArr);
    const undoResources = this.resources.delete(idToString(idsArr));
    return () => {
      undoResources();
      undoRels();
    };
  }
}

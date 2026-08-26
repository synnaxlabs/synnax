// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor } from "@synnaxlabs/x";

import {
  type ID,
  idToString,
  PARENT_OF_RELATIONSHIP_TYPE,
  type Relationship,
  relationshipToString,
  type Resource,
} from "@/ontology/payload";
import { query } from "@/query";

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

  /** Returns the cached parent ID of the given ontology ID, or null if unknown. */
  parentID(id: ID): ID | null {
    const rel = this.relationshipsTo(id).find(
      (r) => r.type === PARENT_OF_RELATIONSHIP_TYPE,
    );
    return rel?.from ?? null;
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
    const idsArr = Array.isArray(ids) ? ids : [ids];
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
    const idsArr = Array.isArray(ids) ? ids : [ids];
    const undoRels = this.deleteRelationships(idsArr);
    const undoResources = this.resources.delete(idToString(idsArr));
    return () => {
      undoResources();
      undoRels();
    };
  }
}

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type change } from "@synnaxlabs/x";

import { QueryError } from "@/errors";
import { type Client as FrameClient } from "@/framer/client";
import { type Frame } from "@/framer/frame";
import { type Streamer as FrameStreamer } from "@/framer/streamer";
import {
  ID,
  parseRelationship,
  type Relationship,
  type Resource,
} from "@/ontology/payload";
import { type Retriever } from "@/ontology/retriever";

export type ResourceChange = change.Change<ID, Resource>;
export type ResourceSet = change.Set<ID, Resource>;
export type ResourceDelete = change.Delete<ID, Resource>;
export type RelationshipChange = change.Change<Relationship, undefined>;
export type RelationshipSet = change.Set<Relationship, undefined>;
export type RelationshipDelete = change.Delete<Relationship, undefined>;

const RESOURCE_SET_NAME = "sy_ontology_resource_set";
const RESOURCE_DELETE_NAME = "sy_ontology_resource_delete";
const RELATIONSHIP_SET_NAME = "sy_ontology_relationship_set";
const RELATIONSHIP_DELETE_NAME = "sy_ontology_relationship_delete";

export class ChangeTracker {
  private readonly resourceObs: observe.Observer<ResourceChange[]>;
  private readonly relationshipObs: observe.Observer<RelationshipChange[]>;

  readonly relationships: observe.Observable<RelationshipChange[]>;
  readonly resources: observe.Observable<ResourceChange[]>;

  private readonly streamer: FrameStreamer;
  private readonly retriever: Retriever;
  private readonly closePromise: Promise<void>;

  constructor(streamer: FrameStreamer, retriever: Retriever) {
    this.relationshipObs = new observe.Observer<RelationshipChange[]>();
    this.relationships = this.relationshipObs;
    this.resourceObs = new observe.Observer<ResourceChange[]>();
    this.resources = this.resourceObs;
    this.retriever = retriever;
    this.streamer = streamer;
    this.closePromise = this.start();
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  private async start(): Promise<void> {
    for await (const frame of this.streamer) {
      await this.update(frame);
    }
  }

  private async update(frame: Frame): Promise<void> {
    const resSets = await this.parseResourceSets(frame);
    const resDeletes = this.parseResourceDeletes(frame);
    const allResources = resSets.concat(resDeletes);
    if (allResources.length > 0) this.resourceObs.notify(resSets.concat(resDeletes));
    const relSets = this.parseRelationshipSets(frame);
    const relDeletes = this.parseRelationshipDeletes(frame);
    const allRels = relSets.concat(relDeletes);
    if (allRels.length > 0) this.relationshipObs.notify(relSets.concat(relDeletes));
  }

  private parseRelationshipSets(frame: Frame): RelationshipChange[] {
    const relationships = frame.get(RELATIONSHIP_SET_NAME);
    if (relationships.length === 0) return [];
    // We should only ever get one series of relationships
    return relationships[0].toStrings().map((rel) => ({
      variant: "set",
      key: parseRelationship(rel),
      value: undefined,
    }));
  }

  private parseRelationshipDeletes(frame: Frame): RelationshipChange[] {
    const relationships = frame.get(RELATIONSHIP_DELETE_NAME);
    if (relationships.length === 0) return [];
    // We should only ever get one series of relationships
    return relationships[0].toStrings().map((rel) => ({
      variant: "delete",
      key: parseRelationship(rel),
    }));
  }

  private async parseResourceSets(frame: Frame): Promise<ResourceChange[]> {
    const sets = frame.get(RESOURCE_SET_NAME);
    if (sets.length === 0) return [];
    // We should only ever get one series of sets
    const ids = sets[0].toStrings().map((id) => new ID(id));
    try {
      const resources = await this.retriever.retrieve(ids);
      return resources.map((resource) => ({
        variant: "set",
        key: resource.id,
        value: resource,
      }));
    } catch (e) {
      if (e instanceof QueryError) {
        console.warn(e);
        return [];
      }
      throw e;
    }
  }

  private parseResourceDeletes(frame: Frame): ResourceChange[] {
    const deletes = frame.get(RESOURCE_DELETE_NAME);
    if (deletes.length === 0) return [];
    // We should only ever get one series of deletes
    return deletes[0]
      .toStrings()
      .map((str) => ({ variant: "delete", key: new ID(str) }));
  }

  static async open(client: FrameClient, retriever: Retriever): Promise<ChangeTracker> {
    const streamer = await client.openStreamer([
      RESOURCE_SET_NAME,
      RESOURCE_DELETE_NAME,
      RELATIONSHIP_SET_NAME,
      RELATIONSHIP_DELETE_NAME,
    ]);
    return new ChangeTracker(streamer, retriever);
  }
}

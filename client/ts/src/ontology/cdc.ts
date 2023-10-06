// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type change, type Destructor } from "@synnaxlabs/x";

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
export type RelationshipChange = change.Change<Relationship, undefined>;

export const parseIDsFromBuffer = (buf: ArrayBufferLike): ID[] =>
  new TextDecoder()
    .decode(buf)
    .split("\n")
    .slice(0, -1)
    .map((id) => new ID(id));

export const parseRelationshipsFromBuffer = (buf: ArrayBufferLike): Relationship[] =>
  new TextDecoder()
    .decode(buf)
    .split("\n")
    .slice(0, -1)
    .map((rel) => parseRelationship(rel));

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
    const relationships = frame.get("sy_ontology_relationship_set");
    if (relationships.length === 0) return [];
    // We should only ever get one series of relationships
    const rels = parseRelationshipsFromBuffer(relationships[0].buffer);
    return rels.map((rel) => ({ variant: "set", key: rel, value: undefined }));
  }

  private parseRelationshipDeletes(frame: Frame): RelationshipChange[] {
    const relationships = frame.get("sy_ontology_relationship_delete");
    if (relationships.length === 0) return [];
    // We should only ever get one series of relationships
    const rels = parseRelationshipsFromBuffer(relationships[0].buffer);
    return rels.map((rel) => ({ variant: "delete", key: rel }));
  }

  private async parseResourceSets(frame: Frame): Promise<ResourceChange[]> {
    const sets = frame.get("sy_ontology_set");
    if (sets.length === 0) return [];
    // We should only ever get one series of sets
    const ids = parseIDsFromBuffer(sets[0].buffer);
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
    const deletes = frame.get("sy_ontology_delete");
    if (deletes.length === 0) return [];
    // We should only ever get one series of deletes
    const ids = parseIDsFromBuffer(deletes[0].buffer);
    return ids.map((id) => ({ variant: "delete", key: id }));
  }

  static async open(client: FrameClient, retriever: Retriever): Promise<ChangeTracker> {
    const streamer = await client.newStreamer([
      "sy_ontology_set",
      "sy_ontology_delete",
      "sy_ontology_relationship_set",
      "sy_ontology_relationship_delete",
    ]);
    return new ChangeTracker(streamer, retriever);
  }
}

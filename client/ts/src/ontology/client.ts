// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { observe, toArray } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { z } from "zod";

import { QueryError } from "@/errors";
import { framer } from "@/framer";
import { Frame } from "@/framer/frame";
import { group } from "@/ontology/group";
import {
  CrudeID,
  ID,
  IDPayload,
  idZ,
  parseRelationship,
  RelationshipChange,
  type Resource,
  ResourceChange,
  resourceSchemaZ,
} from "@/ontology/payload";
import { Writer } from "@/ontology/writer";

const RETRIEVE_ENDPOINT = "/ontology/retrieve";

const retrieveReqZ = z.object({
  ids: idZ.array().optional(),
  children: z.boolean().optional(),
  parents: z.boolean().optional(),
  includeSchema: z.boolean().optional(),
  excludeFieldData: z.boolean().optional(),
  term: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
});

type RetrieveRequest = z.infer<typeof retrieveReqZ>;

export type RetrieveOptions = Pick<
  RetrieveRequest,
  "includeSchema" | "excludeFieldData"
>;

const retrieveResZ = z.object({
  resources: resourceSchemaZ.array(),
});

export const parseIDs = (ids: CrudeID | CrudeID[] | string | string[]): IDPayload[] =>
  toArray(ids).map((id) => new ID(id).payload);

/** The core client class for executing queries against a Synnax cluster ontology */
export class Client implements AsyncTermSearcher<string, string, Resource> {
  readonly type: string = "ontology";
  groups: group.Client;
  private readonly client: UnaryClient;
  private readonly writer: Writer;
  private readonly framer: framer.Client;

  constructor(unary: UnaryClient, framer: framer.Client) {
    this.client = unary;
    this.writer = new Writer(unary);
    this.groups = new group.Client(unary);
    this.framer = framer;
  }

  async search(term: string, options?: RetrieveOptions): Promise<Resource[]> {
    return await this.execRetrieve({ term, ...options });
  }

  async retrieve(id: ID | string, options?: RetrieveOptions): Promise<Resource>;

  async retrieve(ids: ID[] | string[], options?: RetrieveOptions): Promise<Resource[]>;

  async retrieve(
    ids: CrudeID | CrudeID[] | string | string[],
    options?: RetrieveOptions,
  ): Promise<Resource | Resource[]> {
    const resources = await this.execRetrieve({ ids: parseIDs(ids), ...options });
    if (Array.isArray(ids)) return resources;
    if (resources.length === 0)
      throw new QueryError(`No resource found with ID ${ids.toString()}`);
    return resources[0];
  }

  async page(
    offset: number,
    limit: number,
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({ offset, limit, ...options });
  }

  async retrieveChildren(
    ids: CrudeID | CrudeID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({ ids: parseIDs(ids), children: true, ...options });
  }

  async retrieveParents(
    ids: CrudeID | CrudeID[],
    options?: RetrieveOptions,
  ): Promise<Resource[]> {
    return await this.execRetrieve({ ids: parseIDs(ids), parents: true, ...options });
  }

  async addChildren(id: CrudeID, ...children: CrudeID[]): Promise<void> {
    return await this.writer.addChildren(id, ...children);
  }

  async removeChildren(id: CrudeID, ...children: CrudeID[]): Promise<void> {
    return await this.writer.removeChildren(id, ...children);
  }

  async moveChildren(
    from: CrudeID,
    to: CrudeID,
    ...children: CrudeID[]
  ): Promise<void> {
    return await this.writer.moveChildren(from, to, ...children);
  }

  async openChangeTracker(): Promise<ChangeTracker> {
    return await ChangeTracker.open(this.framer, this);
  }

  async trackRelationships(
    parent: ID,
    type: string = "parent",
    retrieveInitial?: (parent: ID) => Promise<Resource[]>,
  ): Promise<observe.ObservableAsyncCloseable<Resource[]>> {
    return await RelationshipTracker.open(
      parent,
      this,
      this.framer,
      type,
      retrieveInitial,
    );
  }

  newSearcherWithOptions(
    options: RetrieveOptions,
  ): AsyncTermSearcher<string, string, Resource> {
    return {
      type: this.type,
      search: (term: string) => this.search(term, options),
      retrieve: (ids: string[]) => this.retrieve(ids, options),
      page: (offset: number, limit: number) => this.page(offset, limit, options),
    };
  }

  private async execRetrieve(request: RetrieveRequest): Promise<Resource[]> {
    const { resources } = await sendRequired(
      this.client,
      RETRIEVE_ENDPOINT,
      request,
      retrieveReqZ,
      retrieveResZ,
    );
    return resources;
  }
}

const RESOURCE_SET_NAME = "sy_ontology_resource_set";
const RESOURCE_DELETE_NAME = "sy_ontology_resource_delete";
const RELATIONSHIP_SET_NAME = "sy_ontology_relationship_set";
const RELATIONSHIP_DELETE_NAME = "sy_ontology_relationship_delete";

export class ChangeTracker {
  private readonly resourceObs: observe.Observer<ResourceChange[]>;
  private readonly relationshipObs: observe.Observer<RelationshipChange[]>;

  readonly relationships: observe.Observable<RelationshipChange[]>;
  readonly resources: observe.Observable<ResourceChange[]>;

  private readonly streamer: framer.Streamer;
  private readonly client: Client;
  private readonly closePromise: Promise<void>;

  constructor(streamer: framer.Streamer, client: Client) {
    this.relationshipObs = new observe.Observer<RelationshipChange[]>();
    this.relationships = this.relationshipObs;
    this.resourceObs = new observe.Observer<ResourceChange[]>();
    this.resources = this.resourceObs;
    this.client = client;
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
    const allRelationships = relSets.concat(relDeletes);
    if (allRelationships.length > 0)
      this.relationshipObs.notify(relSets.concat(relDeletes));
  }

  private parseRelationshipSets(frame: Frame): RelationshipChange[] {
    const relationships = frame.get(RELATIONSHIP_SET_NAME);
    if (relationships.length === 0) return [];
    return Array.from(relationships.as("string")).map((rel) => ({
      variant: "set",
      key: parseRelationship(rel),
      value: undefined,
    }));
  }

  private parseRelationshipDeletes(frame: Frame): RelationshipChange[] {
    const relationships = frame.get(RELATIONSHIP_DELETE_NAME);
    if (relationships.length === 0) return [];
    return Array.from(relationships.as("string")).map((rel) => ({
      variant: "delete",
      key: parseRelationship(rel),
    }));
  }

  private async parseResourceSets(frame: Frame): Promise<ResourceChange[]> {
    const sets = frame.get(RESOURCE_SET_NAME);
    if (sets.length === 0) return [];
    // We should only ever get one series of sets
    const ids = Array.from(sets.as("string")).map((id: string) => new ID(id));
    try {
      const resources = await this.client.retrieve(ids);
      return resources.map((resource) => ({
        variant: "set",
        key: resource.id,
        value: resource,
      }));
    } catch (e) {
      if (e instanceof QueryError) return [];
      throw e;
    }
  }

  private parseResourceDeletes(frame: Frame): ResourceChange[] {
    const deletes = frame.get(RESOURCE_DELETE_NAME);
    if (deletes.length === 0) return [];
    // We should only ever get one series of deletes
    return Array.from(deletes.as("string")).map((str) => ({
      variant: "delete",
      key: new ID(str),
    }));
  }

  static async open(client: framer.Client, retriever: Client): Promise<ChangeTracker> {
    const streamer = await client.openStreamer([
      RESOURCE_SET_NAME,
      RESOURCE_DELETE_NAME,
      RELATIONSHIP_SET_NAME,
      RELATIONSHIP_DELETE_NAME,
    ]);
    return new ChangeTracker(streamer, retriever);
  }
}

export class RelationshipTracker
  extends observe.Observer<Resource[]>
  implements observe.ObservableAsyncCloseable<Resource[]>
{
  private readonly internal: ChangeTracker;
  private readonly from: ID;
  private children: Resource[];
  private readonly client: Client;
  private readonly type: string;

  private constructor(
    from: ID,
    internal: ChangeTracker,
    to: Resource[],
    client: Client,
    type: string = "parent",
  ) {
    super();
    this.internal = internal;
    this.from = from;
    this.children = to;
    this.client = client;
    this.type = type;
    this.internal.resources.onChange(this.handleResourceChange);
    this.internal.relationships.onChange(this.handleRelationshipChange);
  }
  static async open(
    from: ID,
    client: Client,
    framer: framer.Client,
    type: string = "parent",
    retrieveInitial?: (parent: ID) => Promise<Resource[]>,
  ): Promise<RelationshipTracker> {
    const internal = await ChangeTracker.open(framer, client);
    let currChildren: Resource[];
    if (retrieveInitial != null) currChildren = await retrieveInitial(from);
    else currChildren = await client.retrieveChildren(from);
    return new RelationshipTracker(from, internal, currChildren, client, type);
  }

  private handleResourceChange = (changes: ResourceChange[]): void => {
    this.children = this.children.map((child) => {
      const change = changes.find((c) => c.key.toString() == child.id.toString());
      if (change == null || change.variant === "delete") return child;
      return change.value;
    });
    this.notify(this.children);
  };

  private handleRelationshipChange = (changes: RelationshipChange[]): void => {
    const deletes = changes.filter(
      (c) => c.variant === "delete" && c.key.from.toString() === this.from.toString(),
    );
    this.children = this.children.filter(
      (child) =>
        !deletes.some(
          (del) =>
            del.key.to.toString() === child.id.toString() && del.key.type === this.type,
        ),
    );
    const sets = changes.filter(
      (c) =>
        c.variant === "set" &&
        c.key.type === this.type &&
        c.key.from.toString() === this.from.toString(),
    );
    if (sets.length === 0) return this.notify(this.children);
    this.client.retrieve(sets.map((s) => s.key.to)).then((resources) => {
      this.children = this.children.concat(resources);
      this.notify(this.children);
    });
  };

  async close(): Promise<void> {
    await this.internal.close();
  }
}

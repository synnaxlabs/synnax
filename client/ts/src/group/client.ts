// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, zod } from "@synnaxlabs/x";
import z from "zod";

import {
  type Group,
  groupZ,
  type Key,
  keyZ,
  ontologyID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Where,
} from "@/group/types.gen";
import { matches, serverComputed } from "@/group/where.gen";
import { ontology } from "@/ontology";
import { idZ as ontologyIDZ } from "@/ontology/payload";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_group_set";
export const DELETE_CHANNEL_NAME = "sy_group_delete";

const resZ = z.object({ group: groupZ });

const createReqZ = z.object({
  parent: ontologyIDZ,
  key: keyZ.optional(),
  name: z.string(),
});

const renameReqZ = z.object({ key: keyZ, name: z.string() });

const deleteReqZ = z.object({ keys: z.array(keyZ) });

export interface CreateParams extends z.infer<typeof createReqZ> {}

// The Core knows nothing of parents, so the client resolves parent through the ontology
// graph.
const retrieveMultipleParamsZ = retrieveRequestZ.extend({
  parent: ontologyIDZ.optional(),
});

export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = z.input<typeof retrieveMultipleParamsZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

interface MultipleRequest extends z.infer<typeof retrieveMultipleParamsZ> {}

const isChildOf = (rel: ontology.Relationship, parent: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: parent,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "group" },
  });

export interface ClientConfig {
  unary: UnaryClient;
  ontology: ontology.Client;
  cache: query.Cache;
}

export class Client extends query.Retriever<
  typeof retrieveMultipleParamsZ,
  Key,
  Group
> {
  /** Public so a client that deletes a group through an endpoint of its own can drop
   * the group from the cache. */
  readonly store: query.Table<Key, Group>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache, ontology: ontologyClient } = cfg;
    const store = cache.createTable<Key, Group>({
      name: "groups",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, groupZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "group",
      table: store,
      request: {
        schema: retrieveMultipleParamsZ,
        fetch: async (req) => await this.fetchRequest(req),
        matches: (g, req) =>
          matches(g, req.where) &&
          (req.parent == null || this.isCachedChild(req.parent, g.key)),
        serverComputed: (req) => serverComputed(req.where),
        watch: [
          query.watch(
            ontologyClient.cache.relationships,
            (event, req: MultipleRequest) => {
              if (req.parent == null) return null;
              const rel =
                event.variant === "set"
                  ? event.value
                  : ontology.relationshipZ.parse(event.key);
              if (!isChildOf(rel, req.parent)) return null;
              return [rel.to.key];
            },
          ),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async create(params: CreateParams): Promise<Group> {
    const res = await this.cfg.unary.send(
      "/ontology/create-group",
      params,
      createReqZ,
      resZ,
    );
    this.store.set(res.group);
    const rel: ontology.Relationship = {
      from: params.parent,
      type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
      to: ontologyID(res.group.key),
    };
    this.cfg.ontology.cache.relationships.set(ontology.relationshipToString(rel), rel);
    return res.group;
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/ontology/rename-group",
          { key, name },
          renameReqZ,
          z.object({}),
        ),
    });
    rename();
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => this.store.delete(keysArr);
    await query.optimistic({
      rollbacks: [drop()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/ontology/delete-group",
          { keys: keysArr },
          deleteReqZ,
          z.object({}),
        ),
    });
    drop();
  }

  private isCachedChild(parent: ontology.ID, key: Key): boolean {
    return this.cfg.ontology.cache.relationships.has(
      ontology.relationshipToString({
        from: parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(key),
      }),
    );
  }

  private requireOntology(): ontology.Client {
    if (this.cfg.ontology == null)
      throw new Error("ontology client is not available on this client");
    return this.cfg.ontology;
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Group[]> {
    const res = await this.cfg.unary.send(
      "/ontology/retrieve-group",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.groups;
  }

  private async fetchRequest({ parent, ...req }: MultipleRequest): Promise<Group[]> {
    if (parent == null) return await this.execRetrieve(req);
    const keys = (await this.fetchChildren(parent)).map((g) => g.key);
    if (keys.length === 0) return [];
    const children: Where = { key: { in: keys } };
    const where = req.where == null ? children : { and: [req.where, children] };
    return await this.execRetrieve({ ...req, where });
  }

  private async fetchChildren(parent: ontology.ID): Promise<Group[]> {
    const res = await this.requireOntology().children.retrieve({
      ids: parent,
      types: ["group"],
    });
    const groups = res.map((r) => zod.parse(groupZ, r.data, { label: "group" }));
    const rels = this.cfg.ontology.cache.relationships;
    groups.forEach((g) => {
      const rel: ontology.Relationship = {
        from: parent,
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: ontologyID(g.key),
      };
      rels.set(ontology.relationshipToString(rel), rel);
    });
    return groups;
  }
}

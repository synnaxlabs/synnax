// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x";
import z from "zod";

import { type Group, groupZ, type Key, keyZ, ontologyID } from "@/group/types.gen";
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

const retrieveChildrenParamsZ = z.object({
  parent: ontologyIDZ,
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});

export type RetrieveSingleParams = { key: Key };
export type RetrieveChildrenParams = z.input<typeof retrieveChildrenParamsZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveChildrenParams;

interface ChildrenRequest extends z.infer<typeof retrieveChildrenParamsZ> {}

const isChildOf = (rel: ontology.Relationship, parent: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: parent,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "group" },
  });

export interface ClientParams {
  unary: UnaryClient;
  ontology: ontology.Client;
  cache: query.Cache;
}

export class Client extends query.Retriever<
  typeof retrieveChildrenParamsZ,
  Key,
  Group
> {
  private readonly cfg: ClientParams;
  private readonly store: query.Table<Key, Group>;

  constructor(cfg: ClientParams) {
    const { cache, ontology: ontologyClient } = cfg;
    const store = cache.createTable<Key, Group>({
      name: "groups",
      fetch: async (keys) => await this.execRetrieveKeys(keys),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, groupZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "group",
      table: store,
      request: {
        schema: retrieveChildrenParamsZ,
        fetch: async (req) => await this.fetchChildren(req),
        matches: (g, req) => this.isCachedChild(req.parent, g.key),
        watch: [
          query.watch(
            ontologyClient.cache.relationships,
            (event, req: ChildrenRequest) => {
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

  private async execRetrieveKeys(keys: Key[]): Promise<Group[]> {
    const res = await this.requireOntology().retrieve({
      ids: keys.map((key) => ontologyID(key)),
    });
    return res.map((r) => groupZ.parse(r.data));
  }

  private async fetchChildren(req: ChildrenRequest): Promise<Group[]> {
    const { parent, ...options } = req;
    const res = await this.requireOntology().children.retrieve({
      ids: parent,
      ...options,
      types: ["group"],
    });
    const groups = res.map((r) => groupZ.parse(r.data));
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

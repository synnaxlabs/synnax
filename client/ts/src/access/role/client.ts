// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Role,
  roleZ,
} from "@/access/role/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { user } from "@/user";

export const SET_CHANNEL_NAME = "sy_role_set";
export const DELETE_CHANNEL_NAME = "sy_role_delete";

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  internal: z.boolean().optional(),
});
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

const keyRetrieveRequestZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const singleCreateParamsZ = roleZ.transform((r) => ({ roles: [r] }));
export type SingleCreateParams = z.input<typeof singleCreateParamsZ>;

export const multipleCreateParamsZ = roleZ.array().transform((roles) => ({ roles }));

export const createParamsZ = z.union([singleCreateParamsZ, multipleCreateParamsZ]);
export type CreateParams = z.input<typeof createParamsZ>;

const createResZ = z.object({ roles: roleZ.array() });
const retrieveResZ = z.object({ roles: roleZ.array().default(() => []) });

export type RetrieveSingleParams = z.input<typeof keyRetrieveRequestZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

export const retrieveParamsZ = z.union([keyRetrieveRequestZ, retrieveRequestZ]);
export type RetrieveParams = z.input<typeof retrieveParamsZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const deleteResZ = z.object({});

const deleteParamsZ = keyZ
  .transform((key) => ({ keys: [key] }))
  .or(keyZ.array().transform((keys) => ({ keys })));
export type DeleteParams = z.input<typeof deleteParamsZ>;

const assignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type AssignParams = z.input<typeof assignReqZ>;

const assignResZ = z.object({});

const unassignReqZ = z.object({
  user: user.keyZ,
  role: keyZ,
});
export type UnassignParams = z.input<typeof unassignReqZ>;

const unassignResZ = z.object({});

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["limit", "offset"] as const;

/**
 * Client-side matching for a request: key sets and the internal flag.
 * Server-computed shapes (pagination) never reach this filter; they refetch
 * instead.
 */
const requestFilter = (req: RetrieveRequest): ((r: Role) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (r) => {
    if (keySet != null && !keySet.has(r.key)) return false;
    if (req.internal != null && r.internal !== req.internal) return false;
    return true;
  };
};

const assignmentRel = (role: Key, userKey: user.Key): ontology.Relationship => ({
  from: ontologyID(role),
  type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
  to: user.ontologyID(userKey),
});

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Role> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Role>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Role>({
      name: "roles",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, roleZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "role",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (role, req) => requestFilter(req)(role),
        serverFields: SERVER_FIELDS,
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async create(role: New): Promise<Role>;
  async create(roles: New[]): Promise<Role[]>;
  async create(roles: New | New[]): Promise<Role | Role[]> {
    const isMany = Array.isArray(roles);
    const res = await this.cfg.unary.send(
      "/access/role/create",
      roles,
      createParamsZ,
      createResZ,
    );
    this.store.set(res.roles);
    return isMany ? res.roles : res.roles[0];
  }

  async delete(params: DeleteParams, opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(params);
    const ids = ontologyID(keysArr);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(ids),
      this.store.delete(keysArr),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/access/role/delete",
          params,
          deleteParamsZ,
          deleteResZ,
        ),
    });
    drop();
    this.cfg.ontology.cache.relationships.delete((r) =>
      ids.some((id) => ontology.idsEqual(r.from, id) || ontology.idsEqual(r.to, id)),
    );
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const existing = await this.retrieve(key);
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () => {
        await this.create({ ...existing, name });
      },
    });
    rename();
  }

  async assign(params: AssignParams, opts: query.WriteOptions = {}): Promise<void> {
    const rel = assignmentRel(params.role, params.user);
    await query.optimistic({
      rollbacks: [
        this.cfg.ontology.cache.relationships.set(
          ontology.relationshipToString(rel),
          rel,
        ),
      ],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/access/role/assign",
          params,
          assignReqZ,
          assignResZ,
        ),
    });
  }

  async unassign(params: UnassignParams, opts: query.WriteOptions = {}): Promise<void> {
    await query.optimistic({
      rollbacks: [
        this.cfg.ontology.cache.relationships.delete(
          ontology.relationshipToString(assignmentRel(params.role, params.user)),
        ),
      ],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/access/role/unassign",
          params,
          unassignReqZ,
          unassignResZ,
        ),
    });
  }

  private async execRetrieve(params: RetrieveMultipleParams): Promise<Role[]> {
    const res = await this.cfg.unary.send(
      "/access/role/retrieve",
      params,
      retrieveRequestZ,
      retrieveResZ,
    );
    return res.roles;
  }
}

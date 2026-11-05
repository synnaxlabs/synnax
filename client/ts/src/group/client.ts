// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, primitive, zod } from "@synnaxlabs/x";
import z from "zod";

import { type batch } from "@/batch";
import { type Group, groupZ, type Key, keyZ } from "@/group/types.gen";
import { query } from "@/query";
import { resource } from "@/resource";

export const SET_CHANNEL_NAME = "sy_group_set";
export const DELETE_CHANNEL_NAME = "sy_group_delete";

const createReqZ = z.object({
  key: keyZ.optional(),
  name: z.string(),
  scope: resource.typeZ.optional(),
  parent: resource.idZ.optional(),
});
const createResZ = z.object({ group: groupZ });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const setParentReqZ = z.object({
  keys: keyZ.array(),
  parent: resource.idZ.optional(),
});
const emptyResZ = z.object({});

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  /** Keeps groups whose parent is any of these resources. */
  parents: resource.idZ.array().optional(),
  /** Keeps groups whose scope is any of these types. */
  scopes: resource.typeZ.array().optional(),
  /** Keeps groups with a parent when true, and top-level groups when false. */
  nested: z.boolean().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
});
const retrieveMultiParamsZ = retrieveReqZ.or(query.keyListZ(keyZ));
const wireRetrieveReqZ = retrieveReqZ.extend({ ancestors: z.boolean().optional() });
const retrieveResZ = z.object({
  groups: groupZ.array().default(() => []),
  ancestors: groupZ.array().default(() => []),
});

export interface CreateParams extends z.input<typeof createReqZ> {}
export type RetrieveRequest = z.input<typeof retrieveReqZ>;

/**
 * Client-side matching for a request. Server-computed shapes (search, pagination) never
 * reach this filter; they refetch instead.
 */
const requestFilter = (req: z.output<typeof retrieveReqZ>): ((g: Group) => boolean) => {
  const keys = primitive.isNonZero(req.keys) ? new Set(req.keys) : null;
  const parents = primitive.isNonZero(req.parents)
    ? new Set(req.parents.map((id) => resource.idToString(id)))
    : null;
  const scopes = primitive.isNonZero(req.scopes) ? new Set(req.scopes) : null;
  const { nested } = req;
  return (g) => {
    if (keys != null && !keys.has(g.key)) return false;
    if (scopes != null && !scopes.has(g.scope)) return false;
    if (nested != null && (g.parent != null) !== nested) return false;
    if (
      parents != null &&
      (g.parent == null || !parents.has(resource.idToString(g.parent)))
    )
      return false;
    return true;
  };
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Group> {
  /** Public so a client that deletes a group through an endpoint of its own can drop
   * the group from the cache. */
  readonly store: query.Table<Key, Group>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Group>({
      name: "groups",
      fetch: async (keys) => (await this.execRetrieve({ keys })).groups,
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, groupZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "group",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => (await this.execRetrieve(req)).groups,
        matches: (g, req) => requestFilter(req)(g),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  /**
   * Builds a request that creates a group. The Core derives the scope from the parent;
   * a group with no parent needs a scope.
   */
  createRequest(params: CreateParams): batch.Request<z.infer<typeof createReqZ>> {
    return {
      endpoint: "/group/create",
      payload: zod.parse(createReqZ, params, { label: "group create request" }),
    };
  }

  /** Builds a request that renames a group. Its cache write renames the cached one. */
  renameRequest(key: Key, name: string): batch.Request<z.infer<typeof renameReqZ>> {
    return {
      endpoint: "/group/rename",
      payload: { key, name },
      apply: () => query.partialUpdate(this.store, key, { name }),
    };
  }

  /**
   * Builds a request that deletes groups. The Core refuses to delete a non-empty group.
   * Its cache write drops the cached groups.
   */
  deleteRequest(keys: Key | Key[]): batch.Request<z.infer<typeof deleteReqZ>> {
    const payload = { keys: array.toArray(keys) };
    return {
      endpoint: "/group/delete",
      payload,
      apply: () => this.store.delete(payload.keys),
    };
  }

  /**
   * Builds a request that moves groups under the parent, or to the top level when the
   * parent is null or undefined. Its cache write moves the cached groups.
   */
  setParentRequest(
    keys: Key | Key[],
    parent?: resource.ID | null,
  ): batch.Request<z.infer<typeof setParentReqZ>> {
    const payload = { keys: array.toArray(keys), parent: parent ?? undefined };
    return {
      endpoint: "/group/set-parent",
      payload,
      apply: () => {
        const prev = payload.keys.map(
          (key) => [key, this.store.get(key)?.parent] as const,
        );
        payload.keys.forEach((key) =>
          query.partialUpdate(this.store, key, { parent: payload.parent }),
        );
        return () =>
          prev.forEach(([key, p]) =>
            query.partialUpdate(this.store, key, { parent: p }),
          );
      },
    };
  }

  /**
   * Creates a group. Give a parent to nest it under a group, project, or rack, or a
   * scope to create a top-level group.
   * @throws {ValidationError} if neither a parent nor a scope is given, or if the scope
   * does not match the parent.
   */
  async create(params: CreateParams): Promise<Group> {
    const { endpoint, payload } = this.createRequest(params);
    const res = await this.cfg.unary.send(endpoint, payload, createReqZ, createResZ);
    this.store.set(res.group);
    return res.group;
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    await this.send(this.renameRequest(key, name), renameReqZ, opts);
  }

  /**
   * Deletes the groups.
   * @throws {ValidationError} if a group holds members or groups the call does not also
   * delete.
   */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    await this.send(this.deleteRequest(keys), deleteReqZ, opts);
  }

  /**
   * Moves the groups under the parent, or to the top level when the parent is null or
   * undefined.
   * @throws {ValidationError} if the move makes a cycle or the scope of a group does
   * not match the parent.
   */
  async setParent(
    keys: Key | Key[],
    parent?: resource.ID | null,
    opts: query.WriteOptions = {},
  ): Promise<void> {
    await this.send(this.setParentRequest(keys, parent), setParentReqZ, opts);
  }

  /**
   * Retrieves every group on the parent chain above the groups, each once. The chain
   * stops at a project or rack, which the topmost group names as its parent.
   */
  async retrieveAncestors(keys: Key | Key[]): Promise<Group[]> {
    const since = this.store.stamp();
    const res = await this.execRetrieve({
      keys: array.toArray(keys),
      ancestors: true,
    });
    this.store.ingest([...res.groups, ...res.ancestors], { since });
    return res.ancestors;
  }

  private async send<P>(
    request: batch.Request<P>,
    reqZ: z.ZodType<P>,
    opts: query.WriteOptions,
  ): Promise<void> {
    const { endpoint, payload, apply } = request;
    await query.optimistic({
      rollbacks: apply == null ? [] : [apply()],
      onOptimistic: opts.onOptimistic,
      commit: async () => await this.cfg.unary.send(endpoint, payload, reqZ, emptyResZ),
    });
    apply?.();
  }

  private async execRetrieve(
    req: z.input<typeof wireRetrieveReqZ>,
  ): Promise<z.infer<typeof retrieveResZ>> {
    return await this.cfg.unary.send(
      "/group/retrieve",
      req,
      wireRetrieveReqZ,
      retrieveResZ,
    );
  }
}

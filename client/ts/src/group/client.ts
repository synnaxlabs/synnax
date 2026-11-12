// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, filter, zod } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import { type batch } from "@/batch";
import { createOf, reduceAll } from "@/group/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
  setParent,
} from "@/group/actions.gen";
import { type MemberType } from "@/group/members.gen";
import { isMember } from "@/group/move";
import {
  type Group,
  groupZ,
  type Key,
  keyZ,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/group/types.gen";
import { LINES } from "@/group/where";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/group/where.gen";
import { query } from "@/query";
import { resource } from "@/resource";

export const ACTION_CHANNEL_NAME = "sy_group_action_set";
export const DELETE_CHANNEL_NAME = "sy_group_delete";

const createReqZ = z.object({
  key: keyZ.optional(),
  name: z.string(),
  scope: resource.typeZ.optional(),
  parent: resource.idZ.optional(),
});
const createResZ = z.object({ group: groupZ });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);
const retrieveAncestorsReqZ = z.object({ keys: keyZ.array() });

export interface CreateParams extends z.input<typeof createReqZ> {}
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  batch: batch.Client;
  /**
   * Builds the requests that move records of each type into a group, or out of their
   * group when it is undefined.
   */
  setGroup: Record<MemberType, (keys: string[], to?: Key) => batch.Request[]>;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Group> {
  /** Public so a client that deletes a group through an endpoint of its own can drop
   * the group from the cache. */
  readonly store: query.Table<Key, Group>;
  private readonly dispatcher: actions.Controller<Key, Group, Action>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Group>({
      name: "groups",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a group reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Group, Action>({
      store,
      listed: (action) =>
        action.type === "set_parent" || actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/group/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "group",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (g, req) => matches(LINES, g, req.where),
        serverComputed: (req) => serverComputed(LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /**
   * Builds a request that creates a group. The Core derives the scope from the parent;
   * a group with no parent needs a scope. A request with a key and a scope writes the
   * group to the cache.
   */
  createRequest(params: CreateParams): batch.Request<z.infer<typeof createReqZ>> {
    const payload = zod.parse(createReqZ, params, { label: "group create request" });
    const { key, name, scope, parent } = payload;
    if (key == null || scope == null) return { endpoint: "/group/create", payload };
    return {
      endpoint: "/group/create",
      payload,
      apply: () => this.store.set({ key, name, scope, parent }),
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
   * Builds one batch entry per group that moves it under the parent, or to the top
   * level when the parent is null or undefined. The cache write of each entry moves its
   * cached group.
   */
  setParentRequest(
    keys: Key | Key[],
    parent?: resource.ID | null,
  ): batch.Request<actions.DispatchPayload<Key, Action>>[] {
    const action = setParent({ parent: parent ?? undefined });
    return array.toArray(keys).map((key) => ({
      endpoint: "/group/dispatch",
      payload: { key, actions: [action] },
      apply: () => {
        const prev = this.store.get(key)?.parent;
        query.partialUpdate(this.store, key, { parent: parent ?? undefined });
        return () => query.partialUpdate(this.store, key, { parent: prev });
      },
    }));
  }

  /**
   * Builds the requests that move resources under a parent: a group, project, or rack,
   * or the top of their tree when the parent is undefined. A group takes the parent as
   * its parent. Any other resource joins the parent when it is a group, and leaves its
   * group when it is not.
   * @throws {Error} if a resource is of a type that cannot sit in a group.
   */
  moveRequests(ids: resource.ID[], parent?: resource.ID): batch.Request[] {
    const byType = new Map<resource.Type, string[]>();
    ids.forEach(({ type, key }) =>
      byType.set(type, [...(byType.get(type) ?? []), key]),
    );
    const to = parent?.type === "group" ? parent.key : undefined;
    return [...byType].flatMap(([type, keys]) => {
      if (type === "group") return this.setParentRequest(keys, parent);
      if (!isMember(type)) throw new Error(`a ${type} cannot sit in a group`);
      return this.cfg.setGroup[type](keys, to);
    });
  }

  /**
   * Creates a group. Give a parent to nest it under a group, project, or rack, or a
   * scope to create a top-level group.
   * @throws {ValidationError} if neither a parent nor a scope is given, or if the scope
   * does not match the parent.
   */
  async create(params: CreateParams): Promise<Group> {
    const { endpoint, payload, apply } = this.createRequest(params);
    const { group } = await query.optimistic({
      rollbacks: apply == null ? [] : [apply()],
      commit: async () =>
        await this.cfg.unary.send(endpoint, payload, createReqZ, createResZ),
    });
    this.store.set(group);
    return group;
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached group until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Actions hold the snake_case form that the
   * Core stores. A refused send drops the actions from the cached group.
   * @throws {Error} if the cached group cannot hold an action, before any send.
   * @throws {ValidationError} if the group cannot hold an action.
   * @throws {NotFoundError} if the group does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
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
    await this.cfg.batch.send(this.setParentRequest(keys, parent), opts);
  }

  /**
   * Retrieves every group above the groups that is not one of them, each once. The
   * chain stops at a project or rack, which the topmost group names as its parent.
   */
  async retrieveAncestors(keys: Key | Key[]): Promise<Group[]> {
    return await this.store.track(async (since) => {
      const { groups } = await this.cfg.unary.send(
        "/group/retrieve-ancestors",
        { keys: array.toArray(keys) },
        retrieveAncestorsReqZ,
        retrieveResponseZ,
      );
      this.store.ingest(groups, { since });
      return groups;
    });
  }

  /**
   * Returns every cached group above the groups that is not one of them, each once, or
   * undefined when a group on the chain is not cached.
   */
  getCachedAncestors(keys: Key | Key[]): Group[] | undefined {
    const starts = array.toArray(keys);
    const visited = new Set<string>(starts);
    const ancestors: Group[] = [];
    for (const key of starts) {
      const start = this.store.get(key);
      if (start == null) return undefined;
      let parent = start.parent;
      while (parent?.type === "group" && !visited.has(parent.key)) {
        const g = this.store.get(parent.key);
        if (g == null) return undefined;
        visited.add(g.key);
        ancestors.push(g);
        parent = g.parent;
      }
    }
    return ancestors;
  }

  /**
   * Calls the handler with the cached ancestors of the groups after each change to a
   * cached group, since a rename or a move of any group can change them.
   */
  onAncestorsChange(
    keys: Key | Key[],
    handler: query.ChangeHandler<Group[]>,
  ): destructor.Destructor {
    return this.store.subscribe(() => handler(this.getCachedAncestors(keys)));
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

  private async execRetrieve(req: z.input<typeof retrieveRequestZ>): Promise<Group[]> {
    const res = await this.cfg.unary.send(
      "/group/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.groups;
  }
}

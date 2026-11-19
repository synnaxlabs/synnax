// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, caseconv, filter, id } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { query } from "@/query";
import { resource } from "@/resource";
import { createOf, reduceAll } from "@/secret/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/secret/actions.gen";
import {
  fileSecretZ,
  type Key,
  keyZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Secret,
  secretZ,
  storedSecretZ,
} from "@/secret/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/secret/where.gen";
import { task } from "@/task";

export const ACTION_CHANNEL_NAME = "sy_secret_action_set";
export const DELETE_CHANNEL_NAME = "sy_secret_delete";

const newStoredZ = storedSecretZ.omit({ updated: true, uses: true }).extend({
  /** value is the hidden value. No client can read it back. */
  value: z.string(),
});
const newFileZ = fileSecretZ.omit({ updated: true, uses: true });

/**
 * A secret to create, or to replace when its key exists. A stored secret carries its
 * value, which the Core encrypts and never returns.
 */
export const newZ = z.discriminatedUnion("source", [newStoredZ, newFileZ]);
export type New = z.input<typeof newZ>;

const setReqZ = z.object({ secrets: newZ.array() });
const setResZ = z.object({ secrets: secretZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const grantKeyZ = z.uuid();
/** GrantKey is the key of a grant that lets a subject read secrets once. */
export type GrantKey = z.infer<typeof grantKeyZ>;
const grantCreateReqZ = z.object({
  subject: resource.idZ,
  secrets: keyZ.array(),
  command: task.commandZ,
});

/** A grant to create. */
export interface GrantParams {
  /** subject is the rack that may run the command with the secrets. */
  subject: resource.ID;
  /** secrets are the secrets that the rack reads to run the command. */
  secrets: Key[];
  /**
   * command is the command that the rack runs once. Its task must be on the rack. The
   * rack runs this command, not the arguments of the command that carries the grant.
   */
  command: task.NewCommand;
}
const grantCreateResZ = z.object({ grant: grantKeyZ });

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

export type RetrieveSingleParams = {
  key: Key;
  /**
   * Fills `uses` with the resources that use the secret and that the caller can read.
   */
  includeUses?: boolean;
};
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  /**
   * The channels that announce changes to resources that can use a secret. A change on
   * one refetches the uses the client holds.
   */
  usesChannels: string[];
}

export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Secret,
  Secret,
  RetrieveSingleParams,
  RetrieveSingleParams
> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Secret>;
  private readonly dispatcher: actions.Controller<Key, Secret, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Secret>({
      name: "secrets",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const uses = new query.Uses<Key, Secret, RetrieveRequest>({
      name: "secret",
      cache,
      table: store,
      key: keyZ,
      retrieve: async (req) => await this.execRetrieve(req),
      request: (key) => ({ where: { key }, includeUses: true }),
      channels: cfg.usesChannels,
    });
    // Every write to a secret reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Secret, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/secret/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "secret",
      table: store,
      shortcut: shortcutZ,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await uses.fetch(req),
        // A secret whose uses the client does not hold fetches the query again.
        matches: (secret, req) =>
          uses.lacks(secret, req) ? null : matches(secret, req.where),
        serverComputed: (req) => serverComputed(req.where),
        triggers: [uses.trigger],
      },
      compose: (record, q) => uses.compose(record, q),
      single: uses.single,
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /**
   * Creates secrets, or replaces the secrets whose keys exist.
   * @returns the saved secrets, without their values.
   * @throws {PathError} if a secret is not valid, such as a stored secret without a
   * value, or if a replace changes the source of a secret.
   * @throws {AccessDeniedError} if the caller cannot create secrets, or cannot update a
   * secret that the call replaces.
   */
  async set(secret: New): Promise<Secret>;
  async set(secrets: New[]): Promise<Secret[]>;
  async set(secrets: New | New[]): Promise<Secret | Secret[]> {
    const isMany = Array.isArray(secrets);
    const res = await this.cfg.unary.send(
      "/secret/set",
      { secrets: array.toArray(secrets) },
      setReqZ,
      setResZ,
    );
    this.store.set(res.secrets);
    return isMany ? res.secrets : res.secrets[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached secret until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Path actions edit only the name.
   * @throws {Error} if the cached secret cannot hold an action, before any send.
   * @throws {ValidationError} if the secret cannot hold an action.
   * @throws {NotFoundError} if the secret does not exist.
   * @throws {AccessDeniedError} if the caller cannot update the secret.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  /**
   * Deletes secrets. Unknown keys are ignored.
   * @throws {ValidationError} if a resource uses one of the secrets.
   * @throws {AccessDeniedError} if the caller cannot delete one of the secrets.
   */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/secret/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /**
   * Lets a subject run a command once with secrets, within a minute, such as for a rack
   * to test a connection before the device that uses them is saved. Only racks can be
   * subjects.
   * @returns the key of the grant to send to the task of the command.
   * @throws {AccessDeniedError} if the caller cannot create devices or cannot retrieve
   * one of the secrets.
   * @throws {NotFoundError} if a secret or the task of the command does not exist.
   * @throws {PathError} if the task of the command is not on the subject rack.
   */
  async grant({ command, ...rest }: GrantParams): Promise<GrantKey> {
    // A task reads args in snake case, as the command writer sends them.
    const args = caseconv.camelToSnake(command.args ?? {});
    const res = await this.cfg.unary.send(
      "/secret/grant/create",
      { ...rest, command: { ...command, args, key: id.create() } },
      grantCreateReqZ,
      grantCreateResZ,
    );
    return res.grant;
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Secret[]> {
    const res = await this.cfg.unary.send(
      "/secret/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.secrets;
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, filter, sync } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { isConnectionError, NotFoundError } from "@/errors";
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
  type Base,
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
const grantCreateReqZ = z.object({ subject: resource.idZ, secrets: keyZ.array() });
export interface GrantParams extends z.input<typeof grantCreateReqZ> {}
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

/** Drops includeUses when unset so equivalent queries hash identically. */
const normalizeSingle = ({
  key,
  includeUses,
}: RetrieveSingleParams): RetrieveSingleParams =>
  includeUses === true ? { key, includeUses } : { key };

const singleQueryZ = z
  .union([
    z.strictObject({ key: keyZ, includeUses: z.boolean().optional() }),
    keyZ.transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

/** The resources that use a secret, keyed by the secret. */
type Uses = Pick<Base, "key" | "uses">;

/** The deletes of secrets that land while fetches of uses run. */
interface Deletes {
  inFlight: number;
  seq: number;
  at: Map<Key, number>;
}

const stripUses = ({ uses: _, ...secret }: Secret): Secret => secret;
const toUses = ({ key, uses }: Secret): Uses => ({ key, uses });

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
  private readonly usesStore: query.Table<Key, Uses>;
  private readonly deletes: Deletes;
  private readonly usesChanges: { count: number };
  private readonly refetchUses: () => void;
  private readonly dispatcher: actions.Controller<Key, Secret, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Secret>({
      name: "secrets",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // The secret table never holds uses. A retrieve that asks for them reads this one.
    const usesStore = cache.createTable<Key, Uses>({
      name: "secret.uses",
      fetch: async (keys) =>
        (
          await this.retrieveUses({ ...query.keysRequest(keys), includeUses: true })
        ).map(toUses),
    });
    // An answer that holds a secret composes its uses too.
    usesStore.hold({ held: () => store.held() });
    const deletes: Deletes = { inFlight: 0, seq: 0, at: new Map() };
    // A secret created again under a deleted key reads its uses again.
    store.subscribe((event) => {
      if (event.variant !== "delete") return;
      usesStore.evict(event.key);
      if (deletes.inFlight > 0) deletes.at.set(event.key, ++deletes.seq);
    });
    // The Core announces no change to uses. Refetches run outside the change stream and
    // one at a time, so an older reply cannot overwrite a newer one. A change that
    // arrives while a refetch waits joins it.
    const refetchMu = sync.newMutex({ waiting: false });
    const refetchUses = () => {
      if (refetchMu.waiting) return;
      refetchMu.waiting = true;
      refetchMu
        .runExclusive(async () => {
          refetchMu.waiting = false;
          await usesStore.reconcile();
        })
        .catch((exc: unknown) => {
          // The reconcile after the next reconnect fetches the uses again.
          if (isConnectionError(exc)) return;
          cache.onError(new Error("failed to refetch secret uses", { cause: exc }));
        });
    };
    const usesChanges = { count: 0 };
    cfg.usesChannels.forEach((channel) =>
      cache.listen({
        channel,
        schema: z.unknown(),
        onChange: () => {
          usesChanges.count++;
          refetchUses();
        },
      }),
    );
    const usesTrigger = query.trigger(
      usesStore,
      (event, q: RetrieveRequest | RetrieveSingleParams) =>
        q.includeUses === true ? [event.key] : null,
      { contentOnly: true },
    );
    const single = cache.queries<RetrieveSingleParams, Secret, Key, Secret>({
      name: "secret",
      table: store,
      fetch: async (q) => [await this.fetchSingle(q)],
      compose: (records, q) => this.compose(records[0], q.includeUses === true),
      keyOf: (q) => q.key,
      covers: (_, q) => q.includeUses !== true,
      triggers: [usesTrigger],
      single: true,
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
        fetch: async (req) => await this.fetchThrough(req),
        // A secret whose uses the client does not hold fetches the query again.
        matches: (secret, req) =>
          req.includeUses === true && usesStore.get(secret.key) == null
            ? null
            : matches(secret, req.where),
        serverComputed: (req) => serverComputed(req.where),
        triggers: [usesTrigger],
      },
      compose: (record, q) =>
        this.compose(record, (q as { includeUses?: boolean }).includeUses === true),
      single: { schema: singleQueryZ, space: single },
    });
    this.cfg = cfg;
    this.store = store;
    this.usesStore = usesStore;
    this.deletes = deletes;
    this.usesChanges = usesChanges;
    this.refetchUses = refetchUses;
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
   * Lets a subject read secrets once, within a minute, such as for a rack to test a
   * connection before the device that uses them is saved. Only racks can be subjects.
   * @returns the key of the grant to send to the subject with the command.
   * @throws {AccessDeniedError} if the caller cannot create devices or cannot retrieve
   * one of the secrets.
   * @throws {NotFoundError} if a secret does not exist.
   */
  async grant(params: GrantParams): Promise<GrantKey> {
    const res = await this.cfg.unary.send(
      "/secret/grant/create",
      params,
      grantCreateReqZ,
      grantCreateResZ,
    );
    return res.grant;
  }

  private compose(secret: Secret, includeUses: boolean): Secret {
    if (!includeUses) return secret;
    return { ...secret, uses: this.usesStore.get(secret.key)?.uses };
  }

  private async fetchThrough(req: RetrieveRequest): Promise<Secret[]> {
    if (req.includeUses !== true) return await this.execRetrieve(req);
    const changes = this.usesChanges.count;
    const secrets = await this.retrieveUses(req);
    this.usesStore.set(secrets.map(toUses));
    // A refetch that a change started can miss these keys or land first.
    if (this.usesChanges.count !== changes) this.refetchUses();
    return secrets.map(stripUses);
  }

  // A delete of a secret that a retrieve returns makes its uses stale, so the retrieve
  // runs again.
  private async retrieveUses(req: RetrieveRequest): Promise<Secret[]> {
    const { deletes } = this;
    const since = deletes.seq;
    deletes.inFlight++;
    let secrets: Secret[];
    try {
      secrets = await this.execRetrieve(req);
    } finally {
      deletes.inFlight--;
    }
    const overtaken = secrets.some(({ key }) => (deletes.at.get(key) ?? 0) > since);
    if (deletes.inFlight === 0) deletes.at.clear();
    return overtaken ? await this.retrieveUses(req) : secrets;
  }

  private async fetchSingle({ key, includeUses }: RetrieveSingleParams): Promise<Key> {
    if (includeUses === true)
      return await this.store.track(async (since) => {
        const secrets = await this.fetchThrough({
          where: { key: { eq: key } },
          includeUses,
        });
        if (secrets.length === 0) {
          this.store.ingestDeletes([key], since);
          throw new NotFoundError(`secret with key ${key} not found`);
        }
        this.store.ingest(secrets, { since });
        return key;
      });
    if ((await this.store.retrieve([key])).length === 0)
      throw new NotFoundError(`secret with key ${key} not found`);
    return key;
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

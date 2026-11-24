// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { UnaryClient } from "@synnaxlabs/freighter";
import { array, filter, zod } from "@synnaxlabs/x";
import z from "zod";

import { actions } from "@/actions";
import { createOf, encode, reduceAll } from "@/calculation/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
} from "@/calculation/actions.gen";
import {
  type Calculation,
  calculationZ,
  type Key,
  keyZ,
  type New,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  statusZ,
} from "@/calculation/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/calculation/where.gen";
import type { channel } from "@/channel";
import { group } from "@/group";
import { query } from "@/query";
import type { status } from "@/status";

export const ACTION_CHANNEL_NAME = "sy_calculation_action_set";
export const DELETE_CHANNEL_NAME = "sy_calculation_delete";

const inputZ = z.object({
  calculation: calculationZ,
  name: z.string(),
  group: group.keyZ.optional(),
});
const createReqZ = z.object({ calculations: inputZ.array(), dispatchKey: z.string() });
const createResZ = z.object({ calculations: calculationZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveMultiParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

/** Names one calculation. */
export type RetrieveSingleParams = { key: Key };
/** Everything a multi-calculation retrieval can filter on. */
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
/** Params for a calculation retrieval, single or multiple. */
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

/**
 * A calculation to create. With an output, the calculation writes that virtual channel.
 * Without one, the create makes a virtual output with name in group, typed by the
 * expression.
 */
export interface CreateInput extends Omit<New, "output"> {
  output?: channel.Key;
  name?: string;
  group?: group.Key;
}

/** A calculation as the cache holds it. The status table is the home of its status. */
type Stored = Omit<Calculation, "status">;

/** The calculation that writes the channel with key, or null when none does. */
export interface Output {
  key: channel.Key;
  calculation: Key | null;
}

const stripStatus = ({ status: _, ...calc }: Calculation): Stored => calc;

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  statuses: status.Client;
}

/**
 * Creates, reads, changes, and deletes calculations on a Core. Reach it through
 * `client.calculations`.
 */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Stored,
  Calculation
> {
  readonly type: string = "calculation";
  private readonly unary: UnaryClient;
  private readonly statuses: status.Client;
  private readonly store: query.Table<Key, Stored>;
  /**
   * Maps each channel to the calculation that writes it. Its fetch merges the lookups
   * of a tick into one retrieve by output. A lookup that answers fetches again each
   * channel whose last lookup failed. An absent entry is not yet loaded.
   */
  readonly outputs: query.Table<channel.Key, Output>;
  private readonly dispatcher: actions.Controller<Key, Stored, Action>;

  constructor({ unary, cache, statuses }: ClientConfig) {
    const store = cache.createTable<Key, Stored>({
      name: "calculations",
      fetch: async (keys) =>
        (await this.execRetrieve(query.keysRequest(keys))).map(stripStatus),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Stored, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(unary, "/calculation/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      encode,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    const failed = new Set<channel.Key>();
    const outputs: query.Table<channel.Key, Output> = cache.createTable<
      channel.Key,
      Output
    >({
      name: "calculation.outputs",
      fetch: async (channels) => {
        let answered: Output[] | undefined;
        try {
          // A reply older than a delete, a reset, or a newer write leaves the record as
          // the table holds it, and maps no channel to a record the table dropped.
          answered = await store.track(async (since) => {
            const calcs = await this.execRetrieve({
              where: { output: { in: channels } },
            });
            store.ingest(calcs.map(stripStatus), { since });
            const writers = new Map(
              calcs.flatMap(({ key, output }) =>
                store.has(key) ? [[output, key]] : [],
              ),
            );
            return channels.map((key) => ({
              key,
              calculation: writers.get(key) ?? null,
            }));
          });
        } finally {
          if (answered == null) channels.forEach((key) => failed.add(key));
        }
        if (failed.size > 0) {
          const asked = new Set(channels);
          const retry = [...failed].filter((key) => !asked.has(key));
          failed.clear();
          outputs.refetch(retry);
        }
        return answered;
      },
    });
    // The output of a calculation never changes, so a set or a delete is all that moves
    // a channel between written and not written. A delete clears only the outputs that
    // still name the deleted calculation.
    store.subscribe((event) => {
      if (event.variant === "set") {
        outputs.set({ key: event.value.output, calculation: event.key });
        return;
      }
      const named = outputs.get((output) => output.calculation === event.key);
      if (named.length > 0)
        outputs.set(named.map(({ key }) => ({ key, calculation: null })));
    });
    statuses.holdOwned(store, resourceID);
    super(cache, {
      name: "calculation",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => (await this.execRetrieve(req)).map(stripStatus),
        matches: (calc, req) => matches(calc, req.where),
        serverComputed: (req) => serverComputed(req.where),
        triggers: [
          query.trigger(
            statuses.store,
            (event, req: RetrieveRequest) =>
              req.includeStatus === true
                ? statuses.ownerKeys(event, "calculation")
                : null,
            { contentOnly: true },
          ),
        ],
      },
      compose: (calc, req) =>
        this.compose(calc, (req as RetrieveRequest).includeStatus === true),
    });
    this.unary = unary;
    this.statuses = statuses;
    this.store = store;
    this.outputs = outputs;
    this.dispatcher = dispatcher;
  }

  /**
   * Creates the calculations. Each input names its output, or holds the name, and the
   * group if any, of an output for the create to make.
   * @throws {ValidationError} if a calculation is not valid, if an input names an
   * output and also holds a name or a group, or if it holds neither.
   */
  async create(input: CreateInput): Promise<Calculation>;
  async create(inputs: CreateInput[]): Promise<Calculation[]>;
  async create(
    inputs: CreateInput | CreateInput[],
  ): Promise<Calculation | Calculation[]> {
    const isMany = Array.isArray(inputs);
    const parsed = array.toArray(inputs).map(({ output, name, group, ...rest }) => ({
      calculation: zod.parse(
        calculationZ,
        { ...rest, output: output ?? 0 },
        { label: "calculation" },
      ),
      name: name ?? "",
      group,
    }));
    const res = await this.dispatcher.create(
      parsed.map(({ calculation }) => calculation.key),
      async (dispatchKey) =>
        await this.unary.send(
          "/calculation/create",
          { calculations: parsed, dispatchKey },
          createReqZ,
          createResZ,
        ),
    );
    this.store.set(res.calculations.map(stripStatus));
    return isMany ? res.calculations : res.calculations[0];
  }

  /**
   * Applies actions to the cached calculation and sends them to the Core. Returns false
   * without side effects when the calculation is not cached.
   * @throws {Error} if the calculation cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached calculation drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  /** Deletes the calculations with keys. Their outputs stay as channels. */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.unary.send(
          "/calculation/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Returns cached with its cached status when includeStatus is true. */
  private compose(cached: Stored, includeStatus: boolean): Calculation {
    if (!includeStatus) return cached;
    const parsed = statusZ.safeParse(this.statuses.ownedBy(resourceID(cached.key)));
    return parsed.success ? { ...cached, status: parsed.data } : cached;
  }

  /** Fetches calculations and saves the statuses the Core returns. */
  private async execRetrieve(params: RetrieveRequest): Promise<Calculation[]> {
    const { store } = this.statuses;
    // A status that streams in during the fetch is newer than the fetched one.
    return await store.track(async (since) => {
      const res = await this.unary.send(
        "/calculation/retrieve",
        params,
        retrieveRequestZ,
        retrieveResponseZ,
      );
      res.calculations.forEach(({ status }) => {
        if (status != null) store.ingest(status, { since });
      });
      return res.calculations;
    });
  }
}

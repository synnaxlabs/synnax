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
  type HiddenOutput,
  hiddenOutputZ,
  type Key,
  keyZ,
  type New,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  type RetrieveResponse,
  retrieveResponseZ,
  type Status,
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
export const SET_CHANNEL_NAME = "sy_calculation_set";
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

/**
 * Whether a calculation writes the channel with key. A hidden output has a calculation
 * that the reader cannot retrieve. Otherwise calculation is the one that writes the
 * channel, or null when none does.
 */
export type Output =
  | { key: channel.Key; hidden: true }
  | { key: channel.Key; hidden: false; calculation: Key | null };

/** @returns whether a calculation writes the channel of output, hidden or not. */
export const isOutput = (output: Output): boolean =>
  output.hidden || output.calculation != null;

/**
 * @returns the key of the calculation that writes the channel of output, or null when
 * none does or the reader cannot retrieve it.
 */
export const keyOf = (output: Output): Key | null =>
  output.hidden ? null : output.calculation;

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
        (await this.execRetrieve(query.keysRequest(keys))).calculations.map(
          stripStatus,
        ),
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
            const { calculations, hiddenOutputs } = await this.execRetrieve({
              where: { output: { in: channels } },
            });
            store.ingest(calculations.map(stripStatus), { since });
            const writers = new Map(
              calculations.flatMap(({ key, output }) =>
                store.has(key) ? [[output, key]] : [],
              ),
            );
            const hidden = new Set(hiddenOutputs);
            return channels.map((key): Output =>
              hidden.has(key)
                ? { key, hidden: true }
                : { key, hidden: false, calculation: writers.get(key) ?? null },
            );
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
        const output = {
          key: event.value.output,
          hidden: false,
          calculation: event.key,
        };
        // A calculation that a fetch found is new to the client, not to the Core.
        if (event.discovered === true) outputs.ingest(output);
        else outputs.set(output);
        return;
      }
      const named = outputs.get(
        (output) => !output.hidden && output.calculation === event.key,
      );
      if (named.length > 0)
        outputs.set(
          named.map(({ key }) => ({ key, hidden: false, calculation: null })),
        );
    });
    // The set channel tells a reader the outputs of the calculations that it cannot
    // retrieve. Every other change reaches the store through the action channel.
    cache.listen({
      channel: SET_CHANNEL_NAME,
      schema: hiddenOutputZ.or(z.object({ key: keyZ })),
      onChange: (changed) => {
        const hidden = changed.filter((c): c is HiddenOutput => "hidden" in c);
        if (hidden.length === 0) return;
        const named = new Set(hidden.map(({ output }) => output));
        const held = store.get((calc) => named.has(calc.output));
        if (held.length > 0) store.mirrorDelete(held.map(({ key }) => key));
        outputs.set(
          hidden.map(({ output, hidden }): Output =>
            hidden
              ? { key: output, hidden: true }
              : { key: output, hidden: false, calculation: null },
          ),
        );
      },
    });
    statuses.holdOwned(store, resourceID);
    super(cache, {
      name: "calculation",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) =>
          (await this.execRetrieve(req)).calculations.map(stripStatus),
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

  /**
   * Retrieves the status of the calculation that writes the channel with key output.
   * Lookups made in the same few milliseconds share one request for the outputs and one
   * for the statuses.
   * @returns the status, or null when no calculation that the reader can retrieve
   * writes the channel, or the calculation has no status.
   */
  async retrieveStatus(output: channel.Key): Promise<Status | null> {
    const [writer] = await this.outputs.retrieve([output]);
    const calc = keyOf(writer);
    if (calc == null) return null;
    await this.statuses.retrieveOwned([resourceID(calc)]);
    return this.statusOf(calc) ?? null;
  }

  /**
   * Returns the cached status of the calculation that writes the channel with key
   * output, or null when no calculation that the reader can retrieve writes it. Returns
   * undefined when the cache cannot tell.
   */
  getCachedStatus(output: channel.Key): Status | null | undefined {
    const writer = this.outputs.get(output);
    if (writer == null) return undefined;
    const calc = keyOf(writer);
    return calc == null ? null : this.statusOf(calc);
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
    const status = this.statusOf(cached.key);
    return status == null ? cached : { ...cached, status };
  }

  // A cached status that does not parse as a calculation status is left out.
  private statusOf(key: Key): Status | undefined {
    const parsed = statusZ.safeParse(this.statuses.ownedBy(resourceID(key)));
    return parsed.success ? parsed.data : undefined;
  }

  /** Fetches calculations and saves the statuses the Core returns. */
  private async execRetrieve(params: RetrieveRequest): Promise<RetrieveResponse> {
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
      return res;
    });
  }
}

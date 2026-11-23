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
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
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

/** Config for {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

/**
 * Creates, reads, changes, and deletes calculations on a Core. Reach it through
 * `client.calculations`.
 */
export class Client extends query.Retriever<
  typeof retrieveMultiParamsZ,
  Key,
  Calculation
> {
  readonly type: string = "calculation";
  private readonly unary: UnaryClient;
  private readonly store: query.Table<Key, Calculation>;
  private readonly dispatcher: actions.Controller<Key, Calculation, Action>;

  constructor({ unary, cache }: ClientConfig) {
    const store = cache.createTable<Key, Calculation>({
      name: "calculations",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Calculation, Action>({
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
    super(cache, {
      name: "calculation",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (calc, req) => matches(calc, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.unary = unary;
    this.store = store;
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
    this.store.set(res.calculations);
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

  private async execRetrieve(params: RetrieveRequest): Promise<Calculation[]> {
    const res = await this.unary.send(
      "/calculation/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.calculations;
  }
}

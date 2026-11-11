// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type FileTransport,
  type UnaryClient,
  type UploadBody,
} from "@synnaxlabs/freighter";
import { array, filter } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { group } from "@/group";
import { imex } from "@/imex";
import { query } from "@/query";
import { createOf, reduceAll } from "@/schematic/symbol/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/schematic/symbol/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Symbol,
  symbolZ,
} from "@/schematic/symbol/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/schematic/symbol/where.gen";

export const ACTION_CHANNEL_NAME = "sy_schematic_symbol_action_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_symbol_delete";

const createReqZ = z.object({ symbols: symbolZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

export type RetrieveSingleParams = { key: Key };
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

const createResZ = z.object({ symbols: symbolZ.array() });
const emptyResZ = z.object({});
const exportGroupReqZ = z.object({ key: group.keyZ, encoding: imex.encodingZ });
const importGroupResZ = z.object({ group: group.groupZ });
const deleteGroupReqZ = z.object({ key: group.keyZ });

/** A symbol to create. Its group places it; with no group it sits in none. */
export interface CreateParams extends New {}

export interface CreateMultipleParams {
  symbols: New[];
}

export interface ClientConfig {
  unary: UnaryClient;
  file: FileTransport;
  cache: query.Cache;
  groupStore: query.Table<group.Key, group.Group>;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Symbol> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Symbol>;
  private readonly dispatcher: actions.Controller<Key, Symbol, Action>;
  /** Builds requests that move symbols between groups. */
  readonly setGroupRequest: group.MoveRequest<Key>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Symbol>({
      name: "schematicSymbols",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a symbol reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Symbol, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(
        cfg.unary,
        "/schematic/symbol/dispatch",
        dispatchReqZ,
        dispatchResZ,
      ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "schematic symbol",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (sym, req) => matches(group.MEMBER_LINES, sym, req.where),
        serverComputed: (req) => serverComputed(group.MEMBER_LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
    this.setGroupRequest = group.createMoveRequest(
      "/schematic/symbol/dispatch",
      (keys, to) => group.moveCached(store, keys, to),
    );
  }

  async create(options: CreateParams): Promise<Symbol>;
  async create(options: CreateMultipleParams): Promise<Symbol[]>;
  async create(
    options: CreateParams | CreateMultipleParams,
  ): Promise<Symbol | Symbol[]> {
    const isMany = "symbols" in options;
    const symbols = isMany ? options.symbols : [options];
    const res = await this.cfg.unary.send(
      "/schematic/symbol/create",
      { symbols },
      createReqZ,
      createResZ,
    );
    this.store.set(res.symbols);
    return isMany ? res.symbols : res.symbols[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached symbol until the Core echoes them, and sends them
   * to the Core to apply in one transaction. Actions hold the snake_case form that the
   * Core stores. A refused send drops the actions from the cached symbol.
   * @throws {Error} if the cached symbol cannot hold an action, before any send.
   * @throws {ValidationError} if the symbol cannot hold an action.
   * @throws {NotFoundError} if the symbol does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => this.store.delete(keysArr);
    await query.optimistic({
      rollbacks: [drop()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/schematic/symbol/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /**
   * Exports every symbol in the group as a bundle: a zip archive holding one JSON file
   * per symbol beside a manifest.json naming the group. Two symbols that take the same
   * file name keep distinct names through a numeric suffix, and children that are not
   * symbols are skipped. The caller pipes the stream wherever it likes without the
   * client buffering the whole archive.
   * @param options - the export options, including the serialization member files are
   * written in.
   * @returns the bundle as a stream of zip bytes.
   */
  async exportGroup(
    key: group.Key,
    options: imex.Options,
  ): Promise<ReadableStream<Uint8Array>> {
    return await this.cfg.file.download(
      "/schematic/symbol/group/export",
      { key, encoding: options.encoding },
      exportGroupReqZ,
      { encoding: "ZIP" },
    );
  }

  /**
   * Imports a symbol group bundle: a zip archive holding one JSON envelope per symbol
   * beside a manifest.json naming the group. The Core creates a fresh top-level symbol
   * group and imports every member in a single transaction, so a failure leaves nothing
   * behind.
   * @returns the created group.
   * @throws {ValidationError} if the manifest is missing, malformed, or of another
   * bundle kind, if two member names collide, or if a member is not a symbol.
   */
  async importGroup(data: UploadBody): Promise<group.Group> {
    const res = await this.cfg.file.upload(
      "/schematic/symbol/group/import",
      data,
      { encoding: "ZIP" },
      importGroupResZ,
    );
    this.cfg.groupStore.set(res.group);
    return res.group;
  }

  /**
   * Deletes the group and every symbol in it. The Core removes both in a single
   * transaction, so a failure leaves the group and its symbols untouched.
   */
  async deleteGroup(key: group.Key, opts: query.WriteOptions = {}): Promise<void> {
    const memberKeys = this.store.get((s) => s.group === key).map((s) => s.key);
    const drop = () => [this.store.delete(memberKeys), this.cfg.groupStore.delete(key)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/schematic/symbol/group/delete",
          { key },
          deleteGroupReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Symbol[]> {
    const res = await this.cfg.unary.send(
      "/schematic/symbol/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.symbols;
  }
}

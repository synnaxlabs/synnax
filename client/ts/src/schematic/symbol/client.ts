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
import { array, primitive } from "@synnaxlabs/x";
import { z } from "zod";

import { group } from "@/group";
import { imex } from "@/imex";
import { query } from "@/query";
import {
  type Key,
  keyZ,
  type New,
  type Symbol,
  symbolZ,
} from "@/schematic/symbol/types.gen";

export const SET_CHANNEL_NAME = "sy_schematic_symbol_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_symbol_delete";

const createReqZ = z.object({ symbols: symbolZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveRequestZ = group.memberFilterZ.extend({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
});
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ symbols: symbolZ.array().default(() => []) });
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

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm"] as const;

/**
 * Client-side matching for a request: key and group sets. Server-computed shapes
 * (search) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((s: Symbol) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const inGroup = group.createMemberMatcher(req);
  return (s) => (keySet == null || keySet.has(s.key)) && inGroup(s);
};

export interface ClientConfig {
  unary: UnaryClient;
  file: FileTransport;
  cache: query.Cache;
  groupStore: query.Table<group.Key, group.Group>;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Symbol> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Symbol>;
  /** Builds requests that move symbols between groups. */
  readonly setGroupRequest: group.MoveRequest<Key>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Symbol>({
      name: "schematicSymbols",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, symbolZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "schematic symbol",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (sym, req) => requestFilter(req)(sym),
        serverFields: SERVER_FIELDS,
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.setGroupRequest = group.createMoveRequest(
      "/schematic/symbol/set-group",
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

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => query.partialUpdate(this.store, key, { name });
    await query.optimistic({
      rollbacks: [rename()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/schematic/symbol/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
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

  private async execRetrieve(params: RetrieveParams): Promise<Symbol[]> {
    const res = await this.cfg.unary.send(
      "/schematic/symbol/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.symbols;
  }
}

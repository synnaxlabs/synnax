// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, caseconv, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { createOf, encode, reduceAll } from "@/icd/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/icd/actions.gen";
import {
  type ICD,
  icdZ,
  type Key,
  keyZ,
  type New,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/icd/types.gen";
import {
  compareCursors,
  cursor,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/icd/where.gen";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_icd_action_set";
export const DELETE_CHANNEL_NAME = "sy_icd_delete";

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
]);
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

const createReqZ = z.object({ icds: icdZ.array(), dispatchKey: z.string() });
const createResZ = z.object({ icds: icdZ.array() });
/** One rule an ICD breaks. */
export const problemZ = z.object({
  /**
   * The dotted location of the bad value, such as "messages.0.name", or empty for a
   * problem with the whole ICD.
   */
  path: z.string(),
  /** What is wrong. */
  message: z.string(),
});
export interface Problem extends z.infer<typeof problemZ> {}

const checkReqZ = z.object({ icd: icdZ });
const checkResZ = z.object({ problems: problemZ.array() });

/**
 * Words an issue with a rule of the ICD schema as a sentence that names its field, as
 * the Core words the problems with its schema.
 */
const issueMessage = (issue: z.core.$ZodIssue): string => {
  const name = issue.path.findLast((p) => typeof p === "string") ?? "";
  const field = caseconv.toSentence(name).toLowerCase();
  if (issue.code === "too_small")
    return issue.origin === "string"
      ? `${field} is required`
      : `${field} must be greater than or equal to ${issue.minimum}`;
  if (issue.code === "too_big")
    return `${field} must be less than or equal to ${issue.maximum}`;
  return `${field} is not valid`;
};
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

/** The file formats an interface control document is imported from. */
export const IMPORT_FORMATS = ["dbc", "csv", "xlsx"] as const;
export const importFormatZ = z.enum(IMPORT_FORMATS);
export type ImportFormat = z.infer<typeof importFormatZ>;

const importReqZ = z.object({
  key: keyZ,
  format: importFormatZ,
  data: z.string(),
  dispatchKey: z.string(),
});
const importResZ = z.object({ icd: icdZ });

// btoa takes a binary string, and spreading a large array into fromCharCode overflows
// the stack, so the bytes convert in chunks.
const BASE64_CHUNK_SIZE = 0x8000;

const encodeBase64 = (data: Uint8Array): string => {
  let binary = "";
  for (let i = 0; i < data.length; i += BASE64_CHUNK_SIZE)
    binary += String.fromCharCode(...data.subarray(i, i + BASE64_CHUNK_SIZE));
  return btoa(binary);
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, ICD> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, ICD>;
  private readonly dispatcher: actions.Controller<Key, ICD, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, ICD>({
      name: "icds",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to an ICD reaches the client as actions.
    const dispatcher = new actions.Controller<Key, ICD, Action>({
      store,
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/icd/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      encode,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "icd",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (icd, req) => matches(icd, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  /**
   * Creates the given ICDs, or replaces those whose keys already exist. The Core gives
   * keys to messages and fields that have none.
   * @throws {ValidationError} if an ICD or one of its messages has a bad shape.
   */
  async create(icd: New, opts?: query.WriteOptions<ICD[]>): Promise<ICD>;
  async create(icds: New[], opts?: query.WriteOptions<ICD[]>): Promise<ICD[]>;
  async create(
    icds: New | New[],
    opts: query.WriteOptions<ICD[]> = {},
  ): Promise<ICD | ICD[]> {
    const isMany = Array.isArray(icds);
    const optimistic = array
      .toArray(icds)
      .map((i) => zod.parse(icdZ, i, { label: "icd" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/icd/create",
              { icds: optimistic, dispatchKey },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.icds);
    return isMany ? res.icds : res.icds[0];
  }

  /**
   * Checks an ICD for problems, such as two fields that use the same bits. The Core
   * stores an ICD that has problems.
   * @returns Every problem in the ICD, or an empty array when it has none. Each path is
   * dotted and camel case, such as "messages.0.payload.fields.1.bitLength". An ICD that
   * breaks a rule of its schema returns only those problems, as the Core cannot read
   * it.
   */
  async check(icd: New): Promise<Problem[]> {
    const parsed = icdZ.safeParse(icd);
    if (!parsed.success)
      return parsed.error.issues.map((issue) => ({
        path: issue.path.join("."),
        message: issueMessage(issue),
      }));
    const res = await this.cfg.unary.send(
      "/icd/check",
      { icd: parsed.data },
      checkReqZ,
      checkResZ,
    );
    return res.problems.map((p) => ({
      ...p,
      path: caseconv
        .snakeToCamelPath(p.path.split("."), { schema: icdZ })
        .path.join("."),
    }));
  }

  /**
   * Replaces the messages of an ICD with those parsed from an interface control
   * document. Messages and fields that match by name keep their keys.
   * @param key - The key of the ICD to import into.
   * @param data - The raw bytes of the document.
   * @param format - The file format of the document.
   * @returns The ICD as the Core stored it.
   * @throws {ValidationError} if the format is unknown or the document is invalid.
   */
  async import(key: Key, data: Uint8Array, format: ImportFormat): Promise<ICD> {
    const res = await this.dispatcher.create(
      [key],
      async (dispatchKey) =>
        await this.cfg.unary.send(
          "/icd/import",
          { key, format, data: encodeBase64(data), dispatchKey },
          importReqZ,
          importResZ,
        ),
    );
    this.store.set([res.icd]);
    return res.icd;
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatcher.send(key, [set({ path: ["name"], value: name })]);
  }

  /**
   * Applies actions to the cached ICD and sends them to the Core. Paths and values take
   * the camelCase form of the cached ICD. Returns false without side effects when the
   * ICD is not cached.
   * @throws {Error} if the ICD cannot hold an action, before any send.
   * @throws {Error} if the send fails, after the cached ICD drops the actions.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions));
  }

  /** Stages actions that commit atomically as one dispatch. */
  beginTransaction(key: Key): actions.Transaction<Action> {
    return this.dispatcher.transaction(key);
  }

  /** Deletes the ICDs with the given keys. */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/icd/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(req: RetrieveRequest): Promise<ICD[]> {
    const res = await this.cfg.unary.send(
      "/icd/retrieve",
      req,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.icds;
  }
}

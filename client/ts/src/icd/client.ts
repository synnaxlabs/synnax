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
import { z } from "zod";

import { type ICD, icdZ, type Key, keyZ, type New, ontologyID } from "@/icd/types.gen";
import { type ontology } from "@/ontology";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_icd_set";
export const DELETE_CHANNEL_NAME = "sy_icd_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
const retrieveMultiParamsZ = retrieveReqZ.or(query.keyListZ(keyZ));
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

const createReqZ = z.object({ icds: icdZ.array() });
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
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const retrieveResZ = z.object({ icds: icdZ.array().default(() => []) });
const emptyResZ = z.object({});

/** The file formats an interface control document is imported from. */
export const IMPORT_FORMATS = ["dbc", "csv", "xlsx"] as const;
export const importFormatZ = z.enum(IMPORT_FORMATS);
export type ImportFormat = z.infer<typeof importFormatZ>;

const importReqZ = z.object({ key: keyZ, format: importFormatZ, data: z.string() });
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

/**
 * Client-side matching for a request: key sets. Server-computed shapes (search,
 * limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((i: ICD) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (i) => keySet == null || keySet.has(i.key);
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, ICD> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, ICD>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, ICD>({
      name: "icds",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, icdZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "icd",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (icd, req) => requestFilter(req)(icd),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  /**
   * Creates the given ICDs, or replaces those whose keys already exist. A replacement
   * must carry the hash of the ICD as the caller read it.
   * @throws {ValidationError} if an ICD or one of its messages is invalid.
   * @throws {StaleICDError} if a replaced ICD changed since the caller read it.
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
        await this.cfg.unary.send(
          "/icd/create",
          { icds: optimistic },
          createReqZ,
          createResZ,
        ),
    });
    this.store.set(res.icds);
    return isMany ? res.icds : res.icds[0];
  }

  /**
   * Checks an ICD against the rules a save enforces, without saving it.
   * @returns Every problem in the ICD, or an empty array when it is valid. An ICD that
   * breaks a rule of its schema returns only those problems, as the Core cannot read
   * it.
   */
  async check(icd: New): Promise<Problem[]> {
    const parsed = icdZ.safeParse(icd);
    if (!parsed.success)
      return parsed.error.issues.map(({ path, message }) => ({
        path: path.join("."),
        message,
      }));
    const res = await this.cfg.unary.send(
      "/icd/check",
      { icd: parsed.data },
      checkReqZ,
      checkResZ,
    );
    return res.problems;
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
    const res = await this.cfg.unary.send(
      "/icd/import",
      { key, format, data: encodeBase64(data) },
      importReqZ,
      importResZ,
    );
    this.store.set([res.icd]);
    return res.icd;
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => [
      query.partialUpdate(this.store, key, { name }),
      this.cfg.ontology.cache.renameResource(ontologyID(key), name),
    ];
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send("/icd/rename", { key, name }, renameReqZ, emptyResZ),
    });
    rename();
  }

  /** Deletes the ICDs with the given keys. */
  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteResources(ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
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
      retrieveReqZ,
      retrieveResZ,
    );
    return res.icds;
  }
}

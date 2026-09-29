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

import {
  type Key,
  keyZ,
  type Library,
  libraryZ,
  type New,
  ontologyID,
} from "@/library/types.gen";
import { type ontology } from "@/ontology";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_library_set";
export const DELETE_CHANNEL_NAME = "sy_library_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});
const retrieveMultiParamsZ = retrieveReqZ.or(query.keyListZ(keyZ));
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}

const createReqZ = z.object({ libraries: libraryZ.array() });
const createResZ = z.object({ libraries: libraryZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const retrieveResZ = z.object({ libraries: libraryZ.array().default(() => []) });
const emptyResZ = z.object({});

/** The file formats an interface control document is imported from. */
export const IMPORT_FORMATS = ["dbc", "csv", "xlsx"] as const;
export const importFormatZ = z.enum(IMPORT_FORMATS);
export type ImportFormat = z.infer<typeof importFormatZ>;

const importReqZ = z.object({ key: keyZ, format: importFormatZ, data: z.string() });
const importResZ = z.object({ library: libraryZ });

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
const requestFilter = (req: RetrieveRequest): ((l: Library) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (l) => keySet == null || keySet.has(l.key);
};

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Library> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Library>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Library>({
      name: "libraries",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, libraryZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "library",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (library, req) => requestFilter(req)(library),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  /**
   * Creates the given libraries, or replaces those whose keys already exist.
   * @throws {ValidationError} if a library or one of its entries is invalid.
   */
  async create(library: New, opts?: query.WriteOptions<Library[]>): Promise<Library>;
  async create(
    libraries: New[],
    opts?: query.WriteOptions<Library[]>,
  ): Promise<Library[]>;
  async create(
    libraries: New | New[],
    opts: query.WriteOptions<Library[]> = {},
  ): Promise<Library | Library[]> {
    const isMany = Array.isArray(libraries);
    const optimistic = array
      .toArray(libraries)
      .map((l) => zod.parse(libraryZ, l, { label: "library" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.cfg.unary.send(
          "/library/create",
          { libraries: optimistic },
          createReqZ,
          createResZ,
        ),
    });
    this.store.set(res.libraries);
    return isMany ? res.libraries : res.libraries[0];
  }

  /**
   * Replaces the entries of a library with those parsed from an interface control
   * document. Entries and fields that match by name keep their keys.
   * @param key - The key of the library to import into.
   * @param format - The file format of the document.
   * @param data - The raw bytes of the document.
   * @returns The library as the Core stored it.
   * @throws {ValidationError} if the format is unknown or the document is invalid.
   */
  async import(key: Key, format: ImportFormat, data: Uint8Array): Promise<Library> {
    const res = await this.cfg.unary.send(
      "/library/import",
      { key, format, data: encodeBase64(data) },
      importReqZ,
      importResZ,
    );
    this.store.set([res.library]);
    return res.library;
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
        await this.cfg.unary.send(
          "/library/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  /**
   * Deletes the libraries with the given keys.
   * @throws {ValidationError} if a task uses one of the libraries.
   */
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
          "/library/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Library[]> {
    const res = await this.cfg.unary.send(
      "/library/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.libraries;
  }
}

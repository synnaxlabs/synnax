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
import {
  array,
  caseconv,
  type destructor,
  primitive,
  record,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { imex } from "@/imex";
import { type ontology } from "@/ontology";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Project,
  projectZ,
} from "@/project/types.gen";
import { query } from "@/query";

export const SET_CHANNEL_NAME = "sy_project_set";
export const DELETE_CHANNEL_NAME = "sy_project_delete";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
  ignoreNotFoundError: z.boolean().optional(),
});
const retrieveMultiParamsZ = retrieveReqZ.or(query.keyListZ(keyZ));
export interface RetrieveRequest extends z.infer<typeof retrieveReqZ> {}
const createReqZ = z.object({ projects: projectZ.array() });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const setLayoutReqZ = z.object({
  key: keyZ,
  layout: caseconv.preserveCase(record.unknownZ()),
});
const deleteReqZ = z.object({ keys: keyZ.array() });
const exportReqZ = z.object({ key: keyZ, encoding: imex.encodingZ });

const importParamsZ = z.object({ fileName: z.string().min(1) });

const retrieveResZ = z.object({ projects: projectZ.array().default(() => []) });
const createResZ = z.object({ projects: projectZ.array() });
const importResZ = z.object({ project: projectZ });
const emptyResZ = z.object({});

/** Options for a project import call. */
export interface ImportOptions {
  /**
   * The name of the uploaded archive or picked directory. Its extension-stripped form
   * names the project when the bundle carries no name.
   */
  fileName: string;
}

export interface SetLayoutParams extends z.input<typeof setLayoutReqZ> {}

/**
 * Client-side matching for a request: key sets. Server-computed shapes
 * (search, limit/offset) never reach this filter; they refetch instead.
 */
const requestFilter = (req: RetrieveRequest): ((p: Project) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  return (p) => keySet == null || keySet.has(p.key);
};

export interface ClientConfig {
  unary: UnaryClient;
  file: FileTransport;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Project> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Project>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Project>({
      name: "projects",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, projectZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "project",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (project, req) => requestFilter(req)(project),
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async create(project: New, opts?: query.WriteOptions<Project[]>): Promise<Project>;
  async create(
    projects: New[],
    opts?: query.WriteOptions<Project[]>,
  ): Promise<Project[]>;
  async create(
    projects: New | New[],
    opts: query.WriteOptions<Project[]> = {},
  ): Promise<Project | Project[]> {
    const isMany = Array.isArray(projects);
    const optimistic = array
      .toArray(projects)
      .map((p) => zod.parse(projectZ, p, { label: "project" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.cfg.unary.send(
          "/project/create",
          { projects: optimistic },
          createReqZ,
          createResZ,
        ),
    });
    this.store.set(res.projects);
    return isMany ? res.projects : res.projects[0];
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
          "/project/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  async setLayout(
    key: Key,
    layout: record.Unknown,
    opts: query.WriteOptions = {},
  ): Promise<void> {
    await query.optimistic({
      rollbacks: [query.partialUpdate(this.store, key, { layout })],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/project/set-layout",
          { key, layout },
          setLayoutReqZ,
          emptyResZ,
        ),
    });
    this.mergeThrough(key, { layout });
  }

  async delete(key: Key, opts?: query.WriteOptions): Promise<void>;
  async delete(keys: Key[], opts?: query.WriteOptions): Promise<void>;
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
          "/project/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /**
   * Exports the project and its contents as a bundle: a zip archive holding one JSON
   * file per document and panel, group children as directories, and a manifest.json
   * naming the project. Two members of one directory that take the same file name keep
   * distinct names through a numeric suffix. The caller pipes the stream wherever it
   * likes without the client buffering the whole archive.
   * @param options - the export options, including the serialization member files are
   * written in.
   * @returns the bundle as a stream of zip bytes.
   */
  async export(key: Key, options: imex.Options): Promise<ReadableStream<Uint8Array>> {
    return await this.cfg.file.download(
      "/project/export",
      { key, encoding: options.encoding },
      exportReqZ,
      { encoding: "ZIP" },
    );
  }

  /**
   * Imports a project bundle: a zip archive holding one JSON envelope per document and
   * panel, group children as directories, and a manifest.json naming the project. The
   * Core creates a fresh project and imports every member in a single transaction, so a
   * failure leaves nothing behind. Legacy project directories (a root LAYOUT.json with
   * no manifest) import through the same call.
   *
   * @param data - the bundle as zip bytes.
   * @param options - the import options, including the source file's name.
   * @returns the created project.
   * @throws {ValidationError} if the manifest is missing, malformed, or of another
   * bundle kind, if two member names collide, or if a member cannot be imported.
   */
  async import(data: UploadBody, options: ImportOptions): Promise<Project> {
    const res = await this.cfg.file.upload(
      "/project/import",
      data,
      { encoding: "ZIP", params: options, paramsSchema: importParamsZ },
      importResZ,
    );
    this.store.set(res.project);
    return res.project;
  }

  /** Subscribes to every project delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Project>): void {
    const prev = this.store.get(key);
    if (prev != null)
      this.store.set(key, { ...prev, ...record.purgeUndefined(changes) });
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Project[]> {
    const res = await this.cfg.unary.send(
      "/project/retrieve",
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return res.projects;
  }
}

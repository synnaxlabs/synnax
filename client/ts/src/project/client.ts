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
import { array, type destructor, filter, record, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { group } from "@/group";
import { imex } from "@/imex";
import { createOf, reduceAll } from "@/project/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  scopedActionZ,
  set,
} from "@/project/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  type Project,
  projectZ,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
} from "@/project/types.gen";
import {
  compareCursors,
  cursor,
  LIST_FIELDS,
  matches,
  retrieveInputZ,
  serverComputed,
} from "@/project/where.gen";
import { query } from "@/query";

export const ACTION_CHANNEL_NAME = "sy_project_action_set";
export const DELETE_CHANNEL_NAME = "sy_project_delete";

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;
const createReqZ = z.object({ projects: projectZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const exportReqZ = z.object({ key: keyZ, encoding: imex.encodingZ });

const importParamsZ = z.object({ fileName: z.string().min(1) });

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

export interface ClientConfig {
  unary: UnaryClient;
  file: FileTransport;
  cache: query.Cache;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Project> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Project>;
  private readonly dispatcher: actions.Controller<Key, Project, Action>;
  /** Builds requests that move projects between groups. */
  readonly setGroupRequest: group.MoveRequest<Key>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    const store = cache.createTable<Key, Project>({
      name: "projects",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    // Every write to a project reaches the client as actions.
    const dispatcher = new actions.Controller<Key, Project, Action>({
      store,
      listed: (action) => actions.changesField(action, LIST_FIELDS),
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      onStreamDown: (callback) => cache.onStreamDown(callback),
      send: actions.sender(cfg.unary, "/project/dispatch", dispatchReqZ, dispatchResZ),
      reduce: reduceAll,
      createOf,
    });
    cache.listen(dispatcher.listener(ACTION_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "project",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (project, req) => matches(group.MEMBER_LINES, project, req.where),
        serverComputed: (req) => serverComputed(group.MEMBER_LINES, req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
    this.setGroupRequest = group.createMoveRequest("/project/dispatch", (keys, to) =>
      group.moveCached(store, keys, to),
    );
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

  async rename(key: Key, name: string): Promise<void> {
    await this.dispatch(key, set({ path: ["name"], value: name }));
  }

  /**
   * Applies the actions to the cached project until the Core echoes them, and sends
   * them to the Core to apply in one transaction. Actions hold the snake_case form that
   * the Core stores. A refused send drops the actions from the cached project.
   * @throws {Error} if the cached project cannot hold an action, before any send.
   * @throws {ValidationError} if the project cannot hold an action.
   * @throws {NotFoundError} if the project does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.dispatcher.send(key, array.toArray(actions));
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [this.store.delete(keysArr)];
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
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.projects;
  }
}

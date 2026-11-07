// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, errors, filter, zod } from "@synnaxlabs/x";
import { z } from "zod";

import { actions } from "@/actions";
import { type ontology } from "@/ontology";
import { project } from "@/project";
import { query } from "@/query";
import { createOf, kindOf, reduceAll } from "@/video/actions";
import {
  type Action,
  dispatchReqZ,
  dispatchResZ,
  rename as renameAction,
  scopedActionZ,
} from "@/video/actions.gen";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type RetrieveRequest,
  retrieveRequestZ,
  retrieveResponseZ,
  type Video,
  videoZ,
} from "@/video/types.gen";
import { matches, retrieveInputZ, serverComputed } from "@/video/where.gen";

export const SET_CHANNEL_NAME = "sy_video_set";
export const DELETE_CHANNEL_NAME = "sy_video_delete";

const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveMultiParamsZ = z.union([
  query.keysRequestZ(keyZ),
  filter.requestInputZ(
    retrieveRequestZ,
    retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
  ),
]);

export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;
export type RetrieveSingleParams = { key: Key };
export type RetrieveMultipleParams = query.RequestInput<typeof retrieveMultiParamsZ>;

const createReqZ = z.object({
  project: project.keyZ,
  dispatchKey: z.string(),
  videos: videoZ.array(),
});
const createResZ = z.object({ videos: videoZ.array() });

const emptyResZ = z.object({});

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontology: ontology.Client;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Video> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Video>;
  private readonly dispatcher: actions.Controller<Key, Video, Action>;

  constructor(cfg: ClientConfig) {
    const { cache } = cfg;
    // Dispatch mutates documents server-side, so fetched copies never clobber
    // a doc holding locally replayed edits: the table hydrates if-absent.
    const store = cache.createTable<Key, Video>({
      name: "videos",
      hydrate: "if-absent",
      fetch: async (keys) => await this.execRetrieve(query.keysRequest(keys)),
      listen: [query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ)],
    });
    const dispatcher = new actions.Controller<Key, Video, Action>({
      store,
      onError: cache.onError,
      onEpoch: (callback) => cache.onEpoch(callback),
      reduce: reduceAll,
      send: actions.sender(cfg.unary, "/video/dispatch", dispatchReqZ, dispatchResZ),
      kindOf,
      createOf,
    });
    cache.listen(dispatcher.listener(SET_CHANNEL_NAME, scopedActionZ));
    super(cache, {
      name: "video",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.execRetrieve(req),
        matches: (video, req) => matches(video, req.where),
        serverComputed: (req) => serverComputed(req.where),
      },
    });
    this.cfg = cfg;
    this.store = store;
    this.dispatcher = dispatcher;
  }

  async create(
    project: project.Key,
    video: New,
    opts?: query.WriteOptions<Video[]>,
  ): Promise<Video>;
  async create(
    project: project.Key,
    videos: New[],
    opts?: query.WriteOptions<Video[]>,
  ): Promise<Video[]>;
  async create(
    project: project.Key,
    videos: New | New[],
    opts: query.WriteOptions<Video[]> = {},
  ): Promise<Video | Video[]> {
    const isMany = Array.isArray(videos);
    const optimistic = array
      .toArray(videos)
      .map((v) => zod.parse(videoZ, v, { label: "video" }));
    const res = await query.optimistic({
      rollbacks: [this.store.set(optimistic)],
      onOptimistic: () => opts.onOptimistic?.(optimistic),
      commit: async () =>
        await this.dispatcher.create(
          optimistic.map(({ key }) => key),
          async (dispatchKey) =>
            await this.cfg.unary.send(
              "/video/create",
              { project, dispatchKey, videos: optimistic },
              createReqZ,
              createResZ,
            ),
        ),
    });
    this.store.set(res.videos);
    return isMany ? res.videos : res.videos[0];
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    // Restores only the name, so the rollback cannot undo edits made after it.
    const rename = (): destructor.Destructor => {
      const prev = this.store.get(key)?.name ?? name;
      query.partialUpdate(this.store, key, { name });
      const resource = this.cfg.ontology.cache.renameResource(ontologyID(key), name);
      return () => {
        query.partialUpdate(this.store, key, { name: prev });
        resource();
      };
    };
    const rollback = rename();
    try {
      await opts.onOptimistic?.();
    } catch (e) {
      rollback();
      throw errors.fromUnknown(e);
    }
    await this.dispatcher.send(key, [renameAction({ name })], {
      rollback,
      reapply: rename,
    });
    rename();
  }

  /**
   * Applies actions to the cached video view and sends them to the server, recording an
   * undoable entry. Returns false without side effects when the view isn't cached. Rolls
   * back the local apply and rethrows on send failure.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: actions.Options<Video, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(key, array.toArray(actions), opts);
  }

  /**
   * Reverts the most recent undoable entry of the view. Returns false when
   * nothing is undoable.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo(key);
  }

  /**
   * Re-applies the most recently undone entry of the view. Returns false when
   * nothing is redoable.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo(key);
  }

  /** Whether the view has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the view has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the undo and redo stacks of the view. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher.onUndoStateChange(callback, key);
  }

  /** Stages actions committed atomically as one undoable entry. */
  beginTransaction(key: Key, kind?: string): actions.Transaction<Action> {
    return this.dispatcher.transaction(key, kind);
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.cfg.ontology.cache.deleteRelationships(ontologyID(keysArr)),
      this.store.delete(keysArr),
    ];
    await query.optimistic({
      rollbacks: drop(),
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/video/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
  }

  /** Subscribes to every video view delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
  }

  private async execRetrieve(params: RetrieveRequest): Promise<Video[]> {
    const res = await this.cfg.unary.send(
      "/video/retrieve",
      params,
      retrieveRequestZ,
      retrieveResponseZ,
    );
    return res.videos;
  }
}

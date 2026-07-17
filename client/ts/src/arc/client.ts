// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Stream,
  type StreamClient,
  type UnaryClient,
} from "@synnaxlabs/freighter";
import { array, deep, type destructor, errors, id, primitive } from "@synnaxlabs/x";
import { z } from "zod/v4";

import { type Action, dispatchReqZ, rename as renameAction } from "@/arc/actions.gen";
import { bindStore, STORE_KEY } from "@/arc/store";
import { type Arc, arcZ, type Key, keyZ, type New, ontologyID } from "@/arc/types.gen";
import { cache } from "@/cache";
import { type dispatch } from "@/dispatch";
import { NotFoundError } from "@/errors";
import { ontology } from "@/ontology";
import { status } from "@/status";
import { task } from "@/task";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeStatus: z.boolean().optional(),
});
const createReqZ = z.object({ arcs: arcZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });

const retrieveResZ = z.object({ arcs: arcZ.array().default(() => []) });
const createResZ = z.object({ arcs: arcZ.array() });
const emptyResZ = z.object({});

export const lspMessageZ = z.object({ content: z.string() });
export type LSPMessage = z.infer<typeof lspMessageZ>;

export type RetrieveRequest = z.input<typeof retrieveReqZ>;

const keyRetrieveRequestZ = z
  .object({
    key: keyZ,
    includeStatus: z.boolean().optional(),
  })
  .transform(({ key, includeStatus }) => ({ keys: [key], includeStatus }));

const nameRetrieveRequestZ = z
  .object({
    name: z.string(),
    includeStatus: z.boolean().optional(),
  })
  .transform(({ name, includeStatus }) => ({ names: [name], includeStatus }));

export const singleRetrieveParamsZ = z.union([
  keyRetrieveRequestZ,
  nameRetrieveRequestZ,
]);

export type SingleRetrieveParams = z.input<typeof singleRetrieveParamsZ>;

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveReqZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

const MOUNT_SCOPE = "arc.mounts";

const isSingleParams = (params: RetrieveParams): params is SingleRetrieveParams =>
  "key" in params || "name" in params;

export interface CreateParams extends New {
  /** Rack to deploy the arc on. Ensures a deployment task exists for it. */
  rack?: number;
}

const TASK_TYPE = "arc";

const taskStatusDataZ = z.null().optional();

const configuringStatus = (taskKey: task.Key): task.Status<typeof taskStatusDataZ> =>
  status.create<ReturnType<typeof task.statusDetailsZ<typeof taskStatusDataZ>>>({
    key: task.statusKey(taskKey),
    name: "Configuring task",
    variant: "loading",
    message: "Configuring task...",
    details: { task: taskKey, running: false, cmd: "", data: undefined },
  });

const TASK_SCHEMAS = {
  type: z.literal(TASK_TYPE),
  config: z.object({ arcKey: z.string() }),
  statusData: taskStatusDataZ,
} as const satisfies task.Schemas;

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key and name sets, permissive for server-computed shapes (search,
 * pagination), which accept every change and drift toward the server's answer.
 */
const requestFilter = (req: RetrieveRequest): ((a: Arc) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  return (a) => {
    if (keySet != null && !keySet.has(a.key)) return false;
    if (nameSet != null && !nameSet.has(a.name)) return false;
    return true;
  };
};

export class Client {
  private readonly client: UnaryClient;
  private readonly streamClient: StreamClient;
  private readonly engine_?: cache.Engine;
  private readonly dispatcher_?: dispatch.Controller<Key, Arc, Action>;
  private readonly ontologyClient: ontology.Client;
  private readonly taskClient: task.Client;
  private readonly queries_?: {
    single: cache.Queries<SingleRetrieveParams, Arc>;
    request: cache.Queries<RetrieveRequest, Arc[]>;
    task: cache.Queries<Key, task.Task | null>;
  };

  constructor(
    client: UnaryClient,
    streamClient: StreamClient,
    ontologyClient: ontology.Client,
    taskClient: task.Client,
    engine?: cache.Engine,
  ) {
    this.client = client;
    this.streamClient = streamClient;
    this.ontologyClient = ontologyClient;
    this.taskClient = taskClient;
    if (engine == null) return;
    this.dispatcher_ = bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "arc",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "arcs",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
      task: new cache.Queries({
        name: "arc task",
        fetch: async (query) => await this.fetchTask(query),
        mount: (params) => this.mountTask(params),
        ensureStreaming,
      }),
    };
  }

  private get dispatcher(): dispatch.Controller<Key, Arc, Action> {
    if (this.dispatcher_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.dispatcher_;
  }

  async create(arc: CreateParams, opts?: cache.WriteOptions<Arc[]>): Promise<Arc>;
  async create(arcs: CreateParams[], opts?: cache.WriteOptions<Arc[]>): Promise<Arc[]>;
  async create(
    arcs: CreateParams | CreateParams[],
    opts: cache.WriteOptions<Arc[]> = {},
  ): Promise<Arc | Arc[]> {
    const isMany = Array.isArray(arcs);
    const params = array.toArray(arcs);
    // Resolve the deployment task for each arc targeting a rack. A task
    // previously deployed to a different rack is deleted and recreated.
    const taskKeys = new Map<number, task.Key>();
    for (let i = 0; i < params.length; i++) {
      const { rack, key } = params[i];
      if (rack == null) continue;
      let taskKey = task.newKey(rack, 0);
      if (key != null) {
        const tsk = await this.retrieveTask(key);
        if (tsk != null)
          if (task.rackKey(tsk.key) !== rack) await this.taskClient.delete([tsk.key]);
          else taskKey = tsk.key;
      }
      taskKeys.set(i, taskKey);
    }
    const optimistic = params.map((a) => arcZ.parse(a));
    const rollback = new cache.Rollback();
    if (this.writes != null) rollback.add(this.writes.set(optimistic));
    await opts.onOptimistic?.(optimistic);
    const res = await rollback.guard(
      async () =>
        await this.client.send(
          "/arc/create",
          { arcs: optimistic },
          createReqZ,
          createResZ,
        ),
    );
    this.writes?.set(res.arcs);
    for (const [i, taskKey] of taskKeys) {
      const created = res.arcs[i];
      const newTsk = await this.taskClient.create(
        {
          key: taskKey,
          name: created.name,
          type: TASK_TYPE,
          config: { arcKey: created.key },
          status: configuringStatus(taskKey),
        },
        TASK_SCHEMAS,
      );
      await this.ontologyClient.addChildren(
        ontologyID(created.key),
        task.ontologyID(newTsk.key),
      );
    }
    return isMany ? res.arcs : res.arcs[0];
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const tsk = await this.retrieveTask(key);
    if (tsk != null) await this.taskClient.rename(tsk.key, name);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (writes != null) rollback.add(cache.partialUpdate(writes, key, { name }));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () => await this.sendDispatch(key, id.create(), [renameAction({ name })]),
    );
  }

  private async retrieveTask(key: Key): Promise<task.Task | null> {
    if (this.queries_ != null) return await this.queries_.task.retrieve(key);
    return await this.fetchTask(key);
  }

  async retrieve(params: SingleRetrieveParams): Promise<Arc>;
  async retrieve(params: RetrieveParams): Promise<Arc[]>;
  async retrieve(params: RetrieveParams): Promise<Arc | Arc[]> {
    const isSingle = isSingleParams(params);
    if (this.queries_ == null) {
      const arcs = await this.execRetrieve(params);
      checkForMultipleOrNoResults("Arc", params, arcs, isSingle);
      return isSingle ? arcs[0] : arcs;
    }
    if (isSingleParams(params)) return await this.queries_.single.retrieve(params);
    return await this.queries_.request.retrieve(retrieveReqZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver an arc; every other shape delivers the matching arcs.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: SingleRetrieveParams,
    handler: cache.ChangeHandler<Arc>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveRequest,
    handler: cache.ChangeHandler<Arc[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Arc> | cache.ChangeHandler<Arc[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if (isSingleParams(params))
      return queries.single.onChange(params, handler as cache.ChangeHandler<Arc>);
    return queries.request.onChange(
      retrieveReqZ.parse(params),
      handler as cache.ChangeHandler<Arc[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: SingleRetrieveParams): cache.Cached<Arc> | undefined;
  getCached(params: RetrieveRequest): cache.Cached<Arc[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Arc> | cache.Cached<Arc[]> | undefined {
    const queries = this.requireQueries();
    if (isSingleParams(params)) return queries.single.getCached(params);
    return queries.request.getCached(retrieveReqZ.parse(params));
  }

  /**
   * Cached queries for the task deployed for an arc, keyed by the arc's key.
   * Resolves null when the arc has no task.
   * @throws when the cache was disabled at client construction.
   */
  get task(): cache.Queries<Key, task.Task | null> {
    return this.requireQueries().task;
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    if (this.writes != null) rollback.add(this.writes.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send("/arc/delete", { keys: keysArr }, deleteReqZ, emptyResZ),
    );
  }

  async openLSP(): Promise<Stream<typeof lspMessageZ, typeof lspMessageZ>> {
    return await this.streamClient.stream("/arc/lsp", lspMessageZ, lspMessageZ);
  }

  /**
   * Applies actions to the cached arc and sends them to the server,
   * recording an undoable entry. Returns false without side effects when the
   * arc isn't cached. Rolls back the local apply and rethrows on send
   * failure.
   * @throws when the cache was disabled at client construction.
   */
  async dispatch(
    key: Key,
    actions: Action | Action[],
    opts: dispatch.Options<Arc, Action> = {},
  ): Promise<boolean> {
    return await this.dispatcher.dispatch(
      "",
      key,
      array.toArray(actions),
      this.dispatchSender(key),
      opts.preprocess,
    );
  }

  /**
   * Reverts the arc's most recent undoable entry. Returns false when
   * nothing is undoable.
   * @throws when the cache was disabled at client construction.
   */
  async undo(key: Key): Promise<boolean> {
    return await this.dispatcher.undo("", key, this.dispatchSender(key));
  }

  /**
   * Re-applies the arc's most recently undone entry. Returns false when
   * nothing is redoable.
   * @throws when the cache was disabled at client construction.
   */
  async redo(key: Key): Promise<boolean> {
    return await this.dispatcher.redo("", key, this.dispatchSender(key));
  }

  /** Whether the arc has a live undo entry. */
  hasUndo(key: Key): boolean {
    return this.dispatcher.hasUndo(key);
  }

  /** Whether the arc has a live redo entry. */
  hasRedo(key: Key): boolean {
    return this.dispatcher.hasRedo(key);
  }

  /**
   * Subscribes to changes in the arc's undo/redo stacks. Returns a
   * destructor that unsubscribes.
   */
  onUndoStateChange(callback: () => void, key?: Key): destructor.Destructor {
    return this.dispatcher.onUndoStateChange(MOUNT_SCOPE, callback, key);
  }

  /**
   * Stages actions committed atomically as one undoable entry.
   * @throws when the cache was disabled at client construction.
   */
  beginTransaction(key: Key, kind?: string): dispatch.Transaction<Action> {
    return this.dispatcher.transaction("", key, this.dispatchSender(key), kind);
  }

  private dispatchSender(key: Key): dispatch.SendDispatch<Action> {
    return async (actions, dispatchKey) =>
      await this.sendDispatch(key, dispatchKey, actions);
  }

  private async sendDispatch(
    key: Key,
    dispatchKey: string,
    actions: Action[],
  ): Promise<void> {
    await this.client.send(
      "/arc/dispatch",
      { key, dispatchKey, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }

  private get writes(): cache.UnaryStore<Key, Arc> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get arcStore(): cache.UnaryStore<Key, Arc> {
    return this.requireEngine().store(STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get arcEvents(): cache.UnaryStore<Key, Arc> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private requireEngine(): cache.Engine {
    if (this.engine_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.engine_;
  }

  private requireQueries(): NonNullable<typeof this.queries_> {
    if (this.queries_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.queries_;
  }

  private async execRetrieve(params: RetrieveParams): Promise<Arc[]> {
    const res = await this.client.send(
      "/arc/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.arcs;
  }

  // Dispatch mutates documents server-side (including materialized text), so
  // a cached copy is only as fresh as the streamer. Fetches always hit the
  // network; only the server materializes text.
  private async fetchSingle(query: SingleRetrieveParams): Promise<Arc> {
    const arcs = await this.execRetrieve(query);
    checkForMultipleOrNoResults("Arc", query, arcs, true);
    return this.hydrate(arcs[0]);
  }

  // Answers reuse the identical store doc so selector references stay
  // stable; a fresher network doc replaces it and answers. While a locally
  // replayed dispatch awaits its echo the replayed doc stays, but the
  // network doc answers: it carries the server-materialized text.
  private hydrate(a: Arc): Arc {
    if (this.dispatcher_?.hasOutstanding(a.key) === true) {
      this.arcStore.setIfAbsent(a);
      return a;
    }
    const prev = this.arcStore.get(a.key);
    if (prev != null && deep.equal(prev, a)) return prev;
    this.arcStore.set(a.key, a);
    return a;
  }

  private mountSingle({
    query,
    update,
    remove,
  }: cache.MountParams<SingleRetrieveParams, Arc>) {
    const matches = (a: Arc) =>
      "key" in query ? a.key === query.key : a.name === query.name;
    return [
      this.arcEvents.onSet((arc) => {
        if (matches(arc)) update(arc);
      }),
      this.arcEvents.onDelete((key) => {
        const corpse = this.arcStore.getTombstone(key)?.corpse;
        const deleted =
          "key" in query ? key === query.key : corpse != null && matches(corpse);
        if (deleted) remove(corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Arc[]> {
    const arcs = await this.execRetrieve(query);
    return arcs.map((a) => this.hydrate(a));
  }

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Arc[]>) {
    const matches = requestFilter(query);
    return [
      this.arcEvents.onSet((arc) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((a) => a.key === arc.key);
          if (!matches(arc))
            return existing ? prev.filter((a) => a.key !== arc.key) : prev;
          if (existing) return prev.map((a) => (a.key === arc.key ? arc : a));
          return [...prev, arc];
        });
      }),
      this.arcEvents.onDelete((key) => {
        update((prev) => prev?.filter((a) => a.key !== key));
      }),
    ];
  }

  private get statusStore(): cache.UnaryStore<status.Key, status.Status> {
    return this.requireEngine().store(status.STORE_KEY);
  }

  private get taskEvents(): cache.UnaryStore<task.Key, Omit<task.Task, "status">> {
    return this.requireEngine().store(task.STORE_KEY, MOUNT_SCOPE);
  }

  private get statusEvents(): cache.UnaryStore<status.Key, status.Status> {
    return this.requireEngine().store(status.STORE_KEY, MOUNT_SCOPE);
  }

  private get relationshipEvents(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY, MOUNT_SCOPE);
  }

  /** Rebuilds a cached task with its cached status attached. */
  private composeTask(cached: Omit<task.Task, "status">): task.Task {
    const st = this.statusStore.get(task.statusKey(cached.key));
    const payload = (cached as task.Task).payload;
    if (st == null) return this.taskClient.sugar(payload);
    return this.taskClient.sugar({ ...payload, status: st as unknown as task.Status });
  }

  private async fetchTask(arcKey: Key): Promise<task.Task | null> {
    let children: ontology.Resource[];
    try {
      children = await this.ontologyClient.retrieveChildren(ontologyID(arcKey), {
        types: ["task"],
      });
    } catch (e) {
      // An arc that does not exist cannot have a task.
      if (NotFoundError.matches(e)) return null;
      throw errors.fromUnknown(e);
    }
    const child = children[0];
    if (child == null) return null;
    return await this.taskClient.retrieve({ key: child.id.key });
  }

  private mountTask({ query, update }: cache.MountParams<Key, task.Task | null>) {
    const isTaskChild = (rel: ontology.Relationship): boolean =>
      ontology.matchRelationship(rel, {
        from: ontologyID(query),
        type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
        to: { type: "task" },
      });
    return [
      this.relationshipEvents.onSet((rel) => {
        if (!isTaskChild(rel)) return;
        this.taskClient
          .retrieve({ key: rel.to.key })
          .then((tsk) => update(() => tsk))
          .catch(console.error);
      }),
      this.relationshipEvents.onDelete((relKey) => {
        if (isTaskChild(ontology.relationshipZ.parse(relKey))) update(() => null);
      }),
      this.taskEvents.onSet((tsk) => {
        update((prev) => (prev?.key === tsk.key ? this.composeTask(tsk) : prev));
      }),
      this.statusEvents.onSet((st) => {
        update((prev) =>
          prev != null && st.key === task.statusKey(prev.key)
            ? this.composeTask(prev)
            : prev,
        );
      }),
    ];
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive } from "@synnaxlabs/x";
import z from "zod";

import { cache } from "@/cache";
import { label } from "@/label";
import { ontology } from "@/ontology";
import { type Key, keyZ, ontologyID, TYPE_ONTOLOGY_ID } from "@/status/payload";
import { bindStore, STORE_KEY } from "@/status/store";
import { type New, type Status, statusZ } from "@/status/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const setReqZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  z.object({
    parent: ontology.idZ.optional(),
    statuses: statusZ({ details: detailsSchema }).array(),
  });
const setResZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) => z.object({ statuses: statusZ({ details: detailsSchema }).array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
  includeLabels: z.boolean().optional(),
  hasLabels: label.keyZ.array().optional(),
  variants: z.string().array().optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ, includeLabels: z.boolean().optional() })
  .transform(({ key, includeLabels }) => ({ keys: [key], includeLabels }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type SingleRetrieveParams = z.input<typeof singleRetrieveParamsZ>;
export type MultiRetrieveParams = z.input<typeof retrieveRequestZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResponseZ = <DetailsSchema extends z.ZodType = z.ZodNever>(
  detailsSchema?: DetailsSchema,
) =>
  z.object({
    statuses: statusZ({ details: detailsSchema })
      .array()
      .default(() => []),
  });

export interface SetOptions {
  parent?: ontology.ID;
}

const MOUNT_SCOPE = "status.mounts";

const BASE_REQUEST: Partial<RetrieveRequest> = { includeLabels: true };

const normalizeRequest = (params: MultiRetrieveParams): RetrieveRequest =>
  retrieveRequestZ.parse(params);

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.searchTerm == null &&
  req.hasLabels == null &&
  req.variants == null &&
  req.limit == null &&
  req.offset == null;

export class Client {
  readonly type: string = "status";
  private readonly client: UnaryClient;
  private readonly labelClient?: label.Client;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key, Status>;
    request: cache.Queries<RetrieveRequest, Status[]>;
  };

  constructor(client: UnaryClient, labelClient?: label.Client, engine?: cache.Engine) {
    this.client = client;
    this.labelClient = labelClient;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "status",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "statuses",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  async retrieve<DetailsSchema extends z.ZodType>(
    params: SingleRetrieveParams & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async retrieve(params: SingleRetrieveParams): Promise<Status>;
  async retrieve(params: MultiRetrieveParams): Promise<Status[]>;
  async retrieve<DetailsSchema extends z.ZodType = z.ZodNever>(
    params: RetrieveParams & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    const isSingle = "key" in params;
    // Schema-parametrized retrieves validate details for one caller; their
    // results are not shared through the cache.
    if (this.queries_ == null || params.detailsSchema != null) {
      const statuses = await this.execRetrieve<DetailsSchema>(params);
      checkForMultipleOrNoResults("Status", params, statuses, isSingle);
      return isSingle ? statuses[0] : statuses;
    }
    if (isSingle)
      return (await this.queries_.single.retrieve(params.key)) as Status<DetailsSchema>;
    return (await this.queries_.request.retrieve(
      normalizeRequest(params),
    )) as Status<DetailsSchema>[];
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a status; every other shape delivers the matching
   * statuses.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: SingleRetrieveParams,
    handler: cache.ChangeHandler<Status>,
  ): destructor.Destructor;
  onChange(
    params: MultiRetrieveParams,
    handler: cache.ChangeHandler<Status[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Status> | cache.ChangeHandler<Status[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(
        params.key,
        handler as cache.ChangeHandler<Status>,
      );
    return queries.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Status[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: SingleRetrieveParams): cache.Cached<Status> | undefined;
  getCached(params: MultiRetrieveParams): cache.Cached<Status[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Status> | cache.Cached<Status[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.request.getCached(normalizeRequest(params));
  }

  async set<DetailsSchema extends z.ZodType>(
    status: New<DetailsSchema>,
    opts?: SetOptions & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>>;
  async set(status: New, opts?: SetOptions): Promise<Status>;
  async set(statuses: New[], opts?: SetOptions): Promise<Status[]>;
  async set<DetailsSchema extends z.ZodType = z.ZodNever>(
    statuses: New<DetailsSchema> | New<DetailsSchema>[],
    opts: SetOptions & { detailsSchema?: DetailsSchema } = {},
  ): Promise<Status<DetailsSchema> | Status<DetailsSchema>[]> {
    const isMany = Array.isArray(statuses);
    const res = await this.client.send(
      "/status/set",
      {
        statuses: array.toArray(statuses) as z.input<
          ReturnType<typeof setReqZ<DetailsSchema>>
        >["statuses"],
        parent: opts.parent,
      },
      setReqZ(opts.detailsSchema),
      setResZ(opts.detailsSchema),
    );
    const created = res.statuses as Status<DetailsSchema>[];
    this.writes?.set(created);
    return isMany ? created : created[0];
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const stat = await this.retrieve({ key });
    const renamed = { ...stat, name };
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (this.engine_ != null && writes != null) {
      rollback.add(writes.set(renamed));
      rollback.add(ontology.renameCachedResource(this.engine_, ontologyID(key), name));
    }
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.set(renamed);
    });
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send("/status/delete", { keys: keysArr }, deleteReqZ, emptyResZ);
    this.writes?.delete(keysArr);
    this.relationshipWrites?.delete((r) =>
      keysArr.some((key) => label.matchLabeledBy(r, ontologyID(key))),
    );
  }

  private get writes(): cache.UnaryStore<Key, Status> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get relationshipWrites():
    cache.UnaryStore<string, ontology.Relationship> | undefined {
    return this.engine_?.store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get statusStore(): cache.UnaryStore<Key, Status> {
    return this.requireEngine().store(STORE_KEY);
  }

  private get labelStore(): cache.UnaryStore<label.Key, label.Label> {
    return this.requireEngine().store(label.STORE_KEY);
  }

  private get relationshipStore(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  /** Subscribes to every status set delivered to the cache. */
  onSet(handler: (status: Status) => void): destructor.Destructor {
    return this.statusEvents.onSet(handler);
  }

  /** Subscribes to every status delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.statusEvents.onDelete(handler);
  }

  private get statusEvents(): cache.UnaryStore<Key, Status> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private get relationshipEvents(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY, MOUNT_SCOPE);
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

  private async execRetrieve<DetailsSchema extends z.ZodType = z.ZodNever>(
    params: RetrieveParams & { detailsSchema?: DetailsSchema },
  ): Promise<Status<DetailsSchema>[]> {
    const res = await this.client.send(
      "/status/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ<DetailsSchema>(params.detailsSchema),
    );
    return res.statuses as Status<DetailsSchema>[];
  }

  /** Rebuilds a cached status with its cached labels attached. */
  private compose(cached: Status): Status {
    const labels = label.cachedLabelsOf(
      this.relationshipStore,
      this.labelStore,
      ontologyID(cached.key),
    );
    return { ...cached, labels };
  }

  /** Writes a fetched status and its included label relationships. */
  private writeThrough(status: Status): void {
    this.statusStore.set(status.key, status);
    if (status.labels == null) return;
    this.labelStore.set(status.labels);
    const id = ontologyID(status.key);
    status.labels.forEach((l) => {
      const rel: ontology.Relationship = {
        from: id,
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: label.ontologyID(l.key),
      };
      this.relationshipStore.set(ontology.relationshipToString(rel), rel);
    });
  }

  /**
   * Fetches the given keys, serving composed cached entries and fetching only
   * the misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Status[]> {
    const results: Status[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.statusStore.get(key);
      if (cached != null) results.push(this.compose(cached));
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ ...BASE_REQUEST, keys: misses });
      fetched.forEach((s) => this.writeThrough(s));
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (s) => s.key);
  }

  private async fetchSingle(query: Key): Promise<Status> {
    const cached = this.statusStore.get(query);
    if (cached != null) return this.compose(cached);
    const statuses = await this.execRetrieve({ ...BASE_REQUEST, keys: [query] });
    checkForMultipleOrNoResults("Status", query, statuses, true);
    this.writeThrough(statuses[0]);
    return statuses[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Status>) {
    const id = ontologyID(query);
    return [
      this.statusEvents.onSet((status) => {
        if (status.key === query) update(this.compose(status));
      }),
      this.statusEvents.onDelete((key) => {
        if (key === query) remove(this.statusStore.getTombstone(key)?.corpse);
      }),
      this.relationshipEvents.onSet((rel) => {
        if (!label.matchLabeledBy(rel, id)) return;
        const recompose = () =>
          update((prev) => (prev == null ? prev : this.compose(prev)));
        recompose();
        void this.ensureLabel(rel).then((fetched) => {
          if (fetched) recompose();
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        if (!label.matchLabeledBy(rel, id)) return;
        update((prev) => (prev == null ? prev : this.compose(prev)));
      }),
    ];
  }

  /**
   * Fetches a labeled-by target the cache is missing so composition sees it.
   * Returns whether a fetch occurred, so callers can recompose.
   */
  private async ensureLabel(rel: ontology.Relationship): Promise<boolean> {
    if (this.labelClient == null) return false;
    if (rel.to.type !== "label" || this.labelStore.has(rel.to.key)) return false;
    const fetched = await this.labelClient.retrieve({ key: rel.to.key });
    this.labelStore.set(rel.to.key, fetched);
    return true;
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Status[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const statuses = await this.execRetrieve({ ...BASE_REQUEST, ...query });
    statuses.forEach((s) => this.writeThrough(s));
    return statuses;
  }

  /**
   * Client-side approximation of the server's matching for a request: exact
   * for key, variant, and label filters, permissive for server-computed
   * shapes (search), which accept every change and drift toward the server's
   * answer.
   */
  private requestFilter(req: RetrieveRequest): (s: Status) => boolean {
    const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
    const labelSet = primitive.isNonZero(req.hasLabels)
      ? new Set(req.hasLabels)
      : undefined;
    const variantSet = primitive.isNonZero(req.variants)
      ? new Set(req.variants)
      : undefined;
    return (s) => {
      if (keySet != null && !keySet.has(s.key)) return false;
      if (variantSet != null && !variantSet.has(s.variant)) return false;
      if (labelSet != null && !(s.labels ?? []).some((l) => labelSet.has(l.key)))
        return false;
      return true;
    };
  }

  private mountRequest({
    query,
    update,
  }: cache.MountParams<RetrieveRequest, Status[]>) {
    const matches = this.requestFilter(query);
    return [
      this.statusEvents.onSet((status) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((s) => s.key === status.key);
          const merged = this.compose(status);
          if (!matches(merged))
            return existing ? prev.filter((s) => s.key !== status.key) : prev;
          if (existing) return prev.map((s) => (s.key === status.key ? merged : s));
          return [...prev, merged];
        });
      }),
      this.statusEvents.onDelete((key) => {
        update((prev) => prev?.filter((s) => s.key !== key));
      }),
      this.relationshipEvents.onSet((rel) => {
        const apply = () => update((prev) => this.applyLabeledBy(prev, rel, matches));
        apply();
        void this.ensureLabel(rel).then((fetched) => {
          if (fetched) apply();
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        update((prev) => this.applyLabeledBy(prev, rel, matches));
      }),
    ];
  }

  /**
   * Applies a labeled-by change to a request answer: recomposes the affected
   * member, drops it when it stops matching, and admits a stored status that
   * now matches.
   */
  private applyLabeledBy(
    prev: Status[] | undefined,
    rel: ontology.Relationship,
    matches: (s: Status) => boolean,
  ): Status[] | undefined {
    if (prev == null) return prev;
    if (
      rel.type !== label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE ||
      rel.from.type !== TYPE_ONTOLOGY_ID.type
    )
      return prev;
    const key = rel.from.key;
    const member = prev.find((s) => s.key === key);
    if (member != null) {
      const composed = this.compose(member);
      if (!matches(composed)) return prev.filter((s) => s.key !== key);
      return prev.map((s) => (s.key === key ? composed : s));
    }
    const stored = this.statusStore.get(key);
    if (stored == null) return prev;
    const composed = this.compose(stored);
    if (!matches(composed)) return prev;
    return [...prev, composed];
  }
}

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
import {
  DELETE_CHANNEL_NAME,
  type Key,
  keyZ,
  ontologyID,
  SET_CHANNEL_NAME,
  TYPE_ONTOLOGY_ID,
} from "@/status/payload";
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

const BASE_REQUEST: Partial<RetrieveRequest> = { includeLabels: true };

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const normalizeRequest = (params: MultiRetrieveParams): RetrieveRequest =>
  retrieveRequestZ.parse(params);

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.searchTerm == null &&
  req.hasLabels == null &&
  req.variants == null &&
  req.limit == null &&
  req.offset == null;

export class Client {
  readonly type: string = "status";
  /** The status record table; injected into sibling clients at wiring. */
  readonly store: cache.Table<Key, Status>;
  private readonly client: UnaryClient;
  private readonly labelClient?: label.Client;
  private readonly engine: cache.Cache;
  private readonly labels: cache.Table<label.Key, label.Label>;
  private readonly ontology: ontology.Stores;
  private readonly answers: {
    single: cache.Answers<Key, Status, Key, Status>;
    request: cache.Answers<RetrieveRequest, Status[], Key, Status>;
  };

  constructor(
    client: UnaryClient,
    engine: cache.Cache,
    labels: cache.Table<label.Key, label.Label>,
    ontologyStores: ontology.Stores,
    labelClient?: label.Client,
  ) {
    this.client = client;
    this.labelClient = labelClient;
    this.engine = engine;
    this.labels = labels;
    this.ontology = ontologyStores;
    this.store = this.createTable(engine, ontologyStores.relationships, labels);
    this.answers = {
      single: engine.answers({
        name: "status",
        table: this.store,
        fetch: async (query) => [(await this.fetchSingle(query)).key],
        compose: (records) => this.compose(records[0]),
        keyOf: (query) => query,
        single: true,
        watch: [
          cache.watch(this.ontology.relationships, (event, query: Key) => {
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!label.matchLabeledBy(rel, ontologyID(query))) return null;
            if (event.variant === "set") this.ensureLabel(rel);
            return [query];
          }),
          cache.watch(this.labels, (event, query: Key) =>
            this.isLabeledBy(query, event.key) ? [query] : null,
          ),
        ],
      }),
      request: engine.answers({
        name: "statuses",
        table: this.store,
        fetch: async (query) => (await this.fetchRequest(query)).map((s) => s.key),
        compose: (records) => records.map((s) => this.compose(s)),
        matches: (status, query) => this.requestFilter(query)(status),
        serverFields: SERVER_FIELDS,
        watch: [
          cache.watch(this.ontology.relationships, (event, _: RetrieveRequest) => {
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (
              rel.type !== label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE ||
              rel.from.type !== TYPE_ONTOLOGY_ID.type
            )
              return null;
            if (event.variant === "set") this.ensureLabel(rel);
            return [rel.from.key];
          }),
          cache.watch(this.labels, (event, _: RetrieveRequest) =>
            this.statusesLabeledBy(event.key),
          ),
        ],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
    };
  }

  private createTable(
    engine: cache.Cache,
    relationships: cache.Table<string, ontology.Relationship>,
    labels: cache.Table<label.Key, label.Label>,
  ): cache.Table<Key, Status> {
    const table = engine.createTable<Key, Status>({ name: "statuses" });
    const set: cache.ChannelListener<ReturnType<typeof statusZ>> = {
      channel: SET_CHANNEL_NAME,
      schema: statusZ(),
      onChange: (changed) =>
        table.set(changed.key, (p) => {
          const next = { ...p, ...changed };
          const id = ontologyID(changed.key);
          next.labels = label.cachedLabelsOf(relationships, labels, id);
          return next;
        }),
    };
    const del: cache.ChannelListener<typeof keyZ> = {
      channel: DELETE_CHANNEL_NAME,
      schema: keyZ,
      onChange: (changed) => table.delete(changed),
    };
    engine.addListeners(table, set, del);
    return table;
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
    if (params.detailsSchema != null) {
      const statuses = await this.execRetrieve<DetailsSchema>(params);
      checkForMultipleOrNoResults("Status", params, statuses, isSingle);
      return isSingle ? statuses[0] : statuses;
    }
    if (isSingle)
      return (await this.answers.single.retrieve(params.key)) as Status<DetailsSchema>;
    return (await this.answers.request.retrieve(
      normalizeRequest(params),
    )) as Status<DetailsSchema>[];
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a status; every other shape delivers the matching
   * statuses.
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
    const { request, single } = this.answers;
    if ("key" in params)
      return single.onChange(params.key, handler as cache.ChangeHandler<Status>);
    return request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Status[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: SingleRetrieveParams): cache.Cached<Status> | undefined;
  getCached(params: MultiRetrieveParams): cache.Cached<Status[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Status> | cache.Cached<Status[]> | undefined {
    const { request, single } = this.answers;
    if ("key" in params) return single.getCached(params.key);
    return request.getCached(normalizeRequest(params));
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
    this.store.set(created);
    return isMany ? created : created[0];
  }

  async rename(key: Key, name: string, opts: cache.WriteOptions = {}): Promise<void> {
    const stat = await this.retrieve({ key });
    const renamed = { ...stat, name };
    const rollback = new cache.Rollback();
    rollback.add(this.store.set(renamed.key, renamed));
    rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
    await opts.onOptimistic?.();
    await rollback.guard(async () => {
      await this.set(renamed);
    });
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send("/status/delete", { keys: keysArr }, deleteReqZ, emptyResZ);
    this.store.delete(keysArr);
    this.ontology.relationships.delete((r) =>
      keysArr.some((key) => label.matchLabeledBy(r, ontologyID(key))),
    );
  }

  /** Subscribes to every status set delivered to the cache. */
  onSet(handler: (status: Status) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "set") handler(event.value);
    });
  }

  /** Subscribes to every status delete delivered to the cache. */
  onDelete(handler: (key: Key) => void): destructor.Destructor {
    return this.store.subscribe((event) => {
      if (event.variant === "delete") handler(event.key);
    });
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
      this.ontology.relationships,
      this.labels,
      ontologyID(cached.key),
    );
    return { ...cached, labels };
  }

  /** Writes a fetched status and its included label relationships. */
  private writeThrough(status: Status): void {
    this.store.set(status.key, status);
    if (status.labels == null) return;
    this.labels.set(status.labels);
    const id = ontologyID(status.key);
    status.labels.forEach((l) => {
      const rel: ontology.Relationship = {
        from: id,
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: label.ontologyID(l.key),
      };
      this.ontology.relationships.set(ontology.relationshipToString(rel), rel);
    });
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Status[]> {
    const results: Status[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.store.get(key);
      if (cached != null) results.push(cached);
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
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const statuses = await this.execRetrieve({ ...BASE_REQUEST, keys: [query] });
    checkForMultipleOrNoResults("Status", query, statuses, true);
    this.writeThrough(statuses[0]);
    return statuses[0];
  }

  /**
   * Fetches a labeled-by target the cache is missing so composition sees it.
   * The fetched label lands in the label table, which recomposes answers.
   */
  private ensureLabel(rel: ontology.Relationship): void {
    if (this.labelClient == null) return;
    if (rel.to.type !== "label" || this.labels.has(rel.to.key)) return;
    void this.labelClient
      .retrieve({ key: rel.to.key })
      .catch((exc: unknown) =>
        this.engine.onError(new Error("failed to fetch status label", { cause: exc })),
      );
  }

  /** Reports whether the status is labeled by the given label. */
  private isLabeledBy(status: Key, labelKey: label.Key): boolean {
    return this.ontology.relationships.has(
      ontology.relationshipToString({
        from: ontologyID(status),
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: label.ontologyID(labelKey),
      }),
    );
  }

  /** Returns the keys of cached statuses labeled by the given label. */
  private statusesLabeledBy(labelKey: label.Key): Key[] {
    return this.ontology.relationships
      .get(
        (r) =>
          r.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
          r.from.type === TYPE_ONTOLOGY_ID.type &&
          r.to.type === "label" &&
          r.to.key === labelKey,
      )
      .map((r) => r.from.key);
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Status[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const statuses = await this.execRetrieve({ ...BASE_REQUEST, ...query });
    statuses.forEach((s) => this.writeThrough(s));
    return statuses;
  }

  /**
   * Client-side matching for a request: key sets, variants, and label
   * membership via the relationship table. Server-computed shapes (search,
   * limit/offset) never reach this filter; they refetch instead.
   */
  private requestFilter(req: RetrieveRequest): (s: Status) => boolean {
    const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
    const variantSet = primitive.isNonZero(req.variants)
      ? new Set(req.variants)
      : undefined;
    return (s) => {
      if (keySet != null && !keySet.has(s.key)) return false;
      if (variantSet != null && !variantSet.has(s.variant)) return false;
      if (
        primitive.isNonZero(req.hasLabels) &&
        !req.hasLabels.some((key) => this.isLabeledBy(s.key, key))
      )
        return false;
      return true;
    };
  }
}

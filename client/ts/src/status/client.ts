// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor, primitive } from "@synnaxlabs/x";
import z from "zod";

import { label } from "@/label";
import { ontology } from "@/ontology";
import { query } from "@/query";
import {
  DELETE_CHANNEL_NAME,
  type Key,
  keyZ,
  ontologyID,
  SET_CHANNEL_NAME,
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

export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  ontologyStores: ontology.Stores;
  labels: label.Client;
}

export class Client extends query.Retriever<
  typeof retrieveRequestZ,
  Key,
  Status,
  Status,
  SingleRetrieveParams
> {
  readonly type: string = "status";
  readonly store: query.Table<Key, Status>;
  private readonly cfg: ClientConfig;

  constructor(cfg: ClientConfig) {
    const { cache, ontologyStores, labels } = cfg;
    const { relationships } = ontologyStores;
    const store = cache.createTable<Key, Status>({
      name: "statuses",
      fetch: async (keys) => await this.fetchThrough({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, statusZ(), {
          value: (changed, prev) => {
            const next = { ...prev, ...changed };
            const id = ontologyID(changed.key);
            next.labels = label.cachedLabelsOf(relationships, labels.store, id);
            return next;
          },
        }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    const single = cache.queries<Key, Status, Key, Status>({
      name: "status",
      table: store,
      fetch: async (key) => [(await this.fetchSingle(key)).key],
      compose: (records) => this.compose(records[0]),
      keyOf: (key) => key,
      single: true,
      watch: [
        query.watch(relationships, (event, key: Key) => {
          const rel =
            event.variant === "set"
              ? event.value
              : ontology.relationshipZ.parse(event.key);
          if (!label.matchLabeledBy(rel, ontologyID(key))) return null;
          if (event.variant === "set") this.ensureLabel(rel);
          return [key];
        }),
        query.watch(labels.store, (event, key: Key) =>
          this.isLabeledBy(key, event.key) ? [key] : null,
        ),
      ],
    });
    super(cache, {
      name: "status",
      table: store,
      request: {
        schema: retrieveRequestZ,
        fetch: async (req) => await this.fetchThrough(req),
        matches: (status, req) => this.requestFilter(req)(status),
        watch: [
          query.watch(relationships, (event, _: RetrieveRequest) => {
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (
              rel.type !== label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE ||
              rel.from.type !== "status"
            )
              return null;
            if (event.variant === "set") this.ensureLabel(rel);
            return [rel.from.key];
          }),
          query.watch(labels.store, (event) => this.statusesLabeledBy(event.key)),
        ],
      },
      compose: (record) => this.compose(record),
      single: {
        is: (params): params is SingleRetrieveParams =>
          typeof params === "object" && params !== null && "key" in params,
        normalize: ({ key }) => key,
        space: single as query.Retrieves<query.Params, Status>,
      },
    });
    this.cfg = cfg;
    this.store = store;
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
    if (isSingle) return (await super.retrieve(params)) as Status<DetailsSchema>;
    return (await super.retrieve(
      params as MultiRetrieveParams,
    )) as Status<DetailsSchema>[];
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
    const res = await this.cfg.unary.send(
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

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const stat = await this.retrieve({ key });
    const renamed = { ...stat, name };
    const rollback = new destructor.Chain();
    rollback.add(
      this.store.set(renamed),
      ontology.renameCachedResource(this.cfg.ontologyStores, ontologyID(key), name),
    );
    await opts.onOptimistic?.();
    await rollback.guard(async () => await this.set(renamed));
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => [
      this.store.delete(keysArr),
      this.cfg.ontologyStores.relationships.delete((r) =>
        keysArr.some((key) => label.matchLabeledBy(r, ontologyID(key))),
      ),
    ];
    const rollback = new destructor.Chain();
    rollback.add(...drop());
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.cfg.unary.send(
          "/status/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    drop();
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
    const res = await this.cfg.unary.send(
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
      this.cfg.ontologyStores.relationships,
      this.cfg.labels.store,
      ontologyID(cached.key),
    );
    return { ...cached, labels };
  }

  /** Writes a fetched status and its included label relationships. */
  private writeThrough(status: Status): void {
    this.store.set(status);
    if (status.labels == null) return;
    this.cfg.labels.store.set(status.labels);
    const id = ontologyID(status.key);
    status.labels.forEach((l) => {
      const rel: ontology.Relationship = {
        from: id,
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: label.ontologyID(l.key),
      };
      this.cfg.ontologyStores.relationships.set(
        ontology.relationshipToString(rel),
        rel,
      );
    });
  }

  /** Fetches statuses and writes their included labels through the caches. */
  private async fetchThrough(req: RetrieveRequest): Promise<Status[]> {
    const statuses = await this.execRetrieve({ ...BASE_REQUEST, ...req });
    statuses.forEach((s) => this.writeThrough(s));
    return statuses;
  }

  private async fetchSingle(key: Key): Promise<Status> {
    const cached = this.store.get(key);
    if (cached != null) return cached;
    const statuses = await this.fetchThrough({ keys: [key] });
    checkForMultipleOrNoResults("Status", key, statuses, true);
    return statuses[0];
  }

  /**
   * Fetches a labeled-by target the cache is missing so composition sees it.
   * The fetched label lands in the label table, which recomposes answers.
   */
  private ensureLabel(rel: ontology.Relationship): void {
    if (rel.to.type !== "label" || this.cfg.labels.store.has(rel.to.key)) return;
    void this.cfg.labels
      .retrieve({ key: rel.to.key })
      .catch((exc: unknown) =>
        this.cfg.cache.onError(
          new Error("failed to fetch status label", { cause: exc }),
        ),
      );
  }

  /** Reports whether the status is labeled by the given label. */
  private isLabeledBy(status: Key, labelKey: label.Key): boolean {
    return this.cfg.ontologyStores.relationships.has(
      ontology.relationshipToString({
        from: ontologyID(status),
        type: label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
        to: label.ontologyID(labelKey),
      }),
    );
  }

  /** Returns the keys of cached statuses labeled by the given label. */
  private statusesLabeledBy(labelKey: label.Key): Key[] {
    return this.cfg.ontologyStores.relationships
      .get(
        (r) =>
          r.type === label.LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
          r.from.type === "status" &&
          r.to.type === "label" &&
          r.to.key === labelKey,
      )
      .map((r) => r.from.key);
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

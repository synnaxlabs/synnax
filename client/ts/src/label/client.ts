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
import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { bindStore, matchLabeledBy, STORE_KEY } from "@/label/store";
import { type Key, keyZ, type Label, labelZ, type New } from "@/label/types.gen";
import { ontology } from "@/ontology";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const createReqZ = z.object({ labels: labelZ.array() });
const createResZ = z.object({ labels: labelZ.array() });
const deleteReqZ = z.object({ keys: keyZ.array() });
const setReqZ = z.object({
  id: ontology.idZ,
  labels: keyZ.array(),
  replace: z.boolean().optional(),
});

interface SetReq extends z.infer<typeof setReqZ> {}
export interface SetOptions extends Pick<SetReq, "replace"> {}

const removeReqZ = setReqZ.omit({ replace: true });
const emptyResZ = z.object({});

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  for: ontology.idZ.optional(),
  searchTerm: z.string().optional(),
  offset: z.int().optional(),
  limit: z.int().optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, retrieveRequestZ]);

export type RetrieveParams = z.input<typeof retrieveParamsZ>;
export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const normalizeRequest = (params: RetrieveMultipleParams): RetrieveRequest =>
  retrieveRequestZ.parse(params);

const isKeysOnly = (req: RetrieveRequest): boolean =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.for == null &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

export class Client {
  readonly type: string = "label";
  private readonly client: UnaryClient;
  private readonly cache_: cache.Cache;
  private readonly answers_: {
    single: cache.Answers<Key, Label, Key, Label>;
    request: cache.Answers<RetrieveRequest, Label[], Key, Label>;
  };

  constructor(client: UnaryClient, engine: cache.Cache) {
    this.client = client;
    this.cache_ = engine;
    bindStore(engine, async (keys) => await this.execRetrieve({ keys }));
    this.answers_ = {
      single: engine.answers({
        name: "label",
        table: this.labelStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((l) => l.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "labels",
        table: this.labelStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((l) => l.key),
        compose: (records) => records,
        matches: (label, query) => this.requestFilter(query)(label),
        serverFields: SERVER_FIELDS,
        watch: [
          cache.watch(this.relationshipStore, (event, query: RetrieveRequest) => {
            if (query.for == null) return null;
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!matchLabeledBy(rel, query.for)) return null;
            return [rel.to.key];
          }),
        ],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
    };
  }

  async retrieve(params: RetrieveSingleParams): Promise<Label>;
  async retrieve(params: RetrieveMultipleParams): Promise<Label[]>;
  async retrieve(params: RetrieveParams): Promise<Label | Label[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a label; every other shape delivers the matching labels.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Label>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Label[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveParams,
    handler: cache.ChangeHandler<Label> | cache.ChangeHandler<Label[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(params.key, handler as cache.ChangeHandler<Label>);
    return answers.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Label[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Label> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Label[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Label> | cache.Cached<Label[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(normalizeRequest(params));
  }

  async label(id: ontology.ID, labels: Key[], opts: SetOptions = {}): Promise<void> {
    await this.client.send(
      "/label/set",
      { id, labels, replace: opts.replace },
      setReqZ,
      emptyResZ,
    );
    const rels = this.relationshipStore;
    if (opts.replace === true) rels.delete((r) => matchLabeledBy(r, id));
    labels.forEach((key) => {
      const rel = labeledByRel(id, key);
      rels.set(ontology.relationshipToString(rel), rel);
    });
  }

  async remove(id: ontology.ID, labels: Key[]): Promise<void> {
    await this.client.send("/label/remove", { id, labels }, removeReqZ, emptyResZ);
    this.relationshipStore.delete(
      (r) => matchLabeledBy(r, id) && labels.includes(r.to.key),
    );
  }

  async create(label: New): Promise<Label>;
  async create(labels: New[]): Promise<Label[]>;
  async create(labels: New | New[]): Promise<Label | Label[]> {
    const isMany = Array.isArray(labels);
    const res = await this.client.send(
      "/label/create",
      { labels: array.toArray(labels) },
      createReqZ,
      createResZ,
    );
    this.labelStore.setMany(res.labels);
    return isMany ? res.labels : res.labels[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send("/label/delete", { keys: keysArr }, deleteReqZ, emptyResZ);
    this.labelStore.delete(keysArr);
    this.relationshipStore.delete(
      (r) =>
        r.type === LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
        r.to.type === "label" &&
        keysArr.includes(r.to.key),
    );
  }

  private get labelStore(): cache.Table<Key, Label> {
    return this.cache_.table(STORE_KEY);
  }

  private get relationshipStore(): cache.Table<string, ontology.Relationship> {
    return this.cache_.table(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private async execRetrieve(params: RetrieveParams): Promise<Label[]> {
    const res = await this.client.send(
      "/label/retrieve",
      params,
      retrieveParamsZ,
      retrieveResponseZ,
    );
    return res.labels;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Label[]> {
    const results: Label[] = [];
    const misses: Key[] = [];
    for (const key of keys) {
      const cached = this.labelStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.labelStore.setMany(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (l) => l.key);
  }

  private async fetchSingle(query: Key): Promise<Label> {
    const cached = this.labelStore.get(query);
    if (cached != null) return cached;
    const labels = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Label", query, labels, true);
    this.labelStore.setMany(labels);
    return labels[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Label[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const labels = await this.execRetrieve(query);
    this.labelStore.setMany(labels);
    return labels;
  }

  /**
   * Client-side matching for a request: key sets, names, and labeled-entity
   * membership via the relationship table. Server-computed shapes (search,
   * limit/offset) never reach this filter; they refetch instead.
   */
  private requestFilter(req: RetrieveRequest): (l: Label) => boolean {
    const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
    const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
    return (l) => {
      if (keySet != null && !keySet.has(l.key)) return false;
      if (nameSet != null && !nameSet.has(l.name)) return false;
      if (req.for != null && !this.isLabelOf(req.for, l.key)) return false;
      return true;
    };
  }

  private isLabelOf(id: ontology.ID, key: Key): boolean {
    return this.relationshipStore.has(
      ontology.relationshipToString(labeledByRel(id, key)),
    );
  }
}

const retrieveResponseZ = z.object({ labels: labelZ.array().default(() => []) });

const labeledByRel = (id: ontology.ID, key: Key): ontology.Relationship => ({
  from: id,
  type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  to: ontologyID(key),
});

export const ontologyID = ontology.createIDFactory<Key>("label");
export const TYPE_ONTOLOGY_ID = ontologyID("");

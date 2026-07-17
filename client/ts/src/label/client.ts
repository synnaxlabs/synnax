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

const retrieveResponseZ = z.object({ labels: labelZ.array().default(() => []) });

const MOUNT_SCOPE = "label.mounts";

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
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key, Label>;
    request: cache.Queries<RetrieveRequest, Label[]>;
  };

  constructor(client: UnaryClient, engine?: cache.Engine) {
    this.client = client;
    if (engine == null) return;
    bindStore(engine, async (keys) => await this.execRetrieve({ keys }));
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "label",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "labels",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
  }

  async retrieve(params: RetrieveSingleParams): Promise<Label>;
  async retrieve(params: RetrieveMultipleParams): Promise<Label[]>;
  async retrieve(params: RetrieveParams): Promise<Label | Label[]> {
    const isSingle = "key" in params;
    if (this.queries_ == null) {
      const labels = await this.execRetrieve(params);
      checkForMultipleOrNoResults("Label", params, labels, isSingle);
      return isSingle ? labels[0] : labels;
    }
    if (isSingle) return await this.queries_.single.retrieve(params.key);
    return await this.queries_.request.retrieve(normalizeRequest(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a label; every other shape delivers the matching labels.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(params.key, handler as cache.ChangeHandler<Label>);
    return queries.request.onChange(
      normalizeRequest(params),
      handler as cache.ChangeHandler<Label[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Label> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Label[]> | undefined;
  getCached(
    params: RetrieveParams,
  ): cache.Cached<Label> | cache.Cached<Label[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.request.getCached(normalizeRequest(params));
  }

  async label(id: ontology.ID, labels: Key[], opts: SetOptions = {}): Promise<void> {
    await this.client.send(
      "/label/set",
      { id, labels, replace: opts.replace },
      setReqZ,
      emptyResZ,
    );
    const rels = this.relationshipWrites;
    if (rels == null) return;
    if (opts.replace === true) rels.delete((r) => matchLabeledBy(r, id));
    labels.forEach((key) => {
      const rel = labeledByRel(id, key);
      rels.set(ontology.relationshipToString(rel), rel);
    });
  }

  async remove(id: ontology.ID, labels: Key[]): Promise<void> {
    await this.client.send("/label/remove", { id, labels }, removeReqZ, emptyResZ);
    this.relationshipWrites?.delete(
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
    this.writes?.set(res.labels);
    return isMany ? res.labels : res.labels[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send("/label/delete", { keys: keysArr }, deleteReqZ, emptyResZ);
    this.writes?.delete(keysArr);
    this.relationshipWrites?.delete(
      (r) =>
        r.type === LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
        r.to.type === "label" &&
        keysArr.includes(r.to.key),
    );
  }

  private get writes(): cache.UnaryStore<Key, Label> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get relationshipWrites():
    cache.UnaryStore<string, ontology.Relationship> | undefined {
    return this.engine_?.store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get labelStore(): cache.UnaryStore<Key, Label> {
    return this.requireEngine().store(STORE_KEY);
  }

  private get relationshipStore(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get labelEvents(): cache.UnaryStore<Key, Label> {
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
      this.labelStore.set(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (l) => l.key);
  }

  private async fetchSingle(query: Key): Promise<Label> {
    const cached = this.labelStore.get(query);
    if (cached != null) return cached;
    const labels = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Label", query, labels, true);
    this.labelStore.set(labels);
    return labels[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Label>) {
    return [
      this.labelEvents.onSet((label) => {
        if (label.key === query) update(label);
      }),
      this.labelEvents.onDelete((key) => {
        if (key === query) remove(this.labelStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Label[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const labels = await this.execRetrieve(query);
    this.labelStore.set(labels);
    return labels;
  }

  /**
   * Client-side approximation of the server's matching for a request: exact
   * for key sets, names, and labeled-entity queries, permissive for
   * server-computed shapes (search), which accept every change and drift
   * toward the server's answer.
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

  private mountRequest({ query, update }: cache.MountParams<RetrieveRequest, Label[]>) {
    const matches = this.requestFilter(query);
    return [
      this.labelEvents.onSet((label) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((l) => l.key === label.key);
          if (!matches(label))
            return existing ? prev.filter((l) => l.key !== label.key) : prev;
          if (existing) return prev.map((l) => (l.key === label.key ? label : l));
          return [...prev, label];
        });
      }),
      this.labelEvents.onDelete((key) => {
        update((prev) => prev?.filter((l) => l.key !== key));
      }),
      this.relationshipEvents.onSet((rel) => {
        if (query.for == null || !matchLabeledBy(rel, query.for)) return;
        void this.fetchKeys([rel.to.key]).then(([label]) => {
          if (label == null) return;
          update((prev) => {
            if (prev == null) return prev;
            if (prev.some((l) => l.key === label.key))
              return prev.map((l) => (l.key === label.key ? label : l));
            return [...prev, label];
          });
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        if (query.for == null || !matchLabeledBy(rel, query.for)) return;
        update((prev) => prev?.filter((l) => l.key !== rel.to.key));
      }),
    ];
  }
}

const labeledByRel = (id: ontology.ID, key: Key): ontology.Relationship => ({
  from: id,
  type: LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE,
  to: ontologyID(key),
});

export const ontologyID = ontology.createIDFactory<Key>("label");
export const TYPE_ONTOLOGY_ID = ontologyID("");

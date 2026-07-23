// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, primitive } from "@synnaxlabs/x";
import z from "zod";

import { LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE } from "@/label/payload";
import { matchLabeledBy } from "@/label/store";
import { type Key, keyZ, type Label, labelZ, type New } from "@/label/types.gen";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_label_set";
export const DELETE_CHANNEL_NAME = "sy_label_delete";

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

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.for == null &&
  req.searchTerm == null &&
  req.limit == null &&
  req.offset == null;

const createTable = (
  cache: query.Cache,
  refetch: (keys: Key[]) => Promise<Label[]>,
): query.Table<Key, Label> => {
  const table = cache.createTable<Key, Label>({ name: "labels", refetch });
  const set: query.ChannelListener<typeof labelZ> = {
    channel: SET_CHANNEL_NAME,
    schema: labelZ,
    onChange: table.set.bind(table),
  };
  const del: query.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: table.delete.bind(table),
  };
  cache.addListeners(table, set, del);
  return table;
};

export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Label
> {
  readonly type: string = "label";
  /** The label record table; injected into sibling clients at wiring. */
  readonly store: query.Table<Key, Label>;
  private readonly client: UnaryClient;
  private readonly relationships: query.Table<string, ontology.Relationship>;

  constructor(
    client: UnaryClient,
    cache: query.Cache,
    relationships: query.Table<string, ontology.Relationship>,
  ) {
    const store = createTable(cache, async (keys) => await this.execRetrieve({ keys }));
    super({
      single: cache.queries({
        name: "label",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((l) => l.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "labels",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((l) => l.key),
        compose: (records) => records,
        matches: (label, query) => this.requestFilter(query)(label),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(relationships, (event, query: RetrieveRequest) => {
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
      isSingle: (params) => "key" in params,
      normalizeSingle: ({ key }) => key,
      normalizeRequest,
    });
    this.client = client;
    this.relationships = relationships;
    this.store = store;
  }

  async label(id: ontology.ID, labels: Key[], opts: SetOptions = {}): Promise<void> {
    await this.client.send(
      "/label/set",
      { id, labels, replace: opts.replace },
      setReqZ,
      emptyResZ,
    );
    if (opts.replace === true) this.relationships.delete((r) => matchLabeledBy(r, id));
    labels.forEach((key) => {
      const rel = labeledByRel(id, key);
      this.relationships.set(ontology.relationshipToString(rel), rel);
    });
  }

  async remove(id: ontology.ID, labels: Key[]): Promise<void> {
    await this.client.send("/label/remove", { id, labels }, removeReqZ, emptyResZ);
    this.relationships.delete(
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
    this.store.set(res.labels);
    return isMany ? res.labels : res.labels[0];
  }

  async delete(keys: Key | Key[]): Promise<void> {
    const keysArr = array.toArray(keys);
    await this.client.send("/label/delete", { keys: keysArr }, deleteReqZ, emptyResZ);
    this.store.delete(keysArr);
    this.relationships.delete(
      (r) =>
        r.type === LABELED_BY_ONTOLOGY_RELATIONSHIP_TYPE &&
        r.to.type === "label" &&
        keysArr.includes(r.to.key),
    );
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
      const cached = this.store.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.store.set(fetched);
      results.push(...fetched);
    }
    return query.orderByKeys(keys, results, (l) => l.key);
  }

  private async fetchSingle(query: Key): Promise<Label> {
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const labels = await this.execRetrieve({ keys: [query] });
    checkForMultipleOrNoResults("Label", query, labels, true);
    this.store.set(labels);
    return labels[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Label[]> {
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const labels = await this.execRetrieve(query);
    this.store.set(labels);
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
    return this.relationships.has(ontology.relationshipToString(labeledByRel(id, key)));
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

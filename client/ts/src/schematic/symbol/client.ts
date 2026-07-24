// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, destructor, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

import { group } from "@/group";
import { ontology } from "@/ontology";
import { query } from "@/query";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Symbol,
  symbolZ,
} from "@/schematic/symbol/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_schematic_symbol_set";
export const DELETE_CHANNEL_NAME = "sy_schematic_symbol_delete";

const createReqZ = z.object({ symbols: symbolZ.array(), parent: ontology.idZ });
const renameReqZ = z.object({ key: keyZ, name: z.string() });
const deleteReqZ = z.object({ keys: keyZ.array() });

// The server only understands keys and searchTerm; parent scoping is resolved
// client-side through the ontology, so it never reaches the wire.
const wireRetrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  searchTerm: z.string().optional(),
});

const retrieveRequestZ = wireRetrieveRequestZ.extend({
  parent: ontology.idZ.optional(),
});

const singleRetrieveParamsZ = z
  .object({ key: keyZ })
  .transform(({ key }) => ({ keys: [key] }));

const retrieveParamsZ = z.union([singleRetrieveParamsZ, wireRetrieveRequestZ]);

export type RetrieveSingleParams = z.input<typeof singleRetrieveParamsZ>;
export type RetrieveMultipleParams = z.input<typeof retrieveRequestZ>;
export type RetrieveParams = RetrieveSingleParams | RetrieveMultipleParams;

interface RetrieveRequest extends z.infer<typeof retrieveRequestZ> {}

const retrieveResZ = z.object({ symbols: symbolZ.array().default(() => []) });
const createResZ = z.object({ symbols: symbolZ.array() });
const emptyResZ = z.object({});
const retrieveGroupReqZ = z.object({});
const retrieveGroupResZ = z.object({ group: group.groupZ });

export interface CreateParams extends New {
  parent: ontology.ID;
}

export interface CreateMultipleParams {
  symbols: New[];
  parent: ontology.ID;
}

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm"] as const;

const isKeysOnly = (req: RetrieveRequest): req is RetrieveRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) && req.searchTerm == null && req.parent == null;

const childRel = (parent: ontology.ID, key: Key): ontology.Relationship => ({
  from: parent,
  type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
  to: ontologyID(key),
});

const matchChildRel = (rel: ontology.Relationship, parent: ontology.ID): boolean =>
  ontology.matchRelationship(rel, {
    from: parent,
    type: ontology.PARENT_OF_RELATIONSHIP_TYPE,
    to: { type: "schematic_symbol" },
  });

const createTable = (cache: query.Cache): query.Table<Key, Symbol> => {
  const table = cache.createTable<Key, Symbol>({ name: "schematicSymbols" });
  const set: query.ChannelListener<typeof symbolZ> = {
    channel: SET_CHANNEL_NAME,
    schema: symbolZ,
    onChange: (changed) => table.set(changed),
  };
  const del: query.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: (changed) => table.delete(changed),
  };
  cache.addListeners(table, set, del);
  return table;
};

export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveMultipleParams,
  Key,
  RetrieveRequest,
  Symbol
> {
  private readonly client: UnaryClient;
  private readonly ontologyClient: ontology.Client;
  private readonly store: query.Table<Key, Symbol>;
  private readonly ontology: ontology.Stores;

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    cache: query.Cache,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(cache);
    super({
      single: cache.queries({
        name: "schematic symbol",
        table: store,
        fetch: async (query) => [await this.fetchSingle(query)].map((s) => s.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: cache.queries({
        name: "schematic symbols",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((s) => s.key),
        compose: (records) => records,
        matches: (symbol, query) => this.requestFilter(query)(symbol),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(ontologyStores.relationships, (event, query: RetrieveRequest) => {
            if (query.parent == null) return null;
            const rel =
              event.variant === "set"
                ? event.value
                : ontology.relationshipZ.parse(event.key);
            if (!matchChildRel(rel, query.parent)) return null;
            return [rel.to.key];
          }),
        ],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
      isSingle: (params) => "key" in params,
      normalizeSingle: ({ key }) => key,
      normalizeRequest: (params) => retrieveRequestZ.parse(params),
    });
    this.client = client;
    this.ontologyClient = ontologyClient;
    this.store = store;
    this.ontology = ontologyStores;
  }

  async create(options: CreateParams): Promise<Symbol>;
  async create(options: CreateMultipleParams): Promise<Symbol[]>;
  async create(
    options: CreateParams | CreateMultipleParams,
  ): Promise<Symbol | Symbol[]> {
    const isMany = "symbols" in options;
    const symbols = isMany ? options.symbols : [options];
    const res = await this.client.send(
      "/schematic/symbol/create",
      { symbols, parent: options.parent },
      createReqZ,
      createResZ,
    );
    // Relationships first: parent-scoped answers check membership when the
    // symbol write lands.
    const rels = this.ontology.relationships;
    res.symbols.forEach((s) => {
      const rel = childRel(options.parent, s.key);
      rels.set(ontology.relationshipToString(rel), rel);
    });
    this.store.set(res.symbols);
    return isMany ? res.symbols : res.symbols[0];
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.client.send(
      "/schematic/symbol/rename",
      { key, name },
      renameReqZ,
      emptyResZ,
    );
    this.mergeThrough(key, { name });
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new destructor.Chain();
    rollback.add(this.store.delete(keysArr));
    await opts.onOptimistic?.();
    await rollback.guard(
      async () =>
        await this.client.send(
          "/schematic/symbol/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    );
    this.store.delete(keysArr);
    this.ontology.relationships.delete(
      (r) =>
        r.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
        r.to.type === "schematic_symbol" &&
        keysArr.includes(r.to.key),
    );
  }

  async retrieveGroup(): Promise<group.Group> {
    const res = await this.client.send(
      "/schematic/symbol/retrieve-group",
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return res.group;
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Symbol>): void {
    const prev = this.store.get(key);
    if (prev != null)
      this.store.set(key, { ...prev, ...record.purgeUndefined(changes) });
  }

  private async execRetrieve(
    params: RetrieveSingleParams | z.input<typeof wireRetrieveRequestZ>,
  ): Promise<Symbol[]> {
    const res = await this.client.send(
      "/schematic/symbol/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.symbols;
  }

  /** Resolves a parent scope to its symbol children's keys via the ontology. */
  private async resolveChildKeys(parent: ontology.ID): Promise<Key[]> {
    const children = await this.ontologyClient.retrieveChildren(parent, {
      types: ["schematic_symbol"],
    });
    return children.map(({ id: { key } }) => key);
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Symbol[]> {
    const results: Symbol[] = [];
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
    return query.orderByKeys(keys, results, (s) => s.key);
  }

  private async fetchSingle(query: Key): Promise<Symbol> {
    const cached = this.store.get(query);
    if (cached != null) return cached;
    const symbols = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Schematic Symbol", query, symbols, true);
    this.store.set(symbols);
    return symbols[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Symbol[]> {
    const { parent } = query;
    if (parent != null) return await this.fetchChildren({ ...query, parent });
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys);
    const symbols = await this.execRetrieve(query);
    this.store.set(symbols);
    return symbols;
  }

  private async fetchChildren(
    query: RetrieveRequest & { parent: ontology.ID },
  ): Promise<Symbol[]> {
    // retrieveChildren writes the parent relationships through, so membership
    // checks in requestFilter see them.
    const keys = await this.resolveChildKeys(query.parent);
    if (keys.length === 0) return [];
    if (primitive.isZero(query.searchTerm)) return await this.fetchKeys(keys);
    const symbols = await this.execRetrieve({ keys, searchTerm: query.searchTerm });
    this.store.set(symbols);
    return symbols;
  }

  /**
   * Client-side matching for a request: exact for key sets and parent
   * membership via the relationship table. Server-computed shapes (search)
   * never reach this filter; they refetch instead.
   */
  private requestFilter(req: RetrieveRequest): (s: Symbol) => boolean {
    const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
    return (s) => {
      if (keySet != null && !keySet.has(s.key)) return false;
      if (req.parent != null && !this.isChildOf(req.parent, s.key)) return false;
      return true;
    };
  }

  private isChildOf(parent: ontology.ID, key: Key): boolean {
    return this.ontology.relationships.has(
      ontology.relationshipToString(childRel(parent, key)),
    );
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { array, type destructor, primitive, record } from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { group } from "@/group";
import { ontology } from "@/ontology";
import { bindStore, STORE_KEY } from "@/schematic/symbol/store";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Symbol,
  symbolZ,
} from "@/schematic/symbol/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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

const isKeysOnly = (req: RetrieveRequest): boolean =>
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

export class Client {
  private readonly client: UnaryClient;
  private readonly ontologyClient: ontology.Client;
  private readonly cache_: cache.Cache;
  private readonly answers_: {
    single: cache.Answers<Key, Symbol, Key, Symbol>;
    request: cache.Answers<RetrieveRequest, Symbol[], Key, Symbol>;
  };

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    engine: cache.Cache,
  ) {
    this.client = client;
    this.ontologyClient = ontologyClient;
    bindStore(engine);
    this.cache_ = engine;
    this.answers_ = {
      single: engine.answers({
        name: "schematic symbol",
        table: this.symbolStore,
        fetch: async (query) => [await this.fetchSingle(query)].map((s) => s.key),
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      }),
      request: engine.answers({
        name: "schematic symbols",
        table: this.symbolStore,
        fetch: async (query) => (await this.fetchRequest(query)).map((s) => s.key),
        compose: (records) => records,
        matches: (symbol, query) => this.requestFilter(query)(symbol),
        serverFields: SERVER_FIELDS,
        watch: [
          cache.watch(this.relationshipStore, (event, query: RetrieveRequest) => {
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
    };
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
    const rels = this.relationshipStore;
    res.symbols.forEach((s) => {
      const rel = childRel(options.parent, s.key);
      rels.set(ontology.relationshipToString(rel), rel);
    });
    this.symbolStore.setMany(res.symbols);
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

  async retrieve(params: RetrieveSingleParams): Promise<Symbol>;
  async retrieve(params: RetrieveMultipleParams): Promise<Symbol[]>;
  async retrieve(params: RetrieveParams): Promise<Symbol | Symbol[]> {
    const isSingle = "key" in params;
    if (isSingle) return await this.answers_.single.retrieve(params.key);
    return await this.answers_.request.retrieve(retrieveRequestZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a symbol; every other shape delivers the matching symbols.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Symbol>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveMultipleParams,
    handler: cache.ChangeHandler<Symbol[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveMultipleParams,
    handler: cache.ChangeHandler<Symbol> | cache.ChangeHandler<Symbol[]>,
  ): destructor.Destructor {
    const answers = this.answers_;
    if ("key" in params)
      return answers.single.onChange(
        params.key,
        handler as cache.ChangeHandler<Symbol>,
      );
    return answers.request.onChange(
      retrieveRequestZ.parse(params),
      handler as cache.ChangeHandler<Symbol[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Symbol> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Symbol[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Symbol> | cache.Cached<Symbol[]> | undefined {
    const answers = this.answers_;
    if ("key" in params) return answers.single.getCached(params.key);
    return answers.request.getCached(retrieveRequestZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    rollback.add(this.symbolStore.delete(keysArr));
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
    this.symbolStore.delete(keysArr);
    this.relationshipStore.delete(
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

  private get symbolStore(): cache.Table<Key, Symbol> {
    return this.cache_.table(STORE_KEY);
  }

  private get relationshipStore(): cache.Table<string, ontology.Relationship> {
    return this.cache_.table(ontology.RELATIONSHIPS_STORE_KEY);
  }

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Symbol>): void {
    const store = this.symbolStore;
    const prev = store.get(key);
    if (prev != null) store.set(key, { ...prev, ...record.purgeUndefined(changes) });
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
      const cached = this.symbolStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    }
    if (misses.length > 0) {
      const fetched = await this.execRetrieve({ keys: misses });
      this.symbolStore.setMany(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (s) => s.key);
  }

  private async fetchSingle(query: Key): Promise<Symbol> {
    const cached = this.symbolStore.get(query);
    if (cached != null) return cached;
    const symbols = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Schematic Symbol", query, symbols, true);
    this.symbolStore.setMany(symbols);
    return symbols[0];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Symbol[]> {
    if (query.parent != null) return await this.fetchChildren(query);
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const symbols = await this.execRetrieve(query);
    this.symbolStore.setMany(symbols);
    return symbols;
  }

  private async fetchChildren(query: RetrieveRequest): Promise<Symbol[]> {
    // retrieveChildren writes the parent relationships through, so membership
    // checks in requestFilter see them.
    const keys = await this.resolveChildKeys(query.parent as ontology.ID);
    if (keys.length === 0) return [];
    if (primitive.isZero(query.searchTerm)) return await this.fetchKeys(keys);
    const symbols = await this.execRetrieve({ keys, searchTerm: query.searchTerm });
    this.symbolStore.setMany(symbols);
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
    return this.relationshipStore.has(
      ontology.relationshipToString(childRel(parent, key)),
    );
  }
}

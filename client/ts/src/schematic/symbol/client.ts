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

const MOUNT_SCOPE = "schematic.symbol.mounts";

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
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<Key, Symbol>;
    request: cache.Queries<RetrieveRequest, Symbol[]>;
  };

  constructor(
    client: UnaryClient,
    ontologyClient: ontology.Client,
    engine?: cache.Engine,
  ) {
    this.client = client;
    this.ontologyClient = ontologyClient;
    if (engine == null) return;
    bindStore(engine);
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "schematic symbol",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "schematic symbols",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
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
    // Relationships first: parent-scoped listeners check membership when the
    // symbol write lands.
    const rels = this.relationshipWrites;
    if (rels != null)
      res.symbols.forEach((s) => {
        const rel = childRel(options.parent, s.key);
        rels.set(ontology.relationshipToString(rel), rel);
      });
    this.writes?.set(res.symbols);
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
    if (this.queries_ == null) {
      const symbols = await this.execResolved(params);
      checkForMultipleOrNoResults("Schematic Symbol", params, symbols, isSingle);
      return isSingle ? symbols[0] : symbols;
    }
    if (isSingle) return await this.queries_.single.retrieve(params.key);
    return await this.queries_.request.retrieve(retrieveRequestZ.parse(params));
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a symbol; every other shape delivers the matching symbols.
   * @throws when the cache was disabled at client construction.
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
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(
        params.key,
        handler as cache.ChangeHandler<Symbol>,
      );
    return queries.request.onChange(
      retrieveRequestZ.parse(params),
      handler as cache.ChangeHandler<Symbol[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Symbol> | undefined;
  getCached(params: RetrieveMultipleParams): cache.Cached<Symbol[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveMultipleParams,
  ): cache.Cached<Symbol> | cache.Cached<Symbol[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(params.key);
    return queries.request.getCached(retrieveRequestZ.parse(params));
  }

  async delete(keys: Key | Key[], opts: cache.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const rollback = new cache.Rollback();
    const writes = this.writes;
    if (writes != null) rollback.add(writes.delete(keysArr));
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
    this.writes?.delete(keysArr);
    this.relationshipWrites?.delete(
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

  private get writes(): cache.UnaryStore<Key, Symbol> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get symbolStore(): cache.UnaryStore<Key, Symbol> {
    return this.requireEngine().store(STORE_KEY);
  }

  private get relationshipWrites():
    cache.UnaryStore<string, ontology.Relationship> | undefined {
    return this.engine_?.store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  private get relationshipStore(): cache.UnaryStore<string, ontology.Relationship> {
    return this.requireEngine().store(ontology.RELATIONSHIPS_STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get symbolEvents(): cache.UnaryStore<Key, Symbol> {
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

  // Undefined fields are dropped: the server keeps prior values for them.
  private mergeThrough(key: Key, changes: Partial<Symbol>): void {
    const store = this.writes;
    if (store == null) return;
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

  /** Cache-free retrieve that still honors parent scoping. */
  private async execResolved(params: RetrieveParams): Promise<Symbol[]> {
    if ("key" in params || params.parent == null)
      return await this.execRetrieve(params);
    const keys = await this.resolveChildKeys(ontology.idZ.parse(params.parent));
    if (keys.length === 0) return [];
    return await this.execRetrieve({ keys, searchTerm: params.searchTerm });
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
      this.symbolStore.set(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (s) => s.key);
  }

  private async fetchSingle(query: Key): Promise<Symbol> {
    const cached = this.symbolStore.get(query);
    if (cached != null) return cached;
    const symbols = await this.execRetrieve({ key: query });
    checkForMultipleOrNoResults("Schematic Symbol", query, symbols, true);
    this.symbolStore.set(symbols);
    return symbols[0];
  }

  private mountSingle({ query, update, remove }: cache.MountParams<Key, Symbol>) {
    return [
      this.symbolEvents.onSet((symbol) => {
        if (symbol.key === query) update(symbol);
      }),
      this.symbolEvents.onDelete((key) => {
        if (key === query) remove(this.symbolStore.getTombstone(key)?.corpse);
      }),
    ];
  }

  private async fetchRequest(query: RetrieveRequest): Promise<Symbol[]> {
    if (query.parent != null) return await this.fetchChildren(query);
    if (isKeysOnly(query)) return await this.fetchKeys(query.keys as Key[]);
    const symbols = await this.execRetrieve(query);
    this.symbolStore.set(symbols);
    return symbols;
  }

  private async fetchChildren(query: RetrieveRequest): Promise<Symbol[]> {
    // retrieveChildren writes the parent relationships through, so membership
    // checks in requestFilter see them.
    const keys = await this.resolveChildKeys(query.parent as ontology.ID);
    if (keys.length === 0) return [];
    if (primitive.isZero(query.searchTerm)) return await this.fetchKeys(keys);
    const symbols = await this.execRetrieve({ keys, searchTerm: query.searchTerm });
    this.symbolStore.set(symbols);
    return symbols;
  }

  /**
   * Client-side approximation of the server's matching for a request: exact
   * for key sets and parent scopes, permissive for server-computed shapes
   * (search), which accept every change and drift toward the server's answer.
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

  private mountRequest({
    query,
    update,
  }: cache.MountParams<RetrieveRequest, Symbol[]>) {
    const matches = this.requestFilter(query);
    const listeners = [
      this.symbolEvents.onSet((symbol) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((s) => s.key === symbol.key);
          if (!matches(symbol))
            return existing ? prev.filter((s) => s.key !== symbol.key) : prev;
          if (existing) return prev.map((s) => (s.key === symbol.key ? symbol : s));
          return [...prev, symbol];
        });
      }),
      this.symbolEvents.onDelete((key) => {
        update((prev) => prev?.filter((s) => s.key !== key));
      }),
    ];
    const { parent } = query;
    if (parent == null) return listeners;
    return [
      ...listeners,
      this.relationshipEvents.onSet((rel) => {
        if (!matchChildRel(rel, parent)) return;
        void this.fetchKeys([rel.to.key]).then(([symbol]) => {
          if (symbol == null) return;
          update((prev) => {
            if (prev == null) return prev;
            if (prev.some((s) => s.key === symbol.key))
              return prev.map((s) => (s.key === symbol.key ? symbol : s));
            return [...prev, symbol];
          });
        });
      }),
      this.relationshipEvents.onDelete((relKey) => {
        const rel = ontology.relationshipZ.parse(relKey);
        if (!matchChildRel(rel, parent)) return;
        update((prev) => prev?.filter((s) => s.key !== rel.to.key));
      }),
    ];
  }
}

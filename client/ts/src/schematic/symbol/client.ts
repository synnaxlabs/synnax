// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FileTransport, type UnaryClient } from "@synnaxlabs/freighter";
import { array, primitive } from "@synnaxlabs/x";
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
const retrieveMultiParamsZ = retrieveRequestZ.or(query.keyListZ(keyZ));

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
const exportGroupReqZ = z.object({ key: group.keyZ });
const deleteGroupReqZ = z.object({ key: group.keyZ });

export interface CreateParams extends New {
  parent: ontology.ID;
}

export interface CreateMultipleParams {
  symbols: New[];
  parent: ontology.ID;
}

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm"] as const;

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

export interface ClientConfig {
  unary: UnaryClient;
  file: FileTransport;
  ontology: ontology.Client;
  cache: query.Cache;
  groupStore: query.Table<group.Key, group.Group>;
}

export class Client extends query.Retriever<typeof retrieveMultiParamsZ, Key, Symbol> {
  private readonly cfg: ClientConfig;
  private readonly store: query.Table<Key, Symbol>;

  constructor(cfg: ClientConfig) {
    const { cache, ontology: ontologyClient } = cfg;
    const store = cache.createTable<Key, Symbol>({
      name: "schematicSymbols",
      fetch: async (keys) => await this.execRetrieve({ keys }),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, symbolZ),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
    });
    super(cache, {
      name: "schematic symbol",
      table: store,
      request: {
        schema: retrieveMultiParamsZ,
        fetch: async (req) => await this.fetchRequest(req),
        matches: (sym, req) => this.requestFilter(req)(sym),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(
            ontologyClient.cache.relationships,
            (event, req: RetrieveRequest) => {
              if (req.parent == null) return null;
              const rel =
                event.variant === "set"
                  ? event.value
                  : ontology.relationshipZ.parse(event.key);
              if (!matchChildRel(rel, req.parent)) return null;
              return [rel.to.key];
            },
          ),
        ],
      },
    });
    this.cfg = cfg;
    this.store = store;
  }

  async create(options: CreateParams): Promise<Symbol>;
  async create(options: CreateMultipleParams): Promise<Symbol[]>;
  async create(
    options: CreateParams | CreateMultipleParams,
  ): Promise<Symbol | Symbol[]> {
    const isMany = "symbols" in options;
    const symbols = isMany ? options.symbols : [options];
    const res = await this.cfg.unary.send(
      "/schematic/symbol/create",
      { symbols, parent: options.parent },
      createReqZ,
      createResZ,
    );
    // Relationships first: parent-scoped answers check membership when the
    // symbol write lands.
    const rels = this.cfg.ontology.cache.relationships;
    res.symbols.forEach((s) => {
      const rel = childRel(options.parent, s.key);
      rels.set(ontology.relationshipToString(rel), rel);
    });
    this.store.set(res.symbols);
    return isMany ? res.symbols : res.symbols[0];
  }

  async rename(key: Key, name: string, opts: query.WriteOptions = {}): Promise<void> {
    const rename = () => query.partialUpdate(this.store, key, { name });
    await query.optimistic({
      rollbacks: [rename()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/schematic/symbol/rename",
          { key, name },
          renameReqZ,
          emptyResZ,
        ),
    });
    rename();
  }

  async delete(keys: Key | Key[], opts: query.WriteOptions = {}): Promise<void> {
    const keysArr = array.toArray(keys);
    const drop = () => this.store.delete(keysArr);
    await query.optimistic({
      rollbacks: [drop()],
      onOptimistic: opts.onOptimistic,
      commit: async () =>
        await this.cfg.unary.send(
          "/schematic/symbol/delete",
          { keys: keysArr },
          deleteReqZ,
          emptyResZ,
        ),
    });
    drop();
    this.cfg.ontology.cache.relationships.delete(
      (r) =>
        r.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
        r.to.type === "schematic_symbol" &&
        keysArr.includes(r.to.key),
    );
  }

  /**
   * Exports every symbol in the group as a bundle: a zip archive holding one JSON file
   * per symbol beside a manifest.json naming the group. The caller pipes the stream
   * wherever it likes without the client buffering the whole archive.
   *
   * @param key - the key of the group to export.
   * @returns the bundle as a stream of zip bytes.
   * @throws {ValidationError} if the group holds a resource that is not a symbol, or if
   * two symbols take the same file name.
   */
  async exportGroup(key: group.Key): Promise<ReadableStream<Uint8Array>> {
    return await this.cfg.file.download(
      "/schematic/symbol/group/export",
      { key },
      exportGroupReqZ,
      { encoding: "ZIP" },
    );
  }

  /**
   * Deletes the group and every symbol in it. The Core removes both in a single
   * transaction, so a failure leaves the group and its symbols untouched.
   *
   * @param key - the key of the group to delete.
   * @throws {ValidationError} if the group holds a resource that is not a symbol.
   */
  async deleteGroup(key: group.Key): Promise<void> {
    const groupID = group.ontologyID(key);
    const rels = this.cfg.ontology.cache.relationships;
    // Read the members before the delete drops the relationships naming them.
    const memberKeys = rels.get((r) => matchChildRel(r, groupID)).map((r) => r.to.key);
    await this.cfg.unary.send(
      "/schematic/symbol/group/delete",
      { key },
      deleteGroupReqZ,
      emptyResZ,
    );
    this.store.delete(memberKeys);
    this.cfg.groupStore.delete(key);
    // Both the relationships to the group's children and the one naming the group as
    // its parent's child: the group and every symbol in it are gone.
    rels.delete(
      (r) =>
        r.type === ontology.PARENT_OF_RELATIONSHIP_TYPE &&
        (ontology.idsEqual(r.from, groupID) || ontology.idsEqual(r.to, groupID)),
    );
  }

  async retrieveGroup(): Promise<group.Group> {
    const res = await this.cfg.unary.send(
      "/schematic/symbol/retrieve-group",
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return res.group;
  }

  private async execRetrieve(
    params: RetrieveSingleParams | z.input<typeof wireRetrieveRequestZ>,
  ): Promise<Symbol[]> {
    const res = await this.cfg.unary.send(
      "/schematic/symbol/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    return res.symbols;
  }

  /** Resolves a parent scope to its symbol children's keys via the ontology. */
  private async resolveChildKeys(parent: ontology.ID): Promise<Key[]> {
    // The children space writes the parent relationships through, so
    // membership checks in requestFilter see them.
    const children = await this.cfg.ontology.children.retrieve({
      ids: parent,
      types: ["schematic_symbol"],
    });
    return children.map(({ id: { key } }) => key);
  }

  private async fetchRequest(req: RetrieveRequest): Promise<Symbol[]> {
    const { parent, ...rest } = req;
    if (parent == null) return await this.execRetrieve(rest);
    const keys = await this.resolveChildKeys(parent);
    if (keys.length === 0) return [];
    if (primitive.isZero(rest.searchTerm)) return await this.store.retrieve(keys);
    return await this.execRetrieve({ keys, searchTerm: rest.searchTerm });
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
    return this.cfg.ontology.cache.relationships.has(
      ontology.relationshipToString(childRel(parent, key)),
    );
  }
}

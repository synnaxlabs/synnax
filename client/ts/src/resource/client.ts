// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

import { query } from "@/query";
import {
  type ID,
  idToString,
  idZ,
  parseID,
  type Resource,
  resourceZ,
} from "@/resource/payload";
import { type Type, typeZ } from "@/resource/types.gen";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

const searchReqZ = z.object({
  /** The text that a resource must match. An empty term lists resources by type. */
  searchTerm: z.string().optional(),
  /** Limits the search to resources of these types. Empty searches every type. */
  types: typeZ.array().optional(),
  /** The maximum number of resources to return, at most 1000. Zero returns 100. */
  limit: z.int().optional(),
});
/** Selects the resources that {@link Client.search} returns. */
export interface SearchParams extends z.input<typeof searchReqZ> {}

/** A search with its zero fields left out, so equal searches share one answer. */
type Search = {
  searchTerm?: string;
  types?: Type[];
  limit: number;
};

type ListQuery = query.ListQuery<never, string> & {
  readonly types?: Type[];
  readonly [key: string]: query.Params;
};

/** The number of resources that the Core returns for a search that sets no limit. */
const DEFAULT_LIMIT = 100;

// A search with no limit still has the Core's cap, so the cache must know it to fill a
// slot that a delete empties.
const normalize = ({ searchTerm, types, limit }: SearchParams): Search => ({
  ...(searchTerm != null && searchTerm !== "" && { searchTerm }),
  ...(types != null && types.length > 0 && { types }),
  limit: limit == null || limit === 0 ? DEFAULT_LIMIT : limit,
});

const retrieveReqZ = z.object({ ids: idZ.array() });
const resZ = z.object({ resources: resourceZ.array().default(() => []) });

const keyZ = z.union([z.string(), z.number()]);
const keyedZ = z.object({ key: keyZ });

/** The channels on which the Core announces the changes to the records of a type. */
export interface Channels {
  /** Carries a JSON sample with the key of each record set. */
  set: string;
  /** Carries the key of each record deleted. */
  delete: string;
}

const isID = (target: ID | SearchParams): target is ID => "type" in target;

/** Configuration for a {@link Client}. */
export interface ClientConfig {
  unary: UnaryClient;
  cache: query.Cache;
  /**
   * The change channels of each type. The searches of a type with null channels do not
   * stay current.
   */
  channels: Record<Type, Channels | null>;
}

/**
 * Searches and names records of every type on a Core. Reach it through
 * `client.resources`. A subscribed search stays current for the types that the client
 * has change channels for.
 */
export class Client {
  private readonly client: UnaryClient;
  private readonly table: query.Table<string, Resource>;
  private readonly searches: query.Space<Search, Resource[], string, Resource>;
  private readonly byKey: query.Space<string, Resource, string, Resource>;
  private readonly byKeys: query.Space<string[], Resource[], string, Resource>;
  private readonly lists: query.Lists<
    ListQuery,
    Resource,
    string,
    Resource,
    string,
    never
  >;

  constructor({ unary, cache, channels }: ClientConfig) {
    this.client = unary;
    this.table = cache.createTable<string, Resource>({
      name: "resources",
      fetch: async (keys) => await this.execRetrieve(keys.map((k) => parseID(k))),
      // A record deleted and created again keeps its key.
      revivable: true,
      listen: record
        .entries(channels)
        .flatMap(([type, ch]) => (ch == null ? [] : this.listeners(type, ch))),
    });
    // The Core hides records that a Resource cannot show, so it answers every search.
    this.searches = cache.queries<Search, Resource[], string, Resource>({
      name: "resources",
      table: this.table,
      fetch: async (q) =>
        await this.table.track(async (since) => {
          const resources = await this.execSearch(q);
          this.table.ingest(resources, { since });
          return resources.map((r) => r.key);
        }),
      compose: (records) => records,
    });
    // The Core alone ranks a search, so each list is one window in its order.
    this.lists = cache.lists<ListQuery, Resource, string, Resource, string, never>({
      name: "resources",
      table: this.table,
      fetch: async ({ searchTerm, types, limit }) =>
        await this.execSearch(normalize({ searchTerm, types, limit })),
      compose: (record) => record,
      cursor: ({ key }) => key,
      compare: (a, b) => a.localeCompare(b),
    });
    this.byKey = cache.queries<string, Resource, string, Resource>({
      name: "Resource",
      table: this.table,
      fetch: async (key) => {
        const resources = await this.table.retrieve([key]);
        checkForMultipleOrNoResults(
          "resource",
          { where: { id: { eq: key } } },
          resources,
          true,
        );
        return [key];
      },
      compose: ([r]) => r,
      keyOf: (key) => key,
      single: true,
    });
    this.byKeys = cache.queries<string[], Resource[], string, Resource>({
      name: "resources",
      table: this.table,
      fetch: async (keys) => (await this.table.retrieve(keys)).map((r) => r.key),
      compose: (records, keys) => {
        const byKey = new Map(records.map((r) => [r.key, r]));
        return [...new Set(keys)].flatMap((k) => byKey.get(k) ?? []);
      },
      keysOf: (keys) => keys,
      matches: (r, keys) => keys.includes(r.key),
    });
  }

  /**
   * @returns the resources whose text matches the term, closest match first. An empty
   * term lists the resources in order of type.
   * @throws {ValidationError} if a type is unknown, or the limit is out of bounds.
   */
  async search(params: SearchParams): Promise<Resource[]> {
    return await this.searches.retrieve(normalize(params));
  }

  /**
   * Returns the live resource that the ID names, the live resources that a list of IDs
   * names, in the order of the IDs, or the live list of the resources that match the
   * search, closest match first. An empty term lists the resources in order of type.
   * Renames, creates, and deletes of the types that the client has change channels for
   * keep the answer current.
   */
  watch(id: ID): query.Live<Resource>;
  watch(ids: ID[]): query.Live<Resource[]>;
  watch(params: SearchParams): query.List<Resource, string>;
  watch(
    target: ID | ID[] | SearchParams,
  ): query.Live<Resource> | query.Live<Resource[]> | query.List<Resource, string> {
    if (Array.isArray(target)) return this.byKeys.watch(idToString(target));
    if (isID(target)) return this.byKey.watch(idToString(target));
    const { searchTerm, types, limit } = target;
    return this.lists.list({
      ...(searchTerm != null && searchTerm !== "" && { searchTerm }),
      ...(types != null && types.length > 0 && { types }),
      ...(limit != null && { limit }),
    });
  }

  /**
   * Retrieves the ID and name of each resource. The list form returns each resource
   * once, in the order of the IDs, and leaves out each ID that names no resource.
   * @throws {NotFoundError} if the single ID names no resource.
   * @throws {ValidationError} if an ID has an unknown type.
   */
  async retrieve(id: ID): Promise<Resource>;
  async retrieve(ids: ID[]): Promise<Resource[]>;
  async retrieve(ids: ID | ID[]): Promise<Resource | Resource[]> {
    if (Array.isArray(ids)) return await this.byKeys.retrieve(idToString(ids));
    return await this.byKey.retrieve(idToString(ids));
  }

  private listeners(
    type: Type,
    { set, delete: del }: Channels,
  ): query.ListenerSpec<string, Resource>[] {
    const toKey = (key: string | number): string => idToString({ type, key: `${key}` });
    const onSet: query.ListenerSpec<string, Resource> = {
      bind: (table): query.Listener<typeof keyedZ> => ({
        channel: set,
        schema: keyedZ,
        // The caller may not be able to read a record that the table does not hold,
        // so the searches that cover its type and the answers of its ID refetch
        // instead of reading it.
        onChange: (changed) => {
          const keys = changed.map(({ key }) => toKey(key));
          const unheld = keys.filter((k) => !table.has(k));
          if (unheld.length > 0) {
            const covers = (q: { types?: Type[] }) => q.types?.includes(type) ?? true;
            this.searches.refetchMatching(covers);
            this.lists.refetchMatching(covers);
            this.byKey.refetchMatching((k) => unheld.includes(k));
            this.byKeys.refetchMatching((ks) => ks.some((k) => unheld.includes(k)));
          }
          table.refetch(keys.filter((k) => table.has(k)));
        },
      }),
    };
    return [
      onSet,
      query.createDeleteListener<typeof keyZ, string, Resource>(del, keyZ, {
        key: toKey,
      }),
    ];
  }

  private async execSearch(q: Search): Promise<Resource[]> {
    const { resources } = await this.client.send(
      "/resource/search",
      q,
      searchReqZ,
      resZ,
    );
    return resources;
  }

  private async execRetrieve(ids: ID[]): Promise<Resource[]> {
    const { resources } = await this.client.send(
      "/resource/retrieve",
      { ids },
      retrieveReqZ,
      resZ,
    );
    return resources;
  }
}

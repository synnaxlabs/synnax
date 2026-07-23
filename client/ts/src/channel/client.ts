// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import {
  array,
  control,
  type CrudeDensity,
  type CrudeTimeRange,
  type CrudeTimeSpan,
  type CrudeTimeStamp,
  DataType,
  deep,
  type destructor,
  errors,
  type MultiSeries,
  primitive,
  TimeSpan,
  type TypedArray,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Params, type PrimitiveParams, statusKey } from "@/channel/payload";
import {
  analyzeParams,
  DebouncedBatchRetriever,
  type RetrieveOptions,
  type Retriever,
  type RetrieveRequest,
  retrieveRequestZ,
} from "@/channel/retriever";
import {
  type Key,
  keyZ,
  type Name,
  type New,
  ontologyID,
  type Operation,
  type Payload,
  payloadZ,
  statusZ,
} from "@/channel/types.gen";
import { type Writer } from "@/channel/writer";
import { NotFoundError, ValidationError } from "@/errors";
import { type framer } from "@/framer";
import { group } from "@/group";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { type ranger } from "@/ranger";
import { createKey, decodeDeleteChange } from "@/ranger/alias/payload";
import { status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_channel_set";
export const DELETE_CHANNEL_NAME = "sy_channel_delete";

interface CreateOptions {
  retrieveIfNameExists?: boolean;
}

/**
 * Represents a Channel in a Synnax database. Typically, channels should not be
 * instantiated directly, but instead created via the `.channels.create` or retrieved
 * via the `.channels.retrieve` method on a Synnax client.
 *
 * Please refer to the [Synnax
 * documentation](https://docs.synnaxlabs.com/reference/concepts/channels) for detailed
 * information on what channels are and how to use them.
 */
export class Channel {
  private readonly frameClient: framer.Client | null;
  /**
   * A unique key identifying the channel in the Synnax database. This key is
   * automatically assigned by Synnax.
   */
  readonly key: Key;
  /**
   * A human-readable name for the channel. This name is not guaranteed to be unique.
   */
  readonly name: string;
  /**
   * The data type of the channel.
   */
  readonly dataType: DataType;
  /**
   * The key of the node in the Synnax cluster that holds the 'lease' over the channel
   * i.e. it's the only node in the cluster allowed to accept writes to the channel. This
   * property is mostly for internal use.
   */
  readonly leaseholder: number;
  /**
   * The key of the index channel that this channel is associated with i.e. the channel
   * that stores its timestamps.
   */
  readonly index: Key;
  /**
   * This is set to true if the channel is an index channel, and false otherwise.
   */
  readonly isIndex: boolean;
  /**
   * This is set to true if the channel is an internal channel, and false otherwise.
   */
  readonly internal: boolean;
  /**
   * An alias for the channel under a specific range. This parameter is unstable and
   * should not be relied upon in the current version of Synnax.
   */
  alias: string | undefined;
  /**
   * Whether the channel is virtual. Virtual channels do not store any data in the
   * database, but can still be used for streaming purposes.
   */
  readonly virtual: boolean;
  /**
   * Only used for calculated channels. Specifies the Arc expression used to evaluate
   * the calculated value
   */
  readonly expression: string;
  readonly operations: Operation[];
  readonly concurrency: control.Concurrency;
  /**
   * The status of the channel.
   */
  readonly status?: status.Status;

  constructor({
    dataType,
    name,
    leaseholder = 0,
    key = 0,
    isIndex = false,
    index = 0,
    internal = false,
    virtual = false,
    frameClient,
    alias,
    status: argsStatus,
    expression = "",
    operations = [],
    concurrency = control.Concurrency.exclusive,
  }: New & {
    internal?: boolean;
    frameClient?: framer.Client;
    density?: CrudeDensity;
    status?: status.New;
    operations?: Operation[];
  }) {
    this.key = keyZ.parse(key);
    this.name = name;
    this.dataType = new DataType(dataType);
    this.leaseholder = leaseholder;
    this.index = keyZ.parse(index);
    this.isIndex = isIndex;
    this.internal = internal;
    this.alias = alias;
    this.virtual = virtual;
    this.expression = expression;
    this.operations = operations;
    this.concurrency = concurrency;
    if (argsStatus != null) this.status = status.create(argsStatus);
    this.frameClient = frameClient ?? null;
  }

  private get framer(): framer.Client {
    if (this.frameClient == null)
      throw new ValidationError("cannot read from a channel that has not been created");
    return this.frameClient;
  }

  /**
   * Returns the payload representation of this channel i.e. a pure JS object with
   * all of the channel fields but without any methods. This is used internally for
   * network transportation, but also provided to you as a convenience.
   */
  get payload(): Payload {
    return payloadZ.parse({
      key: this.key,
      name: this.name,
      dataType: this.dataType.valueOf(),
      leaseholder: this.leaseholder,
      index: this.index,
      isIndex: this.isIndex,
      internal: this.internal,
      virtual: this.virtual,
      expression: this.expression,
      status: this.status,
      operations: this.operations,
    });
  }

  get isCalculated(): boolean {
    return isCalculated(this.payload);
  }

  /***
   * @returns the ontology ID of the channel
   */
  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   *
   * @param start - The starting timestamp of the range to read from.
   * @param end - The ending timestamp of the range to read from.
   * @returns A typed array containing the retrieved
   */
  async read(tr: CrudeTimeRange): Promise<MultiSeries> {
    return await this.framer.read(tr, this.key);
  }

  /**
   * Writes telemetry to the channel starting at the given timestamp.
   *
   * @param start - The starting timestamp of the first sample in data.
   * @param data - THe telemetry to write to the channel.
   */
  async write(start: CrudeTimeStamp, data: TypedArray): Promise<void> {
    return await this.framer.write(start, this.key, data);
  }
}

const retrieveGroupReqZ = z.object({});

const retrieveGroupResZ = z.object({ group: group.groupZ });

export type RetrieveSingleParams = { key: Key; rangeKey?: ranger.Key };

type NormalizedRequest = z.infer<typeof retrieveRequestZ>;

/** Aliases and statuses are composed from their stores on read. */
const stripComposed = ({ alias: _alias, status: _status, ...rest }: Payload): Payload =>
  rest;

const normalizeSingle = ({ key, rangeKey }: RetrieveSingleParams) =>
  rangeKey == null ? { key } : { key, rangeKey };

const onlyRangeKey = (options?: RetrieveOptions): boolean =>
  options == null ||
  Object.entries(options).every(([k, v]) => v === undefined || k === "rangeKey");

const isKeysOnly = (
  req: NormalizedRequest,
): req is NormalizedRequest & { keys: Key[] } =>
  primitive.isNonZero(req.keys) &&
  req.names == null &&
  req.searchTerm == null &&
  req.nodeKey == null &&
  req.limit == null &&
  req.offset == null &&
  req.dataTypes == null &&
  req.notDataTypes == null &&
  req.virtual == null &&
  req.isIndex == null &&
  req.internal == null &&
  req.legacyCalculated == null;

/** Query fields only the server can evaluate. */
const SERVER_FIELDS = ["searchTerm", "limit", "offset"] as const;

const NAME_LITERAL_PATTERN = /^[a-zA-Z_][a-zA-Z0-9_]*$/;

/**
 * Mirrors the server's name matching: literal names compare exactly, anything
 * else compiles as a regular expression, anchored when unanchored.
 */
const createNameMatcher = (pattern: string): ((name: string) => boolean) => {
  if (NAME_LITERAL_PATTERN.test(pattern)) return (name) => name === pattern;
  const anchored =
    pattern.startsWith("^") || pattern.endsWith("$") ? pattern : `^${pattern}$`;
  try {
    const rx = new RegExp(anchored);
    return (name) => rx.test(name);
  } catch {
    return (name) => name === pattern;
  }
};

/**
 * Client-side matching for a request over every locally evaluable field.
 * Server-computed shapes (search, pagination) never reach this filter; they
 * refetch instead.
 */
const requestFilter = (req: NormalizedRequest): ((ch: Channel) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameMatchers = primitive.isNonZero(req.names)
    ? req.names.map(createNameMatcher)
    : undefined;
  return (ch) => {
    if (keySet != null && !keySet.has(ch.key)) return false;
    if (nameMatchers != null && !nameMatchers.some((m) => m(ch.name))) return false;
    if (primitive.isNonZero(req.nodeKey) && ch.leaseholder !== req.nodeKey)
      return false;
    // The server's virtual bucket excludes calculated channels even though
    // they are stored with virtual=true.
    if (req.virtual != null && (ch.virtual && !ch.isCalculated) !== req.virtual)
      return false;
    if (req.isIndex != null && ch.isIndex !== req.isIndex) return false;
    if (req.internal != null && ch.internal !== req.internal) return false;
    if (
      primitive.isNonZero(req.dataTypes) &&
      !req.dataTypes.some((dt) => dt.equals(ch.dataType))
    )
      return false;
    if (
      primitive.isNonZero(req.notDataTypes) &&
      req.notDataTypes.some((dt) => dt.equals(ch.dataType))
    )
      return false;
    return true;
  };
};

const createTable = (
  cache: query.Cache,
  retriever: Retriever,
  sugar: (payload: Payload) => Channel,
): query.Table<Key, Channel> => {
  const table = cache.createTable<Key, Channel>({
    name: "channels",
    equal: (a, b) => deep.equal(a.payload, b.payload),
    refetch: async (keys) =>
      (await retriever.retrieve(keys)).map((p) => sugar(stripComposed(p))),
  });
  const set: query.ChannelListener<typeof payloadZ> = {
    channel: SET_CHANNEL_NAME,
    schema: payloadZ,
    onChange: (changed) => table.set(changed.key, sugar(stripComposed(changed))),
  };
  const del: query.ChannelListener<typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: (changed) => table.delete(changed),
  };
  cache.addListeners(table, set, del);
  return table;
};

/**
 * The main client class for executing channel operations against a Synnax Core. This
 * class should not be instantiated directly, and instead should be used through the
 * `channels` property of an {@link Synnax} client.
 */
export class Client extends query.Retriever<
  RetrieveSingleParams,
  RetrieveRequest,
  RetrieveSingleParams,
  NormalizedRequest,
  Channel
> {
  private readonly frameClient: framer.Client;
  private readonly client: UnaryClient;
  readonly retriever: Retriever;
  readonly writer: Writer;
  private readonly statuses?: status.Client;
  private readonly ranges?: ranger.Client;
  private readonly store: query.Table<Key, Channel>;
  private readonly statusStore: query.Table<status.Key, status.Status>;
  private readonly aliasStore: query.Table<string, ranger.alias.Alias>;
  private readonly ontology: ontology.Stores;

  constructor(
    frameClient: framer.Client,
    retriever: Retriever,
    client: UnaryClient,
    writer: Writer,
    statuses: status.Client | undefined,
    ranges: ranger.Client | undefined,
    cache: query.Cache,
    statusStore: query.Table<status.Key, status.Status>,
    aliasStore: query.Table<string, ranger.alias.Alias>,
    ontologyStores: ontology.Stores,
  ) {
    const store = createTable(cache, retriever, (payload) => this.sugar(payload));
    super({
      single: cache.queries({
        name: "channel",
        table: store,
        fetch: async (query) => [(await this.fetchSingle(query)).key],
        compose: (records, query) => this.compose(records[0].payload, query.rangeKey),
        keyOf: (query) => query.key,
        single: true,
        watch: [
          query.watch(statusStore, (event, query: RetrieveSingleParams) =>
            event.key === statusKey(query.key) ? [query.key] : null,
          ),
          query.watch(aliasStore, (event, query: RetrieveSingleParams) => {
            if (query.rangeKey == null) return null;
            const aliasKey = createKey({ range: query.rangeKey, channel: query.key });
            return event.key === aliasKey ? [query.key] : null;
          }),
        ],
      }),
      request: cache.queries({
        name: "channels",
        table: store,
        fetch: async (query) => (await this.fetchRequest(query)).map((ch) => ch.key),
        compose: (records, query) => {
          const { rangeKey } = query;
          if (rangeKey == null) return records;
          return records.map((ch) => this.composeAlias(ch.payload, rangeKey));
        },
        matches: (ch, query) => requestFilter(query)(ch),
        serverFields: SERVER_FIELDS,
        watch: [
          query.watch(aliasStore, (event, query: NormalizedRequest) => {
            if (query.rangeKey == null) return null;
            if (event.variant === "set")
              return event.value.range === query.rangeKey
                ? [event.value.channel]
                : null;
            const decoded = decodeDeleteChange(event.key);
            return decoded.range === query.rangeKey ? [decoded.channel] : null;
          }),
        ],
        hydrate: async (keys) => {
          await this.fetchKeys(keys);
        },
      }),
      isSingle: (params) => "key" in params,
      normalizeSingle,
      normalizeRequest: (params) => retrieveRequestZ.parse(params),
    });
    this.frameClient = frameClient;
    this.retriever = retriever;
    this.client = client;
    this.writer = writer;
    this.statuses = statuses;
    this.ranges = ranges;
    this.statusStore = statusStore;
    this.aliasStore = aliasStore;
    this.ontology = ontologyStores;
    this.store = store;
  }

  /**
   * Creates a single channel with the given properties.
   *
   * @param name - A human-readable name for the channel.
   * @param rate - The rate of the channel. This only applies to fixed rate channels.
   * @param dataType - The data type for the samples stored in the channel.
   * @param index - The key of the index channel that this channel should be associated
   * with. An 'index' channel is a channel that stores timestamps for other channels.
   * Refer to the Synnax documentation
   * (https://docs.synnaxlabs.com/reference/concepts/channels) for more information. The
   * index channel must have already been created. This field does not need to be
   * specified if the channel is an index channel, or the channel is a fixed rate
   * channel. If this value is specified, the 'rate' parameter will be ignored.
   * @param isIndex - Set to true if the channel is an index channel, and false
   * otherwise. Index channels must have a data type of `DataType.TIMESTAMP`.
   * @returns the created channel. {@link Channel}
   * @throws {ValidationError} if any of the parameters for creating the channel are
   * invalid.
   *
   * @example
   * ```typescript
   * const indexChannel = await client.channels.create({
   *    name: "time",
   *    dataType: DataType.TIMESTAMP,
   *    isIndex: true,
   * })
   *
   *
   * const dataChannel = await client.channels.create({
   *    name: "temperature",
   *    dataType: DataType.FLOAT,
   *    index: indexChannel.key,
   * });
   * ```
   */
  async create(channel: New, options?: CreateOptions): Promise<Channel>;

  /**
   * Creates multiple channels with the given properties. The order of the channels
   * returned is guaranteed to match the order of the channels passed in.
   *
   * @param channels - An array of channel properties to create.
   * For each channel, the following properties should be considered:
   *
   * @param name - A human-readable name for the channel.
   * @param rate - The rate of the channel. This only applies to fixed rate channels. If
   * the 'index' parameter is specified or 'isIndex' is set to true, this parameter will
   * be ignored.
   * @param dataType - The data type for the samples stored in the channel.
   * @param index - The key of the index channel that this channel should be associated
   * with. An 'index' channel is a channel that stores timestamps for other channels. Refer
   * to the Synnax documentation (https://docs.synnaxlabs.com) for more information. The
   * index channel must have already been created. This field does not need to be specified
   * if the channel is an index channel, or the channel is a fixed rate channel. If this
   * value is specified, the 'rate' parameter will be ignored.
   * @param isIndex - Set to true if the channel is an index channel, and false otherwise.
   * Index channels must have a data type of `DataType.TIMESTAMP`.
   *
   * @param channels
   */
  async create(channels: New[], options?: CreateOptions): Promise<Channel[]>;

  async create(
    channels: New | New[],
    options: CreateOptions = {},
  ): Promise<Channel | Channel[]> {
    const { retrieveIfNameExists = false } = options;
    const single = !Array.isArray(channels);
    let toCreate = array.toArray(channels);
    let created: Channel[] = [];
    if (retrieveIfNameExists) {
      const res = await this.retriever.retrieve(toCreate.map((c) => c.name));
      const existingNames = new Set(res.map((c) => c.name));
      toCreate = toCreate.filter((c) => !existingNames.has(c.name));
      created = this.sugar(res);
    }
    created = created.concat(this.sugar(await this.writer.create(toCreate)));
    this.store.set(created.map((ch) => this.sugar(stripComposed(ch.payload))));
    return single ? created[0] : created;
  }

  /**
   * Retrieves a channel from the database using the given key or name.
   *
   * @param params - The key or name of the channel to retrieve.
   * @param options - Optional parameters to control the retrieval process.
   * @param options.dataTypes - Limits the query to only channels with the specified data
   * type.
   * @param options.notDataTypes - Limits the query to only channels without the specified
   * data type.
   *
   * @returns The retrieved channel.
   * @throws {NotFoundError} if the channel does not exist in the cluster.
   * @throws {MultipleFoundError} is only thrown if the channel is retrieved by name,
   * and multiple channels with the same name exist in the cluster.
   *
   * @example
   *
   * ```typescript
   * const channel = await client.channels.retrieve("temperature");
   * const channel = await client.channels.retrieve(1);
   * ```
   */
  async retrieve(params: Key | Name, options?: RetrieveOptions): Promise<Channel>;

  /**
   * Retrieves multiple channels from the database using the provided keys or the
   * provided names.
   *
   * @param params - The keys or the names of the channels to retrieve. Note that
   * this method does not support mixing keys and names in the same call.
   * @param options - Optional parameters to control the retrieval process.
   * @param options.dataTypes - Limits the query to only channels with the specified data
   * type.
   * @param options.notDataTypes - Limits the query to only channels without the specified
   *
   */
  async retrieve(
    params: PrimitiveParams | Payload[],
    options?: RetrieveOptions,
  ): Promise<Channel[]>;

  async retrieve(params: RetrieveSingleParams): Promise<Channel>;

  async retrieve(params: RetrieveRequest): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given parameters.
   *
   * this will be ignored.
   * @returns The retrieved channel.
   * @raises {QueryError} If the channel does not exist or if multiple results are returned.
   */
  async retrieve(
    params: PrimitiveParams | Payload[] | RetrieveSingleParams | RetrieveRequest,
    options?: RetrieveOptions,
  ): Promise<Channel | Channel[]> {
    if (typeof params === "object" && !Array.isArray(params))
      return await super.retrieve(params);
    const isSingle = !Array.isArray(params);
    const { variant, normalized: rawNormalized } = analyzeParams(params);
    const normalized =
      variant === "keys" ? rawNormalized.filter((k) => k !== 0) : rawNormalized;
    if (normalized.length === 0) {
      checkForMultipleOrNoResults<Params, Channel>("channel", params, [], isSingle);
      return [];
    }
    if (isSingle && variant === "keys" && onlyRangeKey(options))
      return await super.retrieve({
        key: normalized[0] as Key,
        rangeKey: options?.rangeKey,
      });
    const res = await super.retrieve(
      retrieveRequestZ.parse({ [variant]: normalized, ...options }),
    );
    checkForMultipleOrNoResults<Params, Channel>("channel", params, res, isSingle);
    return isSingle ? res[0] : res;
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached. Unfetched filter queries
   * are approximated from the record store when possible.
   */
  getCached(params: RetrieveSingleParams): query.Cached<Channel> | undefined;
  getCached(params: RetrieveRequest): query.Cached<Channel[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveRequest,
  ): query.Cached<Channel> | query.Cached<Channel[]> | undefined {
    if ("key" in params) return super.getCached(params);
    return (
      super.getCached(params) ?? this.approximateCached(retrieveRequestZ.parse(params))
    );
  }

  /**
   * Approximates an unfetched filter query's answer from the record store.
   * Server-computed shapes (search, pagination) cannot be approximated.
   */
  private approximateCached(
    req: NormalizedRequest,
  ): query.Cached<Channel[]> | undefined {
    if (req.searchTerm != null || req.limit != null || req.offset != null)
      return undefined;
    const matches = this.store.get(requestFilter(req));
    if (matches.length === 0) return undefined;
    return { variant: "changed", data: matches };
  }

  /***
   * Deletes channels from the database using the given keys or names.
   * @param params - The keys or names of the channels to delete.
   */
  async delete(params: Params, opts: query.WriteOptions = {}): Promise<void> {
    const { normalized, variant } = analyzeParams(params);
    if (variant === "keys") {
      const keys = normalized;
      const rollback = new query.Rollback();
      const ids = ontologyID(keys);
      rollback.add(ontology.deleteCachedRelationships(this.ontology, ids));
      rollback.add(this.store.delete(keys));
      rollback.add(this.ontology.resources.delete(ontology.idToString(ids)));
      await opts.onOptimistic?.();
      await rollback.guard(async () => await this.writer.delete({ keys }));
      // Re-applied after success: a stale streamer echo may have resurrected
      // an optimistically deleted entry while the send was in flight.
      this.store.delete(keys);
      return;
    }
    const names = normalized;
    await this.writer.delete({ names });
    const cached = this.store.get((ch) => names.includes(ch.name));
    if (cached.length > 0) this.store.delete(cached.map((ch) => ch.key));
  }

  async rename(key: Key, name: string, opts?: query.WriteOptions): Promise<void>;
  async rename(keys: Key[], names: string[], opts?: query.WriteOptions): Promise<void>;
  async rename(
    keys: Key | Key[],
    names: string | string[],
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const keysArr = array.toArray(keys);
    const namesArr = array.toArray(names);
    const rollback = new query.Rollback();
    keysArr.forEach((key, i) => {
      const name = namesArr[i];
      rollback.add(this.renameThrough(key, name));
      rollback.add(ontology.renameCachedResource(this.ontology, ontologyID(key), name));
    });
    await opts.onOptimistic?.();
    await rollback.guard(async () => await this.writer.rename(keysArr, namesArr));
    // Re-applied after success: a stale streamer echo may have clobbered the
    // optimistic write while the send was in flight.
    keysArr.forEach((key, i) => this.renameThrough(key, namesArr[i]));
  }

  /** Renames the cached channel, skipping when absent. Returns a rollback. */
  private renameThrough(key: Key, name: string): destructor.Destructor {
    return this.store.set(key, (p) =>
      p == null ? undefined : this.sugar({ ...p.payload, name }),
    );
  }

  createDebouncedBatchRetriever(
    deb: CrudeTimeSpan = TimeSpan.milliseconds(10),
  ): Retriever {
    const cached: Retriever = {
      retrieve: async (
        channels: Params | RetrieveRequest,
        options?: RetrieveOptions,
      ): Promise<Payload[]> => {
        if (!Array.isArray(channels) && typeof channels === "object")
          return await this.retriever.retrieve(channels);
        const { variant, normalized } = analyzeParams(channels);
        if (variant === "keys" && options == null)
          return (await this.fetchKeys(normalized)).map((ch) => ch.payload);
        return await this.retriever.retrieve(channels, options);
      },
    };
    return new DebouncedBatchRetriever(cached, deb);
  }

  sugar(payload: Payload): Channel;
  sugar(payloads: Payload[]): Channel[];
  sugar(payloads: Payload | Payload[]): Channel | Channel[] {
    const { frameClient } = this;
    if (Array.isArray(payloads))
      return payloads.map((p) => new Channel({ ...p, frameClient }));
    return new Channel({ ...payloads, frameClient });
  }

  async retrieveGroup(): Promise<group.Group> {
    const res = await this.client.send(
      "/channel/retrieve-group",
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return res.group;
  }

  private requireStatuses(): status.Client {
    if (this.statuses == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.statuses;
  }

  private requireRanges(): ranger.Client {
    if (this.ranges == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.ranges;
  }

  /** Rebuilds a cached channel with its cached status and alias attached. */
  private compose(payload: Payload, rangeKey?: ranger.Key): Channel {
    const next: Payload = { ...payload, status: undefined, alias: undefined };
    if (isCalculated(payload)) {
      const parsed = statusZ.safeParse(this.statusStore.get(statusKey(payload.key)));
      if (parsed.success) next.status = parsed.data;
    }
    if (rangeKey != null)
      next.alias = this.aliasStore.get(
        createKey({ range: rangeKey, channel: payload.key }),
      )?.alias;
    return this.sugar(next);
  }

  private composeAlias(payload: Payload, rangeKey: ranger.Key): Channel {
    return this.sugar({
      ...payload,
      alias: this.aliasStore.get(createKey({ range: rangeKey, channel: payload.key }))
        ?.alias,
    });
  }

  /** Fetches aliases under the range for keys the alias store has no entry for. */
  private async ensureAliases(rangeKey: ranger.Key, keys: Key[]): Promise<void> {
    const missing = keys.filter(
      (key) => !this.aliasStore.has(createKey({ range: rangeKey, channel: key })),
    );
    if (missing.length === 0) return;
    const fetched = await this.requireRanges().retrieveAliases(rangeKey, missing);
    Object.entries(fetched).forEach(([channel, alias]) => {
      const entry: ranger.alias.Alias = {
        range: rangeKey,
        channel: Number(channel),
        alias,
      };
      this.aliasStore.set(createKey(entry), entry);
    });
  }

  private async fetchSingle(query: RetrieveSingleParams): Promise<Channel> {
    const { key, rangeKey } = query;
    let ch = this.store.get(key);
    if (ch == null) {
      const payloads = await this.retriever.retrieve([key]);
      checkForMultipleOrNoResults("channel", key, payloads, true);
      ch = this.sugar(stripComposed(payloads[0]));
      this.store.set(key, ch);
    }
    // A cached calculated channel without a cached status is ambiguous: the
    // status may not exist, or may simply never have been fetched.
    if (ch.isCalculated && !this.statusStore.has(statusKey(key)))
      try {
        await this.requireStatuses().retrieve({ key: statusKey(key) });
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      }
    if (rangeKey != null) await this.ensureAliases(rangeKey, [key]);
    return ch;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Channel[]> {
    const results: Channel[] = [];
    const misses: Key[] = [];
    keys.forEach((key) => {
      const cached = this.store.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    });
    if (misses.length > 0) {
      const fetched = (await this.retriever.retrieve(misses)).map((p) =>
        this.sugar(stripComposed(p)),
      );
      this.store.set(fetched);
      results.push(...fetched);
    }
    return query.orderByKeys(keys, results, (ch) => ch.key);
  }

  private async fetchRequest(query: NormalizedRequest): Promise<Channel[]> {
    const { rangeKey } = query;
    let channels: Channel[];
    if (isKeysOnly(query)) channels = await this.fetchKeys(query.keys);
    else {
      channels = (await this.retriever.retrieve(query)).map((p) =>
        this.sugar(stripComposed(p)),
      );
      this.store.set(channels);
    }
    if (rangeKey != null)
      await this.ensureAliases(
        rangeKey,
        channels.map((ch) => ch.key),
      );
    return channels;
  }
}

export const isCalculated = ({ virtual, expression }: Payload): boolean =>
  virtual && expression !== "";

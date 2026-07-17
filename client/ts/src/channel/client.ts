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
  type destructor,
  errors,
  type MultiSeries,
  primitive,
  TimeSpan,
  type TypedArray,
} from "@synnaxlabs/x";
import { z } from "zod";

import { cache } from "@/cache";
import { type Params, statusKey } from "@/channel/payload";
import {
  analyzeParams,
  CacheRetriever,
  ClusterRetriever,
  DebouncedBatchRetriever,
  type RetrieveOptions,
  type Retriever,
  type RetrieveRequest,
  retrieveRequestZ,
} from "@/channel/retriever";
import { bindStore, STORE_KEY } from "@/channel/store";
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
import { type ranger } from "@/ranger";
import { createKey, decodeDeleteChange } from "@/ranger/alias/payload";
import { STORE_KEY as ALIASES_STORE_KEY } from "@/ranger/alias/store";
import { status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

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
  private readonly _frameClient: framer.Client | null;
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
    this._frameClient = frameClient ?? null;
  }

  private get framer(): framer.Client {
    if (this._frameClient == null)
      throw new ValidationError("cannot read from a channel that has not been created");
    return this._frameClient;
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

const MOUNT_SCOPE = "channel.mounts";

/** Aliases and statuses are composed from their stores on read. */
const stripComposed = ({ alias: _alias, status: _status, ...rest }: Payload): Payload =>
  rest;

const normalizeSingle = ({ key, rangeKey }: RetrieveSingleParams) =>
  rangeKey == null ? { key } : { key, rangeKey };

const onlyRangeKey = (options?: RetrieveOptions): boolean =>
  options == null ||
  Object.entries(options).every(([k, v]) => v === undefined || k === "rangeKey");

const isKeysOnly = (req: NormalizedRequest): boolean =>
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

/**
 * Client-side approximation of the server's matching for a request: exact for
 * key, name, and payload-field filters, permissive for server-computed shapes
 * (search), which accept every change and drift toward the server's answer.
 */
const requestFilter = (req: NormalizedRequest): ((ch: Channel) => boolean) => {
  const keySet = primitive.isNonZero(req.keys) ? new Set(req.keys) : undefined;
  const nameSet = primitive.isNonZero(req.names) ? new Set(req.names) : undefined;
  return (ch) => {
    if (keySet != null && !keySet.has(ch.key)) return false;
    if (nameSet != null && !nameSet.has(ch.name)) return false;
    if (req.virtual != null && ch.virtual !== req.virtual) return false;
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

/**
 * The main client class for executing channel operations against a Synnax Core. This
 * class should not be instantiated directly, and instead should be used through the
 * `channels` property of an {@link Synnax} client.
 */
export class Client {
  private readonly frameClient: framer.Client;
  private readonly client: UnaryClient;
  readonly retriever: Retriever;
  readonly writer: Writer;
  private readonly statuses_?: status.Client;
  private readonly ranges_?: ranger.Client;
  private readonly engine_?: cache.Engine;
  private readonly queries_?: {
    single: cache.Queries<RetrieveSingleParams, Channel>;
    request: cache.Queries<NormalizedRequest, Channel[]>;
  };

  constructor(
    frameClient: framer.Client,
    retriever: Retriever,
    client: UnaryClient,
    writer: Writer,
    statuses?: status.Client,
    ranges?: ranger.Client,
    engine?: cache.Engine,
  ) {
    this.frameClient = frameClient;
    this.retriever = retriever;
    this.client = client;
    this.writer = writer;
    this.statuses_ = statuses;
    this.ranges_ = ranges;
    if (engine == null) return;
    bindStore(
      engine,
      (payload) => this.sugar(stripComposed(payload)),
      async (keys) =>
        (await this.retriever.retrieve(keys)).map((p) => this.sugar(stripComposed(p))),
    );
    this.engine_ = engine;
    const ensureStreaming = async () => await engine.ensureStreaming();
    this.queries_ = {
      single: new cache.Queries({
        name: "channel",
        fetch: async (query) => await this.fetchSingle(query),
        mount: (params) => this.mountSingle(params),
        ensureStreaming,
      }),
      request: new cache.Queries({
        name: "channels",
        fetch: async (query) => await this.fetchRequest(query),
        mount: (params) => this.mountRequest(params),
        ensureStreaming,
      }),
    };
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
    this.writes?.set(created.map((ch) => this.sugar(stripComposed(ch.payload))));
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
  async retrieve(params: Params, options?: RetrieveOptions): Promise<Channel[]>;

  async retrieve(params: RetrieveRequest): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given parameters.
   *
   * this will be ignored.
   * @returns The retrieved channel.
   * @raises {QueryError} If the channel does not exist or if multiple results are returned.
   */
  async retrieve(
    params: Params | RetrieveRequest,
    options?: RetrieveOptions,
  ): Promise<Channel | Channel[]> {
    if (typeof params === "object" && !Array.isArray(params)) {
      if (this.queries_ == null)
        return this.sugar(await this.retriever.retrieve(params as RetrieveRequest));
      return await this.queries_.request.retrieve(retrieveRequestZ.parse(params));
    }
    const isSingle = !Array.isArray(params);
    const { variant, normalized: rawNormalized } = analyzeParams(params);
    const normalized =
      variant === "keys"
        ? (rawNormalized as Key[]).filter((k) => k !== 0)
        : rawNormalized;
    if (normalized.length === 0) {
      checkForMultipleOrNoResults<Params, Channel>("channel", params, [], isSingle);
      return [];
    }
    if (this.queries_ == null) {
      const res = this.sugar(await this.retriever.retrieve(params, options));
      checkForMultipleOrNoResults<Params, Channel>("channel", params, res, isSingle);
      return isSingle ? res[0] : res;
    }
    if (isSingle && variant === "keys" && onlyRangeKey(options))
      return await this.queries_.single.retrieve(
        normalizeSingle({ key: normalized[0] as Key, rangeKey: options?.rangeKey }),
      );
    const res = await this.queries_.request.retrieve(
      retrieveRequestZ.parse({ [variant]: normalized, ...options }),
    );
    checkForMultipleOrNoResults<Params, Channel>("channel", params, res, isSingle);
    return isSingle ? res[0] : res;
  }

  /**
   * Subscribes to changes in the cached answer to the given query. Single
   * queries deliver a channel; request queries deliver the matching channels.
   * @throws when the cache was disabled at client construction.
   */
  onChange(
    params: RetrieveSingleParams,
    handler: cache.ChangeHandler<Channel>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveRequest,
    handler: cache.ChangeHandler<Channel[]>,
  ): destructor.Destructor;
  onChange(
    params: RetrieveSingleParams | RetrieveRequest,
    handler: cache.ChangeHandler<Channel> | cache.ChangeHandler<Channel[]>,
  ): destructor.Destructor {
    const queries = this.requireQueries();
    if ("key" in params)
      return queries.single.onChange(
        normalizeSingle(params),
        handler as cache.ChangeHandler<Channel>,
      );
    return queries.request.onChange(
      retrieveRequestZ.parse(params),
      handler as cache.ChangeHandler<Channel[]>,
    );
  }

  /**
   * Returns the cached answer to the given query without touching the
   * network, or undefined when nothing is cached. Unfetched filter queries
   * are approximated from the record store when possible.
   * @throws when the cache was disabled at client construction.
   */
  getCached(params: RetrieveSingleParams): cache.Cached<Channel> | undefined;
  getCached(params: RetrieveRequest): cache.Cached<Channel[]> | undefined;
  getCached(
    params: RetrieveSingleParams | RetrieveRequest,
  ): cache.Cached<Channel> | cache.Cached<Channel[]> | undefined {
    const queries = this.requireQueries();
    if ("key" in params) return queries.single.getCached(normalizeSingle(params));
    const req = retrieveRequestZ.parse(params);
    return queries.request.getCached(req) ?? this.approximateCached(req);
  }

  /**
   * Approximates an unfetched filter query's answer from the record store.
   * Server-computed shapes (search, pagination) cannot be approximated.
   */
  private approximateCached(
    req: NormalizedRequest,
  ): cache.Cached<Channel[]> | undefined {
    if (req.searchTerm != null || req.limit != null || req.offset != null)
      return undefined;
    const matches = this.channelStore.get(requestFilter(req));
    if (matches.length === 0) return undefined;
    return { variant: "changed", data: matches };
  }

  /***
   * Deletes channels from the database using the given keys or names.
   * @param params - The keys or names of the channels to delete.
   */
  async delete(params: Params, opts: cache.WriteOptions = {}): Promise<void> {
    const { normalized, variant } = analyzeParams(params);
    if (variant === "keys") {
      const keys = normalized as Key[];
      const rollback = new cache.Rollback();
      const engine = this.engine_;
      const writes = this.writes;
      if (engine != null && writes != null) {
        const ids = ontologyID(keys);
        rollback.add(ontology.deleteCachedRelationships(engine, ids));
        rollback.add(writes.delete(keys));
        rollback.add(
          engine
            .store<string, ontology.Resource>(ontology.RESOURCES_STORE_KEY)
            .delete(ontology.idToString(ids)),
        );
      }
      await opts.onOptimistic?.();
      await rollback.guard(async () => await this.writer.delete({ keys }));
      // Re-applied after success: a stale streamer echo may have resurrected
      // an optimistically deleted entry while the send was in flight.
      writes?.delete(keys);
      return;
    }
    const names = normalized as string[];
    await this.writer.delete({ names });
    const writes = this.writes;
    if (writes == null) return;
    const cached = writes.get((ch) => names.includes(ch.name));
    if (cached.length > 0) writes.delete(cached.map((ch) => ch.key));
  }

  async rename(key: Key, name: string, opts?: cache.WriteOptions): Promise<void>;
  async rename(keys: Key[], names: string[], opts?: cache.WriteOptions): Promise<void>;
  async rename(
    keys: Key | Key[],
    names: string | string[],
    opts: cache.WriteOptions = {},
  ): Promise<void> {
    const keysArr = array.toArray(keys);
    const namesArr = array.toArray(names);
    const rollback = new cache.Rollback();
    const engine = this.engine_;
    if (engine != null)
      keysArr.forEach((key, i) => {
        const name = namesArr[i];
        rollback.add(this.renameThrough(key, name));
        rollback.add(ontology.renameCachedResource(engine, ontologyID(key), name));
      });
    await opts.onOptimistic?.();
    await rollback.guard(async () => await this.writer.rename(keysArr, namesArr));
    // Re-applied after success: a stale streamer echo may have clobbered the
    // optimistic write while the send was in flight.
    keysArr.forEach((key, i) => this.renameThrough(key, namesArr[i]));
  }

  /** Renames the cached channel, skipping when absent. Returns a rollback. */
  private renameThrough(key: Key, name: string): destructor.Destructor {
    const writes = this.writes;
    if (writes == null) return () => {};
    return writes.set(key, (p) =>
      p == null ? undefined : this.sugar({ ...p.payload, name }),
    );
  }

  createDebouncedBatchRetriever(
    deb: CrudeTimeSpan = TimeSpan.milliseconds(10),
  ): Retriever {
    return new CacheRetriever(
      new DebouncedBatchRetriever(new ClusterRetriever(this.client), deb),
    );
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

  private get writes(): cache.UnaryStore<Key, Channel> | undefined {
    return this.engine_?.store(STORE_KEY);
  }

  private get channelStore(): cache.UnaryStore<Key, Channel> {
    return this.requireEngine().store(STORE_KEY);
  }

  private get statusStore(): cache.UnaryStore<string, status.Status> {
    return this.requireEngine().store(status.STORE_KEY);
  }

  private get aliasStore(): cache.UnaryStore<string, ranger.alias.Alias> {
    return this.requireEngine().store(ALIASES_STORE_KEY);
  }

  // Query mounts subscribe in their own scope: stores suppress notifications
  // to listeners in the writer's scope, and the streamer writes in the default
  // scope, which would silence default-scope subscriptions entirely.
  private get channelEvents(): cache.UnaryStore<Key, Channel> {
    return this.requireEngine().store(STORE_KEY, MOUNT_SCOPE);
  }

  private get statusEvents(): cache.UnaryStore<string, status.Status> {
    return this.requireEngine().store(status.STORE_KEY, MOUNT_SCOPE);
  }

  private get aliasEvents(): cache.UnaryStore<string, ranger.alias.Alias> {
    return this.requireEngine().store(ALIASES_STORE_KEY, MOUNT_SCOPE);
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

  private requireStatuses(): status.Client {
    if (this.statuses_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.statuses_;
  }

  private requireRanges(): ranger.Client {
    if (this.ranges_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.ranges_;
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
    let ch = this.channelStore.get(key);
    if (ch == null) {
      const payloads = await this.retriever.retrieve([key]);
      checkForMultipleOrNoResults("channel", key, payloads, true);
      ch = this.sugar(stripComposed(payloads[0]));
      this.channelStore.set(key, ch);
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
    return this.compose(ch.payload, rangeKey);
  }

  private mountSingle({
    query,
    update,
    remove,
  }: cache.MountParams<RetrieveSingleParams, Channel>) {
    const { key, rangeKey } = query;
    const recompose = () =>
      update((prev) => (prev == null ? null : this.compose(prev.payload, rangeKey)));
    const destructors = [
      this.channelEvents.onSet((ch) => {
        if (ch.key === key) update(this.compose(ch.payload, rangeKey));
      }),
      this.channelEvents.onDelete((k) => {
        if (k === key) remove(this.channelStore.getTombstone(k)?.corpse);
      }),
      this.statusEvents.onSet((st) => {
        if (st.key === statusKey(key)) recompose();
      }),
      this.statusEvents.onDelete((k) => {
        if (k === statusKey(key)) recompose();
      }),
    ];
    if (rangeKey == null) return destructors;
    const aliasKey = createKey({ range: rangeKey, channel: key });
    destructors.push(
      this.aliasEvents.onSet((a) => {
        if (createKey(a) === aliasKey) recompose();
      }),
      this.aliasEvents.onDelete((k) => {
        if (k === aliasKey) recompose();
      }),
    );
    return destructors;
  }

  /**
   * Fetches the given keys, serving cached entries and fetching only the
   * misses. Preserves the caller's key order.
   */
  private async fetchKeys(keys: Key[]): Promise<Channel[]> {
    const results: Channel[] = [];
    const misses: Key[] = [];
    keys.forEach((key) => {
      const cached = this.channelStore.get(key);
      if (cached != null) results.push(cached);
      else misses.push(key);
    });
    if (misses.length > 0) {
      const fetched = (await this.retriever.retrieve(misses)).map((p) =>
        this.sugar(stripComposed(p)),
      );
      this.channelStore.set(fetched);
      results.push(...fetched);
    }
    return cache.orderByKeys(keys, results, (ch) => ch.key);
  }

  private async fetchRequest(query: NormalizedRequest): Promise<Channel[]> {
    const { rangeKey } = query;
    let channels: Channel[];
    if (isKeysOnly(query)) channels = await this.fetchKeys(query.keys as Key[]);
    else {
      channels = (await this.retriever.retrieve(query)).map((p) =>
        this.sugar(stripComposed(p)),
      );
      this.channelStore.set(channels);
    }
    if (rangeKey == null) return channels;
    await this.ensureAliases(
      rangeKey,
      channels.map((ch) => ch.key),
    );
    return channels.map((ch) => this.composeAlias(ch.payload, rangeKey));
  }

  private mountRequest({
    query,
    update,
  }: cache.MountParams<NormalizedRequest, Channel[]>) {
    const { rangeKey } = query;
    const matches = requestFilter(query);
    const withAlias = (ch: Channel): Channel =>
      rangeKey == null ? ch : this.composeAlias(ch.payload, rangeKey);
    const destructors = [
      this.channelEvents.onSet((ch) => {
        update((prev) => {
          if (prev == null) return prev;
          const existing = prev.some((c) => c.key === ch.key);
          if (!matches(ch))
            return existing ? prev.filter((c) => c.key !== ch.key) : prev;
          const next = withAlias(ch);
          if (existing) return prev.map((c) => (c.key === ch.key ? next : c));
          return [...prev, next];
        });
      }),
      this.channelEvents.onDelete((key) => {
        update((prev) => prev?.filter((c) => c.key !== key));
      }),
    ];
    if (rangeKey == null) return destructors;
    const recomposeMember = (channelKey: Key) =>
      update((prev) =>
        prev?.map((c) =>
          c.key === channelKey ? this.composeAlias(c.payload, rangeKey) : c,
        ),
      );
    destructors.push(
      this.aliasEvents.onSet((a) => {
        if (a.range === rangeKey) recomposeMember(a.channel);
      }),
      this.aliasEvents.onDelete((k) => {
        const decoded = decodeDeleteChange(k);
        if (decoded.range === rangeKey) recomposeMember(decoded.channel);
      }),
    );
    return destructors;
  }
}

export const isCalculated = ({ virtual, expression }: Payload): boolean =>
  virtual && expression !== "";

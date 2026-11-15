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
  type CrudeTimeRange,
  type CrudeTimeStamp,
  DataType,
  deep,
  type destructor,
  filter,
  type MultiSeries,
  type TypedArray,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type batch } from "@/batch";
import { type Action, dispatchReqZ, dispatchResZ, rename } from "@/channel/actions";
import {
  analyzeParams,
  isCalculated,
  type Params,
  type PrimitiveParams,
  singleWhere,
} from "@/channel/payload";
import {
  countLimitedResponseZ,
  type Key,
  keyZ,
  type Name,
  type New,
  type Operation,
  operationZ,
  type Payload,
  payloadZ,
  resourceID,
  type RetrieveRequest,
  retrieveRequestZ,
  statusZ,
} from "@/channel/types.gen";
import { LINES } from "@/channel/where";
import {
  compareCursors,
  cursor,
  matches,
  retrieveInputZ,
  serverComputed,
  shortcutZ,
} from "@/channel/where.gen";
import { type Writer } from "@/channel/writer";
import { ValidationError } from "@/errors";
import { type framer } from "@/framer";
import { group } from "@/group";
import { query } from "@/query";
import { type resource } from "@/resource";
import { type status } from "@/status";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_channel_set";
export const DELETE_CHANNEL_NAME = "sy_channel_delete";

interface CreateOptions {
  retrieveIfNameExists?: boolean;
}

/**
 * Represents a Channel in a Synnax database. Typically, channels should not be
 * instantiated directly, but instead created via the `.channels.create` or retrieved
 * via the `.channels.retrieve` method on a Synnax client. Please refer to the [Synnax
 * documentation](https://docs.synnaxlabs.com/reference/channels) for detailed
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
  /** The data type of the channel. */
  readonly dataType: DataType;
  /**
   * The key of the node in the Synnax cluster that holds the 'lease' over the channel
   * i.e. it's the only node in the cluster allowed to accept writes to the channel.
   * This property is mostly for internal use.
   */
  readonly leaseholder: number;
  /**
   * The key of the index channel that this channel is associated with i.e. the channel
   * that stores its timestamps.
   */
  readonly index: Key;
  /** This is set to true if the channel is an index channel, and false otherwise. */
  readonly isIndex: boolean;
  /** This is set to true if the channel is an internal channel, and false otherwise. */
  readonly internal: boolean;
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
  /** The status of the channel. */
  readonly status?: status.Status;
  /** The group the channel sits in, if any. */
  readonly group?: group.Key;

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
    status: argsStatus,
    expression = "",
    operations = [],
    concurrency = control.Concurrency.exclusive,
    group: groupKey,
  }: New & { frameClient?: framer.Client }) {
    this.key = zod.parse(keyZ, key, { label: "channel key" });
    this.name = name;
    this.dataType = new DataType(dataType);
    this.leaseholder = leaseholder;
    this.index = zod.parse(keyZ, index, { label: "channel index" });
    this.isIndex = isIndex;
    this.internal = internal;
    this.virtual = virtual;
    this.expression = expression;
    this.operations = zod.parse(operationZ.array(), operations, {
      label: "channel operations",
    });
    this.concurrency = concurrency;
    this.group = groupKey;
    if (argsStatus != null)
      this.status = zod.parse(statusZ, argsStatus, { label: "channel status" });
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
    return zod.parse(
      payloadZ,
      {
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
        concurrency: this.concurrency,
        group: this.group,
      },
      { label: "channel payload" },
    );
  }

  get isCalculated(): boolean {
    return isCalculated(this.payload);
  }

  /** @returns the resource ID of the channel */
  get resourceID(): resource.ID {
    return resourceID(this.key);
  }

  /**
   * Reads telemetry from the channel between the two timestamps.
   * @returns A typed array containing the retrieved
   */
  async read(tr: CrudeTimeRange): Promise<MultiSeries> {
    return await this.framer.read(tr, this.key);
  }

  /**
   * Writes telemetry to the channel starting at the given timestamp.
   * @param start - The starting timestamp of the first sample in data.
   */
  async write(start: CrudeTimeStamp, data: TypedArray): Promise<void> {
    return await this.framer.write(start, this.key, data);
  }
}

/** Selects the channels to retrieve. */
export type RetrieveMultipleParams = RetrieveRequest;

const retrieveParamsZ = filter.requestInputZ(
  retrieveRequestZ,
  retrieveInputZ(retrieveRequestZ.omit({ where: true }).shape),
);

const retrieveResZ = z.object({ channels: payloadZ.array().default(() => []) });

/** Names one channel. */
export type RetrieveSingleParams = { key: Key };

type OneParams = query.OneInput<typeof retrieveParamsZ>;

const isOneParams = (params: object): params is RetrieveSingleParams | OneParams =>
  query.addressesOne(params);

/** Statuses are composed from their store on read. */
const stripComposed = ({ status: _status, ...rest }: Payload): Payload => rest;

const singleParamsZ = z.union([
  z.strictObject({ key: z.uint32() }),
  z.uint32().transform((key) => ({ key })),
]);

/**
 * @returns the names of a request whose where selects one name or a list of names and
 * nothing else, with no other field of the request set. Returns null for any other
 * request.
 */
const onlyNames = ({ where, ...rest }: RetrieveRequest): string[] | null => {
  if (!Object.values(rest).every((v) => v == null) || where == null) return null;
  const { name, ...others } = where;
  if (name == null || Object.values(others).some((v) => v != null)) return null;
  const { eq, in: names, ...ops } = name;
  if (Object.values(ops).some((v) => v != null)) return null;
  if (eq == null) return names ?? null;
  return names == null ? [eq] : null;
};

/**
 * The main client class for executing channel operations against a Synnax Core. This
 * class should not be instantiated directly, and instead should be used through the
 * `channels` property of an {@link Synnax} client.
 */
export interface ClientConfig {
  framer: framer.Client;
  unary: UnaryClient;
  writer: Writer;
  statuses: status.Client;
  cache: query.Cache;
  batch: batch.Client;
}

/**
 * Creates, reads, and deletes channels on a Core. Reach it through `client.channels`.
 * Retrievals are served from a cache that a change stream keeps current, so a repeated
 * read costs nothing.
 */
export class Client extends query.Retriever<
  typeof retrieveParamsZ,
  Key,
  Channel,
  Channel,
  RetrieveSingleParams,
  RetrieveSingleParams
> {
  private readonly cfg: ClientConfig;
  readonly writer: Writer;
  readonly store: query.Table<Key, Channel>;
  /** Builds requests that move channels between groups. */
  readonly setGroupRequest: group.MoveRequest<Key>;
  private readonly byName: query.LookupIndex<Key, Channel>;

  constructor(cfg: ClientConfig) {
    const { writer, statuses, cache } = cfg;
    const statusStore = statuses.store;
    const affectedChannelKeys = (event: query.TableEvent<status.Key, status.Status>) =>
      statuses.ownerKeys(event, "channel")?.map(Number) ?? null;
    const sugar = (payload: Payload): Channel => this.sugar(payload);
    const byName = new query.LookupIndex<Key, Channel>((ch) => ch.name);
    const store = cache.createTable<Key, Channel>({
      name: "channels",
      equal: (a, b) => deep.equal(a.payload, b.payload),
      fetch: async (keys) =>
        (await this.execRetrieve(query.keysRequest(keys))).map((p) =>
          sugar(stripComposed(p)),
        ),
      listen: [
        query.createSetListener(SET_CHANNEL_NAME, payloadZ, {
          value: (changed) => sugar(stripComposed(changed)),
        }),
        query.createDeleteListener(DELETE_CHANNEL_NAME, keyZ),
      ],
      indexes: [byName],
    });
    const composed = cache.derive<Key, Channel, Channel>({
      name: "channel.composed",
      source: store,
      compose: (record) => this.compose(record.payload),
      equal: (a, b) => deep.equal(a.payload, b.payload),
      triggers: [
        query.deriveTrigger(statusStore, (event) => affectedChannelKeys(event)),
      ],
    });
    statuses.holdOwned(store, resourceID);
    const single = cache.queries<RetrieveSingleParams, Channel, Key, Channel>({
      name: "channel",
      table: composed,
      fetch: async (query) => [(await this.fetchSingle(query)).key],
      compose: (records) => records[0],
      keyOf: (query) => query.key,
      // Status-free fetches and the stream cache calculated channels with no status.
      covers: (ch) => !ch.isCalculated || ch.status != null,
      single: true,
    });
    super(cache, {
      name: "channel",
      table: store,
      request: {
        list: { cursor, compare: compareCursors },
        schema: retrieveParamsZ,
        fetch: async (query) => await this.fetchRequest(query),
        matches: (ch, query) => matches(LINES, ch, query.where),
        serverComputed: (query) => serverComputed(LINES, query.where),
        triggers: [
          query.trigger(
            statusStore,
            (event, query: RetrieveRequest) =>
              query.includeStatus === true ? affectedChannelKeys(event) : null,
            { contentOnly: true },
          ),
        ],
      },
      compose: (ch, query) =>
        (query as RetrieveRequest).includeStatus === true
          ? this.compose(ch.payload)
          : ch,
      single: { schema: singleParamsZ, space: single },
    });
    this.cfg = cfg;
    this.writer = writer;
    this.store = store;
    this.setGroupRequest = group.createMoveRequest("/channel/dispatch", (keys, to) => {
      const rollbacks = keys.map((key) => this.moveThrough(key, to));
      return () => rollbacks.forEach((rollback) => rollback());
    });
    this.byName = byName;
  }

  /**
   * Creates a single channel with the given properties.
   * @param name - A human-readable name for the channel.
   * @param rate - The rate of the channel. This only applies to fixed rate channels.
   * @param dataType - The data type for the samples stored in the channel.
   * @param index - The key of the index channel that this channel should be associated
   * with. An 'index' channel is a channel that stores timestamps for other channels.
   * Refer to the Synnax documentation (https://docs.synnaxlabs.com/reference/channels)
   * for more information. The index channel must have already been created. This field
   * does not need to be specified if the channel is an index channel, or the channel is
   * a fixed rate channel. If this value is specified, the 'rate' parameter will be
   * ignored.
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
   * @param channels - An array of channel properties to create. For each channel, the
   * following properties should be considered:
   * @param name - A human-readable name for the channel.
   * @param rate - The rate of the channel. This only applies to fixed rate channels. If
   * the 'index' parameter is specified or 'isIndex' is set to true, this parameter will
   * be ignored.
   * @param dataType - The data type for the samples stored in the channel.
   * @param index - The key of the index channel that this channel should be associated
   * with. An 'index' channel is a channel that stores timestamps for other channels.
   * Refer to the Synnax documentation (https://docs.synnaxlabs.com) for more
   * information. The index channel must have already been created. This field does not
   * need to be specified if the channel is an index channel, or the channel is a fixed
   * rate channel. If this value is specified, the 'rate' parameter will be ignored.
   * @param isIndex - Set to true if the channel is an index channel, and false
   * otherwise. Index channels must have a data type of `DataType.TIMESTAMP`.
   */
  async create(channels: New[], options?: CreateOptions): Promise<Channel[]>;

  async create(
    channels: New | New[],
    options: CreateOptions = {},
  ): Promise<Channel | Channel[]> {
    const { retrieveIfNameExists = false } = options;
    const single = !Array.isArray(channels);
    const created = this.sugar(
      await this.writer.create(array.toArray(channels), retrieveIfNameExists),
    );
    this.store.set(created.map((ch) => this.sugar(stripComposed(ch.payload))));
    return single ? created[0] : created;
  }

  /**
   * Retrieves a channel from the database using the given key, name, or query.
   * @param params - The key, the name, or a query that selects one channel.
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
  async retrieve(
    params: Key | Name | RetrieveSingleParams | OneParams,
  ): Promise<Channel>;

  /**
   * Retrieves multiple channels from the database using the provided keys, the
   * provided names, or a request that selects them.
   * @param params - The keys or the names of the channels to retrieve, or a request.
   * Note that this method does not support mixing keys and names in the same call. A
   * name matches exactly. To match a pattern, pass `{ where: { name: { regex } } }`.
   */
  async retrieve(
    params: PrimitiveParams | Payload[] | query.RequestInput<typeof retrieveParamsZ>,
  ): Promise<Channel[]>;

  async retrieve(
    params:
      | PrimitiveParams
      | Payload[]
      | RetrieveSingleParams
      | OneParams
      | query.RequestInput<typeof retrieveParamsZ>,
  ): Promise<Channel | Channel[]> {
    if (typeof params === "object" && !Array.isArray(params)) {
      // The branches narrow params onto different base overloads.
      if (isOneParams(params)) return await super.retrieve(params);
      return await super.retrieve(params);
    }
    const isSingle = !Array.isArray(params);
    const { variant, normalized: rawNormalized } = analyzeParams(params);
    const normalized =
      variant === "keys" ? rawNormalized.filter((k) => k !== 0) : rawNormalized;
    if (normalized.length === 0) {
      if (isSingle)
        checkForMultipleOrNoResults(
          "channel",
          { where: singleWhere(rawNormalized[0]) },
          [],
          true,
        );
      return [];
    }
    if (isSingle && variant === "keys")
      return await super.retrieve({ key: normalized[0] as Key });
    const where = isSingle
      ? singleWhere(normalized[0])
      : shortcutZ.parse(normalized).where;
    const res = await super.retrieve({ where });
    checkForMultipleOrNoResults("channel", { where }, res, isSingle);
    return isSingle ? res[0] : res;
  }

  /** * Deletes channels from the database using the given keys or names. */
  async delete(params: Params, opts: query.WriteOptions = {}): Promise<void> {
    const { normalized, variant } = analyzeParams(params);
    if (variant === "keys") {
      const keys = normalized;
      const drop = () => [this.store.delete(keys)];
      await query.optimistic({
        rollbacks: drop(),
        onOptimistic: opts.onOptimistic,
        commit: async () => await this.writer.delete({ keys }),
      });
      drop();
      return;
    }
    const names = normalized;
    await this.writer.delete({ names });
    const cached = this.byName.get(names);
    if (cached.length > 0) this.store.delete(cached.map((ch) => ch.key));
  }

  /**
   * Retrieves how many channels count against the license channel limit (not internal,
   * not virtual).
   * @throws {AccessDeniedError} if the caller cannot retrieve every channel.
   */
  async countLimited(): Promise<number> {
    const { count } = await this.cfg.unary.send(
      "/channel/count-limited",
      {},
      z.object({}),
      countLimitedResponseZ,
    );
    return count;
  }

  /** Renames the channels in one transaction, giving each key the name at its index. */
  async rename(key: Key, name: string, opts?: query.WriteOptions): Promise<void>;
  async rename(keys: Key[], names: string[], opts?: query.WriteOptions): Promise<void>;
  async rename(
    keys: Key | Key[],
    names: string | string[],
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const keysArr = array.toArray(keys);
    const namesArr = array.toArray(names);
    const apply = () => keysArr.map((key, i) => this.renameThrough(key, namesArr[i]));
    const requests = keysArr.map((key, i) => ({
      endpoint: "/channel/dispatch",
      payload: { key, actions: [rename({ name: namesArr[i] })] },
    }));
    await query.optimistic({
      rollbacks: apply(),
      onOptimistic: opts.onOptimistic,
      // A batch prefixes its errors with the request index, so one rename skips it.
      commit: async () =>
        requests.length === 1
          ? await this.send(keysArr[0], requests[0].payload.actions)
          : await this.cfg.batch.send(requests),
    });
    apply();
  }

  /**
   * Applies the actions to the channel in one transaction, then fetches the channel
   * into the cache before it resolves. The fetch catches the fields that the Core sets
   * itself, such as the data type of a calculated channel. A failed fetch goes to the
   * cache's error handler, since the Core applied the actions.
   * @throws {ValidationError} if the channel cannot hold an action.
   * @throws {NotFoundError} if the channel does not exist.
   */
  async dispatch(key: Key, actions: Action | Action[]): Promise<void> {
    await this.send(key, array.toArray(actions));
    try {
      await this.store.retrieve([key], { refresh: true });
    } catch (exc) {
      this.cfg.cache.onError(
        new Error("failed to fetch a dispatched channel", { cause: exc }),
      );
    }
  }

  private async send(key: Key, actions: Action[]): Promise<void> {
    await this.cfg.unary.send(
      "/channel/dispatch",
      { key, dispatchKey: "", actions },
      dispatchReqZ,
      dispatchResZ,
    );
  }

  /**
   * Sets the group of the cached channel, skipping when absent. Returns a rollback that
   * restores only the group.
   */
  private moveThrough(key: Key, to?: group.Key): destructor.Destructor {
    const prev = this.store.get(key)?.group;
    const set = (g?: group.Key) =>
      this.store.set(key, (p) =>
        p == null ? undefined : this.sugar({ ...p.payload, group: g }),
      );
    set(to);
    return () => set(prev);
  }

  /** Renames the cached channel, skipping when absent. Returns a rollback. */
  private renameThrough(key: Key, name: string): destructor.Destructor {
    return this.store.set(key, (p) =>
      p == null ? undefined : this.sugar({ ...p.payload, name }),
    );
  }

  sugar(payload: Payload): Channel;
  sugar(payloads: Payload[]): Channel[];
  sugar(payloads: Payload | Payload[]): Channel | Channel[] {
    const { framer: frameClient } = this.cfg;
    if (Array.isArray(payloads))
      return payloads.map((p) => new Channel({ ...p, frameClient }));
    return new Channel({ ...payloads, frameClient });
  }

  /** Rebuilds a cached channel with its cached status attached. */
  private compose(payload: Payload): Channel {
    const next: Payload = { ...payload, status: undefined };
    if (isCalculated(payload)) {
      const parsed = statusZ.safeParse(
        this.cfg.statuses.ownedBy(resourceID(payload.key)),
      );
      if (parsed.success) next.status = parsed.data;
    }
    return this.sugar(next);
  }

  /**
   * Fetches channel payloads. Missing channels are omitted, not thrown. Saves the
   * statuses the Core returns.
   */
  private async execRetrieve(request: RetrieveRequest): Promise<Payload[]> {
    const { store } = this.cfg.statuses;
    // A status that streams in during the fetch is newer than the fetched one.
    return await store.track(async (since) => {
      const res = await this.cfg.unary.send(
        "/channel/retrieve",
        request,
        retrieveRequestZ,
        retrieveResZ,
      );
      res.channels.forEach(({ status }) => {
        if (status != null) store.ingest(status, { since });
      });
      return res.channels;
    });
  }

  private async fetchSingle({ key }: RetrieveSingleParams): Promise<Channel> {
    let ch = this.store.get(key);
    if (ch == null) {
      // Through the table, so concurrent misses coalesce into one request. A plot
      // panel resolves one channel per telemetry source, all in the same tick.
      const fetched = await this.store.retrieve([key]);
      checkForMultipleOrNoResults(
        "channel",
        { where: { key: { eq: key } } },
        fetched,
        true,
      );
      [ch] = fetched;
    }
    // A cached calculated channel without a cached status is ambiguous: the
    // status may not exist, or may simply never have been fetched.
    if (ch.isCalculated && this.cfg.statuses.ownedBy(resourceID(key)) == null)
      await this.cfg.statuses.retrieveOwned([resourceID(key)]);
    return ch;
  }

  /**
   * Resolves names through the record store. Returns null when a name matches no
   * stored channel or more than one, so the Core answers the whole request.
   */
  private resolveNames(names: string[]): Channel[] | null {
    const resolved: Channel[] = [];
    for (const name of new Set(names)) {
      const matches = this.byName.get(name);
      if (matches.length !== 1) return null;
      resolved.push(matches[0]);
    }
    return resolved;
  }

  private async fetchRequest(req: RetrieveRequest): Promise<Channel[]> {
    const names = onlyNames(req);
    const resolved = names != null ? this.resolveNames(names) : null;
    if (resolved != null) return resolved;
    return (await this.execRetrieve(req)).map((p) => this.sugar(stripComposed(p)));
  }
}

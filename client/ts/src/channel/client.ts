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
  errors,
  type MultiSeries,
  primitive,
  type TypedArray,
  zod,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Condition } from "@/channel/builder.gen";
import {
  analyzeParams,
  isCalculated,
  type Params,
  type PrimitiveParams,
  statusKey,
} from "@/channel/payload";
import {
  type Key,
  keyZ,
  type Name,
  type New,
  ontologyID,
  type Operation,
  operationZ,
  type Payload,
  payloadZ,
  type RetrieveRequest,
  retrieveRequestZ,
  statusZ,
} from "@/channel/types.gen";
import { LINES } from "@/channel/where";
import { matches, serverComputed } from "@/channel/where.gen";
import { type Writer } from "@/channel/writer";
import { NotFoundError, ValidationError } from "@/errors";
import { type framer } from "@/framer";
import { group } from "@/group";
import { ontology } from "@/ontology";
import { query } from "@/query";
import { type ranger } from "@/ranger";
import { createKey, decodeDeleteChange } from "@/ranger/alias/payload";
import { keyZ as rangerKeyZ } from "@/ranger/types.gen";
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
   * i.e. it's the only node in the cluster allowed to accept writes to the channel. This
   * property is mostly for internal use.
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
  /** The status of the channel. */
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
  }: New & { frameClient?: framer.Client }) {
    this.key = zod.parse(keyZ, key, { label: "channel key" });
    this.name = name;
    this.dataType = new DataType(dataType);
    this.leaseholder = leaseholder;
    this.index = zod.parse(keyZ, index, { label: "channel index" });
    this.isIndex = isIndex;
    this.internal = internal;
    this.alias = alias;
    this.virtual = virtual;
    this.expression = expression;
    this.operations = zod.parse(operationZ.array(), operations, {
      label: "channel operations",
    });
    this.concurrency = concurrency;
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
      },
      { label: "channel payload" },
    );
  }

  get isCalculated(): boolean {
    return isCalculated(this.payload);
  }

  /** @returns the ontology ID of the channel */
  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
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

const retrieveMultipleParamsZ = retrieveRequestZ.extend({
  rangeKey: rangerKeyZ.optional(),
});
/**
 * Selects the channels to retrieve. A rangeKey attaches each channel's alias in that
 * range, and a search term then also matches those aliases, ranked first.
 */
export type RetrieveMultipleParams = RetrieveRequest & { rangeKey?: ranger.Key };

const aliasedRetrieveRequestZ = z.object({
  request: retrieveRequestZ,
  rangeKey: rangerKeyZ.optional(),
});

type ListParts = query.ListParts<Condition, typeof retrieveMultipleParamsZ>;
type OneParts = query.OneParts<Condition, typeof retrieveMultipleParamsZ>;

/** The options a retrieval by keys or names accepts. */
export type RetrieveOptions = Pick<RetrieveMultipleParams, "rangeKey">;

const retrieveResZ = z.object({ channels: payloadZ.array().default(() => []) });

const retrieveGroupReqZ = z.object({});

const retrieveGroupResZ = z.object({ group: group.groupZ });

/** Names one channel. Give a `rangeKey` to resolve its alias in that range. */
export type RetrieveSingleParams = { key: Key; rangeKey?: ranger.Key };

/** Aliases and statuses are composed from their stores on read. */
const stripComposed = ({ alias: _alias, status: _status, ...rest }: Payload): Payload =>
  rest;

const normalizeSingle = ({
  key,
  rangeKey,
}: RetrieveSingleParams): RetrieveSingleParams =>
  rangeKey == null ? { key } : { key, rangeKey };

const singleParamsZ = z
  .union([
    z.strictObject({ key: z.uint32(), rangeKey: rangerKeyZ.optional() }),
    z.uint32().transform((key) => ({ key })),
  ])
  .transform(normalizeSingle);

/**
 * @returns the name patterns of a request whose where selects by name patterns and
 * nothing else, with no other field of the request set. Returns null for any other
 * request.
 */
const onlyNamePatterns = ({ where, ...rest }: RetrieveRequest): string[] | null => {
  if (!Object.values(rest).every((v) => v == null) || where == null) return null;
  const { namePatterns, ...others } = where;
  if (namePatterns == null || Object.values(others).some((v) => v != null)) return null;
  return namePatterns;
};

const affectedChannelKeys = (
  event: query.TableEvent<status.Key, status.Status>,
): Key[] | null => {
  const [type, key] = event.key.split(":");
  if (type !== "channel" || !primitive.isNonZero(key)) return null;
  const parsed = keyZ.safeParse(key);
  return parsed.success ? [parsed.data] : null;
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
  ranges: ranger.Client;
  cache: query.Cache;
  ontology: ontology.Client;
}

/**
 * Creates, reads, and deletes channels on a Core. Reach it through `client.channels`.
 * Retrievals are served from a cache that a change stream keeps current, so a repeated
 * read costs nothing.
 */
export class Client extends query.Retriever<
  Condition,
  typeof retrieveMultipleParamsZ,
  Key,
  Channel,
  Channel,
  RetrieveSingleParams,
  RetrieveSingleParams
> {
  private readonly cfg: ClientConfig;
  readonly writer: Writer;
  readonly store: query.Table<Key, Channel>;
  private readonly byName: query.LookupIndex<Key, Channel>;

  constructor(cfg: ClientConfig) {
    const { writer, statuses, ranges, cache } = cfg;
    const statusStore = statuses.store;
    const aliasStore = ranges.aliases;
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
      watch: [query.deriveWatch(statusStore, (event) => affectedChannelKeys(event))],
    });
    const single = cache.queries<RetrieveSingleParams, Channel, Key, Channel>({
      name: "channel",
      table: composed,
      fetch: async (query) => [(await this.fetchSingle(query)).key],
      // Composed rows carry status; the read-time step only glues the alias.
      compose: (records, query) =>
        query.rangeKey == null
          ? records[0]
          : this.composeAlias(records[0].payload, query.rangeKey),
      keyOf: (query) => query.key,
      single: true,
      watch: [
        query.watch(aliasStore, (event, query: RetrieveSingleParams) => {
          if (query.rangeKey == null) return null;
          const aliasKey = createKey({ range: query.rangeKey, channel: query.key });
          return event.key === aliasKey ? [query.key] : null;
        }),
      ],
    });
    super(cache, {
      name: "channel",
      table: store,
      request: {
        schema: retrieveMultipleParamsZ,
        fetch: async (query) => await this.fetchRequest(query),
        matches: (ch, query) => matches(LINES, ch, query.where),
        serverComputed: (query) => serverComputed(LINES, query.where),
        watch: [
          query.watch(aliasStore, (event, query: RetrieveMultipleParams) => {
            if (query.rangeKey == null) return null;
            if (event.variant === "set")
              return event.value.range === query.rangeKey
                ? [event.value.channel]
                : null;
            const decoded = decodeDeleteChange(event.key);
            return decoded.range === query.rangeKey ? [decoded.channel] : null;
          }),
          query.watch(statusStore, (event, query: RetrieveMultipleParams) =>
            query.includeStatus === true ? affectedChannelKeys(event) : null,
          ),
        ],
      },
      compose: (ch, query) => {
        const { rangeKey, includeStatus } = query as RetrieveMultipleParams;
        const composed = includeStatus === true ? this.compose(ch.payload) : ch;
        return rangeKey == null
          ? composed
          : this.composeAlias(composed.payload, rangeKey);
      },
      single: { schema: singleParamsZ, space: single },
    });
    this.cfg = cfg;
    this.writer = writer;
    this.store = store;
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
    let toCreate = array.toArray(channels);
    let created: Channel[] = [];
    if (retrieveIfNameExists) {
      const res = await this.execRetrieve({
        where: { name: { in: toCreate.map((c) => c.name) } },
      });
      const existingNames = new Set(res.map((c) => c.name));
      toCreate = toCreate.filter((c) => !existingNames.has(c.name));
      created = this.sugar(res);
    }
    created = created.concat(this.sugar(await this.writer.create(toCreate)));
    this.store.set(created.map((ch) => this.sugar(stripComposed(ch.payload))));
    return single ? created[0] : created;
  }

  /**
   * Retrieves the channels that match every condition in parts, or one channel when
   * parts end with `one()`.
   */
  async retrieve(...parts: OneParts): Promise<Channel>;
  async retrieve(...parts: ListParts): Promise<Channel[]>;

  /**
   * Retrieves a channel from the database using the given key or name.
   * @param params - The key or the name of the channel to retrieve. A name that is not
   * a valid channel name is a regular expression, anchored when unanchored.
   * @param options - Optional parameters to control the retrieval process.
   * @param options.rangeKey - Attaches the channel's alias in the range.
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
   * @param params - The keys or the names of the channels to retrieve. Note that this
   * method does not support mixing keys and names in the same call. A name that is not
   * a valid channel name is a regular expression, anchored when unanchored.
   * @param options - Optional parameters to control the retrieval process.
   * @param options.rangeKey - Attaches each channel's alias in the range.
   */
  async retrieve(
    params: PrimitiveParams | Payload[],
    options?: RetrieveOptions,
  ): Promise<Channel[]>;

  async retrieve(params: RetrieveSingleParams): Promise<Channel>;

  /** Retrieves the channels that params selects. */
  async retrieve(params: RetrieveMultipleParams): Promise<Channel[]>;

  async retrieve(
    ...args:
      | ListParts
      | OneParts
      | [
          params:
            PrimitiveParams | Payload[] | RetrieveSingleParams | RetrieveMultipleParams,
          options?: RetrieveOptions,
        ]
  ): Promise<Channel | Channel[]> {
    if (this.isParts(args)) return await this.retrieveParts(args);
    const [params, options] = args;
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
    const rangeKey = options?.rangeKey;
    if (isSingle && variant === "keys")
      return await super.retrieve({ key: normalized[0] as Key, rangeKey });
    const where =
      variant === "keys"
        ? { key: { in: normalized as Key[] } }
        : { namePatterns: normalized as string[] };
    const res = await super.retrieve(
      rangeKey == null ? { where } : { where, rangeKey },
    );
    checkForMultipleOrNoResults<Params, Channel>("channel", params, res, isSingle);
    return isSingle ? res[0] : res;
  }

  /** * Deletes channels from the database using the given keys or names. */
  async delete(params: Params, opts: query.WriteOptions = {}): Promise<void> {
    const { normalized, variant } = analyzeParams(params);
    if (variant === "keys") {
      const keys = normalized;
      const ids = ontologyID(keys);
      const drop = () => [
        this.cfg.ontology.cache.deleteRelationships(ids),
        this.store.delete(keys),
        this.cfg.ontology.cache.resources.delete(ontology.idToString(ids)),
      ];
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

  async rename(key: Key, name: string, opts?: query.WriteOptions): Promise<void>;
  async rename(keys: Key[], names: string[], opts?: query.WriteOptions): Promise<void>;
  async rename(
    keys: Key | Key[],
    names: string | string[],
    opts: query.WriteOptions = {},
  ): Promise<void> {
    const keysArr = array.toArray(keys);
    const namesArr = array.toArray(names);
    const rename = () =>
      keysArr.flatMap((key, i) => {
        const name = namesArr[i];
        return [
          this.renameThrough(key, name),
          this.cfg.ontology.cache.renameResource(ontologyID(key), name),
        ];
      });
    await query.optimistic({
      rollbacks: rename(),
      onOptimistic: opts.onOptimistic,
      commit: async () => await this.writer.rename(keysArr, namesArr),
    });
    rename();
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

  async retrieveGroup(): Promise<group.Group> {
    const res = await this.cfg.unary.send(
      "/channel/retrieve-group",
      {},
      retrieveGroupReqZ,
      retrieveGroupResZ,
    );
    return res.group;
  }

  /** Rebuilds a cached channel with its cached status attached. */
  private compose(payload: Payload): Channel {
    const next: Payload = { ...payload, status: undefined, alias: undefined };
    if (isCalculated(payload)) {
      const parsed = statusZ.safeParse(
        this.cfg.statuses.store.get(statusKey(payload.key)),
      );
      if (parsed.success) next.status = parsed.data;
    }
    return this.sugar(next);
  }

  private composeAlias(payload: Payload, rangeKey: ranger.Key): Channel {
    return this.sugar({
      ...payload,
      alias: this.cfg.ranges.aliases.get(
        createKey({ range: rangeKey, channel: payload.key }),
      )?.alias,
    });
  }

  /** Fetches aliases under the range for keys the alias store has no entry for. */
  private async ensureAliases(rangeKey: ranger.Key, keys: Key[]): Promise<void> {
    const missing = keys.filter(
      (key) =>
        !this.cfg.ranges.aliases.has(createKey({ range: rangeKey, channel: key })),
    );
    if (missing.length === 0) return;
    const { aliases } = this.cfg.ranges;
    const since = aliases.stamp();
    const fetched = await this.cfg.ranges.retrieveAliases(rangeKey, missing);
    Object.entries(fetched).forEach(([channel, alias]) => {
      const entry: ranger.alias.Alias = {
        range: rangeKey,
        channel: Number(channel),
        alias,
      };
      aliases.ingest(createKey(entry), entry, { since });
    });
  }

  /**
   * Fetches channel payloads, with their aliases in the range when one is given.
   * Missing channels are omitted, not thrown. Saves the statuses the Core returns.
   */
  private async execRetrieve(
    request: RetrieveRequest,
    rangeKey?: ranger.Key,
  ): Promise<Payload[]> {
    // A status that streams in during the fetch is newer than the fetched one.
    const since = this.cfg.statuses.store.stamp();
    const res = await this.cfg.unary.send(
      "/channel/retrieve",
      { request, rangeKey },
      aliasedRetrieveRequestZ,
      retrieveResZ,
    );
    res.channels.forEach(({ status }) => {
      if (status != null) this.cfg.statuses.store.ingest(status, { since });
    });
    return res.channels;
  }

  private async fetchSingle(query: RetrieveSingleParams): Promise<Channel> {
    const { key, rangeKey } = query;
    let ch = this.store.get(key);
    if (ch == null) {
      // Through the table, so concurrent misses coalesce into one request. A plot
      // panel resolves one channel per telemetry source, all in the same tick.
      const fetched = await this.store.retrieve([key]);
      checkForMultipleOrNoResults("channel", key, fetched, true);
      [ch] = fetched;
    }
    // A cached calculated channel without a cached status is ambiguous: the
    // status may not exist, or may simply never have been fetched.
    if (ch.isCalculated && !this.cfg.statuses.store.has(statusKey(key)))
      try {
        await this.cfg.statuses.retrieve(statusKey(key));
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
      }
    if (rangeKey != null) await this.ensureAliases(rangeKey, [key]);
    return ch;
  }

  /**
   * Resolves names through the record store. Returns null when a name matches no
   * stored channel or more than one, so the Core answers the whole request. Assumes
   * the Core validates names, so a stored name holds no regular expression character.
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

  private async fetchRequest({
    rangeKey,
    ...req
  }: RetrieveMultipleParams): Promise<Channel[]> {
    if (rangeKey == null) return await this.fetchChannels(req);
    const { aliases } = this.cfg.ranges;
    const since = aliases.stamp();
    const payloads = await this.execRetrieve(req, rangeKey);
    payloads.forEach(({ key: channel, alias }) => {
      if (alias == null) return;
      const entry: ranger.alias.Alias = { range: rangeKey, channel, alias };
      aliases.ingest(createKey(entry), entry, { since });
    });
    return payloads.map((p) => this.sugar(stripComposed(p)));
  }

  private async fetchChannels(req: RetrieveRequest): Promise<Channel[]> {
    const names = onlyNamePatterns(req);
    const resolved = names != null ? this.resolveNames(names) : null;
    if (resolved != null) return resolved;
    return (await this.execRetrieve(req)).map((p) => this.sugar(stripComposed(p)));
  }
}

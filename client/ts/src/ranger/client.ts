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
  color,
  type CrudeTimeRange,
  deep,
  type Series,
  TimeRange,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type cache } from "@/cache";
import { type channel } from "@/channel";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { label } from "@/label";
import { type ontology } from "@/ontology";
import { alias } from "@/ranger/alias";
import { type Client as AliasClient } from "@/ranger/alias/client";
import { kv } from "@/ranger/kv";
import { type Client as KVClient } from "@/ranger/kv/client";
import { type Name, type Params } from "@/ranger/payload";
import {
  type Key,
  keyZ,
  type New,
  ontologyID,
  type Payload,
  payloadZ,
} from "@/ranger/types.gen";
import { type Writer } from "@/ranger/writer";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_range_set";
export const DELETE_CHANNEL_NAME = "sy_range_delete";

export const STORE_KEY = "ranges";
export const KV_STORE_KEY = "rangeKV";
export const ALIASES_STORE_KEY = "rangeAliases";

const kvDeleteZ = z
  .string()
  .transform((val) => val.split("<--->"))
  .transform(([range, key]) => ({ key, range }));

const aliasDeleteZ = z.string().transform((val) => alias.decodeDeleteChange(val));

/** Registers the range, range KV, and range alias stores on the given engine. */
const bindStores = (engine: cache.Engine, client: Client): void => {
  const ranges = () => engine.store<Key, Range>(STORE_KEY);
  const setListener: cache.ChannelListener<{}, typeof payloadZ> = {
    channel: SET_CHANNEL_NAME,
    schema: payloadZ,
    onChange: async ({ changed }) => {
      const range = client.sugarOne(changed);
      const prev = ranges().get(changed.key);
      let labels: label.Label[] | undefined;
      if (prev?.labels == null) labels = await range.retrieveLabels();
      let parent: Range | null = null;
      if (prev?.parent == null) parent = await range.retrieveParent();
      ranges().set(changed.key, (p) => {
        const pld: Payload = { ...range.payload };
        pld.labels = p?.labels ?? labels;
        pld.parent = p?.parent ?? parent?.payload;
        return client.sugarOne(pld);
      });
    },
  };
  const deleteListener: cache.ChannelListener<{}, typeof keyZ> = {
    channel: DELETE_CHANNEL_NAME,
    schema: keyZ,
    onChange: ({ changed }) => ranges().delete(changed),
  };
  engine.registerStore<Key, Range>(STORE_KEY, {
    equal: (a, b) => deep.equal(a.payload, b.payload),
    listeners: [setListener, deleteListener],
    refetch: async (keys) => await client.retrieve(keys),
  });

  const pairs = () => engine.store<string, kv.Pair>(KV_STORE_KEY);
  const kvSetListener: cache.ChannelListener<{}, typeof kv.pairZ> = {
    channel: kv.SET_CHANNEL_NAME,
    schema: kv.pairZ,
    onChange: ({ changed }) => pairs().set(kv.createPairKey(changed), changed),
  };
  const kvDeleteListener: cache.ChannelListener<{}, typeof kvDeleteZ> = {
    channel: kv.DELETE_CHANNEL_NAME,
    schema: kvDeleteZ,
    onChange: ({ changed }) => pairs().delete(kv.createPairKey(changed)),
  };
  engine.registerStore<string, kv.Pair>(KV_STORE_KEY, {
    listeners: [kvSetListener, kvDeleteListener],
  });

  const aliases = () => engine.store<string, alias.Alias>(ALIASES_STORE_KEY);
  const aliasSetListener: cache.ChannelListener<{}, typeof alias.aliasZ> = {
    channel: alias.SET_CHANNEL_NAME,
    schema: alias.aliasZ,
    onChange: ({ changed }) => aliases().set(alias.createKey(changed), changed),
  };
  const aliasDeleteListener: cache.ChannelListener<{}, typeof aliasDeleteZ> = {
    channel: alias.DELETE_CHANNEL_NAME,
    schema: aliasDeleteZ,
    onChange: ({ changed }) => aliases().delete(alias.createKey(changed)),
  };
  engine.registerStore<string, alias.Alias>(ALIASES_STORE_KEY, {
    listeners: [aliasSetListener, aliasDeleteListener],
  });
};

interface RangeConstructionOptions {
  frameClient: framer.Client;
  kv: KVClient;
  aliaser: AliasClient;
  channels: channel.Retriever;
  labelClient: label.Client;
  ontologyClient: ontology.Client;
  rangeClient: Client;
}

export class Range {
  key: string;
  name: string;
  readonly kv: KVClient;
  readonly timeRange: TimeRange;
  readonly color?: color.Color;
  readonly parent?: Payload;
  readonly labels?: label.Label[];
  readonly channels: channel.Retriever;
  private readonly aliaser: AliasClient;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: Client;

  constructor(
    { name, timeRange = TimeRange.ZERO, key, color: color_, parent, labels }: Payload,
    {
      frameClient,
      kv,
      aliaser,
      channels,
      labelClient,
      ontologyClient,
      rangeClient,
    }: RangeConstructionOptions,
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.parent = parent;
    this.labels = labels;
    this.frameClient = frameClient;
    this.color = color_;
    this.kv = kv;
    this.aliaser = aliaser;
    this.channels = channels;
    this.labelClient = labelClient;
    this.ontologyClient = ontologyClient;
    this.rangeClient = rangeClient;
  }

  get ontologyID(): ontology.ID {
    return ontologyID(this.key);
  }

  get payload(): Payload {
    const r: Payload = {
      key: this.key,
      name: this.name,
      timeRange: this.timeRange,
      color: this.color,
      labels: this.labels,
    };
    if (this.parent != null)
      if ("payload" in this.parent) r.parent = (this.parent as Range).payload;
      else r.parent = this.parent;
    return r;
  }

  async setAlias(channel: channel.Key | Name, alias: string): Promise<void> {
    const ch = await this.channels.retrieve(channel);
    if (ch.length === 0) throw new QueryError(`Channel ${channel} does not exist`);
    await this.aliaser.set({ [ch[0].key]: alias });
  }

  async deleteAlias(...channels: channel.Key[]): Promise<void> {
    await this.aliaser.delete(channels);
  }

  async listAliases(): Promise<Record<channel.Key, string>> {
    return await this.aliaser.list();
  }

  async resolveAlias(alias: string): Promise<channel.Key> {
    return await this.aliaser.resolve(alias);
  }

  async retrieveParent(): Promise<Range | null> {
    return await this.rangeClient.retrieveParent(this.key);
  }

  async retrieveChildren(): Promise<Range[]> {
    const res = (
      await this.ontologyClient.retrieveChildren(this.ontologyID, {
        excludeFieldData: true,
        types: ["range"],
      })
    ).map((r) => r.id.key);
    return await this.rangeClient.retrieve(res);
  }

  async read(channel: Key | Name): Promise<Series>;
  async read(channels: Params): Promise<framer.Frame>;
  async read(channels: Params): Promise<Series | framer.Frame> {
    return await this.frameClient.read(this.timeRange, channels);
  }

  async retrieveLabels(): Promise<label.Label[]> {
    return await this.labelClient.retrieve({ for: ontologyID(this.key) });
  }

  async addLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.label(ontologyID(this.key), labels);
  }

  async removeLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.remove(ontologyID(this.key), labels);
  }

  static sort(a: Range, b: Range): number {
    return TimeRange.sort(a.timeRange, b.timeRange);
  }
}

const retrieveRequestZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
  searchTerm: z.string().optional(),
  overlapsWith: TimeRange.z.optional(),
  hasLabels: label.keyZ.array().optional(),
  limit: z.int().optional(),
  offset: z.int().optional(),
  includeLabels: z.boolean().optional(),
  includeParent: z.boolean().optional(),
});

export type RetrieveRequest = z.infer<typeof retrieveRequestZ>;

const retrieveParamsZ = retrieveRequestZ
  .or(keyZ.array().transform((keys) => ({ keys })))
  .or(keyZ.transform((key) => ({ keys: [key] })))
  .or(z.string().transform((name) => ({ names: [name] })))
  .or(
    z
      .string()
      .array()
      .transform((names) => ({ names })),
  )
  .or(TimeRange.z.transform((timeRange) => ({ overlapsWith: timeRange })));

export type RetrieveParams = z.input<typeof retrieveParamsZ>;

const retrieveResZ = z.object({ ranges: payloadZ.array().default(() => []) });

export class Client {
  readonly type: string = "range";
  private readonly frameClient: framer.Client;
  private readonly writer: Writer;
  private readonly unaryClient: UnaryClient;
  private readonly channels: channel.Retriever;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly createAliasClient: (key: Key) => AliasClient;
  private readonly createKVClient: (key: Key) => KVClient;
  private readonly store_?: cache.Store<Key, Range>;
  private readonly kvPairs_?: cache.Store<string, kv.Pair>;
  private readonly aliases_?: cache.Store<string, alias.Alias>;

  constructor(
    frameClient: framer.Client,
    writer: Writer,
    unary: UnaryClient,
    channels: channel.Retriever,
    labelClient: label.Client,
    ontologyClient: ontology.Client,
    createAliasClient: (key: Key) => AliasClient,
    createKVClient: (key: Key) => KVClient,
    engine?: cache.Engine,
  ) {
    this.frameClient = frameClient;
    this.writer = writer;
    this.unaryClient = unary;
    this.channels = channels;
    this.labelClient = labelClient;
    this.ontologyClient = ontologyClient;
    this.createAliasClient = createAliasClient;
    this.createKVClient = createKVClient;
    if (engine == null) return;
    bindStores(engine, this);
    this.store_ = engine.store(STORE_KEY);
    this.kvPairs_ = engine.store(KV_STORE_KEY);
    this.aliases_ = engine.store(ALIASES_STORE_KEY);
  }

  /**
   * Read surface of the range cache.
   * @throws when the cache was disabled at client construction.
   */
  get store(): cache.Store<Key, Range> {
    if (this.store_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.store_;
  }

  /**
   * Read surface of the range KV pair cache.
   * @throws when the cache was disabled at client construction.
   */
  get kvPairs(): cache.Store<string, kv.Pair> {
    if (this.kvPairs_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.kvPairs_;
  }

  /**
   * Read surface of the range alias cache.
   * @throws when the cache was disabled at client construction.
   */
  get aliases(): cache.Store<string, alias.Alias> {
    if (this.aliases_ == null)
      throw new Error("cache is disabled on this client (cache: false)");
    return this.aliases_;
  }

  async create(range: New): Promise<Range>;
  async create(ranges: New[]): Promise<Range[]>;
  async create(ranges: New | New[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const res = this.sugarMany(await this.writer.create(array.toArray(ranges)));
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name): Promise<void> {
    await this.writer.rename(key, name);
  }

  async delete(key: Key | Key[]): Promise<void> {
    await this.writer.delete(array.toArray(key));
  }

  async retrieve(params: Key | Name): Promise<Range>;
  async retrieve(params: Key[] | Name[]): Promise<Range[]>;
  async retrieve(params: CrudeTimeRange): Promise<Range[]>;
  async retrieve(params: RetrieveRequest): Promise<Range[]>;
  async retrieve(params: RetrieveParams): Promise<Range | Range[]> {
    const isSingle = typeof params === "string";
    const { ranges } = await this.unaryClient.send(
      "/range/retrieve",
      params,
      retrieveParamsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Range", params, ranges, isSingle);
    if (isSingle) return this.sugarMany(ranges)[0];
    return this.sugarMany(ranges);
  }

  getKV(range: Key): KVClient {
    return this.createKVClient(range);
  }

  async retrieveParent(range: Key): Promise<Range | null> {
    const res = await this.ontologyClient.retrieveParents(ontologyID(range));
    if (res.length === 0) return null;
    const first = res[0];
    if (first.id.type !== "range") return null;
    return await this.retrieve(first.id.key);
  }

  sugarOntologyResource(resource: ontology.Resource): Range {
    return this.sugarOne(convertOntologyResourceToPayload(resource));
  }

  async retrieveAlias(range: Key, channel: channel.Key): Promise<string> {
    const aliaser = this.createAliasClient(range);
    return await aliaser.retrieve(channel);
  }

  async retrieveAliases(
    range: Key,
    channels: channel.Key[],
  ): Promise<Record<channel.Key, string>> {
    return await this.createAliasClient(range).retrieve(channels);
  }

  async listAliases(range: Key): Promise<Record<channel.Key, string>> {
    return await this.createAliasClient(range).list();
  }

  async setAlias(range: Key, channel: channel.Key, alias: string): Promise<void> {
    await this.createAliasClient(range).set({ [channel]: alias });
  }

  async deleteAlias(range: Key, channels: channel.Key | channel.Key[]): Promise<void> {
    await this.createAliasClient(range).delete(channels);
  }

  sugarOne(payload: Payload): Range {
    return new Range(payload, {
      frameClient: this.frameClient,
      kv: this.createKVClient(payload.key),
      aliaser: this.createAliasClient(payload.key),
      channels: this.channels,
      labelClient: this.labelClient,
      ontologyClient: this.ontologyClient,
      rangeClient: this,
    });
  }

  sugarMany(payloads: Payload[]): Range[] {
    return payloads.map((payload) => this.sugarOne(payload));
  }

  resourceToRange(resource: ontology.Resource): Range {
    return this.sugarOne(convertOntologyResourceToPayload(resource));
  }
}

export const aliasOntologyID = (key: Key): ontology.ID => ({
  type: "range-alias",
  key,
});

export const convertOntologyResourceToPayload = ({
  data,
  id: { key },
  name,
}: ontology.Resource): Payload => {
  const timeRange = TimeRange.z.parse(data?.timeRange);
  const c = color.colorZ.safeParse(data?.color);
  return {
    key,
    name,
    timeRange,
    color: c.success ? c.data : undefined,
    labels: [],
    parent: undefined,
  };
};

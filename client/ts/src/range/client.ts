// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array, type CrudeTimeRange, type Series, TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

import { Aliaser } from "@/alias/client";
import { type channel } from "@/channel";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { KV } from "@/kv/client";
import { label } from "@/label";
import { ontology } from "@/ontology";
import {
  type Key,
  type Keys,
  keyZ,
  type Name,
  type Names,
  type New,
  type Params,
  type Payload,
  payloadZ,
} from "@/range/payload";
import { type CreateOptions, type Writer } from "@/range/writer";
import { checkForMultipleOrNoResults } from "@/util/retrieve";

export const SET_CHANNEL_NAME = "sy_range_set";
export const DELETE_CHANNEL_NAME = "sy_range_delete";

interface RangeConstructionOptions {
  frameClient: framer.Client;
  kv: KV;
  aliaser: Aliaser;
  channels: channel.Retriever;
  labelClient: label.Client;
  ontologyClient: ontology.Client;
  rangeClient: Client;
}

export class Range {
  key: string;
  name: string;
  readonly kv: KV;
  readonly timeRange: TimeRange;
  readonly color: string | undefined;
  readonly parent: Payload | null;
  readonly labels?: label.Label[];
  readonly channels: channel.Retriever;
  private readonly aliaser: Aliaser;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: Client;

  constructor(
    { name, timeRange = TimeRange.ZERO, key, color, parent, labels }: Payload,
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
    this.color = color;
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
    let parent: Payload | null = null;
    if (this.parent != null)
      if ("payload" in this.parent) parent = (this.parent as Range).payload;
      else parent = this.parent;
    return {
      key: this.key,
      name: this.name,
      timeRange: this.timeRange,
      color: this.color,
      labels: this.labels,
      parent,
    };
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
    return this.rangeClient.retrieveParent(this.key);
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

const retrieveArgsZ = retrieveRequestZ
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

export type RetrieveArgs = z.input<typeof retrieveArgsZ>;

const retrieveResZ = z.object({ ranges: array.nullishToEmpty(payloadZ) });

export class Client {
  readonly type: string = "range";
  private readonly frameClient: framer.Client;
  private readonly writer: Writer;
  private readonly unaryClient: UnaryClient;
  private readonly channels: channel.Retriever;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;

  constructor(
    frameClient: framer.Client,
    writer: Writer,
    unary: UnaryClient,
    channels: channel.Retriever,
    labelClient: label.Client,
    ontologyClient: ontology.Client,
  ) {
    this.frameClient = frameClient;
    this.writer = writer;
    this.unaryClient = unary;
    this.channels = channels;
    this.labelClient = labelClient;
    this.ontologyClient = ontologyClient;
  }

  async create(range: New, options?: CreateOptions): Promise<Range>;
  async create(ranges: New[], options?: CreateOptions): Promise<Range[]>;
  async create(ranges: New | New[], options?: CreateOptions): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const res = this.sugarMany(
      await this.writer.create(array.toArray(ranges), options),
    );
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name): Promise<void> {
    await this.writer.rename(key, name);
  }

  async delete(key: Key | Keys): Promise<void> {
    await this.writer.delete(array.toArray(key));
  }

  async retrieve(params: Key | Name): Promise<Range>;
  async retrieve(params: Keys | Names): Promise<Range[]>;
  async retrieve(params: CrudeTimeRange): Promise<Range[]>;
  async retrieve(params: RetrieveRequest): Promise<Range[]>;
  async retrieve(params: RetrieveArgs): Promise<Range | Range[]> {
    const isSingle = typeof params === "string";
    const { ranges } = await sendRequired(
      this.unaryClient,
      "/range/retrieve",
      params,
      retrieveArgsZ,
      retrieveResZ,
    );
    checkForMultipleOrNoResults("Range", params, ranges, isSingle);
    if (isSingle) return this.sugarMany(ranges)[0];
    return this.sugarMany(ranges);
  }

  getKV(range: Key): KV {
    return new KV(range, this.unaryClient);
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
    const aliaser = new Aliaser(range, this.unaryClient);
    return await aliaser.retrieve(channel);
  }

  async retrieveAliases(
    range: Key,
    channels: channel.Key[],
  ): Promise<Record<channel.Key, string>> {
    const aliaser = new Aliaser(range, this.unaryClient);
    return await aliaser.retrieve(channels);
  }

  async listAliases(range: Key): Promise<Record<channel.Key, string>> {
    const aliaser = new Aliaser(range, this.unaryClient);
    return await aliaser.list();
  }

  async setAlias(range: Key, channel: channel.Key, alias: string): Promise<void> {
    const aliaser = new Aliaser(range, this.unaryClient);
    await aliaser.set({ [channel]: alias });
  }

  async deleteAlias(range: Key, channels: channel.Key | channel.Key[]): Promise<void> {
    const aliaser = new Aliaser(range, this.unaryClient);
    await aliaser.delete(channels);
  }

  sugarOne(payload: Payload): Range {
    return new Range(payload, {
      frameClient: this.frameClient,
      kv: new KV(payload.key, this.unaryClient),
      aliaser: new Aliaser(payload.key, this.unaryClient),
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
    return this.sugarOne({
      key: resource.id.key,
      name: resource.data?.name as string,
      timeRange: new TimeRange(resource.data?.timeRange as CrudeTimeRange),
      color: resource.data?.color as string,
      labels: [],
      parent: null,
    });
  }
}

export const ontologyID = ontology.createIDFactory<Key>("range");
export const TYPE_ONTOLOGY_ID = ontologyID("");

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
  return {
    key,
    name,
    timeRange,
    color: typeof data?.color === "string" ? data.color : undefined,
    labels: [],
    parent: null,
  };
};

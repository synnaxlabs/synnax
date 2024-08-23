// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { CrudeTimeRange, observe, TimeRange } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { type Series } from "@synnaxlabs/x/telem";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { Key as ChannelKey } from "@/channel/payload";
import { type Retriever as ChannelRetriever } from "@/channel/retriever";
import { MultipleFoundError, NotFoundError } from "@/errors";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { type label } from "@/label";
import { type Label } from "@/label/payload";
import { ontology } from "@/ontology";
import { Resource } from "@/ontology/payload";
import { type Alias, Aliaser } from "@/ranger/alias";
import { KV } from "@/ranger/kv";
import {
  analyzeParams,
  type Key,
  type Keys,
  keyZ,
  type Name,
  type Names,
  type NewPayload,
  type Params,
  type Payload,
  payloadZ,
} from "@/ranger/payload";
import { CreateOptions, type Writer } from "@/ranger/writer";
import { signals } from "@/signals";
import { nullableArrayZ } from "@/util/zod";

const ontologyID = (key: string): ontology.ID =>
  new ontology.ID({ type: "range", key });

export class Range {
  key: string;
  name: string;
  readonly kv: KV;
  readonly timeRange: TimeRange;
  readonly color: string | undefined;
  readonly channels: ChannelRetriever;
  private readonly aliaser: Aliaser;
  private readonly frameClient: framer.Client;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;
  private readonly rangeClient: Client;

  constructor(
    name: string,
    timeRange: TimeRange = TimeRange.ZERO,
    key: string,
    color: string | undefined,
    _frameClient: framer.Client,
    _kv: KV,
    _aliaser: Aliaser,
    _channels: ChannelRetriever,
    _labelClient: label.Client,
    _ontologyClient: ontology.Client,
    _rangeClient: Client,
  ) {
    this.key = key;
    this.name = name;
    this.timeRange = timeRange;
    this.frameClient = _frameClient;
    this.color = color;
    this.kv = _kv;
    this.aliaser = _aliaser;
    this.channels = _channels;
    this.labelClient = _labelClient;
    this.ontologyClient = _ontologyClient;
    this.rangeClient = _rangeClient;
  }

  get ontologyID(): ontology.ID {
    return new ontology.ID({ key: this.key, type: "range" });
  }

  get payload(): Payload {
    return {
      key: this.key,
      name: this.name,
      timeRange: this.timeRange,
      color: this.color,
    };
  }

  async setAlias(channel: ChannelKey | Name, alias: string): Promise<void> {
    const ch = await this.channels.retrieve(channel);
    if (ch.length === 0) {
      throw new QueryError(`Channel ${channel} does not exist`);
    }
    await this.aliaser.set({ [ch[0].key]: alias });
  }

  async deleteAlias(...channels: ChannelKey[]): Promise<void> {
    await this.aliaser.delete(channels);
  }

  async listAliases(): Promise<Record<ChannelKey, string>> {
    return await this.aliaser.list();
  }

  async resolveAlias(alias: string): Promise<ChannelKey> {
    return await this.aliaser.resolve(alias);
  }

  async openAliasTracker(): Promise<signals.Observable<string, Alias>> {
    return await this.aliaser.openChangeTracker();
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

  async labels(): Promise<Label[]> {
    return await this.labelClient.retrieveFor(ontologyID(this.key));
  }

  async addLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.label(ontologyID(this.key), labels);
  }

  async removeLabel(...labels: label.Key[]): Promise<void> {
    await this.labelClient.removeLabels(ontologyID(this.key), labels);
  }

  async openChildRangeTracker(): Promise<observe.ObservableAsyncCloseable<Range[]>> {
    const wrapper = new observe.Observer<Range[]>();
    const initial: ontology.Resource[] = (await this.retrieveChildren()).map((r) => {
      const id = new ontology.ID({ key: r.key, type: "range" });
      return { id, key: id.toString(), name: r.name, data: r.payload };
    });
    const base = await this.ontologyClient.openDependentTracker(
      this.ontologyID,
      initial,
    );
    base.onChange((resources: Resource[]) =>
      wrapper.notify(this.rangeClient.resourcesToRanges(resources)),
    );
    wrapper.setCloser(async () => await base.close());
    return wrapper;
  }

  async openParentRangeTracker(): Promise<observe.ObservableAsyncCloseable<Range> | null> {
    const wrapper = new observe.Observer<Range>();
    const p = await this.retrieveParent();
    if (p == null) return null;
    const id = new ontology.ID({ key: p.key, type: "range" });
    const resourceP = { id, key: id.toString(), name: p.name, data: p.payload };
    const base = await this.ontologyClient.openDependentTracker(
      this.ontologyID,
      [resourceP],
      "parent",
      "to",
    );
    base.onChange((resources: Resource[]) => {
      const ranges = this.rangeClient.resourcesToRanges(resources);
      if (ranges.length === 0) return;
      const p = ranges[0];
      wrapper.notify(p);
    });
    wrapper.setCloser(async () => await base.close());
    return wrapper;
  }
}

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.array(z.string()).optional(),
  term: z.string().optional(),
  overlapsWith: TimeRange.z.optional(),
  limit: z.number().int().optional(),
  offset: z.number().int().optional(),
});

export type RetrieveRequest = z.infer<typeof retrieveReqZ>;

const RETRIEVE_ENDPOINT = "/range/retrieve";

const retrieveResZ = z.object({
  ranges: nullableArrayZ(payloadZ),
});

export class Client implements AsyncTermSearcher<string, Key, Range> {
  readonly type: string = "range";
  private readonly frameClient: framer.Client;
  private readonly writer: Writer;
  private readonly unaryClient: UnaryClient;
  private readonly channels: ChannelRetriever;
  private readonly labelClient: label.Client;
  private readonly ontologyClient: ontology.Client;

  constructor(
    frameClient: framer.Client,
    writer: Writer,
    unary: UnaryClient,
    channels: ChannelRetriever,
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

  async create(range: NewPayload, options?: CreateOptions): Promise<Range>;

  async create(ranges: NewPayload[], options?: CreateOptions): Promise<Range[]>;

  async create(
    ranges: NewPayload | NewPayload[],
    options?: CreateOptions,
  ): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const res = this.sugar(await this.writer.create(toArray(ranges), options));
    return single ? res[0] : res;
  }

  async rename(key: Key, name: Name): Promise<void> {
    await this.writer.rename(key, name);
  }

  async delete(key: Key | Keys): Promise<void> {
    await this.writer.delete(toArray(key));
  }

  async search(term: string): Promise<Range[]> {
    return this.sugar(await this.execRetrieve({ term }));
  }

  async page(offset: number, limit: number): Promise<Range[]> {
    return this.sugar(await this.execRetrieve({ offset, limit }));
  }

  async retrieve(range: CrudeTimeRange): Promise<Range[]>;

  async retrieve(range: Key | Name): Promise<Range>;

  async retrieve(range: Keys | Names): Promise<Range[]>;

  async retrieve(params: Params | CrudeTimeRange): Promise<Range | Range[]> {
    if (typeof params === "object" && "start" in params)
      return await this.execRetrieve({ overlapsWith: new TimeRange(params) });
    const { single, actual, variant, normalized, empty } = analyzeParams(params);
    if (empty) return [];
    const ranges = await this.execRetrieve({ [variant]: normalized });
    if (!single) return ranges;
    if (ranges.length === 0)
      throw new NotFoundError(`range matching ${actual} not found`);
    if (ranges.length > 1)
      throw new MultipleFoundError(`multiple ranges matching ${actual} found`);
    return ranges[0];
  }

  getKV(range: Key): KV {
    return new KV(range, this.unaryClient, this.frameClient);
  }

  private async execRetrieve(req: RetrieveRequest): Promise<Range[]> {
    const { ranges } = await sendRequired<typeof retrieveReqZ, typeof retrieveResZ>(
      this.unaryClient,
      RETRIEVE_ENDPOINT,
      req,
      retrieveReqZ,
      retrieveResZ,
    );
    return this.sugar(ranges);
  }

  async retrieveParent(range: Key): Promise<Range | null> {
    const res = await this.ontologyClient.retrieveParents({
      key: range,
      type: "range",
    });
    if (res.length === 0) return null;
    const first = res[0];
    if (first.id.type !== "range") return null;
    return await this.retrieve(first.id.key);
  }

  sugar(payloads: Payload[]): Range[] {
    return payloads.map((payload) => {
      return new Range(
        payload.name,
        payload.timeRange,
        payload.key,
        payload.color,
        this.frameClient,
        new KV(payload.key, this.unaryClient, this.frameClient),
        new Aliaser(payload.key, this.frameClient, this.unaryClient),
        this.channels,
        this.labelClient,
        this.ontologyClient,
        this,
      );
    });
  }

  async openTracker(): Promise<signals.Observable<string, Range>> {
    return await signals.openObservable<string, Range>(
      this.frameClient,
      "sy_range_set",
      "sy_range_delete",
      (variant, data) => {
        if (variant === "delete")
          return data.toStrings().map((k) => ({ variant, key: k, value: undefined }));
        const sugared = this.sugar(data.parseJSON(payloadZ));
        return sugared.map((r) => ({ variant, key: r.key, value: r }));
      },
    );
  }

  resourcesToRanges(resources: Resource[]): Range[] {
    return this.sugar(
      resources.map((r) => ({
        key: r.id.key,
        name: r.data?.name as string,
        timeRange: new TimeRange(r.data?.timeRange as CrudeTimeRange),
      })),
    );
  }
}

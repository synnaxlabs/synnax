// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { CrudeTimeRange, TimeRange } from "@synnaxlabs/x";
import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { toArray } from "@synnaxlabs/x/toArray";
import { z } from "zod";

import { type Retriever as ChannelRetriever } from "@/channel/retriever";
import { MultipleFoundError, NotFoundError } from "@/errors";
import { type framer } from "@/framer";
import { type label } from "@/label";
import { Active } from "@/ranger/active";
import { Aliaser } from "@/ranger/alias";
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
import { Range } from "@/ranger/range";
import { type Writer } from "@/ranger/writer";
import { signals } from "@/signals";
import { nullableArrayZ } from "@/util/zod";

const retrieveReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.array(z.string()).optional(),
  term: z.string().optional(),
  overlapsWith: TimeRange.z.optional(),
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
  private readonly active: Active;
  private readonly labelClient: label.Client;

  constructor(
    frameClient: framer.Client,
    writer: Writer,
    unary: UnaryClient,
    channels: ChannelRetriever,
    labelClient: label.Client,
  ) {
    this.frameClient = frameClient;
    this.writer = writer;
    this.unaryClient = unary;
    this.channels = channels;
    this.active = new Active(unary);
    this.labelClient = labelClient;
  }

  async create(range: NewPayload): Promise<Range>;

  async create(ranges: NewPayload[]): Promise<Range[]>;

  async create(ranges: NewPayload | NewPayload[]): Promise<Range | Range[]> {
    const single = !Array.isArray(ranges);
    const res = this.sugar(await this.writer.create(toArray(ranges)));
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

  async page(): Promise<Range[]> {
    return [];
  }

  async retrieve(range: CrudeTimeRange): Promise<Range[]>;

  async retrieve(range: Key | Name): Promise<Range>;

  async retrieve(range: Keys | Names): Promise<Range[]>;

  async retrieve(params: Params | CrudeTimeRange): Promise<Range | Range[]> {
    if (typeof params === "object" && "start" in params)
      return await this.execRetrieve({ overlapsWith: new TimeRange(params) });
    const { single, actual, variant, normalized } = analyzeParams(params);
    const ranges = await this.execRetrieve({ [variant]: normalized });
    if (!single) return ranges;
    if (ranges.length === 0)
      throw new NotFoundError(`range matching ${actual} not found`);
    if (ranges.length > 1)
      throw new MultipleFoundError(`multiple ranges matching ${actual} found`);
    return ranges[0];
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

  async setActive(range: Key): Promise<void> {
    await this.active.setActive(range);
  }

  async retrieveActive(): Promise<Range | null> {
    const res = await this.active.retrieveActive();
    if (res == null) return null;
    return this.sugar([res])[0];
  }

  async clearActive(range: Key): Promise<void> {
    await this.active.clearActive(range);
  }

  private sugar(payloads: Payload[]): Range[] {
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
}

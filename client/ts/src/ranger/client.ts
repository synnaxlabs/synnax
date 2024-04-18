// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type AsyncTermSearcher, toArray } from "@synnaxlabs/x";

import { type Retriever as ChannelRetriever } from "@/channel/retriever";
import { QueryError } from "@/errors";
import { type framer } from "@/framer";
import { type label } from "@/label";
import { Active } from "@/ranger/active";
import { Aliaser } from "@/ranger/alias";
import { KV } from "@/ranger/kv";
import {
  type NewPayload,
  type Key,
  type Keys,
  type Name,
  type Names,
  type Params,
  type Payload,
  analyzeParams,
} from "@/ranger/payload";
import { Range } from "@/ranger/range";
import { type Retriever } from "@/ranger/retriever";
import { type Writer } from "@/ranger/writer";

export class Client implements AsyncTermSearcher<string, Key, Range> {
  private readonly frameClient: framer.Client;
  private readonly retriever: Retriever;
  private readonly writer: Writer;
  private readonly unaryClient: UnaryClient;
  private readonly channels: ChannelRetriever;
  private readonly active: Active;
  private readonly labelClient: label.Client;

  constructor(
    frameClient: framer.Client,
    retriever: Retriever,
    writer: Writer,
    unary: UnaryClient,
    channels: ChannelRetriever,
    labelClient: label.Client,
  ) {
    this.frameClient = frameClient;
    this.retriever = retriever;
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
    return this.sugar(await this.retriever.search(term));
  }

  async page(offset: number, limit: number): Promise<Range[]> {
    return [];
  }

  async retrieve(range: Key | Name): Promise<Range>;

  async retrieve(params: Keys | Names): Promise<Range[]>;

  async retrieve(params: Params): Promise<Range | Range[]> {
    const { single, actual } = analyzeParams(params);
    const res = this.sugar(await this.retriever.retrieve(params));
    if (!single) return res;
    if (res.length === 0) throw new QueryError(`range matching ${actual} not found`);
    if (res.length > 1)
      throw new QueryError(`multiple ranges matching ${actual} found`);
    return res[0];
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
        this.frameClient,
        new KV(payload.key, this.unaryClient),
        new Aliaser(payload.key, this.frameClient, this.unaryClient),
        this.channels,
        this.labelClient,
      );
    });
  }
}

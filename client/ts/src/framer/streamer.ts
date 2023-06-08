// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errorZ, Stream, StreamClient } from "@synnaxlabs/freighter";
import { TimeStamp, UnparsedTimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import { ChannelParams } from "@/channel/payload";
import { ChannelRetriever } from "@/channel/retriever";
import { BackwardFrameAdapter } from "@/framer/adapter";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({
  start: TimeStamp.z.optional(),
  keys: z.number().array(),
});

type Request = z.infer<typeof reqZ>;

const resZ = z.object({
  frame: frameZ,
  error: errorZ.optional().nullable(),
});

type Response = z.infer<typeof resZ>;

export class Streamer implements AsyncIterator<Frame>, AsyncIterable<Frame> {
  private static readonly ENDPOINT = "/frame/stream";
  private readonly stream: StreamProxy<Request, Response>;
  private readonly adapter: BackwardFrameAdapter;

  private constructor(
    stream: Stream<Request, Response>,
    adapter: BackwardFrameAdapter
  ) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
  }

  static async _open(
    start: UnparsedTimeStamp,
    channels: ChannelParams[],
    retriever: ChannelRetriever,
    client: StreamClient
  ): Promise<Streamer> {
    const adapter = await BackwardFrameAdapter.open(retriever, channels);
    const stream = await client.stream(Streamer.ENDPOINT, reqZ, resZ);
    const streamer = new Streamer(stream, adapter);
    stream.send({ start: new TimeStamp(start), keys: adapter.keys });
    return streamer;
  }

  async next(): Promise<IteratorResult<Frame, any>> {
    try {
      const frame = await this.read();
      return { done: false, value: frame };
    } catch (EOF) {
      return { done: true, value: undefined };
    }
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
    return this;
  }

  async read(): Promise<Frame> {
    return this.adapter.adapt(new Frame((await this.stream.receive()).frame));
  }

  async update(...params: ChannelParams[]): Promise<void> {
    await this.adapter.update(params);
    this.stream.send({ keys: this.adapter.keys });
  }

  close(): void {
    this.stream.closeSend();
  }
}

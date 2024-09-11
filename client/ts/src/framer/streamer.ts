// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, errorZ, type Stream, type StreamClient } from "@synnaxlabs/freighter";
import { observe } from "@synnaxlabs/x";
import { z } from "zod";

import { type Key, type Params } from "@/channel/payload";
import { type Retriever } from "@/channel/retriever";
import { ReadFrameAdapter } from "@/framer/adapter";
import { Frame, frameZ } from "@/framer/frame";
import { StreamProxy } from "@/framer/streamProxy";

const reqZ = z.object({ keys: z.number().array() });

const resZ = z.object({
  frame: frameZ,
  error: errorZ.optional().nullable(),
});

const ENDPOINT = "/frame/stream";

export interface StreamerConfig {
  channels: Params;
}

export class Streamer implements AsyncIterator<Frame>, AsyncIterable<Frame> {
  private readonly stream: StreamProxy<typeof reqZ, typeof resZ>;
  private readonly adapter: ReadFrameAdapter;

  private constructor(
    stream: Stream<typeof reqZ, typeof resZ>,
    adapter: ReadFrameAdapter,
  ) {
    this.stream = new StreamProxy("Streamer", stream);
    this.adapter = adapter;
  }

  get keys(): Key[] {
    return this.adapter.keys;
  }

  static async _open(
    retriever: Retriever,
    client: StreamClient,
    { channels }: StreamerConfig,
  ): Promise<Streamer> {
    const adapter = await ReadFrameAdapter.open(retriever, channels);
    const stream = await client.stream(ENDPOINT, reqZ, resZ);
    const streamer = new Streamer(stream, adapter);
    stream.send({ keys: adapter.keys });
    return streamer;
  }

  async next(): Promise<IteratorResult<Frame, any>> {
    try {
      const frame = await this.read();
      return { done: false, value: frame };
    } catch (err) {
      if (EOF.matches(err)) return { done: true, value: undefined };
      throw err;
    }
  }

  async read(): Promise<Frame> {
    return this.adapter.adapt(new Frame((await this.stream.receive()).frame));
  }

  async update(channels: Params): Promise<void> {
    await this.adapter.update(channels);
    this.stream.send({ keys: this.adapter.keys });
  }

  close(): void {
    this.stream.closeSend();
  }

  [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
    return this;
  }
}

export class ObservableStreamer<V = Frame>
  extends observe.Observer<Frame, V>
  implements observe.ObservableAsyncCloseable<V>
{
  private readonly streamer: Streamer;
  private readonly closePromise: Promise<void>;

  constructor(streamer: Streamer, transform?: observe.Transform<Frame, V>) {
    super(transform);
    this.streamer = streamer;
    this.closePromise = this.stream();
  }

  async close(): Promise<void> {
    this.streamer.close();
    await this.closePromise;
  }

  private async stream(): Promise<void> {
    for await (const frame of this.streamer) this.notify(frame);
  }
}

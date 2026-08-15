// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, type framer } from "@synnaxlabs/client";
import { id, MultiSeries, Series } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { createFactory } from "@/telem/aether/factory";
import { sourcePipeline } from "@/telem/aether/pipeline";
import { type Client, streamChannelValue } from "@/telem/aether/remote";
import { type BooleanSource } from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/test";
import { withinBounds } from "@/telem/aether/transformers";

class MockClient implements Client {
  key = id.create();
  streamHandler: framer.StreamHandler | null = null;
  streamF = vi.fn();
  channel = new channel.Channel({
    key: 65537,
    name: "virtual",
    dataType: DataType.UINT8,
    isIndex: false,
  });
  channels = { retrieve: async (): Promise<channel.Channel> => this.channel };
  feed = {
    read: async (): Promise<MultiSeries> => new MultiSeries([]),
    stream: (
      handler: framer.StreamHandler,
      keys: channel.Key[],
    ): framer.Subscription => {
      this.streamHandler = handler;
      this.streamF(handler, keys);
      // The streamer delivers the cached leading buffers synchronously at
      // registration. A channel that has never written yields an empty series per key.
      handler(new Map(keys.map((k) => [k, new MultiSeries([])])));
      return telemTest.mockSubscription(() => {});
    },
  };
}

// The source a light, valve, or switch ships with.
const booleanSource = (key: channel.Key) =>
  sourcePipeline("boolean", {
    connections: [{ from: "valueStream", to: "threshold" }],
    segments: {
      valueStream: streamChannelValue({ channel: key }),
      threshold: withinBounds({ trueBound: { lower: 0.9, upper: 1.1 } }),
    },
    outlet: "threshold",
  });

const setup = async () => {
  const c = new MockClient();
  const source = createFactory(c).create(booleanSource(c.channel.key)) as BooleanSource;
  const handler = vi.fn();
  source.onChange(handler);
  source.value();
  await expect.poll(() => c.streamF.mock.calls.length > 0).toBe(true);
  return { c, source, handler };
};

// Staleness counts arrivals, so a boolean symbol pointed at a channel with no data must
// not see one. Anything it sees here turns the symbol stale one timeout later.
describe("boolean symbol source on a channel with no data", () => {
  it("should not report an arrival when the stream registers", async () => {
    const { handler } = await setup();
    expect(handler).not.toHaveBeenCalled();
  });

  it("should still read as empty after the stream registers", async () => {
    const { source } = await setup();
    expect(source.value()).toBe(false);
  });

  it("should report an arrival once real data lands", async () => {
    const { c, source, handler } = await setup();
    c.streamHandler?.(
      new Map([
        [c.channel.key, new MultiSeries([new Series({ data: new Uint8Array([1]) })])],
      ]),
    );
    expect(handler).toHaveBeenCalledTimes(1);
    expect(source.value()).toBe(true);
  });
});

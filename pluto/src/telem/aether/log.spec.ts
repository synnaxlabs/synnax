// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import {
  type destructor,
  id,
  MultiSeries,
  Series,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  StreamMultiChannelLog,
  type StreamMultiChannelLogProps,
} from "@/telem/aether/log";
import { type Source } from "@/telem/aether/telem";
import { type client } from "@/telem/client";

const waitForResolve = async <T>(source: Source<T>): Promise<T> => {
  source.value();
  const handleChange = vi.fn();
  source.onChange(handleChange);
  await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
  return source.value();
};

describe("StreamMultiChannelLog", () => {
  class MockClient implements client.Client {
    key: string = id.create();

    streamHandler: client.StreamHandler | null = null;
    streamKeys: channel.Keys = [];
    streamF = vi.fn();
    streamDestructorF = vi.fn();

    channelA: channel.Channel = new channel.Channel({
      key: 1,
      name: "channel_a",
      dataType: DataType.FLOAT32,
      isIndex: false,
    });

    channelB: channel.Channel = new channel.Channel({
      key: 2,
      name: "channel_b",
      dataType: DataType.FLOAT32,
      isIndex: false,
    });

    async retrieveChannel(key: channel.KeyOrName): Promise<channel.Channel> {
      if (key === this.channelA.key) return this.channelA;
      if (key === this.channelB.key) return this.channelB;
      throw new Error(`Channel ${key} not found`);
    }

    async read(): Promise<MultiSeries> {
      return new MultiSeries([]);
    }

    async stream(
      handler: client.StreamHandler,
      keys: channel.Keys,
    ): Promise<destructor.Async> {
      this.streamHandler = handler;
      this.streamKeys = keys;
      this.streamF(handler, keys);
      return this.streamDestructorF;
    }

    async close(): Promise<void> {}
  }

  let c: MockClient;

  beforeEach(() => {
    c = new MockClient();
    vi.resetAllMocks();
  });

  it("should return an empty array when no channels are configured", () => {
    const props: StreamMultiChannelLogProps = {
      channels: [],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    expect(log.value()).toHaveLength(0);
    expect(c.streamF).not.toHaveBeenCalled();
  });

  it("should open a stream with the correct channel keys", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key, c.channelB.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    expect(c.streamF).toHaveBeenCalledOnce();
    expect(c.streamKeys).toEqual([c.channelA.key, c.channelB.key]);
  });

  it("should append entries when stream data arrives", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([1, 2, 3]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    expect(log.value()).toHaveLength(3);
  });

  it("should populate channelKey, channelName, and value in each entry", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([42]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    const entries = log.value();
    expect(entries).toHaveLength(1);
    expect(entries[0].channelKey).toBe(c.channelA.key);
    expect(entries[0].channelName).toBe("channel_a");
    expect(entries[0].value).toBe("42");
  });

  it("should maintain arrival order and not sort across multiple channels", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key, c.channelB.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);

    // channel_a arrives first, then channel_b
    const seriesA = new Series({ data: new Float32Array([1]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA])]]));
    const seriesB = new Series({ data: new Float32Array([2]) });
    c.streamHandler?.(new Map([[c.channelB.key, new MultiSeries([seriesB])]]));

    const entries = log.value();
    expect(entries).toHaveLength(2);
    expect(entries[0].channelKey).toBe(c.channelA.key);
    expect(entries[1].channelKey).toBe(c.channelB.key);
  });

  it("should notify observers when new data arrives", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const handleChange = vi.fn();
    log.onChange(handleChange);
    const series = new Series({ data: new Float32Array([1]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
  });

  it("should stop the stream on cleanup", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    log.cleanup();
    expect(c.streamDestructorF).toHaveBeenCalled();
  });

  it("should clear all entries on cleanup", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([1, 2, 3]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    expect(log.value()).toHaveLength(3);
    log.cleanup();
    expect(log.value()).toHaveLength(0);
  });

  it("should cap entries at 100,000", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.days(1),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const bigData = new Float32Array(100_001).fill(1);
    const series = new Series({ data: bigData });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    expect(log.value()).toHaveLength(100_000);
  }, 15_000);

  it("should set channelPadding to align values across channels of different name lengths", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key, c.channelB.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const seriesA = new Series({ data: new Float32Array([1]) });
    const seriesB = new Series({ data: new Float32Array([2]) });
    c.streamHandler?.(
      new Map([
        [c.channelA.key, new MultiSeries([seriesA])],
        [c.channelB.key, new MultiSeries([seriesB])],
      ]),
    );
    const entries = log.value();
    // channel_a (9 chars) and channel_b (9 chars) have equal length — both get ""
    expect(entries[0].channelPadding).toBe("");
    expect(entries[1].channelPadding).toBe("");
  });

  it("should pad the shorter channel name to align with the longest", async () => {
    // Give channelB a longer name so we can observe the padding on channelA
    c.channelB = new channel.Channel({
      key: c.channelB.key,
      name: "a_very_long_channel_name",
      dataType: DataType.FLOAT32,
      isIndex: false,
    });
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key, c.channelB.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const seriesA = new Series({ data: new Float32Array([1]) });
    const seriesB = new Series({ data: new Float32Array([2]) });
    c.streamHandler?.(
      new Map([
        [c.channelA.key, new MultiSeries([seriesA])],
        [c.channelB.key, new MultiSeries([seriesB])],
      ]),
    );
    const entries = log.value();
    const maxLen = "a_very_long_channel_name".length;
    const shortLen = c.channelA.name.length; // "channel_a" = 9
    expect(entries[0].channelPadding).toBe(" ".repeat(maxLen - shortLen));
    expect(entries[1].channelPadding).toBe("");
  });

  it("should read entries from subsequent buffer allocations", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series1 = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series1])]]));
    expect(log.value()).toHaveLength(2);
    // Second allocation — simulates buffer filling up and a new one being allocated
    const series2 = new Series({ data: new Float32Array([3, 4]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series2])]]));
    expect(log.value()).toHaveLength(4);
    expect(log.value()[2].value).toBe("3");
    expect(log.value()[3].value).toBe("4");
  });

  it("should read all entries when multiple buffers arrive in a single callback", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    // Simulate a burst that crosses two buffer boundaries in one callback.
    const series1 = new Series({ data: new Float32Array([1, 2, 3]) });
    const series2 = new Series({ data: new Float32Array([4, 5]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series1, series2])]]));
    const entries = log.value();
    expect(entries).toHaveLength(5);
    expect(entries.map((e) => e.value)).toEqual(["1", "2", "3", "4", "5"]);
  });

  it("should not re-read entries when allocated is empty", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    expect(log.value()).toHaveLength(2);
    // Empty allocated — simulates the common case where the dynamic cache writes
    // new samples into the existing buffer in-place and returns no newly allocated
    // buffers. The cursor should not advance and no entries should be duplicated.
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([])]]));
    expect(log.value()).toHaveLength(2);
  });

  it("should garbage collect entries older than keepFor", async () => {
    let now = TimeStamp.milliseconds(1_000);
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
      keepFor: TimeSpan.milliseconds(500),
    };
    const log = new StreamMultiChannelLog(c, props, undefined, () => now);
    await waitForResolve(log);

    // Push entries at t=1000ms
    const seriesA = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA])]]));
    expect(log.value()).toHaveLength(2);

    // Advance time past keepFor — threshold becomes 2000 - 500 = 1500ms
    now = TimeStamp.milliseconds(2_000);

    // Push a new entry — triggers GC, old entries at t=1000ms should be evicted
    const seriesB = new Series({ data: new Float32Array([3]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesB])]]));
    const entries = log.value();
    expect(entries).toHaveLength(1);
    expect(entries[0].value).toBe("3");
  });

  it("should notify when GC evicts as many entries as were pushed in the same callback", async () => {
    let now = TimeStamp.milliseconds(1_000);
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
      keepFor: TimeSpan.milliseconds(500),
    };
    const log = new StreamMultiChannelLog(c, props, undefined, () => now);
    await waitForResolve(log);

    // Fill the window with 2 entries at t=1000ms
    const seriesA = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA])]]));
    expect(log.value()).toHaveLength(2);

    // Advance time so both existing entries are stale (threshold = 2000 - 500 = 1500ms)
    now = TimeStamp.milliseconds(2_000);

    // Push exactly 2 new entries — GC evicts the 2 old ones, net count unchanged.
    // The notify guard must fire because new entries arrived, not because count changed.
    const handleChange = vi.fn();
    log.onChange(handleChange);
    const seriesB = new Series({ data: new Float32Array([3, 4]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesB])]]));
    await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
    const entries = log.value();
    expect(entries).toHaveLength(2);
    expect(entries.map((e) => e.value)).toEqual(["3", "4"]);
  });
});

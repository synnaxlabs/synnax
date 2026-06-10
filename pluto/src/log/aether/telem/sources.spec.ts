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
} from "@/log/aether/telem/sources";
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
    streamKeys: channel.Key[] = [];
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

    channelInt: channel.Channel = new channel.Channel({
      key: 3,
      name: "channel_int",
      dataType: DataType.INT64,
      isIndex: false,
    });

    channelJSON: channel.Channel = new channel.Channel({
      key: 4,
      name: "channel_json",
      dataType: DataType.JSON,
      isIndex: false,
    });

    async retrieveChannel(key: channel.Key | channel.Name): Promise<channel.Channel> {
      if (key === this.channelA.key) return this.channelA;
      if (key === this.channelB.key) return this.channelB;
      if (key === this.channelInt.key) return this.channelInt;
      if (key === this.channelJSON.key) return this.channelJSON;
      throw new Error(`Channel ${key} not found`);
    }

    async read(): Promise<MultiSeries> {
      return new MultiSeries([]);
    }

    async stream(
      handler: client.StreamHandler,
      keys: channel.Key[],
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

  it("should populate channelKey and value in each entry", async () => {
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
    expect(entries[0].value).toBe("42");
  });

  it("should clean f64-widened FLOAT32 values to shortest decimal", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([1.234]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    const entries = log.value();
    expect(entries).toHaveLength(1);
    expect(entries[0].value).toBe("1.234");
  });

  it("should stringify INT64 bigint values", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelInt.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new BigInt64Array([42n, -7n]) });
    c.streamHandler?.(new Map([[c.channelInt.key, new MultiSeries([series])]]));
    const entries = log.value();
    expect(entries).toHaveLength(2);
    expect(entries[0].value).toBe("42");
    expect(entries[1].value).toBe("-7");
  });

  it("should stringify JSON values as their serialized form", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelJSON.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({
      data: [{ name: "Alice", value: 30 }],
      dataType: DataType.JSON,
    });
    c.streamHandler?.(new Map([[c.channelJSON.key, new MultiSeries([series])]]));
    const entries = log.value();
    expect(entries).toHaveLength(1);
    expect(entries[0].value).toBe('{"name":"Alice","value":30}');
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
  }, 30_000);

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

  it("should set evictedCount to 0 before any GC has run", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    expect(log.evictedCount).toBe(0);
  });

  it("should set evictedCount to the number of entries removed by GC", async () => {
    let now = TimeStamp.milliseconds(1_000);
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
      keepFor: TimeSpan.milliseconds(500),
    };
    const log = new StreamMultiChannelLog(c, props, undefined, () => now);
    await waitForResolve(log);

    const seriesA = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA])]]));
    expect(log.evictedCount).toBe(0);

    // Advance time — both entries at t=1000ms are now stale
    now = TimeStamp.milliseconds(2_000);
    const seriesB = new Series({ data: new Float32Array([3]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesB])]]));
    expect(log.evictedCount).toBe(2);
  });

  it("should reset evictedCount to 0 on a subsequent callback where no GC occurs", async () => {
    let now = TimeStamp.milliseconds(1_000);
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
      timeSpan: TimeSpan.seconds(30),
      keepFor: TimeSpan.milliseconds(500),
    };
    const log = new StreamMultiChannelLog(c, props, undefined, () => now);
    await waitForResolve(log);

    // Trigger GC
    const seriesA = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA])]]));
    now = TimeStamp.milliseconds(2_000);
    const seriesB = new Series({ data: new Float32Array([3]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesB])]]));
    expect(log.evictedCount).toBe(2);

    // Next callback — nothing is stale, evictedCount resets to 0
    const seriesC = new Series({ data: new Float32Array([4]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesC])]]));
    expect(log.evictedCount).toBe(0);
  });

  describe("newline handling", () => {
    it("should split a string sample on \\n into multiple entries", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: ["line1\nline2\n"] });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(3);
      expect(entries.map((e) => e.value)).toEqual(["line1", "line2", ""]);
      expect(entries.map((e) => e.continuation === true)).toEqual([false, true, true]);
      expect(entries.every((e) => e.channelKey === c.channelA.key)).toBe(true);
      const ts = entries[0].timestamp;
      expect(entries.every((e) => e.timestamp === ts)).toBe(true);
    });

    it("should split on Windows-style \\r\\n without leaving a stray \\r", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: ["line1\r\nline2"] });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value().map((e) => e.value)).toEqual(["line1", "line2"]);
    });

    it("should preserve consecutive newlines as blank entries", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: ["a\n\nb"] });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value().map((e) => e.value)).toEqual(["a", "", "b"]);
    });

    it("should produce a single non-continuation entry for a string with no \\n", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: ["hello world", ""] });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(2);
      expect(entries[0].value).toBe("hello world");
      expect(entries[0].continuation === true).toBe(false);
      // An explicit empty write is NOT a continuation — the prefix should still
      // render. Only entries produced by a \n split are continuations.
      expect(entries[1].value).toBe("");
      expect(entries[1].continuation === true).toBe(false);
    });

    it("should not split numeric channel values", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: new Float32Array([1, 2, 3]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(3);
      expect(entries.map((e) => e.value)).toEqual(["1", "2", "3"]);
    });

    it("should not split JSON channel values that contain \\n", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.JSON,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: [{ msg: "hi\nthere" }] });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(1);
      expect(entries[0].value).toBe(JSON.stringify({ msg: "hi\nthere" }));
    });

    it("should preserve raw snake_case keys in JSON channel values", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.JSON,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({
        data: [{ user_id: 1, first_name: "alice" }],
      });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(1);
      expect(entries[0].value).toBe('{"user_id":1,"first_name":"alice"}');
    });

    it("should produce a single non-continuation entry for an empty string sample", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: [""] });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(1);
      expect(entries[0].value).toBe("");
      expect(entries[0].continuation === true).toBe(false);
    });

    it("should not split BYTES channel values that contain \\n", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.BYTES,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const payload = new TextEncoder().encode("line1\nline2");
      const buf = new ArrayBuffer(4 + payload.byteLength);
      new DataView(buf).setUint32(0, payload.byteLength, true);
      new Uint8Array(buf).set(payload, 4);
      const series = new Series({ data: buf, dataType: DataType.BYTES });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries).toHaveLength(1);
      expect(entries[0].value).toBe("line1\nline2");
      expect(entries[0].continuation === true).toBe(false);
    });

    it("should not leave orphan continuation entries at the head when maxEntries eviction cuts mid-group", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props, undefined, undefined, 100);
      await waitForResolve(log);
      // 35 three-line samples = 105 entries; excess of 5 cuts mid-group.
      const data = new Array(35).fill("a\nb\nc");
      const series = new Series({ data });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      const entries = log.value();
      expect(entries.length).toBeGreaterThan(0);
      expect(entries[0].continuation === true).toBe(false);
    });
  });

  describe("setChannels", () => {
    it("should be a no-op when channels have not changed", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: new Float32Array([1, 2]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value()).toHaveLength(2);

      // setChannels with the same list should not restart the stream
      log.setChannels([c.channelA.key]);
      expect(c.streamDestructorF).not.toHaveBeenCalled();
      expect(log.value()).toHaveLength(2);
    });

    it("should preserve entries for remaining channels when a channel is removed", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key, c.channelB.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const seriesA = new Series({ data: new Float32Array([1, 2]) });
      const seriesB = new Series({ data: new Float32Array([3]) });
      c.streamHandler?.(
        new Map([
          [c.channelA.key, new MultiSeries([seriesA])],
          [c.channelB.key, new MultiSeries([seriesB])],
        ]),
      );
      expect(log.value()).toHaveLength(3);

      // Remove channel B
      log.setChannels([c.channelA.key]);
      const entries = await waitForResolve(log);
      // Only channel A's 2 entries should remain
      expect(entries).toHaveLength(2);
      expect(entries.every((e) => e.channelKey === c.channelA.key)).toBe(true);
    });

    it("should scrub entries for a removed channel", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key, c.channelB.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const seriesA = new Series({ data: new Float32Array([10]) });
      const seriesB = new Series({ data: new Float32Array([20, 30]) });
      c.streamHandler?.(
        new Map([
          [c.channelA.key, new MultiSeries([seriesA])],
          [c.channelB.key, new MultiSeries([seriesB])],
        ]),
      );
      expect(log.value()).toHaveLength(3);

      // Remove channel A, keep channel B
      log.setChannels([c.channelB.key]);
      const entries = await waitForResolve(log);
      expect(entries).toHaveLength(2);
      expect(entries[0].value).toBe("20");
      expect(entries[1].value).toBe("30");
    });

    it("should stop streaming when all channels are removed", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));

      log.setChannels([]);
      expect(c.streamDestructorF).toHaveBeenCalled();
    });

    it("should not duplicate entries when the stream is restarted", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1, 2, 3]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value()).toHaveLength(3);

      // Add channel B — this restarts the stream. The mock's stream() will
      // call c.streamHandler again with an initial batch. Simulate the batch containing
      // the existing cache data for channel A.
      c.streamF = vi.fn((handler: client.StreamHandler, _keys: channel.Key[]) => {
        // Simulate the initial batch: channel A has cached data we already consumed.
        const initialA = new Series({ data: new Float32Array([1, 2, 3]) });
        handler(new Map([[c.channelA.key, new MultiSeries([initialA])]]));
      });
      log.setChannels([c.channelA.key, c.channelB.key]);
      await waitForResolve(log);

      // Channel A entries should NOT be duplicated (skipInitialBatch should have fired)
      const entries = log.value();
      const channelAEntries = entries.filter((e) => e.channelKey === c.channelA.key);
      expect(channelAEntries).toHaveLength(3);
    });

    it("should skip the initial batch for newly-added channels on restart", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value()).toHaveLength(1);

      // Add channel B. Simulate the initial batch delivering cached data for channel B
      // that was accumulated by another component — we should NOT dump it.
      c.streamF = vi.fn((handler: client.StreamHandler, _keys: channel.Key[]) => {
        const initialB = new Series({ data: new Float32Array([10, 20, 30]) });
        handler(new Map([[c.channelB.key, new MultiSeries([initialB])]]));
      });
      log.setChannels([c.channelA.key, c.channelB.key]);
      await waitForResolve(log);

      // Channel B initial-batch data should be skipped — only channel A's entry remains
      const entries = log.value();
      expect(entries).toHaveLength(1);
      expect(entries[0].channelKey).toBe(c.channelA.key);
    });

    it("should accept new data after skipping the initial batch", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));

      // Restart with channel B added; the initial batch is skipped
      c.streamF = vi.fn((handler: client.StreamHandler, _keys: channel.Key[]) => {
        const initialA = new Series({ data: new Float32Array([1]) });
        const initialB = new Series({ data: new Float32Array([10]) });
        handler(
          new Map([
            [c.channelA.key, new MultiSeries([initialA])],
            [c.channelB.key, new MultiSeries([initialB])],
          ]),
        );
      });
      log.setChannels([c.channelA.key, c.channelB.key]);
      await waitForResolve(log);
      expect(log.value()).toHaveLength(1); // only the original entry

      // Now new data arrives AFTER the initial batch — should be accepted normally
      const newA = new Series({ data: new Float32Array([2]) });
      const newB = new Series({ data: new Float32Array([20]) });
      c.streamHandler?.(
        new Map([
          [c.channelA.key, new MultiSeries([newA])],
          [c.channelB.key, new MultiSeries([newB])],
        ]),
      );
      const entries = log.value();
      expect(entries).toHaveLength(3);
      expect(entries[1].channelKey).toBe(c.channelA.key);
      expect(entries[1].value).toBe("2");
      expect(entries[2].channelKey).toBe(c.channelB.key);
      expect(entries[2].value).toBe("20");
    });

    it("should allow initial-batch data on first start (not a restart)", async () => {
      // Simulate a client that delivers an initial batch on the first stream() call
      c.streamF = vi.fn((handler: client.StreamHandler, _keys: channel.Key[]) => {
        const initial = new Series({ data: new Float32Array([10, 20]) });
        handler(new Map([[c.channelA.key, new MultiSeries([initial])]]));
      });

      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
        timeSpan: TimeSpan.seconds(30),
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      // The initial batch should be accepted on first start
      const entries = log.value();
      expect(entries).toHaveLength(2);
      expect(entries[0].value).toBe("10");
      expect(entries[1].value).toBe("20");
    });
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  DataType,
  type framer,
  type status as cstatus,
} from "@synnaxlabs/client";
import { id, MultiSeries, Series, TimeStamp } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  StreamMultiChannelLog,
  type StreamMultiChannelLogProps,
} from "@/log/aether/telem/sources";
import { type Client } from "@/telem/aether/remote";
import { type Source } from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/test";

const waitForResolve = async <T>(source: Source<T>): Promise<T> => {
  source.value();
  const handleChange = vi.fn();
  source.onChange(handleChange);
  await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
  return source.value();
};

describe("StreamMultiChannelLog", () => {
  class MockClient implements Client {
    now = (): TimeStamp => TimeStamp.now();
    key: string = id.create();

    streamHandler: framer.StreamHandler | null = null;
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

    indexed: channel.Channel[] = [];

    channels = {
      retrieve: async (key: channel.Key | channel.Name): Promise<channel.Channel> => {
        const indexed = this.indexed.find((ch) => ch.key === key);
        if (indexed != null) return indexed;
        if (key === this.channelA.key) return this.channelA;
        if (key === this.channelB.key) return this.channelB;
        if (key === this.channelInt.key) return this.channelInt;
        if (key === this.channelJSON.key) return this.channelJSON;
        throw new Error(`Channel ${key} not found`);
      },
    };

    feed = {
      readLatest: async (): Promise<MultiSeries> => new MultiSeries([]),
      readTile: async (): Promise<MultiSeries> => new MultiSeries([]),
      read: async (): Promise<MultiSeries> => new MultiSeries([]),
      stream: (
        handler: framer.StreamHandler,
        keys: channel.Key[],
      ): framer.Subscription => {
        this.streamHandler = handler;
        this.streamKeys = keys;
        this.streamF(handler, keys);
        return telemTest.mockSubscription(this.streamDestructorF);
      },
    };

    exactFeed = { stream: this.feed.stream };
  }

  let c: MockClient;

  beforeEach(() => {
    c = new MockClient();
    vi.resetAllMocks();
  });

  it("should not subscribe when cleaned up while retrieving channels", async () => {
    let release = (): void => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    c.channels.retrieve = async (): Promise<channel.Channel> => {
      await gate;
      return c.channelA;
    };
    const log = new StreamMultiChannelLog(c, {
      channels: [c.channelA.key],
    });
    log.value();
    log.cleanup();
    release();
    await new Promise((resolve) => setTimeout(resolve, 5));
    expect(c.streamF).not.toHaveBeenCalled();
  });

  it("should return an empty array when no channels are configured", () => {
    const props: StreamMultiChannelLogProps = {
      channels: [],
    };
    const log = new StreamMultiChannelLog(c, props);
    expect(log.value()).toHaveLength(0);
    expect(c.streamF).not.toHaveBeenCalled();
  });

  it("should open a stream with the correct channel keys", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key, c.channelB.key],
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    expect(c.streamF).toHaveBeenCalledOnce();
    expect(c.streamKeys).toEqual([c.channelA.key, c.channelB.key]);
  });

  it("should append entries when stream data arrives", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
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
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);

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
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    log.cleanup();
    expect(c.streamDestructorF).toHaveBeenCalled();
  });

  it("should clear all entries on cleanup", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
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
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const bigData = new Float32Array(100_001).fill(1);
    const series = new Series({ data: bigData });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    expect(log.value()).toHaveLength(100_000);
  });

  it("should read entries from subsequent buffer allocations", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
    };
    const log = new StreamMultiChannelLog(c, props);
    await waitForResolve(log);
    const series1 = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series1])]]));
    expect(log.value()).toHaveLength(2);
    const series2 = new Series({ data: new Float32Array([3, 4]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series2])]]));
    expect(log.value()).toHaveLength(4);
    expect(log.value()[2].value).toBe("3");
    expect(log.value()[3].value).toBe("4");
  });

  it("should read all entries when multiple buffers arrive in a single callback", async () => {
    const props: StreamMultiChannelLogProps = {
      channels: [c.channelA.key],
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

  it("should keep entries of any age", async () => {
    let now = TimeStamp.milliseconds(1_000);
    const log = new StreamMultiChannelLog(
      c,
      { channels: [c.channelA.key] },
      undefined,
      () => now,
    );
    await waitForResolve(log);
    const series = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
    now = TimeStamp.days(365);
    const next = new Series({ data: new Float32Array([3]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([next])]]));
    expect(log.value().map((e) => e.value)).toEqual(["1", "2", "3"]);
  });

  it("should notify when eviction removes as many entries as were pushed", async () => {
    const log = new StreamMultiChannelLog(
      c,
      { channels: [c.channelA.key] },
      undefined,
      undefined,
      2,
    );
    await waitForResolve(log);
    const seriesA = new Series({ data: new Float32Array([1, 2]) });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA])]]));
    expect(log.value()).toHaveLength(2);
    const handleChange = vi.fn();
    log.onChange(handleChange);
    const seriesB = new Series({ data: new Float32Array([3, 4]), alignment: 2n });
    c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([seriesA, seriesB])]]));
    await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
    expect(log.value().map((e) => e.value)).toEqual(["3", "4"]);
  });

  describe("newline handling", () => {
    it("should split a string sample on \\n into multiple entries", async () => {
      c.channelA = new channel.Channel({
        ...c.channelA,
        dataType: DataType.STRING,
      });
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
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
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);
      const series = new Series({ data: new Float32Array([1, 2]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value()).toHaveLength(2);

      log.setChannels([c.channelA.key]);
      expect(c.streamDestructorF).not.toHaveBeenCalled();
      expect(log.value()).toHaveLength(2);
    });

    it("should preserve entries for remaining channels when a channel is removed", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key, c.channelB.key],
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

      log.setChannels([c.channelA.key]);
      const entries = await waitForResolve(log);
      expect(entries).toHaveLength(2);
      expect(entries.every((e) => e.channelKey === c.channelA.key)).toBe(true);
    });

    it("should scrub entries for a removed channel", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key, c.channelB.key],
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

      log.setChannels([c.channelB.key]);
      const entries = await waitForResolve(log);
      expect(entries).toHaveLength(2);
      expect(entries[0].value).toBe("20");
      expect(entries[1].value).toBe("30");
    });

    it("should stop streaming when all channels are removed", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
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
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1, 2, 3]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value()).toHaveLength(3);

      // Add channel B — this restarts the stream. The mock's stream() will
      // call c.streamHandler again with an initial batch. Simulate the batch containing
      // the existing cache data for channel A.
      c.streamF = vi.fn((handler: framer.StreamHandler, _keys: channel.Key[]) => {
        const initialA = new Series({ data: new Float32Array([1, 2, 3]) });
        handler(new Map([[c.channelA.key, new MultiSeries([initialA])]]));
      });
      log.setChannels([c.channelA.key, c.channelB.key]);
      await waitForResolve(log);

      const entries = log.value();
      const channelAEntries = entries.filter((e) => e.channelKey === c.channelA.key);
      expect(channelAEntries).toHaveLength(3);
    });

    it("should skip the initial batch for newly-added channels on restart", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));
      expect(log.value()).toHaveLength(1);

      // Add channel B. Simulate the initial batch delivering cached data for channel B
      // that was accumulated by another component — we should NOT dump it.
      c.streamF = vi.fn((handler: framer.StreamHandler, _keys: channel.Key[]) => {
        const initialB = new Series({ data: new Float32Array([10, 20, 30]) });
        handler(new Map([[c.channelB.key, new MultiSeries([initialB])]]));
      });
      log.setChannels([c.channelA.key, c.channelB.key]);
      await waitForResolve(log);

      const entries = log.value();
      expect(entries).toHaveLength(1);
      expect(entries[0].channelKey).toBe(c.channelA.key);
    });

    it("should accept new data after skipping the initial batch", async () => {
      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const series = new Series({ data: new Float32Array([1]) });
      c.streamHandler?.(new Map([[c.channelA.key, new MultiSeries([series])]]));

      c.streamF = vi.fn((handler: framer.StreamHandler, _keys: channel.Key[]) => {
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
      c.streamF = vi.fn((handler: framer.StreamHandler, _keys: channel.Key[]) => {
        const initial = new Series({ data: new Float32Array([10, 20]) });
        handler(new Map([[c.channelA.key, new MultiSeries([initial])]]));
      });

      const props: StreamMultiChannelLogProps = {
        channels: [c.channelA.key],
      };
      const log = new StreamMultiChannelLog(c, props);
      await waitForResolve(log);

      const entries = log.value();
      expect(entries).toHaveLength(2);
      expect(entries[0].value).toBe("10");
      expect(entries[1].value).toBe("20");
    });
  });

  describe("indexed channels", () => {
    const NOW = TimeStamp.seconds(100);
    const IDX = 10;
    const DATA = 11;
    const IDX_2 = 12;
    const DATA_2 = 13;

    const times = (seconds: number[], alignment: bigint): Series =>
      new Series({
        data: seconds.map((s) => TimeStamp.seconds(s).valueOf()),
        dataType: DataType.TIMESTAMP,
        alignment,
      });

    const strings = (values: string[], alignment: bigint): Series =>
      new Series({ data: values, dataType: DataType.STRING, alignment });

    const create = (
      props: Partial<StreamMultiChannelLogProps> = {},
    ): { log: StreamMultiChannelLog; statuses: cstatus.Crude[] } => {
      const statuses: cstatus.Crude[] = [];
      const log = new StreamMultiChannelLog(
        c,
        {
          channels: [DATA],
          ...props,
        },
        { onStatusChange: (s) => statuses.push(s) },
        () => NOW,
      );
      return { log, statuses };
    };

    const seconds = (log: StreamMultiChannelLog): number[] =>
      log.value().map((e) => e.timestamp.seconds);

    beforeEach(() => {
      c.indexed = [
        new channel.Channel({
          key: IDX,
          name: "time",
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        }),
        new channel.Channel({
          key: DATA,
          name: "messages",
          dataType: DataType.STRING,
          index: IDX,
        }),
        new channel.Channel({
          key: IDX_2,
          name: "time_2",
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        }),
        new channel.Channel({
          key: DATA_2,
          name: "messages_2",
          dataType: DataType.STRING,
          index: IDX_2,
        }),
      ];
    });

    it("should stream the index of each data channel", async () => {
      const { log } = create();
      await waitForResolve(log);
      expect(c.streamKeys).toEqual([DATA, IDX]);
    });

    it("should stamp streamed entries with their sample times", async () => {
      const { log } = create();
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([90, 91], 0n)])],
          [DATA, new MultiSeries([strings(["a", "b"], 0n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["a", "b"]);
      expect(seconds(log)).toEqual([90, 91]);
    });

    it("should hold a streamed sample until its index sample arrives", async () => {
      const { log } = create();
      await waitForResolve(log);
      const index = times([90], 0n);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([index])],
          [DATA, new MultiSeries([strings(["a", "b"], 0n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["a"]);
      const next = times([91], 1n);
      c.streamHandler?.(new Map([[IDX, new MultiSeries([next])]]));
      expect(log.value().map((e) => e.value)).toEqual(["a", "b"]);
    });

    it("should keep a held sample when a new data allocation arrives", async () => {
      const { log } = create();
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([90], 0n)])],
          [DATA, new MultiSeries([strings(["a", "b"], 0n)])],
        ]),
      );
      c.streamHandler?.(new Map([[DATA, new MultiSeries([strings(["c"], 2n)])]]));
      expect(log.value().map((e) => e.value)).toEqual(["a"]);
      c.streamHandler?.(new Map([[IDX, new MultiSeries([times([91, 92], 1n)])]]));
      expect(log.value().map((e) => e.value)).toEqual(["a", "b", "c"]);
    });

    it("should skip replayed samples whose index samples are gone", async () => {
      const { log } = create();
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([92, 93], 2n)])],
          [DATA, new MultiSeries([strings(["a", "b", "c", "d"], 0n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["c", "d"]);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([94], 4n)])],
          [DATA, new MultiSeries([strings(["e"], 4n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["c", "d", "e"]);
    });

    it("should time every sample when one callback carries many index buffers", async () => {
      const { log } = create();
      await waitForResolve(log);
      const values = ["a", "b", "c", "d", "e", "f"];
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries(values.map((_, i) => times([90 + i], BigInt(i))))],
          [DATA, new MultiSeries([strings(values, 0n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(values);
    });

    it("should keep samples 1 ns apart", async () => {
      const { log } = create();
      await waitForResolve(log);
      const start = TimeStamp.seconds(90).valueOf();
      const index = new Series({
        data: [start, start + 1_000_000_000n, start + 1_000_000_001n],
        dataType: DataType.TIMESTAMP,
        alignment: 0n,
      });
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([index])],
          [DATA, new MultiSeries([strings(["a", "b", "c"], 0n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["a", "b", "c"]);
    });

    it("should report dropping held samples past the entry limit", async () => {
      const statuses: cstatus.Crude[] = [];
      const log = new StreamMultiChannelLog(
        c,
        { channels: [DATA] },
        { onStatusChange: (st) => statuses.push(st) },
        () => NOW,
        2,
      );
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([[DATA, new MultiSeries([strings(["a", "b", "c"], 0n)])]]),
      );
      c.streamHandler?.(new Map([[DATA, new MultiSeries([strings(["d"], 3n)])]]));
      expect(statuses.map((st) => st.message)).toEqual([
        "Dropped log samples with no time",
      ]);
      c.streamHandler?.(new Map([[IDX, new MultiSeries([times([92, 93], 2n)])]]));
      expect(log.value().map((e) => e.value)).toEqual(["c", "d"]);
    });

    it("should not repeat samples a restarted stream replays", async () => {
      const { log } = create();
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([80, 81], 0n)])],
          [DATA, new MultiSeries([strings(["x", "y"], 0n)])],
        ]),
      );
      log.setChannels([DATA, DATA_2]);
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([80, 81, 82], 0n)])],
          [DATA, new MultiSeries([strings(["x", "y", "z"], 0n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["x", "y", "z"]);
    });

    it("should sort the entries of several channels by sample time", async () => {
      const { log } = create({ channels: [DATA, DATA_2] });
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([93], 0n)])],
          [DATA, new MultiSeries([strings(["late"], 0n)])],
        ]),
      );
      c.streamHandler?.(
        new Map([
          [IDX_2, new MultiSeries([times([92], 5n)])],
          [DATA_2, new MultiSeries([strings(["early"], 5n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["early", "late"]);
    });

    it("should keep the lines of a multi-line sample together when sorting", async () => {
      const { log } = create({ channels: [DATA, DATA_2] });
      await waitForResolve(log);
      c.streamHandler?.(
        new Map([
          [IDX, new MultiSeries([times([92, 94], 0n)])],
          [DATA, new MultiSeries([strings(["a\nb", "d"], 0n)])],
        ]),
      );
      c.streamHandler?.(
        new Map([
          [IDX_2, new MultiSeries([times([93], 5n)])],
          [DATA_2, new MultiSeries([strings(["c"], 5n)])],
        ]),
      );
      expect(log.value().map((e) => e.value)).toEqual(["a", "b", "c", "d"]);
    });
  });

  describe("disconnected", () => {
    it("should report a disconnected status once instead of throwing", () => {
      const statuses: cstatus.Crude[] = [];
      const log = new StreamMultiChannelLog(
        null,
        { channels: [1] },
        { onStatusChange: (s) => statuses.push(s) },
      );
      expect(log.value()).toHaveLength(0);
      expect(statuses).toHaveLength(1);
      expect(statuses[0].variant).toEqual("warning");
      expect(log.value()).toHaveLength(0);
      expect(statuses).toHaveLength(1);
    });
  });
});

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType, TimeRange } from "@synnaxlabs/client";
import {
  type AsyncDestructor,
  bounds,
  id,
  MultiSeries,
  Series,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  ChannelData,
  type ChannelDataProps,
  StreamChannelData,
  type StreamChannelDataProps,
  StreamChannelValue,
  type StreamChannelValueProps,
} from "@/telem/aether/remote";
import { type Source } from "@/telem/aether/telem";
import { type client } from "@/telem/client";

const waitForResolve = async <T>(source: Source<T>): Promise<T> => {
  source.value();
  const handleChange = vi.fn();
  source.onChange(handleChange);
  await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
  return source.value();
};

describe("remote", () => {
  describe("StreamChannelValue", () => {
    class MockClient implements client.Client {
      key: string = id.create();

      // Stream
      streamHandler: client.StreamHandler | null = null;
      streamKeys: channel.Keys = [];
      streamF = vi.fn();
      streamDestructorF = vi.fn();

      // Channel
      channel: channel.Channel = new channel.Channel({
        key: 65537,
        name: "test",
        dataType: DataType.FLOAT32,
        isIndex: false,
      });

      // Data
      response: MultiSeries = new MultiSeries([]);

      async retrieveChannel(): Promise<channel.Channel> {
        return this.channel;
      }

      async read(): Promise<MultiSeries> {
        return this.response;
      }

      async stream(
        handler: client.StreamHandler,
        keys: channel.Keys,
      ): Promise<AsyncDestructor> {
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

    it("should return a NaN value when no channel has been set", async () => {
      const props: StreamChannelValueProps = {
        channel: 0,
      };
      const scv = new StreamChannelValue(c, props);
      expect(scv.value()).toBe(NaN);
      expect(scv.testingOnlyValid).toBe(false);
    });

    it("should return a NaN value when no leading buffer has been set", async () => {
      const props: StreamChannelValueProps = {
        channel: 0,
      };
      const scv = new StreamChannelValue(c, props);
      expect(scv.value()).toBe(NaN);
      expect(scv.testingOnlyLeadingBuffer).toBeNull();
    });

    it("should open the stream handler when the channel is not zero", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      await waitForResolve(scv);
      expect(c.streamHandler).not.toBeNull();
      expect(c.streamF).toHaveBeenCalled();
      expect(c.streamF).toHaveBeenCalledWith(c.streamHandler, [c.channel.key]);
    });

    it("should destroy the stream handler when cleanup is called", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      await waitForResolve(scv);
      scv.cleanup();
      expect(c.streamDestructorF).toHaveBeenCalled();
    });

    it("should set the leading buffer when onChange is called", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      scv.value();
      await expect.poll(() => handleChange.mock.calls.length === 1).toBe(true);
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      expect(scv.testingOnlyLeadingBuffer).toBeNull();
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length === 2).toBe(true);
      expect(scv.testingOnlyLeadingBuffer).toBe(series);
      expect(scv.value()).toBe(3);
    });

    it("should return the correct value when the leading buffer is appended to", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      expect(scv.value()).toBe(NaN);
      await expect.poll(() => handleChange.mock.calls.length === 1).toBe(true);
      const series = Series.alloc({ dataType: DataType.FLOAT32, capacity: 3 });

      // Call onChange to set the leading buffer
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length === 2).toBe(true);
      // Append to the leading buffer
      series.write(new Series({ data: new Float32Array([1, 2, 5]) }));
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([])]]));
      await expect.poll(() => handleChange.mock.calls.length === 3).toBe(true);
      const v = scv.value();
      expect(v).toBe(5);
    });

    it("should replace the leading buffer when a new one is passed through the streamer", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      scv.value();
      await expect.poll(() => handleChange).toHaveBeenCalledTimes(1);
      const newSeriesOne = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      const newSeriesTwo = new Series({
        data: new Float32Array([4, 5, 6]),
      });
      // Call onChange to set the leading buffer
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([newSeriesOne])]]));
      await expect.poll(() => handleChange).toHaveBeenCalledTimes(2);
      // It should increment the reference count of the buffer
      expect(newSeriesOne.refCount).toBe(1);
      expect(scv.value()).toBe(3);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([newSeriesTwo])]]));
      expect(newSeriesOne.refCount).toBe(0);
      await expect.poll(() => handleChange.mock.calls.length === 3).toBe(true);
      expect(scv.value()).toBe(6);
      expect(newSeriesTwo.refCount).toBe(1);
    });
  });

  describe("ChannelData", () => {
    class MockClient implements client.ReadClient, client.ChannelClient {
      key: string = id.create();
      readMock = vi.fn();
      retrieveChannelMock = vi.fn();

      // Channel
      channel: channel.Channel = new channel.Channel({
        key: 65537,
        name: "test",
        dataType: DataType.FLOAT32,
        isIndex: false,
        index: 65538,
      });

      indexChannel: channel.Channel = new channel.Channel({
        key: 65538,
        name: "test",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      // Data
      response: Record<channel.Key, MultiSeries> = {
        [this.channel.key]: new MultiSeries([]),
        [this.channel.index]: new MultiSeries([]),
      };

      async retrieveChannel(key: channel.KeyOrName): Promise<channel.Channel> {
        this.retrieveChannelMock(key);
        if (key === this.channel.key) return this.channel;
        if (key === this.channel.index) return this.indexChannel;
        throw new Error(`Channel with key ${key} not found`);
      }

      async read(tr: TimeRange, key: channel.Key): Promise<MultiSeries> {
        this.readMock(tr, key);
        return this.response[key];
      }

      close(): void {}
    }

    let c: MockClient;
    beforeEach(() => {
      c = new MockClient();
    });

    it("should return a zero value when no channel has been set", async () => {
      const props = {
        timeRange: TimeRange.MAX,
        channel: 0,
      };
      const cd = new ChannelData(c, props);
      const handleChange = vi.fn();
      cd.onChange(handleChange);
      const [b, data] = cd.value();
      expect(handleChange.mock.calls.length).toBe(0);
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data).toHaveLength(0);
      expect(c.readMock).not.toHaveBeenCalled();
      expect(c.retrieveChannelMock).not.toHaveBeenCalled();
    });

    it("should return a zero value when the time range is empty", async () => {
      const props = {
        timeRange: TimeRange.ZERO,
        channel: c.channel.key,
      };
      const cd = new ChannelData(c, props);
      const handleChange = vi.fn();
      const [b, data] = cd.value();
      expect(handleChange.mock.calls.length).toBe(0);
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data).toHaveLength(0);
      expect(c.readMock).not.toHaveBeenCalled();
      expect(c.retrieveChannelMock).not.toHaveBeenCalled();
    });

    it("should return data when both the channel and time range are set", async () => {
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      c.response = {
        [c.channel.key]: new MultiSeries([series]),
      };
      const props = {
        timeRange: TimeRange.MAX,
        channel: c.channel.key,
      };
      const cd = new ChannelData(c, props);
      const [b, data] = await waitForResolve(cd);
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
    });

    it("should fetch data from the index channel when the channel is not an index and fetchIndex is true", async () => {
      const series = new Series({
        data: new Float32Array([0, 2, 4]),
      });
      c.response = {
        [c.channel.index]: new MultiSeries([series]),
      };
      const props: ChannelDataProps = {
        timeRange: TimeRange.MAX,
        channel: c.channel.key,
        useIndexOfChannel: true,
      };
      const cd = new ChannelData(c, props);
      const [b, data] = await waitForResolve(cd);
      expect(b).toStrictEqual({ lower: 0, upper: 4 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
    });

    it("should fetch data from the same channel when the channel is an index and fetchIndex is true", async () => {
      const series = new Series({
        data: new Float32Array([0, 2, 4]),
      });
      c.response = {
        [c.channel.index]: new MultiSeries([series]),
      };
      const props: ChannelDataProps = {
        timeRange: TimeRange.MAX,
        channel: c.channel.index,
        useIndexOfChannel: true,
      };
      const cd = new ChannelData(c, props);
      const [b, data] = await waitForResolve(cd);
      expect(b).toStrictEqual({ lower: 0, upper: 4 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
    });
  });

  describe("StreamChannelData", () => {
    class MockClient implements client.Client {
      key: string = id.create();

      // Stream
      streamHandler: client.StreamHandler | null = null;
      streamKeys: channel.Keys = [];
      streamF = vi.fn();
      streamDestructorF = vi.fn();

      // Read
      response: MultiSeries = new MultiSeries([]);
      readMock = vi.fn();

      // Channel
      channel: channel.Channel = new channel.Channel({
        key: 65537,
        name: "test",
        dataType: DataType.FLOAT32,
        isIndex: false,
        index: 65538,
      });

      indexChannel: channel.Channel = new channel.Channel({
        key: 65538,
        name: "test",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      async retrieveChannel(key: channel.KeyOrName): Promise<channel.Channel> {
        if (key === this.channel.key) return this.channel;
        if (key === this.channel.index) return this.indexChannel;
        throw new Error(`Channel with key ${key} not found`);
      }

      async read(tr: TimeRange, key: channel.Key): Promise<MultiSeries> {
        this.readMock(tr, key);
        return this.response;
      }

      async stream(
        handler: client.StreamHandler,
        keys: channel.Keys,
      ): Promise<AsyncDestructor> {
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

    it("should return a zero value when no channel has been set", async () => {
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: 0,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = cd.value();
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data).toHaveLength(0);
    });

    it("should return data when the channel is specified", async () => {
      const now = TimeStamp.now();
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(
          now.sub(TimeSpan.milliseconds(3)),
          now.add(TimeSpan.milliseconds(1)),
        ),
      });
      c.response = new MultiSeries([series]);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = await waitForResolve(cd);
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
    });

    it("should bind a stream handler", async () => {
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      await waitForResolve(cd);
      expect(c.streamHandler).not.toBeNull();
      expect(c.streamF).toHaveBeenCalled();
      expect(c.streamF).toHaveBeenCalledWith(c.streamHandler, [c.channel.key]);
    });

    it("should garbage collect data that goes out of range", async () => {
      let now = TimeStamp.milliseconds(10);
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = new MultiSeries([series]);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.milliseconds(2),
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props, {}, () => now);
      const [b, data] = await waitForResolve(cd);
      expect(b).toEqual({ lower: 1, upper: 3 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
      const tr2 = new TimeRange(
        now.add(TimeSpan.milliseconds(1)),
        now.add(TimeSpan.milliseconds(20)),
      );
      // write the new series
      const series2 = new Series({
        data: new Float32Array([4, 5, 6]),
        timeRange: tr2,
      });
      now = TimeStamp.milliseconds(30);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series2])]]));
      expect(series.refCount).toBe(0);
      expect(series2.refCount).toBe(1);
      const [b2, data2] = cd.value();
      expect(b2).toEqual({ lower: 4, upper: 6 });
      expect(data2.series).toHaveLength(1);
      expect(data2.series[0]).toBe(series2);
    });

    it("should adjust the bounds of the data even if it was not garbage collected", async () => {
      let now = TimeStamp.milliseconds(10);
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = new MultiSeries([series]);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.milliseconds(2),
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props, {}, () => now);
      const [b, data] = await waitForResolve(cd);
      expect(b).toEqual({ lower: 1, upper: 3 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
      const tr2 = new TimeRange(
        now.add(TimeSpan.milliseconds(1)),
        now.add(TimeSpan.milliseconds(20)),
      );
      expect(series.refCount).toBe(1);
      // write the new series
      const series2 = new Series({
        data: new Float32Array([4, 5, 6]),
        timeRange: tr2,
      });
      // The old buffer won't be garbage collected yet
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series2])]]));
      now = TimeStamp.milliseconds(20);
      const [b2, data2] = cd.value();
      expect(series2.refCount).toBe(1);
      expect(series.refCount).toBe(1);
      expect(b2).toEqual({ lower: 4, upper: 6 });
      expect(data2.series).toHaveLength(2);
      expect(data2.series[0]).toBe(series);
      expect(data2.series[1]).toBe(series2);
    });

    it("should destroy the stream handler when cleanup is called", async () => {
      const now = TimeStamp.milliseconds(10);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props, {}, () => now);
      await waitForResolve(cd);
      cd.cleanup();
      expect(c.streamDestructorF).toHaveBeenCalled();
    });

    it("should drop the series refcounts to 0 when cleanup is called", async () => {
      const now = TimeStamp.milliseconds(10);
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = new MultiSeries([series]);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.milliseconds(2),
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props, {}, () => now);
      await waitForResolve(cd);
      expect(series.refCount).toBe(1);
      cd.cleanup();
      expect(series.refCount).toBe(0);
    });

    it("should return the index channel data when the channel is not an index and fetchIndex is true", async () => {
      const now = TimeStamp.milliseconds(10);
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = new MultiSeries([series]);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
        useIndexOfChannel: true,
      };
      const cd = new StreamChannelData(c, props, {}, () => now);
      const [b, data] = await waitForResolve(cd);
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
    });

    it("should return the index channel data when the channel is an index and fetchIndex is true", async () => {
      const now = TimeStamp.milliseconds(10);
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = new MultiSeries([series]);
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.index,
        useIndexOfChannel: true,
      };
      const cd = new StreamChannelData(c, props, {}, () => now);
      const [b, data] = await waitForResolve(cd);
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(series);
    });

    it("should read data when the channel is not virtual", async () => {
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.seconds(20),
        channel: c.channel.key,
      };
      const now = TimeStamp.milliseconds(10);
      const cd = new StreamChannelData(c, props, undefined, () => now);
      await waitForResolve(cd);
      expect(c.readMock).toHaveBeenCalled();
      const args = c.readMock.mock.calls[0];
      expect(args).toHaveLength(2);
      const expectedTr = new TimeRange(now.spanRange(-TimeSpan.seconds(20)));
      expect(args[0].equals(expectedTr)).toBe(true);
      expect(args[1]).toBe(c.channel.key);
    });

    it("should not read data when the channel is virtual", async () => {
      c.channel = new channel.Channel({
        ...c.channel,
        virtual: true,
      });
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.seconds(20),
        channel: c.channel.key,
      };
      const now = TimeStamp.milliseconds(10);
      const cd = new StreamChannelData(c, props, undefined, () => now);
      await waitForResolve(cd);
      expect(c.readMock).not.toHaveBeenCalled();
    });

    it("should read data when the channel is calculated", async () => {
      c.channel = new channel.Channel({
        ...c.channel,
        expression: "1 + 2",
      });
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.seconds(1),
        channel: c.channel.key,
      };
      const now = TimeStamp.milliseconds(10);
      const cd = new StreamChannelData(c, props, undefined, () => now);
      await waitForResolve(cd);
      expect(c.readMock).toHaveBeenCalled();
      const args = c.readMock.mock.calls[0];
      expect(args).toHaveLength(2);
      const expectedTr = new TimeRange(now.spanRange(-TimeSpan.seconds(1)));
      expect(args[0].equals(expectedTr)).toBe(true);
      expect(args[1]).toBe(c.channel.key);
    });

    it("should return zero bounds for a channel with a variable length data type", async () => {
      c.channel = new channel.Channel({
        ...c.channel,
        virtual: true,
        dataType: DataType.STRING,
      });
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      await waitForResolve(cd);
      const d = new Series({
        data: ["cat", "in", "the", "hat"],
        timeRange: TimeRange.MAX,
      });
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([d])]]));
      const [b, data] = cd.value();
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data.series).toHaveLength(1);
      expect(data.series[0]).toBe(d);
    });
  });
});

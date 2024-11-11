// Copyright 2024 Synnax Labs, Inc.
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
  Series,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import {
  ChannelData,
  type ChannelDataProps,
  StreamChannelData,
  type StreamChannelDataProps,
  StreamChannelValue,
  type StreamChannelValueProps,
} from "@/telem/aether/remote";
import { client } from "@/telem/client";

describe("remote", () => {
  describe("StreamChannelValue", () => {
    class MockClient implements client.Client {
      key: string = id.id();

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
      response: Record<channel.Key, client.ReadResponse> = {
        [this.channel.key]: new client.ReadResponse(this.channel, []),
      };

      async retrieveChannel(): Promise<channel.Channel> {
        return this.channel;
      }

      async read(): Promise<Record<channel.Key, client.ReadResponse>> {
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

    it("should return a zero value when no channel has been set", async () => {
      const client = new MockClient();
      const props: StreamChannelValueProps = {
        channel: 0,
      };
      const scv = new StreamChannelValue(client, props);
      expect(await scv.value()).toBe(0);
      expect(scv.testingOnlyValid).toBe(false);
    });
    it("should return a zero value when no leading buffer has been set", async () => {
      const client = new MockClient();
      const props: StreamChannelValueProps = {
        channel: 0,
      };
      const scv = new StreamChannelValue(client, props);
      expect(await scv.value()).toBe(0);
      expect(scv.testingOnlyValid).toBe(false);
      expect(scv.testingOnlyLeadingBuffer).toBeNull();
    });
    it("should open the stream handler when the channel is not zero", async () => {
      const c = new MockClient();
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      await scv.value();
      expect(c.streamHandler).not.toBeNull();
      expect(c.streamF).toHaveBeenCalled();
      expect(c.streamF).toHaveBeenCalledWith(c.streamHandler, [c.channel.key]);
    });
    it("should destroy the stream handler when cleanup is called", async () => {
      const c = new MockClient();
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      await scv.value();
      await scv.cleanup();
      expect(c.streamDestructorF).toHaveBeenCalled();
    });
    it("should set the leading buffer when onChange is called", async () => {
      const c = new MockClient();
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      expect(await scv.value()).toBe(0);
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      expect(scv.testingOnlyLeadingBuffer).toBeNull();
      c.streamHandler?.({
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      });
      expect(scv.testingOnlyLeadingBuffer).toBe(series);
      expect(await scv.value()).toBe(3);
    });
    it("should return the correct value when the leading buffer is appended to", async () => {
      const c = new MockClient();
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      expect(await scv.value()).toBe(0);
      const series = Series.alloc({
        dataType: DataType.FLOAT32,
        capacity: 3,
      });
      // Call onChange to set the leading buffer
      c.streamHandler?.({
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      });
      // Append to the leading buffer
      series.write(new Series({ data: new Float32Array([1, 2, 5]) }));
      expect(await scv.value()).toBe(5);
    });
    it("should replace the leading buffer when a new one is passed through the streamer", async () => {
      const c = new MockClient();
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      expect(await scv.value()).toBe(0);
      const newSeriesOne = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      const newSeriesTwo = new Series({
        data: new Float32Array([4, 5, 6]),
      });
      // Call onChange to set the leading buffer
      c.streamHandler?.({
        [c.channel.key]: new client.ReadResponse(c.channel, [newSeriesOne]),
      });
      // It should increment the reference count of the buffer
      expect(newSeriesOne.refCount).toBe(1);
      expect(await scv.value()).toBe(3);
      c.streamHandler?.({
        [c.channel.key]: new client.ReadResponse(c.channel, [newSeriesTwo]),
      });
      expect(newSeriesOne.refCount).toBe(0);
      expect(await scv.value()).toBe(6);
      expect(newSeriesTwo.refCount).toBe(1);
    });
  });
  describe("ChannelData", () => {
    class MockClient implements client.ReadClient, client.ChannelClient {
      key: string = id.id();
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
      response: Record<channel.Key, client.ReadResponse> = {
        [this.channel.key]: new client.ReadResponse(this.channel, []),
        [this.channel.index]: new client.ReadResponse(this.indexChannel, []),
      };

      async retrieveChannel(key: channel.KeyOrName): Promise<channel.Channel> {
        this.retrieveChannelMock(key);
        if (key === this.channel.key) 
          return this.channel;
        
        if (key === this.channel.index) 
          return this.indexChannel;
        
        throw new Error(`Channel with key ${key} not found`);
      }

      async read(
        tr: TimeRange,
        key: channel.Keys,
      ): Promise<Record<channel.Key, client.ReadResponse>> {
        this.readMock(tr, key);
        return this.response;
      }

      close(): void {}
    }

    it("should return a zero value when no channel has been set", async () => {
      const client = new MockClient();
      const props = {
        timeRange: TimeRange.MAX,
        channel: 0,
      };
      const cd = new ChannelData(client, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data).toHaveLength(0);
      expect(client.readMock).not.toHaveBeenCalled();
      expect(client.retrieveChannelMock).not.toHaveBeenCalled();
    });

    it("should return a zero value when the time range is empty", async () => {
      const client = new MockClient();
      const props = {
        timeRange: TimeRange.ZERO,
        channel: client.channel.key,
      };
      const cd = new ChannelData(client, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data).toHaveLength(0);
      expect(client.readMock).not.toHaveBeenCalled();
      expect(client.retrieveChannelMock).not.toHaveBeenCalled();
    });

    it("should return data when both the channel and time range are set", async () => {
      const c = new MockClient();
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      };
      const props = {
        timeRange: TimeRange.MAX,
        channel: c.channel.key,
      };
      const cd = new ChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data).toHaveLength(1);
      expect(data[0]).toBe(series);
    });

    it("should fetch data from the index channel when the channel is not an index and fetchIndex is true", async () => {
      const c = new MockClient();
      const series = new Series({
        data: new Float32Array([0, 2, 4]),
      });
      c.response = {
        [c.channel.index]: new client.ReadResponse(c.indexChannel, [series]),
      };
      const props: ChannelDataProps = {
        timeRange: TimeRange.MAX,
        channel: c.channel.key,
        useIndexOfChannel: true,
      };
      const cd = new ChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual({ lower: 0, upper: 4 });
      expect(data).toHaveLength(1);
      expect(data[0]).toBe(series);
    });

    it("should fetch data from the same channel when the channel is an index and fetchIndex is true", async () => {
      const c = new MockClient();
      const series = new Series({
        data: new Float32Array([0, 2, 4]),
      });
      c.response = {
        [c.channel.index]: new client.ReadResponse(c.indexChannel, [series]),
      };
      const props: ChannelDataProps = {
        timeRange: TimeRange.MAX,
        channel: c.channel.index,
        useIndexOfChannel: true,
      };
      const cd = new ChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual({ lower: 0, upper: 4 });
      expect(data).toHaveLength(1);
      expect(data[0]).toBe(series);
    });
  });
  describe("StreamChannelData", () => {
    class MockClient implements client.Client {
      key: string = id.id();

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
        index: 65538,
      });

      indexChannel: channel.Channel = new channel.Channel({
        key: 65538,
        name: "test",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      // Data
      response: Record<channel.Key, client.ReadResponse> = {
        [this.channel.key]: new client.ReadResponse(this.channel, []),
      };

      async retrieveChannel(key: channel.KeyOrName): Promise<channel.Channel> {
        if (key === this.channel.key) 
          return this.channel;
        
        if (key === this.channel.index) 
          return this.indexChannel;
        
        throw new Error(`Channel with key ${key} not found`);
      }

      async read(): Promise<Record<channel.Key, client.ReadResponse>> {
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

    it("should return a zero value when no channel has been set", async () => {
      const client = new MockClient();
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: 0,
      };
      const cd = new StreamChannelData(client, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual(bounds.ZERO);
      expect(data).toHaveLength(0);
    });

    it("should return data when the channel is specified", async () => {
      const c = new MockClient();
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: new TimeRange(
          TimeStamp.now().sub(TimeSpan.seconds(3)),
          TimeStamp.now().add(TimeSpan.seconds(1)),
        ),
      });
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      };
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data).toHaveLength(1);
      expect(data[0]).toBe(series);
    });

    it("should bind a stream handler", async () => {
      const c = new MockClient();
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      await cd.value();
      expect(c.streamHandler).not.toBeNull();
      expect(c.streamF).toHaveBeenCalled();
      expect(c.streamF).toHaveBeenCalledWith(c.streamHandler, [c.channel.key]);
    });

    it("should garbage collect data that goes out of range", async () => {
      const c = new MockClient();
      const now = TimeStamp.now();
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      };
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.milliseconds(2),
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toEqual({ lower: 1, upper: 3 });
      expect(data).toHaveLength(1);
      const tr2 = new TimeRange(
        now.add(TimeSpan.milliseconds(1)),
        now.add(TimeSpan.milliseconds(20)),
      );
      // write the new series
      const series2 = new Series({
        data: new Float32Array([4, 5, 6]),
        timeRange: tr2,
      });
      // wait for 2 milliseconds
      await new Promise((resolve) => setTimeout(resolve, 10));
      c.streamHandler?.({
        [c.channel.key]: new client.ReadResponse(c.channel, [series2]),
      });
      expect(series.refCount).toBe(0);
      expect(series2.refCount).toBe(1);
      const [b2, data2] = await cd.value();
      expect(b2).toEqual({ lower: 4, upper: 6 });
      expect(data2).toHaveLength(1);
      expect(data2[0]).toBe(series2);
    });

    it("should adjust the bounds of the data even if it was not garbage collected", async () => {
      const c = new MockClient();
      const now = TimeStamp.now();
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      };
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.milliseconds(2),
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toEqual({ lower: 1, upper: 3 });
      expect(data).toHaveLength(1);
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
      // Notice how in this case we call the stream handler before we wait, this means
      // that the old buffer will not be garbage collected
      c.streamHandler?.({
        [c.channel.key]: new client.ReadResponse(c.channel, [series2]),
      });
      await new Promise((resolve) => setTimeout(resolve, 10));
      const [b2, data2] = await cd.value();
      expect(series2.refCount).toBe(1);
      expect(series.refCount).toBe(1);
      expect(b2).toEqual({ lower: 4, upper: 6 });
      expect(data2).toHaveLength(2);
      expect(data2[0]).toBe(series);
      expect(data2[1]).toBe(series2);
    });

    it("should destroy the stream handler when cleanup is called", async () => {
      const c = new MockClient();
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      await cd.value();
      await cd.cleanup();
      expect(c.streamDestructorF).toHaveBeenCalled();
    });

    it("should drop the series refcounts to 0 when cleanup is called", async () => {
      const c = new MockClient();
      const now = TimeStamp.now();
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [series]),
      };
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.milliseconds(2),
        channel: c.channel.key,
      };
      const cd = new StreamChannelData(c, props);
      await cd.value();
      expect(series.refCount).toBe(1);
      await cd.cleanup();
      expect(series.refCount).toBe(0);
    });

    it("should return the index channel data when the channel is not an index and fetchIndex is true", async () => {
      const c = new MockClient();
      const now = TimeStamp.now();
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = {
        [c.channel.index]: new client.ReadResponse(c.indexChannel, [series]),
      };
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.key,
        useIndexOfChannel: true,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data).toHaveLength(1);
      expect(data[0]).toBe(series);
    });

    it("should return the index channel data when the channel is an index and fetchIndex is true", async () => {
      const c = new MockClient();
      const now = TimeStamp.now();
      const tr = new TimeRange(
        now.sub(TimeSpan.milliseconds(3)),
        now.add(TimeSpan.milliseconds(1)),
      );
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
        timeRange: tr,
      });
      c.response = {
        [c.channel.index]: new client.ReadResponse(c.indexChannel, [series]),
      };
      const props: StreamChannelDataProps = {
        timeSpan: TimeSpan.MAX,
        channel: c.channel.index,
        useIndexOfChannel: true,
      };
      const cd = new StreamChannelData(c, props);
      const [b, data] = await cd.value();
      expect(b).toStrictEqual({ lower: 1, upper: 3 });
      expect(data).toHaveLength(1);
      expect(data[0]).toBe(series);
    });
  });
});

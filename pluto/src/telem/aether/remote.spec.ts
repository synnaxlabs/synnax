import { DataType, type TimeRange, channel } from "@synnaxlabs/client";
import { Series, type AsyncDestructor } from "@synnaxlabs/x";
import { nanoid } from "nanoid";
import { describe, expect, it, vi } from "vitest";

import {
  StreamChannelValue,
  type StreamChannelValueProps,
} from "@/telem/aether/remote";
import { client } from "@/telem/client";

describe("remote", () => {
  describe("StreamChannelValue", () => {
    class MockClient implements client.Client {
      key: string = nanoid();

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

      async retrieveChannel(key: channel.Key): Promise<channel.Channel> {
        return this.channel;
      }

      async read(
        tr: TimeRange,
        key: channel.Keys,
      ): Promise<Record<channel.Key, client.ReadResponse>> {
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

      close(): void {}
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
    it("should return a zero value when the leading buffer is empty", async () => {
      const c = new MockClient();
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [
          new Series({
            data: new Float32Array([]),
          }),
        ]),
      };
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      expect(await scv.value()).toBe(0);
      expect(scv.testingOnlyValid).toBe(true);
      expect(scv.testingOnlyLeadingBuffer).not.toBeNull();
    });
    it("should return the last value of the leading buffer when the initial read is successful", async () => {
      const c = new MockClient();
      const s = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      c.response = {
        [c.channel.key]: new client.ReadResponse(c.channel, [s]),
      };
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      expect(await scv.value()).toBe(3);
      expect(s.refCount).toBe(1);
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
});

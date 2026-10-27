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
  framer,
  NotFoundError,
  type status as cstatus,
  TimeRange,
  ValidationError,
} from "@synnaxlabs/client";
import { SyntheticCore } from "@synnaxlabs/client/testutil";
import {
  bounds,
  type breaker,
  dynamicWindow,
  id,
  MultiSeries,
  Series,
  sleep,
  staticWindow,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import {
  afterEach,
  assert,
  beforeEach,
  describe,
  expect,
  it,
  type Mock,
  vi,
} from "vitest";

import { createFactory, type CreateOptions } from "@/telem/aether/factory";
import {
  type Client,
  DISCONNECTED_STATUS,
  StreamChannelStringValue,
  StreamChannelValue,
  streamChannelValue,
  type StreamChannelValueProps,
  TiledChannelData,
  type TiledChannelDataProps,
} from "@/telem/aether/remote";
import { type Source, type ValueProps } from "@/telem/aether/telem";
import { telemTest } from "@/telem/aether/test";
import { line } from "@/vis/line/aether";

const waitForResolve = async <T>(source: Source<T>): Promise<T> => {
  source.value();
  const handleChange = vi.fn();
  source.onChange(handleChange);
  await expect.poll(() => handleChange.mock.calls.length > 0).toBe(true);
  return source.value().data;
};

const loading = (source: Source<unknown>, props?: ValueProps): boolean =>
  source.value(props).status.variant === "loading";

// A value source with nothing to report does not notify when its stream opens, so a
// spec that only needs the read to finish waits on the stream registration. The count
// is captured up front so a re-open after cleanup is not read as the first one.
const waitForStream = async <T>(
  source: Source<T>,
  client: { streamF: Mock },
): Promise<T> => {
  const opens = client.streamF.mock.calls.length;
  source.value();
  await expect.poll(() => client.streamF.mock.calls.length > opens).toBe(true);
  return source.value().data;
};

describe("remote", () => {
  describe("StreamChannelValue", () => {
    class MockClient implements Client {
      now = (): TimeStamp => TimeStamp.now();
      key: string = id.create();

      streamHandler: framer.StreamHandler | null = null;
      streamKeys: channel.Key[] = [];
      streamF = vi.fn();
      streamDestructorF = vi.fn();

      channel: channel.Channel = new channel.Channel({
        key: 65537,
        name: "test",
        dataType: DataType.FLOAT32,
        isIndex: false,
      });

      response: MultiSeries = new MultiSeries([]);

      channels = { retrieve: async (): Promise<channel.Channel> => this.channel };

      feed = {
        readLatest: async (): Promise<MultiSeries> => new MultiSeries([]),
        readTile: async (): Promise<MultiSeries> => new MultiSeries([]),
        read: async (): Promise<MultiSeries> => this.response,
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
      expect(scv.value().data).toBe(NaN);
      expect(scv.testingOnlyValid).toBe(false);
    });

    it("should return a NaN value when no leading buffer has been set", async () => {
      const props: StreamChannelValueProps = {
        channel: 0,
      };
      const scv = new StreamChannelValue(c, props);
      expect(scv.value().data).toBe(NaN);
      expect(scv.testingOnlyLeadingBuffer).toBeNull();
    });

    it("should open the stream handler when the channel is not zero", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      await waitForStream(scv, c);
      expect(c.streamHandler).not.toBeNull();
      expect(c.streamF).toHaveBeenCalled();
      expect(c.streamF).toHaveBeenCalledWith(c.streamHandler, [c.channel.key]);
    });

    // A consumer that counts arrivals reads the open as a sample, and then reports the
    // channel stale one timeout later while it waits for real data.
    it("should not notify when the stream opens with no data", async () => {
      const scv = new StreamChannelValue(c, { channel: c.channel.key });
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      await waitForStream(scv, c);
      expect(handleChange).not.toHaveBeenCalled();
      c.streamHandler?.(
        new Map([
          [
            c.channel.key,
            new MultiSeries([new Series({ data: new Float32Array([1]) })]),
          ],
        ]),
      );
      await expect.poll(() => handleChange.mock.calls.length).toBe(1);
    });

    it("should destroy the stream handler when cleanup is called", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      await waitForStream(scv, c);
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
      await waitForStream(scv, c);
      const series = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      expect(scv.testingOnlyLeadingBuffer).toBeNull();
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length === 1).toBe(true);
      expect(scv.testingOnlyLeadingBuffer).toBe(series);
      expect(scv.value().data).toBe(3);
    });

    it("should read an int64 sample as a number", async () => {
      const scv = new StreamChannelValue(c, { channel: c.channel.key });
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      await waitForStream(scv, c);
      const series = new Series({ data: new BigInt64Array([1n, 2n ** 40n]) });
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length).toBe(1);
      expect(scv.value().data).toBe(2 ** 40);
    });

    it("should return the correct value when the leading buffer is appended to", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      expect(scv.value().data).toBe(NaN);
      await waitForStream(scv, c);
      const series = Series.alloc({ dataType: DataType.FLOAT32, capacity: 3 });

      // Call onChange to set the leading buffer
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length === 1).toBe(true);
      // Append to the leading buffer
      series.write(new Series({ data: new Float32Array([1, 2, 5]) }));
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([])]]));
      await expect.poll(() => handleChange.mock.calls.length === 2).toBe(true);
      const v = scv.value().data;
      expect(v).toBe(5);
    });

    it("should replace the leading buffer when a new one is passed through the streamer", async () => {
      const props: StreamChannelValueProps = {
        channel: c.channel.key,
      };
      const scv = new StreamChannelValue(c, props);
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      await waitForStream(scv, c);
      const newSeriesOne = new Series({
        data: new Float32Array([1, 2, 3]),
      });
      const newSeriesTwo = new Series({
        data: new Float32Array([4, 5, 6]),
      });
      // Call onChange to set the leading buffer
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([newSeriesOne])]]));
      await expect.poll(() => handleChange).toHaveBeenCalledTimes(1);
      // It should increment the reference count of the buffer
      expect(newSeriesOne.refCount).toBe(1);
      expect(scv.value().data).toBe(3);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([newSeriesTwo])]]));
      expect(newSeriesOne.refCount).toBe(0);
      await expect.poll(() => handleChange.mock.calls.length === 2).toBe(true);
      expect(scv.value().data).toBe(6);
      expect(newSeriesTwo.refCount).toBe(1);
    });

    it.each([
      ["float64", new Series({ data: new Float64Array([0.1]) }), 0.1],
      [
        "uint32 above 2^24",
        new Series({ data: new Uint32Array([2 ** 24 + 1]) }),
        2 ** 24 + 1,
      ],
      ["boolean", new Series({ data: [false, true], dataType: DataType.BOOLEAN }), 1],
    ])("should read the exact value of a %s channel", async (_, series, value) => {
      const scv = new StreamChannelValue(c, { channel: c.channel.key });
      const handleChange = vi.fn();
      scv.onChange(handleChange);
      await waitForStream(scv, c);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange).toHaveBeenCalledTimes(1);
      expect(scv.value().data).toBe(value);
    });

    it("should not subscribe when cleaned up while retrieving the channel", async () => {
      let release = (): void => {};
      const gate = new Promise<void>((resolve) => (release = resolve));
      c.channels.retrieve = async (): Promise<channel.Channel> => {
        await gate;
        return c.channel;
      };
      const scv = new StreamChannelValue(c, { channel: c.channel.key });
      scv.value();
      scv.cleanup();
      release();
      await new Promise((resolve) => setTimeout(resolve, 5));
      expect(c.streamF).not.toHaveBeenCalled();
    });

    describe("status", () => {
      it("should report loading until the first read lands", async () => {
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        expect(loading(scv)).toBe(true);
        await expect.poll(() => scv.value().status.variant).toBe("success");
      });

      it("should report an error when the channel cannot be retrieved", async () => {
        c.channels.retrieve = async () => {
          throw new Error("retrieve exploded");
        };
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        await expect
          .poll(() => scv.value().status)
          .toMatchObject({
            variant: "error",
            message: "Failed to stream channel value",
            description: "retrieve exploded",
          });
        expect(scv.value().data).toBe(NaN);
      });

      it("should report loading again after cleanup", async () => {
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        await expect.poll(() => scv.value().status.variant).toBe("success");
        scv.cleanup();
        expect(loading(scv)).toBe(true);
      });
    });

    describe("initializing", () => {
      const SAMPLE_TIME = TimeStamp.seconds(100);
      const stored = (): Series =>
        new Series({
          data: new Float32Array([7]),
          timeRange: new TimeRange(TimeStamp.seconds(90), SAMPLE_TIME),
        });

      it("should initialize the leading buffer from the latest stored sample", async () => {
        c.feed.readLatest = vi.fn(async () => new MultiSeries([stored()]));
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        expect(await waitForResolve(scv)).toBe(7);
        expect(c.feed.readLatest).toHaveBeenCalledWith(c.channel.key);
      });

      it("should notify once when the initial sample lands", async () => {
        c.feed.readLatest = async () => new MultiSeries([stored()]);
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        const handleChange = vi.fn();
        scv.onChange(handleChange);
        await waitForResolve(scv);
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(handleChange).toHaveBeenCalledTimes(1);
      });

      it("should not notify when nothing is stored", async () => {
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        const handleChange = vi.fn();
        scv.onChange(handleChange);
        await waitForStream(scv, c);
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(handleChange).not.toHaveBeenCalled();
        expect(scv.value().data).toBe(NaN);
      });

      it("should not read the latest sample when the stream opens with data", async () => {
        c.feed.readLatest = vi.fn(async () => new MultiSeries([stored()]));
        const live = new Series({ data: new Float32Array([5]) });
        c.feed.stream = (handler, keys) => {
          handler(new Map([[c.channel.key, new MultiSeries([live])]]));
          c.streamF(handler, keys);
          return telemTest.mockSubscription(c.streamDestructorF);
        };
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        expect(await waitForResolve(scv)).toBe(5);
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(c.feed.readLatest).not.toHaveBeenCalled();
      });

      it("should keep a live buffer that arrives while the read is in flight", async () => {
        let release = (): void => {};
        const gate = new Promise<void>((resolve) => (release = resolve));
        c.feed.readLatest = async () => {
          await gate;
          return new MultiSeries([stored()]);
        };
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        const handleChange = vi.fn();
        scv.onChange(handleChange);
        await waitForStream(scv, c);
        const live = new Series({ data: new Float32Array([5]) });
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
        release();
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(scv.value().data).toBe(5);
        expect(scv.testingOnlyLeadingBuffer).toBe(live);
        expect(handleChange).toHaveBeenCalledTimes(1);
      });

      it("should replace the initial buffer when a live buffer arrives", async () => {
        c.feed.readLatest = async () => new MultiSeries([stored()]);
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        await waitForResolve(scv);
        const initial = scv.testingOnlyLeadingBuffer;
        const live = new Series({ data: new Float32Array([5]) });
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
        expect(scv.value().data).toBe(5);
        expect(initial?.refCount).toBe(0);
        expect(live.refCount).toBe(1);
      });

      it("should skip the read for a virtual channel", async () => {
        c.channel = new channel.Channel({ ...c.channel, virtual: true });
        c.feed.readLatest = vi.fn(async () => new MultiSeries([stored()]));
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        await waitForStream(scv, c);
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(c.feed.readLatest).not.toHaveBeenCalled();
        expect(scv.value().data).toBe(NaN);
      });

      it("should read the latest sample for a calculated channel", async () => {
        c.channel = new channel.Channel({
          ...c.channel,
          virtual: true,
          expression: "return 1",
        });
        c.feed.readLatest = vi.fn(async () => new MultiSeries([stored()]));
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        expect(await waitForResolve(scv)).toBe(7);
      });

      it("should release the initial buffer on cleanup", async () => {
        c.feed.readLatest = async () => new MultiSeries([stored()]);
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        await waitForResolve(scv);
        const initial = scv.testingOnlyLeadingBuffer;
        expect(initial?.refCount).toBe(1);
        scv.cleanup();
        expect(initial?.refCount).toBe(0);
        expect(scv.testingOnlyLeadingBuffer).toBeNull();
      });

      it("should not adopt the initial buffer after cleanup", async () => {
        let release = (): void => {};
        const gate = new Promise<void>((resolve) => (release = resolve));
        const initial = stored();
        c.feed.readLatest = async () => {
          await gate;
          return new MultiSeries([initial]);
        };
        const scv = new StreamChannelValue(c, { channel: c.channel.key });
        const handleChange = vi.fn();
        scv.onChange(handleChange);
        await waitForStream(scv, c);
        scv.cleanup();
        release();
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(scv.testingOnlyLeadingBuffer).toBeNull();
        expect(initial.refCount).toBe(0);
        expect(handleChange).not.toHaveBeenCalled();
      });

      it("should post a status and keep streaming when the read fails", async () => {
        const statuses: cstatus.Crude[] = [];
        c.feed.readLatest = async () => {
          throw new Error("read exploded");
        };
        const scv = new StreamChannelValue(
          c,
          { channel: c.channel.key },
          { onStatusChange: (s) => statuses.push(s) },
        );
        await waitForStream(scv, c);
        await expect.poll(() => statuses.length > 0).toBe(true);
        expect(statuses[0].variant).toEqual("error");
        expect(statuses[0].message).toEqual("Failed to read latest value");
        expect(statuses[0].description).toEqual("read exploded");
        expect(c.streamDestructorF).not.toHaveBeenCalled();
        expect(scv.testingOnlyValid).toBe(true);
        expect(scv.value().status).toMatchObject({
          variant: "error",
          message: "Failed to read latest value",
          description: "read exploded",
        });
        const live = new Series({ data: new Float32Array([5]) });
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
        expect(scv.value()).toMatchObject({ data: 5, status: { variant: "success" } });
      });

      it("should not report a failed read once live data arrives", async () => {
        const statuses: cstatus.Crude[] = [];
        let reject: (e: Error) => void = () => {};
        c.feed.readLatest = async () =>
          await new Promise<MultiSeries>((_, r) => {
            reject = r;
          });
        const scv = new StreamChannelValue(
          c,
          { channel: c.channel.key },
          { onStatusChange: (s) => statuses.push(s) },
        );
        await waitForStream(scv, c);
        const live = new Series({ data: new Float32Array([5]) });
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
        reject(new Error("read exploded"));
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(statuses).toEqual([]);
        expect(scv.value()).toMatchObject({ data: 5, status: { variant: "success" } });
      });

      it("should not report a failed read from before a cleanup", async () => {
        const statuses: cstatus.Crude[] = [];
        let reject: (e: Error) => void = () => {};
        c.feed.readLatest = async () =>
          await new Promise<MultiSeries>((_, r) => {
            reject = r;
          });
        const scv = new StreamChannelValue(
          c,
          { channel: c.channel.key },
          { onStatusChange: (s) => statuses.push(s) },
        );
        await waitForStream(scv, c);
        const stale = reject;
        scv.cleanup();
        c.feed.readLatest = async () => new MultiSeries([]);
        await waitForStream(scv, c);
        await expect.poll(() => scv.value().status.variant).toBe("success");
        stale(new Error("read exploded"));
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(statuses).toEqual([]);
        expect(scv.value().status.variant).toBe("success");
      });

      describe("time", () => {
        it("should be undefined before an initial sample lands", () => {
          const scv = new StreamChannelValue(c, { channel: c.channel.key });
          expect(scv.value().time).toBeUndefined();
        });

        it("should be the end of the initial series", async () => {
          c.feed.readLatest = async () => new MultiSeries([stored()]);
          const scv = new StreamChannelValue(c, { channel: c.channel.key });
          await waitForResolve(scv);
          expect(scv.value().time?.equals(SAMPLE_TIME)).toBe(true);
        });

        it("should be undefined when the initial series has no time range", async () => {
          c.feed.readLatest = async () =>
            new MultiSeries([new Series({ data: new Float32Array([7]) })]);
          const scv = new StreamChannelValue(c, { channel: c.channel.key });
          await waitForResolve(scv);
          expect(scv.value().time).toBeUndefined();
        });

        it("should be undefined once a live buffer replaces the initial buffer", async () => {
          c.feed.readLatest = async () => new MultiSeries([stored()]);
          const scv = new StreamChannelValue(c, { channel: c.channel.key });
          await waitForResolve(scv);
          const live = new Series({
            data: new Float32Array([5]),
            timeRange: new TimeRange(SAMPLE_TIME, TimeStamp.MAX),
          });
          c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
          expect(scv.value().time).toBeUndefined();
        });

        it("should be undefined after cleanup", async () => {
          c.feed.readLatest = async () => new MultiSeries([stored()]);
          const scv = new StreamChannelValue(c, { channel: c.channel.key });
          await waitForResolve(scv);
          scv.cleanup();
          expect(scv.value().time).toBeUndefined();
        });
      });
    });
  });

  describe("StreamChannelStringValue", () => {
    class MockClient implements Client {
      now = (): TimeStamp => TimeStamp.now();
      key: string = id.create();

      streamHandler: framer.StreamHandler | null = null;
      streamKeys: channel.Key[] = [];
      streamF = vi.fn();
      streamDestructorF = vi.fn();

      channel: channel.Channel = new channel.Channel({
        key: 65537,
        name: "test",
        dataType: DataType.STRING,
        isIndex: false,
      });

      response: MultiSeries = new MultiSeries([]);

      channels = { retrieve: async (): Promise<channel.Channel> => this.channel };

      feed = {
        readLatest: async (): Promise<MultiSeries> => new MultiSeries([]),
        readTile: async (): Promise<MultiSeries> => new MultiSeries([]),
        read: async (): Promise<MultiSeries> => this.response,
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
    }

    let c: MockClient;

    beforeEach(() => {
      c = new MockClient();
      vi.resetAllMocks();
    });

    it("should return an empty string when no channel has been set", () => {
      const scsv = new StreamChannelStringValue(c, { channel: 0 });
      expect(scsv.value().data).toBe("");
      expect(c.streamF).not.toHaveBeenCalled();
    });

    it("should return an empty string when no data has been received", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      expect(await waitForStream(scsv, c)).toBe("");
      expect(c.streamF).toHaveBeenCalledTimes(1);
      expect(c.streamKeys).toEqual([c.channel.key]);
    });

    it("should report loading until the first read lands", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      expect(loading(scsv)).toBe(true);
      await expect.poll(() => scsv.value().status.variant).toBe("success");
    });

    it("should report an error when the channel cannot be retrieved", async () => {
      c.channels.retrieve = async () => {
        throw new Error("retrieve exploded");
      };
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      await expect
        .poll(() => scsv.value().status)
        .toMatchObject({
          variant: "error",
          message: "Failed to stream channel string value",
          description: "retrieve exploded",
        });
      expect(scsv.value().data).toBe("");
    });

    // A consumer that counts arrivals reads the open as a sample, and then reports the
    // channel stale one timeout later while it waits for real data.
    it("should not notify when the stream opens with no data", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      const handleChange = vi.fn();
      scsv.onChange(handleChange);
      await waitForStream(scsv, c);
      expect(handleChange).not.toHaveBeenCalled();
      c.streamHandler?.(
        new Map([[c.channel.key, new MultiSeries([new Series(["ARMED"])])]]),
      );
      await expect.poll(() => handleChange.mock.calls.length).toBe(1);
    });

    it("should decode the leading sample of a string series", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      const handleChange = vi.fn();
      scsv.onChange(handleChange);
      await waitForStream(scsv, c);
      const series = new Series(["IDLE", "ARMED"]);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length === 1).toBe(true);
      expect(scsv.value().data).toBe("ARMED");
      expect(series.refCount).toBe(1);
    });

    it("should preserve spaces and punctuation", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      await waitForStream(scsv, c);
      c.streamHandler?.(
        new Map([
          [c.channel.key, new MultiSeries([new Series(["Hello, World! (42%)"])])],
        ]),
      );
      await expect.poll(() => scsv.value().data).toBe("Hello, World! (42%)");
    });

    // The client appends into the leading buffer in place and then delivers a
    // response with no series, so a value cached at stream time goes stale.
    it("should return the new value when the leading buffer is appended to", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      const handleChange = vi.fn();
      scsv.onChange(handleChange);
      expect(scsv.value().data).toBe("");
      await waitForStream(scsv, c);
      const series = Series.alloc({ dataType: DataType.STRING, capacity: 64 });
      series.write(new Series(["IDLE"]));
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => handleChange.mock.calls.length === 1).toBe(true);
      expect(scsv.value().data).toBe("IDLE");

      series.write(new Series(["ARMED"]));
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([])]]));
      await expect.poll(() => handleChange.mock.calls.length === 2).toBe(true);
      expect(scsv.value().data).toBe("ARMED");
    });

    it("should replace and release the leading buffer when a new one arrives", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      await waitForStream(scsv, c);
      const first = new Series(["FIRST"]);
      const second = new Series(["SECOND"]);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([first])]]));
      await expect.poll(() => scsv.value().data).toBe("FIRST");
      expect(first.refCount).toBe(1);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([second])]]));
      await expect.poll(() => scsv.value().data).toBe("SECOND");
      expect(first.refCount).toBe(0);
      expect(second.refCount).toBe(1);
    });

    it("should stringify a numeric channel rather than returning NaN", async () => {
      c.channel = new channel.Channel({
        key: 65538,
        name: "numeric",
        dataType: DataType.FLOAT32,
        isIndex: false,
      });
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      await waitForStream(scsv, c);
      c.streamHandler?.(
        new Map([
          [
            c.channel.key,
            new MultiSeries([new Series({ data: new Float32Array([1, 2, 3]) })]),
          ],
        ]),
      );
      await expect.poll(() => scsv.value().data).toBe("3");
    });

    it("should release the leading buffer and stop streaming on cleanup", async () => {
      const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
      await waitForStream(scsv, c);
      const series = new Series(["IDLE"]);
      c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
      await expect.poll(() => scsv.value().data).toBe("IDLE");
      scsv.cleanup();
      expect(c.streamDestructorF).toHaveBeenCalledTimes(1);
      expect(series.refCount).toBe(0);
      expect(scsv.value().data).toBe("");
    });

    // value() runs read() synchronously up to its first await, so a cleanup() on the
    // next line lands between the awaits. Without the generation guard the resumed
    // read subscribes a source that nothing will clean up again.
    describe("cleanup during an in-flight read", () => {
      it("should not subscribe when cleaned up while retrieving the channel", async () => {
        let release = (): void => {};
        const gate = new Promise<void>((resolve) => (release = resolve));
        c.channels.retrieve = async (): Promise<channel.Channel> => {
          await gate;
          return c.channel;
        };
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        scsv.value();
        scsv.cleanup();
        release();
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(c.streamF).not.toHaveBeenCalled();
      });

      it("should not acquire buffers delivered to a stale handler", async () => {
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForStream(scsv, c);
        const staleHandler = c.streamHandler;
        scsv.cleanup();
        const series = new Series(["ARMED"]);
        staleHandler?.(new Map([[c.channel.key, new MultiSeries([series])]]));
        expect(series.refCount).toBe(0);
      });

      it("should not notify listeners from a stale handler", async () => {
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForStream(scsv, c);
        const staleHandler = c.streamHandler;
        const handleChange = vi.fn();
        scsv.onChange(handleChange);
        scsv.cleanup();
        staleHandler?.(
          new Map([[c.channel.key, new MultiSeries([new Series(["ARMED"])])]]),
        );
        expect(handleChange).not.toHaveBeenCalled();
      });

      it("should stream again after cleanup", async () => {
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForStream(scsv, c);
        scsv.cleanup();
        expect(await waitForStream(scsv, c)).toBe("");
        expect(c.streamF).toHaveBeenCalledTimes(2);
        c.streamHandler?.(
          new Map([[c.channel.key, new MultiSeries([new Series(["ARMED"])])]]),
        );
        await expect.poll(() => scsv.value().data).toBe("ARMED");
      });

      it("should not let a stale handler overwrite a later read's buffer", async () => {
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForStream(scsv, c);
        const staleHandler = c.streamHandler;
        scsv.cleanup();
        await waitForStream(scsv, c);
        const fresh = new Series(["ARMED"]);
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([fresh])]]));
        await expect.poll(() => scsv.value().data).toBe("ARMED");
        const stale = new Series(["IDLE"]);
        staleHandler?.(new Map([[c.channel.key, new MultiSeries([stale])]]));
        expect(scsv.value().data).toBe("ARMED");
        expect(stale.refCount).toBe(0);
        expect(fresh.refCount).toBe(1);
      });
    });

    describe("initializing", () => {
      const SAMPLE_TIME = TimeStamp.seconds(100);
      const stored = (): Series =>
        new Series({
          data: ["IDLE", "ARMED"],
          timeRange: new TimeRange(TimeStamp.seconds(90), SAMPLE_TIME),
        });

      it("should initialize the value from the latest stored sample", async () => {
        c.feed.readLatest = vi.fn(async () => new MultiSeries([stored()]));
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        expect(await waitForResolve(scsv)).toBe("ARMED");
        expect(c.feed.readLatest).toHaveBeenCalledWith(c.channel.key);
      });

      it("should not notify when nothing is stored", async () => {
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        const handleChange = vi.fn();
        scsv.onChange(handleChange);
        await waitForStream(scsv, c);
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(handleChange).not.toHaveBeenCalled();
        expect(scsv.value().data).toBe("");
      });

      it("should keep a live buffer that arrives while the read is in flight", async () => {
        let release = (): void => {};
        const gate = new Promise<void>((resolve) => (release = resolve));
        c.feed.readLatest = async () => {
          await gate;
          return new MultiSeries([stored()]);
        };
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForStream(scsv, c);
        const live = new Series(["LIVE"]);
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
        release();
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(scsv.value().data).toBe("LIVE");
        expect(live.refCount).toBe(1);
      });

      it("should replace the initial buffer and release it when a live buffer arrives", async () => {
        const initial = stored();
        c.feed.readLatest = async () => new MultiSeries([initial]);
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForResolve(scsv);
        expect(initial.refCount).toBe(1);
        const live = new Series(["LIVE"]);
        c.streamHandler?.(new Map([[c.channel.key, new MultiSeries([live])]]));
        expect(scsv.value().data).toBe("LIVE");
        expect(initial.refCount).toBe(0);
      });

      it("should skip the read for a virtual channel", async () => {
        c.channel = new channel.Channel({ ...c.channel, virtual: true });
        c.feed.readLatest = vi.fn(async () => new MultiSeries([stored()]));
        const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
        await waitForStream(scsv, c);
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(c.feed.readLatest).not.toHaveBeenCalled();
        expect(scsv.value().data).toBe("");
      });

      it("should post a status and keep streaming when the read fails", async () => {
        const statuses: cstatus.Crude[] = [];
        c.feed.readLatest = async () => {
          throw new Error("read exploded");
        };
        const scsv = new StreamChannelStringValue(
          c,
          { channel: c.channel.key },
          { onStatusChange: (s) => statuses.push(s) },
        );
        await waitForStream(scsv, c);
        await expect.poll(() => statuses.length > 0).toBe(true);
        expect(statuses[0].message).toEqual("Failed to read latest value");
        expect(c.streamDestructorF).not.toHaveBeenCalled();
        c.streamHandler?.(
          new Map([[c.channel.key, new MultiSeries([new Series(["LIVE"])])]]),
        );
        expect(scsv.value().data).toBe("LIVE");
      });

      it("should not report a failed read once live data arrives", async () => {
        const statuses: cstatus.Crude[] = [];
        let reject: (e: Error) => void = () => {};
        c.feed.readLatest = async () =>
          await new Promise<MultiSeries>((_, r) => {
            reject = r;
          });
        const scsv = new StreamChannelStringValue(
          c,
          { channel: c.channel.key },
          { onStatusChange: (s) => statuses.push(s) },
        );
        await waitForStream(scsv, c);
        c.streamHandler?.(
          new Map([[c.channel.key, new MultiSeries([new Series(["LIVE"])])]]),
        );
        reject(new Error("read exploded"));
        await new Promise((resolve) => setTimeout(resolve, 5));
        expect(statuses).toEqual([]);
        expect(scsv.value()).toMatchObject({
          data: "LIVE",
          status: { variant: "success" },
        });
      });

      describe("time", () => {
        it("should be the end of the initial series", async () => {
          c.feed.readLatest = async () => new MultiSeries([stored()]);
          const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
          expect(scsv.value().time).toBeUndefined();
          await waitForResolve(scsv);
          expect(scsv.value().time?.equals(SAMPLE_TIME)).toBe(true);
        });

        it("should be undefined once a live buffer replaces the initial buffer", async () => {
          c.feed.readLatest = async () => new MultiSeries([stored()]);
          const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
          await waitForResolve(scsv);
          c.streamHandler?.(
            new Map([[c.channel.key, new MultiSeries([new Series(["LIVE"])])]]),
          );
          expect(scsv.value().time).toBeUndefined();
        });

        it("should be undefined after cleanup", async () => {
          c.feed.readLatest = async () => new MultiSeries([stored()]);
          const scsv = new StreamChannelStringValue(c, { channel: c.channel.key });
          await waitForResolve(scsv);
          scsv.cleanup();
          expect(scsv.value().time).toBeUndefined();
        });
      });
    });
  });

  describe("TiledChannelData", () => {
    const DATA_KEY = 65537;
    const INDEX_KEY = 65538;
    const XY_KEY = 65539;

    interface Pending {
      spec: framer.TileSpec;
      resolve: (series: MultiSeries) => void;
    }

    // Returns one series per tile. Data tiles hold [level, index], and index tiles hold
    // the tile's start and end. An average tile holds the mean of the two.
    const tileSeries = (spec: framer.TileSpec): MultiSeries => {
      const tr = framer.grid.tileRange(spec);
      const alignment = (BigInt(spec.level) << 32n) + BigInt(spec.index);
      const average = spec.aggregation === "average";
      if (spec.key === INDEX_KEY) {
        const [start, end] = [tr.start.valueOf(), tr.end.valueOf() - 1n];
        return new MultiSeries([
          new Series({
            data: new BigInt64Array(average ? [(start + end) / 2n] : [start, end]),
            dataType: DataType.TIMESTAMP,
            timeRange: tr,
            alignment,
          }),
        ]);
      }
      const { level, index } = spec;
      return new MultiSeries([
        new Series({
          data: new Float32Array(average ? [(level + index) / 2] : [level, index]),
          timeRange: tr,
          alignment,
        }),
      ]);
    };

    const servedKey = ({ level, index, key, aggregation }: framer.TileSpec): string =>
      `${level}/${index}/${key}/${aggregation}`;

    class MockClient implements Client {
      now = (): TimeStamp => TimeStamp.now();
      dataType = DataType.FLOAT32;
      tiles: framer.TileSpec[] = [];
      tileDataTypes: Array<DataType | undefined> = [];
      reads: TimeRange[] = [];
      deferred = false;
      pending: Pending[] = [];
      served = new Map<string, MultiSeries>();
      streamHandler: framer.StreamHandler | null = null;
      streamDestructorF = vi.fn();
      response: MultiSeries = new MultiSeries([]);

      channels = {
        retrieve: async (key: channel.Key | channel.Name): Promise<channel.Channel> => {
          if (key === INDEX_KEY)
            return new channel.Channel({
              key: INDEX_KEY,
              name: "time",
              dataType: DataType.TIMESTAMP,
              isIndex: true,
            });
          return new channel.Channel({
            key,
            name: `data-${key}`,
            dataType: this.dataType,
            index: INDEX_KEY,
          });
        },
      };

      feed = {
        readLatest: async (): Promise<MultiSeries> => new MultiSeries([]),
        read: async (tr: TimeRange): Promise<MultiSeries> => {
          this.reads.push(tr);
          return this.response;
        },
        readTile: async (
          spec: framer.TileSpec,
          dataType?: DataType,
        ): Promise<MultiSeries> => {
          this.tiles.push(spec);
          this.tileDataTypes.push(dataType);
          if (this.deferred)
            return await new Promise<MultiSeries>((resolve) =>
              this.pending.push({ spec, resolve }),
            );
          const series = tileSeries(spec);
          this.served.set(servedKey(spec), series);
          return series;
        },
        stream: (handler: framer.StreamHandler): framer.Subscription => {
          this.streamHandler = handler;
          return telemTest.mockSubscription(this.streamDestructorF);
        },
      };

      resolveAll(): void {
        this.resolveWhere(() => true);
      }

      resolveWhere(match: (spec: framer.TileSpec) => boolean): void {
        const pending = this.pending.filter(({ spec }) => match(spec));
        this.pending = this.pending.filter(({ spec }) => !match(spec));
        pending.forEach(({ spec, resolve }) => {
          const series = tileSeries(spec);
          this.served.set(servedKey(spec), series);
          resolve(series);
        });
      }
    }

    let c: MockClient;
    beforeEach(() => {
      c = new MockClient();
    });

    const HOME = new TimeRange(TimeStamp.SECOND, TimeStamp.seconds(10));

    const create = (
      props: Partial<TiledChannelDataProps> = {},
      now: () => TimeStamp = () => TimeStamp.seconds(100),
      options?: CreateOptions,
      breakerConfig?: breaker.Config,
    ): TiledChannelData =>
      new TiledChannelData(
        c,
        { channel: DATA_KEY, window: staticWindow(HOME), ...props },
        options,
        now,
        breakerConfig,
      );

    // Waits for the source to settle after a value call that starts reads.
    const settle = async (source: TiledChannelData): Promise<void> =>
      await expect.poll(() => loading(source)).toBe(false);

    const view = (lower: number, upper: number, points = 500): ValueProps => ({
      bounds: {
        lower: Number(TimeStamp.milliseconds(lower).valueOf()),
        upper: Number(TimeStamp.milliseconds(upper).valueOf()),
      },
      points,
    });

    const positions = (series: MultiSeries): number[][] =>
      series.series.map((s) => Array.from(s) as number[]);

    it("should return nothing when no channel is set", () => {
      const source = create({ channel: 0 });
      const [b, series] = source.value().data;
      expect(b).toEqual(bounds.INVALID);
      expect(series.length).toBe(0);
      expect(loading(source)).toBe(false);
      expect(c.tiles).toHaveLength(0);
    });

    it("should load the tiles of the home view", async () => {
      const source = create();
      expect(loading(source)).toBe(true);
      await settle(source);
      expect(c.tiles.map((t) => [t.level, t.index, t.pointLimit])).toEqual([
        [13, 0, 1024],
        [13, 1, 1024],
      ]);
      expect(c.tiles.every((t) => t.aggregation === "min_max")).toBe(true);
      expect(positions(source.value().data[1])).toEqual([
        [13, 0],
        [13, 1],
      ]);
    });

    it("should split the points of a read across its tiles", async () => {
      const source = create({ aggregation: "average" });
      source.value({ points: 1000 });
      await settle(source);
      expect(c.tiles.map((t) => [t.pointLimit, t.aggregation])).toEqual([
        [1024, "average"],
        [1024, "average"],
      ]);
      expect(source.value().aggregation).toBe("average");
    });

    it("should keep the exact bounds of a static window past 2^53", async () => {
      const start = 2n ** 53n + 1n;
      const home = new TimeRange(start, start + TimeSpan.seconds(9).valueOf());
      const source = create({
        xChannel: XY_KEY,
        channel: XY_KEY,
        window: staticWindow(home),
      });
      await settle(source);
      expect(c.tiles[0].start?.valueOf()).toBe(start);
    });

    it("should bound an index channel by the home view", async () => {
      const source = create({ useIndexOfChannel: true });
      await settle(source);
      expect(source.value().data[0]).toEqual(HOME.numericBounds);
      expect(c.tiles.every((t) => t.key === INDEX_KEY)).toBe(true);
    });

    it("should bound a data channel by its data", async () => {
      const source = create();
      await settle(source);
      expect(source.value().data[0]).toEqual({ lower: 0, upper: 13 });
    });

    describe("extremes kept", () => {
      it("should bound an average line by min_max tiles of the home view", async () => {
        const source = create({ aggregation: "average" });
        await settle(source);
        expect(source.value().data[0]).toEqual({ lower: 6.5, upper: 7 });
        source.value({ extremesKept: true });
        await expect.poll(() => loading(source, { extremesKept: true })).toBe(false);
        expect(c.tiles.map((t) => [t.level, t.index, t.aggregation])).toEqual([
          [13, 0, "average"],
          [13, 1, "average"],
          [13, 0, "min_max"],
          [13, 1, "min_max"],
        ]);
        expect(source.value({ extremesKept: true }).data[0]).toEqual({
          lower: 0,
          upper: 13,
        });
      });

      it("should bound a min_max line by the tiles it draws", async () => {
        const source = create();
        await settle(source);
        const before = c.tiles.length;
        expect(source.value({ extremesKept: true }).data[0]).toEqual({
          lower: 0,
          upper: 13,
        });
        expect(c.tiles).toHaveLength(before);
      });

      it("should bound an xy line by the decimated tiles it draws", async () => {
        const source = create({ xChannel: XY_KEY, channel: XY_KEY });
        await settle(source);
        const before = c.tiles.length;
        source.value({ extremesKept: true });
        expect(c.tiles).toHaveLength(before);
      });

      it("should keep the min_max home tiles through a zoomed view", async () => {
        const source = create({ aggregation: "average" });
        await settle(source);
        source.value({ extremesKept: true });
        source.value(view(2000, 3000, 1000));
        await expect.poll(() => loading(source, view(2000, 3000, 1000))).toBe(false);
        source.value(view(2000, 3000, 1000));
        const tile = c.served.get(`13/0/${DATA_KEY}/min_max`)!;
        expect(tile.series[0].refCount).toBe(1);
        const before = c.tiles.length;
        source.value({ extremesKept: true });
        expect(c.tiles).toHaveLength(before);
      });

      it("should report loading until a bounds read's min_max tiles land", async () => {
        const source = create({ aggregation: "average" });
        c.deferred = true;
        await expect
          .poll(() => {
            source.value();
            source.value({ extremesKept: true });
            return c.pending.length;
          })
          .toBe(4);
        c.resolveWhere((spec) => spec.aggregation === "average");
        await expect.poll(() => source.value().data[1].length).toBe(2);
        expect(loading(source)).toBe(true);
        c.resolveAll();
        await settle(source);
      });

      it("should draw an average stand-in over a min_max one", async () => {
        const source = create({ aggregation: "average" });
        await expect
          .poll(() => source.value({ extremesKept: true }).data[0])
          .toEqual({ lower: 0, upper: 13 });
        await settle(source);
        c.deferred = true;
        const [, series] = source.value(view(2000, 3000, 1000)).data;
        expect(positions(series)).toEqual([[6.5]]);
      });
    });

    it("should draw the tiles of the view it is given", async () => {
      const source = create();
      await settle(source);
      source.value(view(2000, 3000));
      await expect
        .poll(() => positions(source.value(view(2000, 3000)).data[1]))
        .toEqual([
          [9, 3],
          [9, 4],
          [9, 5],
        ]);
    });

    it("should draw a coarse tile until the view's tiles arrive", async () => {
      const source = create();
      await settle(source);
      c.deferred = true;
      const [, series] = source.value(view(2000, 3000)).data;
      expect(positions(series)).toEqual([[13, 0]]);
      expect(loading(source)).toBe(true);
      c.resolveAll();
      await expect.poll(() => loading(source, view(2000, 3000))).toBe(false);
      expect(positions(source.value(view(2000, 3000)).data[1])).toHaveLength(3);
    });

    it("should draw finer tiles until a zoomed out view's tiles arrive", async () => {
      const source = create();
      await settle(source);
      c.deferred = true;
      const [, series] = source.value(view(0, 60000)).data;
      expect(positions(series)).toEqual([
        [13, 0],
        [13, 1],
      ]);
      c.resolveAll();
      await expect.poll(() => loading(source, view(0, 60000))).toBe(false);
      expect(positions(source.value(view(0, 60000)).data[1])).toEqual([
        [15, 0],
        [15, 1],
      ]);
    });

    it("should request each tile once while it is pending", async () => {
      const source = create();
      await settle(source);
      c.deferred = true;
      const before = c.tiles.length;
      source.value(view(2000, 3000));
      source.value(view(2000, 3000));
      expect(c.tiles.length - before).toBe(3);
    });

    describe("data type changes", () => {
      const bytes = (spec: framer.TileSpec): MultiSeries =>
        new MultiSeries([
          new Series({
            data: new Uint8Array([spec.level, spec.index]),
            timeRange: framer.grid.tileRange(spec),
          }),
        ]);

      it("should ask for tiles in the data type the line holds", async () => {
        const source = create();
        await settle(source);
        expect(c.tileDataTypes).toEqual([undefined, undefined]);
        source.value(view(2000, 3000));
        await expect.poll(() => loading(source, view(2000, 3000))).toBe(false);
        expect(c.tileDataTypes.slice(2)).toEqual([
          DataType.FLOAT32,
          DataType.FLOAT32,
          DataType.FLOAT32,
        ]);
      });

      it("should read the channel again when a tile of a new data type arrives", async () => {
        const source = create();
        await settle(source);
        const home = [...c.served.values()];
        c.feed.readTile = async (spec) => bytes(spec);
        source.value(view(2000, 3000));
        await expect
          .poll(() => home.every((tile) => tile.series[0].refCount === 0))
          .toBe(true);
        await settle(source);
        expect(positions(source.value().data[1])).toEqual([
          [13, 0],
          [13, 1],
        ]);
        expect(source.value().data[1].dataType.equals(DataType.UINT8)).toBe(true);
      });
    });

    it("should release the tiles a view no longer needs", async () => {
      const source = create();
      await settle(source);
      source.value(view(2000, 3000));
      await expect
        .poll(() => source.value(view(2000, 3000)).data[1].series.length)
        .toBe(3);
      const fine = c.served.get(`9/3/${DATA_KEY}/min_max`)!;
      expect(fine.series[0].refCount).toBe(1);
      source.value(view(1000, 10000, 1000));
      expect(fine.series[0].refCount).toBe(0);
      expect(c.served.get(`13/0/${DATA_KEY}/min_max`)!.series[0].refCount).toBe(1);
    });

    it("should release a tile that arrives after its view moved on", async () => {
      const source = create();
      await settle(source);
      c.deferred = true;
      source.value(view(2000, 3000));
      const home = view(1000, 10000, 1000);
      source.value(home);
      c.resolveAll();
      await expect.poll(() => loading(source)).toBe(false);
      const late = c.served.get(`9/3/${DATA_KEY}/min_max`)!;
      expect(late.series[0].refCount).toBe(1);
      expect(positions(source.value(home).data[1])).toEqual([
        [13, 0],
        [13, 1],
      ]);
      expect(late.series[0].refCount).toBe(0);
    });

    it("should keep the home tiles through a view far from home", async () => {
      const source = create();
      await settle(source);
      source.value(view(500_000, 501_000));
      await expect.poll(() => loading(source, view(500_000, 501_000))).toBe(false);
      source.value(view(500_000, 501_000));
      expect(c.served.get(`13/0/${DATA_KEY}/min_max`)!.series[0].refCount).toBe(1);
    });

    it("should read an xy line's home range once with decimate", async () => {
      const source = create({ xChannel: XY_KEY, channel: XY_KEY });
      await settle(source);
      expect(
        c.tiles.map((t) => [
          t.index,
          t.aggregation,
          t.start?.valueOf(),
          t.end?.valueOf(),
        ]),
      ).toEqual([
        [0, "decimate", HOME.start.valueOf(), undefined],
        [1, "decimate", undefined, HOME.end.valueOf()],
      ]);
      expect(source.value().aggregation).toBe("decimate");
      const before = c.tiles.length;
      source.value(view(2000, 3000, 1000));
      expect(c.tiles.length).toBe(before);
    });

    it("should release an xy line's tiles of a former width", async () => {
      const source = create({ xChannel: XY_KEY, channel: XY_KEY });
      await settle(source);
      const former = [...c.served.values()];
      const wide = view(2000, 3000, 4000);
      source.value(wide);
      await expect.poll(() => loading(source, wide)).toBe(false);
      source.value(wide);
      expect(c.tiles).toHaveLength(4);
      expect(former.map((s) => s.series[0].refCount)).toEqual([0, 0]);
    });

    describe("failures", () => {
      const NO_WAIT: breaker.Config = { sleepFn: async () => {} };
      const FAST: breaker.Config = {
        baseInterval: TimeSpan.milliseconds(1),
        scale: 1,
        jitter: 0,
      };

      const failTiles = (error: Error): void => {
        c.feed.readTile = async (spec) => {
          c.tiles.push(spec);
          throw error;
        };
      };

      it("should not request a failed tile again while it backs off", async () => {
        const statuses: cstatus.Crude[] = [];
        failTiles(new Error("transient"));
        const source = create(
          {},
          undefined,
          { onStatusChange: (s) => statuses.push(s) },
          { sleepFn: async () => await new Promise(() => {}) },
        );
        const handleChange = vi.fn();
        source.onChange(handleChange);
        await settle(source);
        const failed = c.tiles.length;
        for (let i = 0; i < 5; i++) source.value();
        await sleep.sleep(TimeSpan.milliseconds(10));
        expect(c.tiles).toHaveLength(failed);
        expect(statuses.map((s) => [s.message, s.description])).toEqual([
          ["Failed to read channel data", "transient"],
        ]);
        // Once when the channel resolves, and once when the first tile read fails.
        expect(handleChange).toHaveBeenCalledTimes(2);
        source.cleanup();
      });

      it("should request a failed tile again once the backoff ends", async () => {
        let failing = true;
        c.feed.readTile = async (spec) => {
          c.tiles.push(spec);
          if (failing) throw new Error("transient");
          return tileSeries(spec);
        };
        let wake = (): void => {};
        const source = create({}, undefined, undefined, {
          sleepFn: async () =>
            await new Promise<void>((resolve) => {
              wake = resolve;
            }),
        });
        await settle(source);
        expect(source.value().status.variant).toBe("error");
        failing = false;
        wake();
        await expect
          .poll(() => positions(source.value().data[1]))
          .toEqual([
            [13, 0],
            [13, 1],
          ]);
        expect(source.value().status.variant).toBe("success");
        source.cleanup();
      });

      it("should park a tile on a definitive rejection", async () => {
        failTiles(new ValidationError("bad tile"));
        const source = create({}, undefined, undefined, NO_WAIT);
        await settle(source);
        const failed = c.tiles.length;
        await sleep.sleep(TimeSpan.milliseconds(10));
        expect(source.value().status).toMatchObject({
          variant: "error",
          message: "Failed to read channel data",
          description: "bad tile",
        });
        expect(c.tiles).toHaveLength(failed);
        source.cleanup();
      });

      it("should stop retrying tiles at cleanup", async () => {
        failTiles(new Error("boom"));
        const source = create({}, undefined, undefined, FAST);
        // The notify at the end of each backoff stands in for the render it starts.
        source.onChange(() => source.value());
        source.value();
        await expect.poll(() => c.tiles.length >= 6).toBe(true);
        source.cleanup();
        const settled = c.tiles.length;
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(c.tiles).toHaveLength(settled);
      });

      it("should retry a failed channel retrieval until it lands", async () => {
        const { retrieve } = c.channels;
        let calls = 0;
        c.channels.retrieve = async (key) => {
          calls++;
          if (calls === 1) throw new Error("transient");
          return await retrieve(key);
        };
        const source = create({}, undefined, undefined, NO_WAIT);
        source.value();
        await expect
          .poll(() => positions(source.value().data[1]))
          .toEqual([
            [13, 0],
            [13, 1],
          ]);
        expect(calls).toBe(2);
        source.cleanup();
      });

      it("should park on a definitive rejection of the channel retrieval", async () => {
        const statuses: cstatus.Crude[] = [];
        let calls = 0;
        c.channels.retrieve = async () => {
          calls++;
          throw new NotFoundError("channel not found");
        };
        const source = create(
          {},
          undefined,
          { onStatusChange: (s) => statuses.push(s) },
          NO_WAIT,
        );
        await settle(source);
        await sleep.sleep(TimeSpan.milliseconds(10));
        expect(source.value().status).toMatchObject({
          variant: "error",
          description: "channel not found",
        });
        expect(calls).toBe(1);
        expect(statuses.map((s) => [s.message, s.description])).toEqual([
          ["Failed to read channel data", "channel not found"],
        ]);
        source.cleanup();
      });

      it("should stop retrying a channel retrieval at cleanup", async () => {
        let calls = 0;
        c.channels.retrieve = async () => {
          calls++;
          throw new Error("boom");
        };
        const source = create({}, undefined, undefined, FAST);
        source.value();
        await expect.poll(() => calls >= 3).toBe(true);
        source.cleanup();
        const settled = calls;
        await sleep.sleep(TimeSpan.milliseconds(20));
        expect(calls).toBe(settled);
      });
    });

    it("should release every tile on cleanup", async () => {
      const source = create();
      await settle(source);
      const home = c.served.get(`13/0/${DATA_KEY}/min_max`)!;
      source.cleanup();
      expect(home.series[0].refCount).toBe(0);
    });

    describe("live", () => {
      const SPAN = TimeSpan.seconds(10);
      let now: TimeStamp;
      const clock = (): TimeStamp => now;
      beforeEach(() => {
        now = TimeStamp.seconds(100);
      });

      const createLive = (props: Partial<TiledChannelDataProps> = {}) =>
        create({ window: dynamicWindow(SPAN), ...props }, clock);

      const raw = (start: number, end: number, alignment = BigInt(start)): Series =>
        new Series({
          data: new Float32Array([start, end]),
          timeRange: new TimeRange(TimeStamp.seconds(start), TimeStamp.seconds(end)),
          alignment,
        });

      // Resolves on the notify that ends the source's first read.
      const firstRead = async (source: TiledChannelData): Promise<void> => {
        const notified = vi.fn();
        source.onChange(notified);
        source.value();
        await expect.poll(() => notified).toHaveBeenCalled();
      };

      // Holds every back-fill read until the returned release is called.
      const gateReads = (): (() => void) => {
        let release = (): void => {};
        const gate = new Promise<void>((resolve) => (release = resolve));
        const { read } = c.feed;
        c.feed.read = async (tr) => {
          await gate;
          return await read(tr);
        };
        return release;
      };

      it("should read the live window at full resolution", async () => {
        c.response = new MultiSeries([raw(82, 100)]);
        const source = createLive();
        await firstRead(source);
        expect(c.reads).toHaveLength(1);
        expect(c.reads[0].start.valueOf()).toBe(
          TimeStamp.milliseconds(81_920).valueOf(),
        );
        expect(c.tiles).toHaveLength(0);
        expect(positions(source.value().data[1])).toEqual([[82, 100]]);
      });

      it("should stream a live xy line's home view without tiles", async () => {
        c.response = new MultiSeries([raw(90, 100)]);
        const source = createLive({ xChannel: XY_KEY, channel: XY_KEY });
        await firstRead(source);
        expect(c.tiles).toHaveLength(0);
        expect(c.reads[0].start.valueOf()).toBe(TimeStamp.seconds(90).valueOf());
        expect(positions(source.value().data[1])).toEqual([[90, 100]]);
        expect(source.value().aggregation).toBe("decimate");
      });

      it("should stream new samples into the live window", async () => {
        const source = createLive();
        await firstRead(source);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([raw(100, 101)])]]));
        expect(positions(source.value().data[1])).toEqual([[100, 101]]);
      });

      it("should draw tiles before the live window and raw data after it", async () => {
        c.response = new MultiSeries([raw(82, 100)]);
        const source = createLive();
        await firstRead(source);
        const v = view(75_000, 100_000, 1000);
        source.value(v);
        await expect.poll(() => loading(source, v)).toBe(false);
        expect(positions(source.value(v).data[1])).toEqual([
          [14, 4],
          [82, 100],
        ]);
      });

      it("should cut a coarse tile at the start of the live window", async () => {
        const source = createLive();
        await firstRead(source);
        source.value(view(0, 100_000, 1000));
        const cut = c.tiles.find((t) => t.end != null);
        expect(cut?.level).toBe(16);
        expect(cut?.end?.valueOf()).toBe(TimeStamp.milliseconds(81_920).valueOf());
      });

      it("should drop raw series that end before the live window", async () => {
        const old = raw(82, 90);
        c.response = new MultiSeries([old]);
        const source = createLive();
        await firstRead(source);
        expect(old.refCount).toBe(1);
        now = TimeStamp.seconds(200);
        source.value();
        expect(old.refCount).toBe(0);
      });

      it("should draw a finished tile where the live window moved past", async () => {
        const left = raw(82, 90);
        c.response = new MultiSeries([left, raw(90, 100)]);
        const source = createLive();
        await firstRead(source);
        now = TimeStamp.seconds(108);
        const v = view(80_000, 108_000, 1000);
        source.value(v);
        await expect.poll(() => loading(source, v)).toBe(false);
        // The tile stops where the held raw series starts, so no span draws twice.
        const cut = c.tiles.find((t) => t.level === 14 && t.index === 5);
        expect(cut?.end?.valueOf()).toBe(TimeStamp.seconds(90).valueOf());
        expect(positions(source.value(v).data[1])).toEqual([
          [14, 4],
          [14, 5],
          [90, 100],
        ]);
        expect(left.refCount).toBe(0);
      });

      it("should draw raw series the live window moved past until their tile arrives", async () => {
        const left = raw(82, 90);
        c.response = new MultiSeries([left, raw(90, 100)]);
        const source = createLive();
        await firstRead(source);
        now = TimeStamp.seconds(108);
        c.deferred = true;
        const v = view(80_000, 108_000, 1000);
        expect(positions(source.value(v).data[1])).toEqual([
          [82, 90],
          [90, 100],
        ]);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([raw(100, 108)])]]));
        expect(left.refCount).toBe(1);
        c.resolveAll();
        await expect.poll(() => loading(source, v)).toBe(false);
        expect(positions(source.value(v).data[1])).toEqual([
          [14, 4],
          [14, 5],
          [90, 100],
          [100, 108],
        ]);
        expect(left.refCount).toBe(0);
      });

      it("should release raw series a line that never draws moved past", async () => {
        const [left, right] = [raw(82, 90), raw(90, 100)];
        c.response = new MultiSeries([left, right]);
        const source = createLive();
        await firstRead(source);
        now = TimeStamp.seconds(120);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([raw(100, 120)])]]));
        expect(left.refCount).toBe(0);
        expect(right.refCount).toBe(1);
      });

      it("should release older tiles when the view returns to the live window", async () => {
        const source = createLive();
        await firstRead(source);
        const older = view(75_000, 100_000, 1000);
        source.value(older);
        await expect.poll(() => loading(source, older)).toBe(false);
        const tile = c.served.get(`14/4/${DATA_KEY}/min_max`)!;
        expect(tile.series[0].refCount).toBe(1);
        source.value(view(95_000, 100_000, 1000));
        expect(tile.series[0].refCount).toBe(0);
        source.cleanup();
      });

      it("should bound a live index by the span before its latest sample", async () => {
        const source = createLive({ useIndexOfChannel: true });
        await firstRead(source);
        const [b] = source.value().data;
        expect(b.upper).toBe(Number(TimeStamp.seconds(100).valueOf()));
        expect(b.upper - b.lower).toBe(Number(SPAN.valueOf()));
      });

      it("should stop streaming on cleanup", async () => {
        const source = createLive();
        await firstRead(source);
        source.cleanup();
        expect(c.streamDestructorF).toHaveBeenCalled();
      });

      it("should keep the buffers of a restarted Core after the older ones", async () => {
        const source = createLive();
        await firstRead(source);
        const before = raw(95, 96, 5n << 32n);
        const after = raw(97, 98, 0n);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([before])]]));
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([after])]]));
        expect(positions(source.value().data[1])).toEqual([
          [95, 96],
          [97, 98],
        ]);
      });

      it("should read the channel again when a new data type meets held tiles", async () => {
        const source = createLive();
        await firstRead(source);
        const v = view(75_000, 100_000, 1000);
        source.value(v);
        await expect.poll(() => loading(source, v)).toBe(false);
        const tile = c.served.get(`14/4/${DATA_KEY}/min_max`)!;
        const opened = c.streamHandler;
        const bytes = new Series({
          data: new Uint8Array([3, 4]),
          timeRange: new TimeRange(TimeStamp.seconds(96), TimeStamp.seconds(97)),
        });
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([bytes])]]));
        expect(tile.series[0].refCount).toBe(0);
        expect(bytes.refCount).toBe(0);
        expect(source.value(v).data[1].series).toEqual([]);
        await expect.poll(() => c.streamHandler !== opened).toBe(true);
        source.cleanup();
      });

      it("should hold streamed float64 series exactly", async () => {
        const source = createLive();
        await firstRead(source);
        const exact = new Series({
          data: new Float64Array([100.1, 100.2]),
          timeRange: new TimeRange(TimeStamp.seconds(100), TimeStamp.seconds(101)),
          alignment: 100n,
        });
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([exact])]]));
        expect(source.value().data[1].series).toEqual([exact]);
        expect(source.value().data[1].at(0)).toBe(100.1);
      });

      it("should read the channel again when its data type changes", async () => {
        const source = createLive();
        await firstRead(source);
        const v = view(75_000, 100_000, 1000);
        source.value(v);
        await expect.poll(() => loading(source, v)).toBe(false);
        const tile = c.served.get(`14/4/${DATA_KEY}/min_max`)!;
        const float = raw(95, 96);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([float])]]));
        const bytes = new Series({
          data: new Uint8Array([3, 4]),
          timeRange: new TimeRange(TimeStamp.seconds(96), TimeStamp.seconds(97)),
        });
        const opened = c.streamHandler;
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([bytes])]]));
        expect(source.value().data[1].series).toEqual([]);
        expect(float.refCount).toBe(0);
        expect(tile.series[0].refCount).toBe(0);
        await expect.poll(() => c.streamHandler !== opened).toBe(true);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([bytes])]]));
        expect(source.value().data[1].series).toEqual([bytes]);
        source.cleanup();
      });

      it("should end the back-fill where the stream opened", async () => {
        let openedAt: TimeStamp | undefined;
        const { stream } = c.feed;
        // The clock moves on once the stream opens, so a back-fill that reads it again
        // overlaps live data.
        c.feed.stream = (handler) => {
          openedAt = now;
          now = now.add(TimeSpan.seconds(1));
          return stream(handler);
        };
        const source = createLive();
        await firstRead(source);
        expect(c.reads.map((r) => r.end)).toEqual([openedAt]);
      });

      it("should read the seam up to where the later of the channel and index streams starts", async () => {
        const source = create({ window: dynamicWindow(SPAN) }, () => TimeStamp.now());
        await firstRead(source);
        expect(c.reads).toHaveLength(1);
        const live = new Series({
          data: new Float32Array([1, 2]),
          timeRange: new TimeRange(
            c.reads[0].end.add(TimeSpan.milliseconds(100)),
            TimeStamp.MAX,
          ),
        });
        const liveStart = c.reads[0].end.add(TimeSpan.milliseconds(200));
        const index = new Series({
          data: new BigInt64Array([1n, 2n]),
          dataType: DataType.TIMESTAMP,
          timeRange: new TimeRange(liveStart, TimeStamp.MAX),
        });
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([live])]]));
        c.streamHandler?.(new Map([[INDEX_KEY, new MultiSeries([index])]]));
        // The seam read waits a second for its samples to commit.
        await sleep.sleep(TimeSpan.milliseconds(300));
        expect(c.reads).toHaveLength(1);
        await expect.poll(() => c.reads.length, { timeout: 3000 }).toBe(2);
        expect(c.reads[1]).toEqual(new TimeRange(c.reads[0].end, liveStart));
        source.cleanup();
      });

      it("should draw streamed data while its back-fill loads", async () => {
        const release = gateReads();
        const source = createLive({
          window: dynamicWindow(TimeSpan.minutes(1)),
        });
        source.value();
        await expect.poll(() => c.streamHandler != null).toBe(true);
        c.streamHandler?.(new Map([[DATA_KEY, new MultiSeries([raw(100, 101)])]]));
        const { data, status } = source.value();
        expect(status.variant).toBe("loading");
        expect(positions(data[1])).toEqual([[100, 101]]);
        release();
        source.cleanup();
      });

      it("should report loading for a live window until its back-fill lands", async () => {
        const release = gateReads();
        now = TimeStamp.seconds(600);
        const source = createLive({
          window: dynamicWindow(TimeSpan.minutes(2)),
        });
        expect(loading(source)).toBe(true);
        await expect.poll(() => c.streamHandler != null).toBe(true);
        expect(loading(source)).toBe(true);
        release();
        await settle(source);
        source.cleanup();
      });

      it("should report loading for tiles in view while the back-fill loads", async () => {
        const release = gateReads();
        c.deferred = true;
        const source = createLive();
        source.value();
        await expect.poll(() => c.streamHandler != null).toBe(true);
        source.value(view(75_000, 100_000, 1000));
        expect(loading(source)).toBe(true);
        c.resolveAll();
        release();
        source.cleanup();
      });
    });

    // A line pairs x and y by alignment, and the Core streams a sample with an
    // alignment that never pairs with the one it reads back.
    describe("live line", () => {
      const START = TimeStamp.seconds(100);
      const PERIOD = TimeSpan.milliseconds(100);
      let core: SyntheticCore;
      let client: ReturnType<typeof telemTest.createClient>;
      beforeEach(() => {
        core = new SyntheticCore({
          rate: 10,
          start: START,
          now: START.add(TimeSpan.seconds(1)),
        });
        client = telemTest.createClient(core);
      });
      afterEach(async () => await client.feed.close());

      // stream subscribes to the channel as another view of it would, then streams the
      // given number of its samples.
      const stream = async (
        key: channel.Key,
        count: number,
      ): Promise<framer.Subscription> => {
        let streaming = false;
        const sub = client.feed.stream(
          (res) => {
            if ((res.get(key)?.length ?? 0) > 0) streaming = true;
          },
          [key],
        );
        // The feed opens its stream after a debounce, and drops samples written before.
        await expect
          .poll(() => {
            core.advance(PERIOD);
            return streaming;
          })
          .toBe(true);
        for (let i = 0; i < count; i++) {
          core.advance(PERIOD);
          await sleep.sleep(TimeSpan.ZERO);
        }
        return sub;
      };

      // createLine opens the x and y sources of a live line of the data channel as a
      // line plot does, with the index as the line's X channel or with no X channel.
      const createLine = (
        timeSpan: TimeSpan,
        indexAsX: boolean,
      ): [TiledChannelData, TiledChannelData] => {
        const yChannel = core.data[0].key;
        const xChannel = indexAsX ? core.index.key : 0;
        const create = (props: Partial<TiledChannelDataProps>): TiledChannelData =>
          new TiledChannelData(
            client,
            {
              channel: yChannel,
              window: dynamicWindow(timeSpan),
              xChannel,
              yChannel,
              ...props,
            },
            undefined,
            core.now,
          );
        const x = create(
          indexAsX ? { channel: xChannel } : { useIndexOfChannel: true },
        );
        return [x, create({})];
      };

      const sample = (t: TimeStamp): number =>
        Math.round(t.span(START).seconds / PERIOD.seconds);

      const samples = (from: number, to: number): number[] =>
        Array.from({ length: to - from }, (_, i) => from + i);

      // drawn returns the index of every sample a line of x and y draws, in order.
      const drawn = (
        x: TiledChannelData,
        y: TiledChannelData,
        shown?: ValueProps,
      ): number[] =>
        line
          .buildDrawOperations(
            x.value(shown).data[1],
            y.value(shown).data[1],
            line.DEFAULT_OVERLAP_THRESHOLD,
          )
          .flatMap((op) =>
            Array.from({ length: op.count }, (_, i) => {
              const t = op.x.at(op.xOffset + i, true);
              assert(typeof t === "bigint");
              return sample(new TimeStamp(t));
            }),
          )
          .sort((a, b) => a - b);

      const STREAMING = [
        { streaming: "index", channels: () => [core.index.key], indexAsX: false },
        {
          streaming: "data channel",
          channels: () => [core.data[0].key],
          indexAsX: false,
        },
        {
          streaming: "index and then its data channel",
          channels: () => [core.index.key, core.data[0].key],
          indexAsX: false,
        },
        {
          streaming: "index, its X channel,",
          channels: () => [core.index.key],
          indexAsX: true,
        },
      ];

      it.each(STREAMING)(
        "should draw every sample of a line added while its $streaming streams",
        async ({ channels, indexAsX }) => {
          const subs: framer.Subscription[] = [];
          for (const key of channels()) subs.push(await stream(key, 20));
          const [x, y] = createLine(TimeSpan.seconds(30), indexAsX);
          await expect.poll(() => drawn(x, y)).toEqual(samples(0, sample(core.now())));
          x.cleanup();
          y.cleanup();
          subs.forEach((s) => s.close());
        },
      );

      // A paused view stays put while the line moves on, so it soon draws from tiles.
      it.each(STREAMING)(
        "should keep every sample of a paused line added while its $streaming streams",
        async ({ channels, indexAsX }) => {
          const subs: framer.Subscription[] = [];
          for (const key of channels()) subs.push(await stream(key, 200));
          const span = TimeSpan.seconds(2);
          const [x, y] = createLine(span, indexAsX);
          const end = core.now();
          const [from, to] = [sample(end.sub(span)), sample(end)];
          await expect
            .poll(() => drawn(x, y))
            .toEqual(expect.arrayContaining(samples(from, to)));
          const paused: ValueProps = {
            bounds: {
              lower: Number(end.sub(span).valueOf()),
              upper: Number(end.valueOf()),
            },
            points: 2000,
          };
          for (let i = 0; i < 10; i++) {
            core.advance(TimeSpan.seconds(1));
            x.value(paused);
            y.value(paused);
            await sleep.sleep(TimeSpan.milliseconds(20));
          }
          await expect
            .poll(() => drawn(x, y, paused).filter((i) => i >= from && i < to))
            .toEqual(samples(from, to));
          x.cleanup();
          y.cleanup();
          subs.forEach((s) => s.close());
        },
      );
    });
  });

  describe("disconnected", () => {
    it("should still create remote sources from a null-client factory", () => {
      const factory = createFactory(null);
      const source = factory.create(streamChannelValue({ channel: 1 }));
      expect(source).not.toBeNull();
    });

    it("should report a disconnected status once instead of throwing", () => {
      const statuses: cstatus.Crude[] = [];
      const scv = new StreamChannelValue(
        null,
        { channel: 1 },
        { onStatusChange: (s) => statuses.push(s) },
      );
      expect(scv.value().data).toBe(NaN);
      expect(statuses).toHaveLength(1);
      expect(statuses[0].variant).toEqual("warning");
      // A repeated read must not re-report: the source is rebuilt on reconnect.
      expect(scv.value().data).toBe(NaN);
      expect(statuses).toHaveLength(1);
    });

    it("should clear loading for a disconnected TiledChannelData source", () => {
      const statuses: cstatus.Crude[] = [];
      const source = new TiledChannelData(
        null,
        {
          channel: 1,
          window: staticWindow(new TimeRange(TimeStamp.SECOND, TimeStamp.seconds(10))),
        },
        { onStatusChange: (s) => statuses.push(s) },
      );
      expect(loading(source)).toBe(false);
      expect(source.value().data[1]).toHaveLength(0);
      expect(statuses).toHaveLength(1);
      expect(statuses[0].variant).toEqual("warning");
    });

    it("should return empty data from a disconnected TiledChannelData source", () => {
      const statuses: cstatus.Crude[] = [];
      const source = new TiledChannelData(
        null,
        { window: staticWindow(TimeRange.MAX), channel: 1 },
        { onStatusChange: (s) => statuses.push(s) },
      );
      const [b, data] = source.value().data;
      expect(b).toStrictEqual(bounds.INVALID);
      expect(data).toHaveLength(0);
      expect(statuses).toEqual([DISCONNECTED_STATUS]);
    });
  });
});

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, Unreachable } from "@synnaxlabs/freighter";
import { DataType, id, Rate, Series, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { type channel } from "@/channel";
import { Frame } from "@/framer/frame";
import {
  HardenedStreamer,
  ObservableStreamer,
  type Streamer,
  streamerConfigZ,
} from "@/framer/streamer";
import { newVirtualChannel } from "@/testutil/channels";
import { createTestClient } from "@/testutil/client";

const client = createTestClient();

describe("Streamer", () => {
  describe("standard", () => {
    test("happy path", async () => {
      const ch = await newVirtualChannel(client);
      const streamer = await client.openStreamer(ch.key);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: ch.key,
      });
      try {
        await writer.write(ch.key, new Float64Array([1, 2, 3]));
      } finally {
        await writer.close();
      }
      const d = await streamer.read();
      expect(Array.from(d.get(ch.key))).toEqual([1, 2, 3]);
    });
    test("open with config", async () => {
      const ch = await newVirtualChannel(client);
      await expect(client.openStreamer({ channels: ch.key })).resolves.not.toThrow();
    });
    it("should not throw an error when the streamer is opened with zero channels", async () => {
      await expect(
        (async () => {
          const s = await client.openStreamer([]);
          s.close();
        })(),
      ).resolves.not.toThrow();
    });
    it("should throw an error when the streamer is opened with a channel that does not exist", async () => {
      await expect(client.openStreamer([5678])).rejects.toThrow("not found");
    });
    describe("downsampling", () => {
      test("downsample factor of 1", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          downsampleFactor: 1,
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
        });
        try {
          await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5]));
        } finally {
          await writer.close();
        }
        const d = await streamer.read();
        expect(Array.from(d.get(ch.key))).toEqual([1, 2, 3, 4, 5]);
      });
      test("downsample factor of 2", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          downsampleFactor: 2,
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
        });
        try {
          await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10]));
        } finally {
          await writer.close();
        }
        const d = await streamer.read();
        expect(Array.from(d.get(ch.key))).toEqual([1, 3, 5, 7, 9]);
      });
      test("downsample factor of 10", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          downsampleFactor: 10,
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
        });
        try {
          await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10]));
        } finally {
          await writer.close();
        }
        const d = await streamer.read();
        expect(Array.from(d.get(ch.key))).toEqual([1]);
      });
    });

    describe("throttling", () => {
      test("throttle at 60Hz", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          throttleRate: 60,
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
        });
        try {
          const startTime = Date.now();
          // Write data rapidly
          for (let i = 0; i < 10; i++) {
            await writer.write(ch.key, new Float64Array([i]));
            await sleep.sleep(TimeSpan.milliseconds(5));
          }

          // Read frames - should be throttled
          const receivedFrames: Frame[] = [];
          const timeout = Date.now() + 500;
          while (Date.now() < timeout)
            try {
              const frame = await Promise.race([
                streamer.read(),
                sleep.sleep(TimeSpan.milliseconds(100)).then(() => null),
              ]);
              if (frame) receivedFrames.push(frame);
              else break;
            } catch {
              break;
            }

          expect(receivedFrames.length).toBeGreaterThan(0);
          const elapsed = Date.now() - startTime;
          // Should take at least the throttle period
          expect(elapsed).toBeGreaterThanOrEqual(16); // ~1/60Hz
        } finally {
          await writer.close();
          streamer.close();
        }
      });

      test("no throttling with rate of 0", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          throttleRate: 0,
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
        });
        try {
          await writer.write(ch.key, new Float64Array([1, 2, 3]));
          const d = await streamer.read();
          expect(Array.from(d.get(ch.key))).toEqual([1, 2, 3]);
        } finally {
          await writer.close();
          streamer.close();
        }
      });

      test("combine throttling and downsampling", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          downsampleFactor: 2,
          throttleRate: 10,
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
        });
        try {
          await writer.write(ch.key, new Float64Array([1, 2, 3, 4, 5, 6]));
          const d = await streamer.read();
          // Should be downsampled to [1, 3, 5] and throttled
          expect(Array.from(d.get(ch.key))).toEqual([1, 3, 5]);
        } finally {
          await writer.close();
          streamer.close();
        }
      });
    });

    describe("calculations", () => {
      test("basic calculated channel streaming", async () => {
        // Create a timestamp index channel
        const timeChannel = await client.channels.create({
          name: id.create(),
          isIndex: true,
          dataType: DataType.TIMESTAMP,
        });

        // Create source channels with the timestamp index
        const [channelA, channelB] = await client.channels.create([
          {
            name: id.create(),
            dataType: DataType.FLOAT64,
            index: timeChannel.key,
          },
          {
            name: id.create(),
            dataType: DataType.FLOAT64,
            index: timeChannel.key,
          },
        ]);

        // Create calculated channel that adds the two source channels
        const calcChannel = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT64,
          virtual: true,
          expression: `return ${channelA.name} + ${channelB.name}`,
        });

        // Set up streamer to listen for calculated results
        const streamer = await client.openStreamer(calcChannel.key);
        await sleep.sleep(TimeSpan.milliseconds(10));

        // Write test data
        const startTime = TimeStamp.now();
        const writer = await client.openWriter({
          start: startTime,
          channels: [timeChannel.key, channelA.key, channelB.key],
        });

        try {
          // Write test values - each source gets 2.5 so sum should be 5.0
          await writer.write({
            [timeChannel.key]: [startTime],
            [channelA.key]: new Float64Array([2.5]),
            [channelB.key]: new Float64Array([2.5]),
          });

          // Read from streamer
          const frame = await streamer.read();

          // Verify calculated results
          const calcData = Array.from(frame.get(calcChannel.key));
          expect(calcData).toEqual([5.0]);
        } finally {
          await writer.close();
          streamer.close();
        }
      });

      test("calculated channel with constant", async () => {
        // Create an index channel for timestamps
        const timeChannel = await client.channels.create({
          name: id.create(),
          isIndex: true,
          dataType: DataType.TIMESTAMP,
        });

        // Create base channel with index
        const baseChannel = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT64,
          index: timeChannel.key,
        });

        // Create calculated channel that adds 5
        const calcChannel = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT64,
          virtual: true,
          expression: `return ${baseChannel.name} + 5`,
        });

        const streamer = await client.openStreamer(calcChannel.key);
        await sleep.sleep(TimeSpan.milliseconds(20));

        const startTime = TimeStamp.now();
        const writer = await client.openWriter({
          start: startTime,
          channels: [timeChannel.key, baseChannel.key],
        });

        try {
          const timestamps = [
            startTime,
            new TimeStamp(startTime.valueOf() + BigInt(1000000000)),
            new TimeStamp(startTime.valueOf() + BigInt(2000000000)),
          ];

          await writer.write({
            [timeChannel.key]: timestamps,
            [baseChannel.key]: new Float64Array([1, 2, 3]),
          });

          const frame = await streamer.read();
          const calcData = Array.from(frame.get(calcChannel.key));
          expect(calcData).toEqual([6, 7, 8]); // Original values + 5
        } finally {
          await writer.close();
          streamer.close();
        }
      });

      test("calculated channel with multiple operations", async () => {
        // Create timestamp channel
        const timeChannel = await client.channels.create({
          name: id.create(),
          isIndex: true,
          dataType: DataType.TIMESTAMP,
        });

        // Create source channels
        const names = [id.create(), id.create()];
        const [channelA, channelB] = await client.channels.create([
          { name: names[0], dataType: DataType.FLOAT64, index: timeChannel.key },
          { name: names[1], dataType: DataType.FLOAT64, index: timeChannel.key },
        ]);

        // Create calculated channel with multiple operations
        const calcChannel = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT64,
          virtual: true,
          expression: `return (${names[0]} * 2) + (${names[1]} / 2)`,
        });

        const streamer = await client.openStreamer(calcChannel.key);
        await sleep.sleep(TimeSpan.milliseconds(5));

        const startTime = TimeStamp.now();
        const writer = await client.openWriter({
          start: startTime,
          channels: [timeChannel.key, channelA.key, channelB.key],
        });

        try {
          await writer.write({
            [timeChannel.key]: [startTime],
            [channelA.key]: new Float64Array([2.0]), // Will be multiplied by 2 = 4.0
            [channelB.key]: new Float64Array([4.0]), // Will be divided by 2 = 2.0
          });

          const frame = await streamer.read();
          const calcData = Array.from(frame.get(calcChannel.key));
          expect(calcData).toEqual([6.0]); // (2.0 * 2) + (4.0 / 2) = 4.0 + 2.0 = 6.0
        } finally {
          await writer.close();
          streamer.close();
        }
      });

      describe("legacy calculations", async () => {
        it("should correctly execute a calculation with a requires field", async () => {
          const timeChannel = await client.channels.create({
            name: id.create(),
            isIndex: true,
            dataType: DataType.TIMESTAMP,
          });

          const [channelA, channelB] = await client.channels.create([
            {
              name: id.create(),
              dataType: DataType.FLOAT64,
              index: timeChannel.key,
            },
            {
              name: id.create(),
              dataType: DataType.FLOAT64,
              index: timeChannel.key,
            },
          ]);

          const calcChannel = await client.channels.create({
            name: id.create(),
            dataType: DataType.FLOAT64,
            virtual: true,
            expression: `return ${channelA.name} + ${channelB.name}`,
          });

          const streamer = await client.openStreamer(calcChannel.key);
          await sleep.sleep(TimeSpan.milliseconds(10));

          const startTime = TimeStamp.now();
          const writer = await client.openWriter({
            start: startTime,
            channels: [timeChannel.key, channelA.key, channelB.key],
          });

          try {
            await writer.write({
              [timeChannel.key]: [startTime],
              [channelA.key]: new Float64Array([2.5]),
              [channelB.key]: new Float64Array([2.5]),
            });

            const frame = await streamer.read();

            const calcData = Array.from(frame.get(calcChannel.key));
            expect(calcData).toEqual([5.0]);
          } finally {
            await writer.close();
            streamer.close();
          }
        });
      });
    });
  });

  class MockStreamer implements Streamer {
    keys: channel.Key[] = [];
    updateMock = vi.fn();
    readMock = vi.fn();
    closeMock = vi.fn();
    responses: [Frame, Error | null][] = [];
    updateErrors: (Error | null)[] = [];

    update(channels: channel.Params): Promise<void> {
      if (this.updateErrors.length > 0) {
        const err = this.updateErrors.shift()!;
        if (err) throw err;
      }
      this.updateMock(channels);
      return Promise.resolve();
    }

    close(): void {
      this.closeMock();
    }

    async read(): Promise<Frame> {
      this.readMock();
      if (this.responses.length === 0) throw new EOF();
      const [frame, err] = this.responses.shift()!;
      if (err) throw err;
      return frame;
    }

    async next(): Promise<IteratorResult<Frame, any>> {
      try {
        const fr = await this.read();
        return { done: false, value: fr };
      } catch (err) {
        if (EOF.matches(err)) return { done: true, value: undefined };
        throw err;
      }
    }

    [Symbol.asyncIterator](): AsyncIterator<Frame, any, undefined> {
      return this;
    }
  }

  describe("hardened", () => {
    it("should correctly call the underlying streamer methods", async () => {
      const streamer = new MockStreamer();
      const openMock = vi.fn();
      const config = { channels: [1, 2, 3], useHighPerformanceCodec: true };
      const fr = new Frame({ 1: new Series([1]) });
      const hardened = await HardenedStreamer.open(
        async (cfg) => {
          openMock(cfg);
          const cfg_ = streamerConfigZ.parse(cfg);
          streamer.responses = [[fr, null]];
          streamer.keys = cfg_.channels as channel.Key[];
          return streamer;
        },
        { channels: [1, 2, 3] },
      );
      expect(hardened.keys).toEqual([1, 2, 3]);
      expect(openMock).toHaveBeenCalledWith({
        ...config,
        downsampleFactor: 1,
        throttleRate: new Rate(0),
      });
      await hardened.update([1, 2, 3]);
      expect(streamer.updateMock).toHaveBeenCalledWith([1, 2, 3]);
      const fr2 = await hardened.read();
      expect(streamer.readMock).toHaveBeenCalled();
      expect(fr2).toEqual(fr);
      hardened.close();
      expect(streamer.closeMock).toHaveBeenCalled();
    });

    it("should correctly iterate over the streamer", async () => {
      const streamer = new MockStreamer();
      const fr = new Frame({ 1: new Series([1]) });
      const fr2 = new Frame({ 1: new Series([2]) });
      streamer.responses = [
        [fr, null],
        [fr2, null],
      ];
      const hardened = await HardenedStreamer.open(async () => streamer, {
        channels: [1],
      });
      const first = await hardened.next();
      expect(first.value).toEqual(fr);
      const second = await hardened.next();
      expect(second.value).toEqual(fr2);
      const third = await hardened.next();
      expect(third.done).toBe(true);
      expect(streamer.readMock).toHaveBeenCalledTimes(3);
    });

    it("should try to re-open the streamer when read fails", async () => {
      const streamer1 = new MockStreamer();
      const streamer2 = new MockStreamer();
      const fr1 = new Frame({ 1: new Series([1]) });
      const fr2 = new Frame({ 1: new Series([2]) });
      streamer1.responses = [
        [fr1, null],
        [fr2, new Unreachable({ message: "cat" })],
      ];
      streamer2.responses = [[fr2, null]];
      let count = 0;
      const openerMock = vi.fn();
      const hardened = await HardenedStreamer.open(
        async () => {
          count++;
          openerMock();
          if (count === 1) return streamer1;
          return streamer2;
        },
        {
          channels: [1],
        },
      );
      const fr = await hardened.read();
      expect(streamer1.readMock).toHaveBeenCalledTimes(1);
      expect(fr).toEqual(fr1);
      const fr3 = await hardened.read();
      expect(fr3).toEqual(fr2);
      expect(streamer2.readMock).toHaveBeenCalledTimes(1);
      expect(openerMock).toHaveBeenCalledTimes(2);
    });

    it("should repeatedly try re-opening the streamer when read fails", async () => {
      const streamer1 = new MockStreamer();
      const streamer5 = new MockStreamer();
      const fr1 = new Frame({ 1: new Series([1]) });
      const fr5 = new Frame({ 1: new Series([4]) });
      streamer1.responses = [
        [fr1, null],
        [fr5, new Unreachable({ message: "cat" })],
      ];
      streamer5.responses = [[fr5, null]];
      const openerMock = vi.fn();
      let count = 0;
      const hardened = await HardenedStreamer.open(
        async () => {
          count++;
          openerMock();
          if (count === 1) return streamer1;
          if (count < 5) throw new Unreachable({ message: "very unreachable" });
          return streamer5;
        },
        { channels: [1] },
        { baseInterval: TimeSpan.milliseconds(1) },
      );
      const fr = await hardened.read();
      expect(fr).toEqual(fr1);
      const fr2 = await hardened.read();
      expect(fr2).toEqual(fr5);
      expect(openerMock).toHaveBeenCalledTimes(5);
    });

    it("should rethrow the error when the breaker exceeds the max retries", async () => {
      const streamer = new MockStreamer();
      const fr = new Frame({ 1: new Series([1]) });
      streamer.responses = [[fr, null]];
      const openerMock = vi.fn();
      await expect(
        HardenedStreamer.open(
          async () => {
            openerMock();
            throw new Unreachable({ message: "very unreachable" });
          },
          { channels: [1] },
          { maxRetries: 3, baseInterval: TimeSpan.milliseconds(1) },
        ),
      ).rejects.toThrow("very unreachable");
    });

    it("should retry update when the underlying streamer fails", async () => {
      const streamer1 = new MockStreamer();
      streamer1.updateErrors = [null, new Unreachable({ message: "cat" })];
      const streamer2 = new MockStreamer();
      const fr1 = new Frame({ 1: new Series([1]) });
      const fr2 = new Frame({ 1: new Series([2]) });
      streamer1.responses = [[fr1, null]];
      streamer2.responses = [[fr2, null]];
      let count = 0;
      const openerMock = vi.fn();
      const hardened = await HardenedStreamer.open(
        async () => {
          count++;
          openerMock();
          if (count === 1) return streamer1;
          return streamer2;
        },
        { channels: [1] },
      );

      await hardened.update([1, 2]);
      expect(streamer1.updateMock).toHaveBeenCalledWith([1, 2]);

      await hardened.update([2, 3]);
      expect(openerMock).toHaveBeenCalledTimes(2);
    });
  });

  describe("observable", () => {
    it("should notify observers when frames are received", async () => {
      const mockStreamer = new MockStreamer();
      const frame1 = new Frame({ 1: new Series([1, 2, 3]) });
      const frame2 = new Frame({ 1: new Series([4, 5, 6]) });

      mockStreamer.responses = [
        [frame1, null],
        [frame2, null],
      ];
      mockStreamer.keys = [1];

      const observable = new ObservableStreamer(mockStreamer);

      const receivedFrames: Frame[] = [];
      observable.onChange((frame) => {
        receivedFrames.push(frame);
      });

      await expect.poll(() => receivedFrames.length).toBe(2);
      expect(receivedFrames[0]).toEqual(frame1);
      expect(receivedFrames[1]).toEqual(frame2);

      await observable.close();
      expect(mockStreamer.closeMock).toHaveBeenCalled();
    });

    test("should apply transform function to frames", async () => {
      const mockStreamer = new MockStreamer();
      const frame1 = new Frame({ 1: new Series([1, 2, 3]) });
      const frame2 = new Frame({ 1: new Series([4, 5, 6]) });

      mockStreamer.responses = [
        [frame1, null],
        [frame2, null],
      ];
      mockStreamer.keys = [1];

      const transform = (frame: Frame): [number, true] | [null, false] => {
        try {
          const data = Array.from(frame.get(1));
          const firstValue = data[0] as number;
          return [firstValue, true];
        } catch {
          return [null, false];
        }
      };

      const observable = new ObservableStreamer(mockStreamer, transform);

      const receivedValues: number[] = [];
      observable.onChange((value) => {
        if (value !== null) receivedValues.push(value);
      });

      await expect.poll(() => receivedValues.length).toBe(2);
      expect(receivedValues[0]).toBe(1);
      expect(receivedValues[1]).toBe(4);

      await observable.close();
    });

    test("should handle multiple observers", async () => {
      const mockStreamer = new MockStreamer();
      const frame1 = new Frame({ 1: new Series([10, 20]) });

      mockStreamer.responses = [[frame1, null]];
      mockStreamer.keys = [1];

      const observable = new ObservableStreamer(mockStreamer);

      const observer1Results: Frame[] = [];
      const observer2Results: Frame[] = [];

      observable.onChange((frame) => {
        observer1Results.push(frame);
      });

      observable.onChange((frame) => {
        observer2Results.push(frame);
      });

      await expect.poll(() => observer1Results.length).toBe(1);
      expect(observer2Results).toHaveLength(1);
      expect(observer1Results[0]).toEqual(frame1);
      expect(observer2Results[0]).toEqual(frame1);

      await observable.close();
    });

    test("should update channels on underlying streamer", async () => {
      const mockStreamer = new MockStreamer();
      mockStreamer.keys = [1, 2];

      const observable = new ObservableStreamer(mockStreamer);

      await observable.update([3, 4, 5]);

      expect(mockStreamer.updateMock).toHaveBeenCalledWith([3, 4, 5]);

      await observable.close();
    });

    test("should handle empty frame stream gracefully", async () => {
      const mockStreamer = new MockStreamer();
      mockStreamer.keys = [1];

      const observable = new ObservableStreamer(mockStreamer);

      const receivedFrames: Frame[] = [];
      observable.onChange((frame) => {
        receivedFrames.push(frame);
      });

      await expect.poll(() => receivedFrames.length).toBe(0);
      expect(receivedFrames).toHaveLength(0);

      await observable.close();
    });

    test("should properly close and cleanup resources", async () => {
      const mockStreamer = new MockStreamer();
      const frame1 = new Frame({ 1: new Series([1]) });

      mockStreamer.responses = [[frame1, null]];
      mockStreamer.keys = [1];

      const observable = new ObservableStreamer(mockStreamer);

      const receivedFrames: Frame[] = [];
      observable.onChange((frame) => {
        receivedFrames.push(frame);
      });

      await observable.close();

      expect(mockStreamer.closeMock).toHaveBeenCalled();
    });
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF, Unreachable } from "@synnaxlabs/freighter";
import {
  DataType,
  errors,
  id,
  Series,
  sleep,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { type channel } from "@/channel";
import { AuthError, ExpiredTokenError, SessionEndedError } from "@/errors";
import { framer } from "@/framer";
import { AccessDeniedError, MissingLicenseError } from "@/index";
import {
  createProxiedTestClient,
  createTestClient,
  FAST_RETRY,
  newIndexedPair,
  newVirtualBoolChannel,
  newVirtualChannel,
  secondsLinspace,
} from "@/testutil";

const quiet = (): void => {};

const client = createTestClient();

const write = async (ch: channel.Channel, values: number[]): Promise<void> => {
  const writer = await client.openWriter({ start: TimeStamp.now(), channels: ch.key });
  try {
    await writer.write(ch.key, new Float64Array(values));
  } finally {
    await writer.close();
  }
};

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

    test("bool stream round-trip", async () => {
      const ch = await newVirtualBoolChannel(client);
      const streamer = await client.openStreamer(ch.key);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: ch.key,
      });
      const samples = [true, false, true, true, false, false, false, true, true];
      try {
        await writer.write(
          ch.key,
          new Series({ data: samples, dataType: DataType.BOOLEAN }),
        );
      } finally {
        await writer.close();
      }
      const d = await streamer.read();
      expect(Array.from(d.get(ch.key))).toEqual(samples);
    });
    test("should preserve non-zero time ranges through codec round-trip", async () => {
      const ch = await newVirtualChannel(client);
      const streamer = await client.openStreamer(ch.key);
      const writer = await client.openWriter({
        start: TimeStamp.now(),
        channels: ch.key,
      });
      const start = new TimeStamp(1000000000n);
      const end = new TimeStamp(6000000000n);
      try {
        const fr = new framer.Frame(
          [ch.key],
          [
            new Series({
              data: new Float64Array([1, 2, 3]),
              dataType: DataType.FLOAT64,
              timeRange: start.spanRange(TimeSpan.seconds(5)),
            }),
          ],
        );
        await writer.write(fr);
      } finally {
        await writer.close();
      }
      const d = await streamer.read();
      const series = d.series[0];
      expect(series.timeRange).toBeDefined();
      expect(series.timeRange?.start.valueOf()).toEqual(start.valueOf());
      expect(series.timeRange?.end.valueOf()).toEqual(end.valueOf());
    });
    test("should deliver series stamped with the index's time range", async () => {
      const channels = await newIndexedPair(client);
      const [index, data] = channels;
      const streamer = await client.openStreamer(channels.map((c) => c.key));
      const writer = await client.openWriter({ start: TimeStamp.seconds(1), channels });
      try {
        await writer.write({
          [index.key]: secondsLinspace(1, 3),
          [data.key]: new Float64Array([1, 2, 3]),
        });
      } finally {
        await writer.close();
      }
      const fr = await streamer.read();
      const expected = TimeStamp.seconds(1).range(
        TimeStamp.seconds(3).add(TimeSpan.nanoseconds(1)),
      );
      expect(fr.get(index.key).series[0].timeRange?.equals(expected)).toBe(true);
      expect(fr.get(data.key).series[0].timeRange?.equals(expected)).toBe(true);
      streamer.close();
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
    describe("update acks", () => {
      it("should deliver a write issued right after an acked update", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer([]);
        try {
          const frame = streamer.read();
          await streamer.update(ch.key, { waitForAck: true });
          const writer = await client.openWriter({
            start: TimeStamp.now(),
            channels: ch.key,
          });
          try {
            await writer.write(ch.key, new Float64Array([1, 2, 3]));
          } finally {
            await writer.close();
          }
          expect(Array.from((await frame).get(ch.key))).toEqual([1, 2, 3]);
        } finally {
          streamer.close();
        }
      });

      it("should ack an update that leaves the channels unchanged", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer(ch.key);
        const pending = streamer.read().catch((e: unknown) => e);
        try {
          await streamer.update(ch.key, { waitForAck: true });
          expect(streamer.keys).toEqual([ch.key]);
        } finally {
          streamer.close();
        }
        expect(EOF.matches(await pending)).toBe(true);
      });

      it("should stream samples written after an unchanged acked update", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer(ch.key);
        try {
          const frame = streamer.read();
          await streamer.update(ch.key, { waitForAck: true });
          await write(ch, [1, 2, 3]);
          expect(Array.from((await frame).get(ch.key))).toEqual([1, 2, 3]);
        } finally {
          streamer.close();
        }
      });

      it("should stream samples after an update to no channels and back", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer(ch.key);
        try {
          await streamer.update([]);
          const frame = streamer.read();
          await streamer.update(ch.key, { waitForAck: true });
          await write(ch, [1, 2, 3]);
          expect(Array.from((await frame).get(ch.key))).toEqual([1, 2, 3]);
        } finally {
          streamer.close();
        }
      });

      it("should resolve several acked updates that wait at once", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer(ch.key);
        try {
          const updates = [
            streamer.update(ch.key, { waitForAck: true }),
            streamer.update(ch.key, { waitForAck: true }),
          ];
          const pending = streamer.read().catch((e: unknown) => e);
          await Promise.all(updates);
          streamer.close();
          expect(EOF.matches(await pending)).toBe(true);
        } finally {
          streamer.close();
        }
      });

      it("should reject a pending acked update when the streamer closes", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer([]);
        const update = streamer
          .update(ch.key, { waitForAck: true })
          .catch((e: unknown) => e);
        await expect.poll(() => streamer.keys).toEqual([ch.key]);
        streamer.close();
        expect(EOF.matches(await update)).toBe(true);
      });

      it("should end a read with EOF when an ack arrives after the streamer closes", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer([]);
        const pending = streamer.read().catch((e: unknown) => e);
        const first = streamer.update(ch.key, { waitForAck: true });
        const second = streamer
          .update(ch.key, { waitForAck: true })
          .catch((e: unknown) => e);
        await first;
        // The Core applied the second update before the close reached it, so its ack
        // arrives after the close.
        streamer.close();
        await second;
        expect(EOF.matches(await pending)).toBe(true);
      });
    });
    describe("excludeGroups", () => {
      test("should filter out frames from a writer whose group matches", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          excludeGroups: [42],
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
          controlSubject: { name: "grouped", key: id.create(), group: 42 },
        });
        try {
          await writer.write(ch.key, new Float64Array([1, 2, 3]));
          const result = await Promise.race([
            streamer.read().then((f) => f),
            sleep.sleep(TimeSpan.milliseconds(500)).then(() => null),
          ]);
          expect(result).toBeNull();
        } finally {
          await writer.close();
          streamer.close();
        }
      });

      test("should deliver frames from a writer whose group does not match", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          excludeGroups: [99],
        });
        const writer = await client.openWriter({
          start: TimeStamp.now(),
          channels: ch.key,
          controlSubject: { name: "grouped", key: id.create(), group: 42 },
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

      test("should deliver frames from a writer with group 0 even if 0 is excluded", async () => {
        const ch = await newVirtualChannel(client);
        const streamer = await client.openStreamer({
          channels: ch.key,
          excludeGroups: [0],
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
    });

    describe("downsampling", () => {
      // The wire field is unsigned, so a factor the schema lets through would be
      // reinterpreted by the Core instead of rejected.
      test.each([-1, 1.5, 2 ** 32])("rejects a factor of %s", (factor) => {
        expect(() =>
          framer.streamerConfigZ.parse({ channels: [], downsampleFactor: factor }),
        ).toThrow();
      });
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
          for (let i = 0; i < 10; i++) {
            await writer.write(ch.key, new Float64Array([i]));
            await sleep.sleep(TimeSpan.milliseconds(5));
          }

          const receivedFrames: framer.Frame[] = [];
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
          expect(Array.from(d.get(ch.key))).toEqual([1, 3, 5]);
        } finally {
          await writer.close();
          streamer.close();
        }
      });
    });

    describe("calculations", () => {
      test("basic calculated channel streaming", async () => {
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
          // Write test values - each source gets 2.5 so sum should be 5.0
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

      test("calculated channel with constant", async () => {
        const timeChannel = await client.channels.create({
          name: id.create(),
          isIndex: true,
          dataType: DataType.TIMESTAMP,
        });

        const baseChannel = await client.channels.create({
          name: id.create(),
          dataType: DataType.FLOAT64,
          index: timeChannel.key,
        });

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
        const timeChannel = await client.channels.create({
          name: id.create(),
          isIndex: true,
          dataType: DataType.TIMESTAMP,
        });

        const names = [id.create(), id.create()];
        const [channelA, channelB] = await client.channels.create([
          { name: names[0], dataType: DataType.FLOAT64, index: timeChannel.key },
          { name: names[1], dataType: DataType.FLOAT64, index: timeChannel.key },
        ]);

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

  class MockStreamer implements framer.Streamer {
    keys: channel.Key[] = [];
    updateMock = vi.fn();
    readMock = vi.fn();
    closeMock = vi.fn();
    responses: [framer.Frame, Error | null][] = [];
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

    async read(): Promise<framer.Frame> {
      this.readMock();
      if (this.responses.length === 0) throw new EOF();
      const [frame, err] = this.responses.shift()!;
      if (err) throw err;
      return frame;
    }

    async next(): Promise<IteratorResult<framer.Frame, any>> {
      try {
        const fr = await this.read();
        return { done: false, value: fr };
      } catch (err) {
        if (EOF.matches(err)) return { done: true, value: undefined };
        throw errors.fromUnknown(err);
      }
    }

    [Symbol.asyncIterator](): AsyncIterator<framer.Frame, any, undefined> {
      return this;
    }
  }

  // Pins silent-death detection on /frame/stream: the Core emits keep-alive responses
  // on request, and the client fails a silent read with Unreachable instead of hanging
  // forever, which the hardened streamer turns into a reconnect.
  describe("keep alive", () => {
    // The Core's minimum accepted cadence, so the specs pin the fastest detection a
    // client can actually get.
    const KEEP_ALIVE = TimeSpan.seconds(2);
    // KEEP_ALIVE_DEADLINE_FACTOR x KEEP_ALIVE: how long a read may stay silent once
    // armed.
    const DEADLINE = TimeSpan.seconds(6);
    // One keep-alive interval plus slack, so the client has seen one and armed.
    const ARMED = TimeSpan.milliseconds(2500);
    // Covers a full deadline trip plus the reconnect that follows it.
    const POLL = { timeout: DEADLINE.milliseconds * 2 };

    it("should keep keep-alives out of the frames a streamer serves", async () => {
      const ch = await newVirtualChannel(client);
      const streamer = await client.openStreamer({
        channels: ch.key,
        keepAlive: KEEP_ALIVE,
      });
      try {
        // Let several keep-alives queue up so the read has to skip past them.
        await sleep.sleep(KEEP_ALIVE.mult(2.5));
        await write(ch, [1, 2, 3]);
        const frame = await streamer.read();
        expect(Array.from(frame.get(ch.key))).toEqual([1, 2, 3]);
      } finally {
        streamer.close();
      }
    });

    it("should reject an interval below the Core's minimum", async () => {
      const ch = await newVirtualChannel(client);
      await expect(
        client.openStreamer({ channels: ch.key, keepAlive: TimeSpan.seconds(1) }),
      ).rejects.toThrow("keep_alive: must be greater than or equal to 2s");
    });

    it("should reject a silent read with Unreachable after the deadline", async () => {
      const { proxy, client: proxied } = await createProxiedTestClient({
        onRetry: quiet,
      });
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: ch.key,
        keepAlive: KEEP_ALIVE,
      });
      // Receive at least one keep-alive so the deadline is armed.
      await sleep.sleep(ARMED);
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      const started = performance.now();
      await expect(streamer.read()).rejects.toSatisfy(
        (exc) =>
          Unreachable.matches(exc) &&
          exc.message === `streamer received no response for ${DEADLINE.toString()}`,
      );
      // The deadline must actually elapse: an instant rejection would mean the
      // deadline armed wrong, not that silence was detected.
      expect(performance.now() - started).toBeGreaterThanOrEqual(
        DEADLINE.milliseconds - 50,
      );
      streamer.close();
    }, 30_000);

    it("should reject an acked update that follows a failed read", async () => {
      const { proxy, client: proxied } = await createProxiedTestClient({
        onRetry: quiet,
      });
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: [],
        keepAlive: KEEP_ALIVE,
      });
      await sleep.sleep(ARMED);
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      const err = await streamer.read().catch((e: unknown) => e);
      expect(Unreachable.matches(err)).toBe(true);
      await expect(streamer.update(ch.key, { waitForAck: true })).rejects.toBe(err);
      streamer.close();
    }, 30_000);

    it("should reconnect and resume streaming after a silent death", async () => {
      const { proxy, client: proxied } = await createProxiedTestClient({
        onRetry: quiet,
      });
      const ch = await newVirtualChannel(client);
      const onDrop = vi.fn();
      const onReopen = vi.fn();
      const hardened = await framer.HardenedStreamer.open(
        async (cfg) => await proxied.openStreamer(cfg),
        { channels: ch.key, keepAlive: KEEP_ALIVE },
        { breaker: FAST_RETRY, onReopen, onDrop, onRetry: quiet },
      );
      try {
        await write(ch, [1]);
        expect(Array.from((await hardened.read()).get(ch.key))).toEqual([1]);
        await sleep.sleep(ARMED);
        expect(proxy.blackholeStreams()).toBeGreaterThan(0);
        // The proxy still forwards new connections, so the deadline trip inside this
        // read reconnects and the read stays pending for the next frame.
        const pending = hardened.read();
        await expect.poll(() => onDrop.mock.calls.length, POLL).toBe(1);
        expect(Unreachable.matches(onDrop.mock.calls[0][0])).toBe(true);
        await expect.poll(() => onReopen.mock.calls.length, POLL).toBe(1);
        await write(ch, [2]);
        expect(Array.from((await pending).get(ch.key))).toEqual([2]);
      } finally {
        hardened.close();
      }
    }, 30_000);

    it("should reject a pending update ack when the read fails", async () => {
      const { proxy, client: proxied } = await createProxiedTestClient({
        onRetry: quiet,
      });
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: [],
        keepAlive: KEEP_ALIVE,
      });
      await sleep.sleep(ARMED);
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      const read = streamer.read().catch((e: unknown) => e);
      const update = streamer
        .update(ch.key, { waitForAck: true })
        .catch((e: unknown) => e);
      const err = await read;
      expect(Unreachable.matches(err)).toBe(true);
      expect(await update).toBe(err);
      streamer.close();
    }, 30_000);

    it("should leave a silent read pending when keep-alive is disabled", async () => {
      const { proxy, client: proxied } = await createProxiedTestClient({
        onRetry: quiet,
      });
      const ch = await newVirtualChannel(client);
      const streamer = await proxied.openStreamer({
        channels: ch.key,
        keepAlive: TimeSpan.ZERO,
      });
      await sleep.sleep(TimeSpan.milliseconds(250));
      expect(proxy.blackholeStreams()).toBeGreaterThan(0);
      // Without keep-alives the deadline never arms, which is also how a client
      // behaves against a Core that predates them: the read hangs, as before.
      const pending = streamer.read();
      pending.catch(() => {});
      const result = await Promise.race([
        pending.then(() => "settled"),
        sleep.sleep(TimeSpan.seconds(1)).then(() => "pending"),
      ]);
      expect(result).toEqual("pending");
      streamer.close();
    });
  });

  describe("hardened", () => {
    it("should correctly call the underlying streamer methods", async () => {
      const streamer = new MockStreamer();
      const openMock = vi.fn();
      const config = { channels: [1, 2, 3] };
      const fr = new framer.Frame({ 1: new Series([1]) });
      const hardened = await framer.HardenedStreamer.open(
        async (cfg) => {
          openMock(cfg);
          const cfg_ = framer.streamerConfigZ.parse(cfg);
          streamer.responses = [[fr, null]];
          streamer.keys = cfg_.channels as channel.Key[];
          return streamer;
        },
        { channels: [1, 2, 3] },
      );
      expect(hardened.keys).toEqual([1, 2, 3]);
      expect(openMock).toHaveBeenCalledWith(config);
      await hardened.update([1, 2, 3]);
      expect(streamer.updateMock).toHaveBeenCalledWith([1, 2, 3]);
      const fr2 = await hardened.read();
      expect(streamer.readMock).toHaveBeenCalled();
      expect(fr2).toEqual(fr);
      hardened.close();
      expect(streamer.closeMock).toHaveBeenCalled();
    });

    it("should correctly iterate over the streamer and end after close", async () => {
      const streamer = new MockStreamer();
      const fr = new framer.Frame({ 1: new Series([1]) });
      const fr2 = new framer.Frame({ 1: new Series([2]) });
      streamer.responses = [
        [fr, null],
        [fr2, null],
      ];
      const hardened = await framer.HardenedStreamer.open(async () => streamer, {
        channels: [1],
      });
      const first = await hardened.next();
      expect(first.value).toEqual(fr);
      const second = await hardened.next();
      expect(second.value).toEqual(fr2);
      hardened.close();
      const third = await hardened.next();
      expect(third.done).toBe(true);
      expect(streamer.readMock).toHaveBeenCalledTimes(2);
    });

    it("should reconnect when the server ends the stream unexpectedly", async () => {
      const streamer1 = new MockStreamer();
      const streamer2 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      const fr2 = new framer.Frame({ 1: new Series([2]) });
      // streamer1 EOFs after one frame without the client asking for it
      streamer1.responses = [[fr1, null]];
      streamer2.responses = [[fr2, null]];
      const onDrop = vi.fn();
      let count = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          count++;
          if (count === 1) return streamer1;
          return streamer2;
        },
        { channels: [1] },
        { onDrop },
      );
      expect(await hardened.read()).toEqual(fr1);
      expect(await hardened.read()).toEqual(fr2);
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(count).toBe(2);
    });

    it("should back off when reopened streams die immediately", async () => {
      let opens = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          opens++;
          // no responses: every stream EOFs on its first read
          return new MockStreamer();
        },
        { channels: [1] },
        { breaker: { baseInterval: TimeSpan.milliseconds(30), jitter: 0 } },
      );
      const pending = hardened.read().catch((e: unknown) => e);
      await new Promise((resolve) => setTimeout(resolve, 150));
      // a hot loop would dial hundreds of times in this window
      expect(opens).toBeLessThan(6);
      hardened.close();
      expect(EOF.matches(await pending)).toBe(true);
    });

    it("should interrupt a pending reconnect backoff on close", async () => {
      const streamer1 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      streamer1.responses = [
        [fr1, null],
        [fr1, new Unreachable({ message: "down" })],
      ];
      let opens = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          opens++;
          if (opens === 1) return streamer1;
          throw new Unreachable({ message: "still down" });
        },
        { channels: [1] },
        { breaker: { baseInterval: TimeSpan.seconds(60) } },
      );
      expect(await hardened.read()).toEqual(fr1);
      // drops, fails one reopen, then sleeps out the 60s backoff
      const pending = hardened.read().catch((e: unknown) => e);
      await new Promise((resolve) => setTimeout(resolve, 20));
      const start = performance.now();
      hardened.close();
      expect(EOF.matches(await pending)).toBe(true);
      expect(performance.now() - start).toBeLessThan(1000);
      const settled = opens;
      await new Promise((resolve) => setTimeout(resolve, 50));
      expect(opens).toBe(settled);
    });

    it("should not report a failed reconnect when closed during the backoff", async () => {
      const streamer1 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      streamer1.responses = [
        [fr1, null],
        [fr1, new Unreachable({ message: "down" })],
      ];
      let opens = 0;
      const onRetry = vi.fn<(error: Error) => void>();
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          opens++;
          if (opens === 1) return streamer1;
          throw new Unreachable({ message: "still down" });
        },
        { channels: [1] },
        {
          breaker: {
            baseInterval: TimeSpan.seconds(60),
            maxInterval: TimeSpan.milliseconds(5),
          },
          onRetry,
        },
      );
      expect(await hardened.read()).toEqual(fr1);
      // maxInterval also sets stableAfter, so a stream older than it counts as
      // healthy: the drop skips the pre-loop backoff and reaches the retry
      // loop's own, which still sleeps the full baseInterval on its first wait.
      await new Promise((resolve) => setTimeout(resolve, 20));
      const pending = hardened.read().catch((e: unknown) => e);
      await expect.poll(() => opens).toBe(2);
      hardened.close();
      expect(EOF.matches(await pending)).toBe(true);
      expect(onRetry).not.toHaveBeenCalled();
    });

    it("should not throw when closed while reconnecting", async () => {
      const streamer1 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      streamer1.responses = [
        [fr1, null],
        [fr1, new Unreachable({ message: "down" })],
      ];
      let opens = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          opens++;
          if (opens === 1) return streamer1;
          throw new Unreachable({ message: "still down" });
        },
        { channels: [1] },
        { breaker: { baseInterval: TimeSpan.seconds(60) } },
      );
      expect(await hardened.read()).toEqual(fr1);
      const pending = hardened.read().catch((e: unknown) => e);
      await new Promise((resolve) => setTimeout(resolve, 20));
      expect(() => hardened.close()).not.toThrow();
      expect(() => hardened.close()).not.toThrow();
      expect(EOF.matches(await pending)).toBe(true);
    });

    it("should try to re-open the streamer when read fails", async () => {
      const streamer1 = new MockStreamer();
      const streamer2 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      const fr2 = new framer.Frame({ 1: new Series([2]) });
      streamer1.responses = [
        [fr1, null],
        [fr2, new Unreachable({ message: "cat" })],
      ];
      streamer2.responses = [[fr2, null]];
      let count = 0;
      const openerMock = vi.fn();
      const hardened = await framer.HardenedStreamer.open(
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
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      const fr5 = new framer.Frame({ 1: new Series([4]) });
      streamer1.responses = [
        [fr1, null],
        [fr5, new Unreachable({ message: "cat" })],
      ];
      streamer5.responses = [[fr5, null]];
      const openerMock = vi.fn();
      const onRetry = vi.fn<(error: Error) => void>();
      let count = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          count++;
          openerMock();
          if (count === 1) return streamer1;
          if (count < 5) throw new Unreachable({ message: "very unreachable" });
          return streamer5;
        },
        { channels: [1] },
        { breaker: { baseInterval: TimeSpan.milliseconds(1) }, onRetry },
      );
      const fr = await hardened.read();
      expect(fr).toEqual(fr1);
      const fr2 = await hardened.read();
      expect(fr2).toEqual(fr5);
      expect(openerMock).toHaveBeenCalledTimes(5);
      expect(onRetry).toHaveBeenCalledTimes(3);
    });

    it("should rethrow the error when the breaker exceeds the max retries", async () => {
      const streamer = new MockStreamer();
      const fr = new framer.Frame({ 1: new Series([1]) });
      streamer.responses = [[fr, null]];
      const openerMock = vi.fn();
      const onRetry = vi.fn<(error: Error) => void>();
      await expect(
        framer.HardenedStreamer.open(
          async () => {
            openerMock();
            throw new Unreachable({ message: "very unreachable" });
          },
          { channels: [1] },
          {
            breaker: { maxRetries: 3, baseInterval: TimeSpan.milliseconds(1) },
            onRetry,
          },
        ),
      ).rejects.toThrow("very unreachable");
      expect(onRetry).toHaveBeenCalledTimes(3);
    });

    it("should retry an open the auth middleware could not refresh", async () => {
      const openerMock = vi.fn();
      const onRetry = vi.fn<(error: Error) => void>();
      await expect(
        framer.HardenedStreamer.open(
          async () => {
            openerMock();
            throw new ExpiredTokenError("token expired");
          },
          { channels: [1] },
          {
            breaker: { maxRetries: 2, baseInterval: TimeSpan.milliseconds(1) },
            onRetry,
          },
        ),
      ).rejects.toThrow(ExpiredTokenError);
      expect(openerMock.mock.calls.length).toBeGreaterThan(1);
      expect(onRetry).toHaveBeenCalled();
    });

    it("should give up on the first denial instead of retrying", async () => {
      const openerMock = vi.fn();
      await expect(
        framer.HardenedStreamer.open(
          async () => {
            openerMock();
            throw new AccessDeniedError("no permission to stream");
          },
          { channels: [1] },
          { breaker: { maxRetries: 3, baseInterval: TimeSpan.milliseconds(1) } },
        ),
      ).rejects.toThrow(AccessDeniedError);
      expect(openerMock).toHaveBeenCalledTimes(1);
    });

    it("should give up on rejected credentials instead of retrying", async () => {
      const openerMock = vi.fn();
      await expect(
        framer.HardenedStreamer.open(
          async () => {
            openerMock();
            throw new AuthError("invalid credentials");
          },
          { channels: [1] },
          { breaker: { maxRetries: 3, baseInterval: TimeSpan.milliseconds(1) } },
        ),
      ).rejects.toThrow(AuthError);
      expect(openerMock).toHaveBeenCalledTimes(1);
    });

    it("should keep retrying an ended session", async () => {
      const openerMock = vi.fn();
      await expect(
        framer.HardenedStreamer.open(
          async () => {
            openerMock();
            throw new SessionEndedError("session ended");
          },
          { channels: [1] },
          {
            breaker: { maxRetries: 2, baseInterval: TimeSpan.milliseconds(1) },
            onRetry: quiet,
          },
        ),
      ).rejects.toThrow(SessionEndedError);
      expect(openerMock).toHaveBeenCalledTimes(3);
    });

    it("should give up on the first license refusal instead of retrying", async () => {
      const openerMock = vi.fn();
      await expect(
        framer.HardenedStreamer.open(
          async () => {
            openerMock();
            throw new MissingLicenseError("No license is active on this Core");
          },
          { channels: [1] },
          { breaker: { maxRetries: 3, baseInterval: TimeSpan.milliseconds(1) } },
        ),
      ).rejects.toThrow(MissingLicenseError);
      expect(openerMock).toHaveBeenCalledTimes(1);
    });

    it("should fire onDrop when the stream fails and onReopen after recovery", async () => {
      const streamer1 = new MockStreamer();
      const streamer2 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      const fr2 = new framer.Frame({ 1: new Series([2]) });
      streamer1.responses = [
        [fr1, null],
        [fr2, new Unreachable({ message: "cat" })],
      ];
      streamer2.responses = [[fr2, null]];
      const onDrop = vi.fn();
      const onReopen = vi.fn();
      let count = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          count++;
          if (count === 1) return streamer1;
          return streamer2;
        },
        { channels: [1] },
        { onReopen, onDrop },
      );
      expect(onDrop).not.toHaveBeenCalled();
      expect(onReopen).not.toHaveBeenCalled();
      await hardened.read();
      await hardened.read();
      expect(onDrop).toHaveBeenCalledTimes(1);
      expect(onDrop.mock.calls[0][0]).toBeInstanceOf(Error);
      expect(onReopen).toHaveBeenCalledTimes(1);
    });

    it("should retry update when the underlying streamer fails", async () => {
      const streamer1 = new MockStreamer();
      streamer1.updateErrors = [null, new Unreachable({ message: "cat" })];
      const streamer2 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      const fr2 = new framer.Frame({ 1: new Series([2]) });
      streamer1.responses = [[fr1, null]];
      streamer2.responses = [[fr2, null]];
      let count = 0;
      const openerMock = vi.fn();
      const hardened = await framer.HardenedStreamer.open(
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

    it("should rethrow a definitive update rejection and keep its channels", async () => {
      const rejected = new AccessDeniedError("no access to channel 2");
      const streamer1 = new MockStreamer();
      streamer1.updateErrors = [rejected];
      streamer1.responses = [
        [new framer.Frame(), new Unreachable({ message: "down" })],
      ];
      const streamer2 = new MockStreamer();
      const fr = new framer.Frame({ 1: new Series([1]) });
      streamer2.responses = [[fr, null]];
      const opened: channel.Params[] = [];
      const onDrop = vi.fn();
      const hardened = await framer.HardenedStreamer.open(
        async (config) => {
          opened.push(framer.streamerConfigZ.parse(config).channels);
          return opened.length === 1 ? streamer1 : streamer2;
        },
        { channels: [1] },
        { onDrop, breaker: FAST_RETRY, onRetry: quiet },
      );
      await expect(hardened.update([1, 2])).rejects.toThrow(rejected);
      expect(onDrop).not.toHaveBeenCalled();
      expect(await hardened.read()).toBe(fr);
      expect(opened).toEqual([[1], [1]]);
      hardened.close();
    });

    it("should retry an acked update on the reopened stream", async () => {
      const streamer1 = new MockStreamer();
      streamer1.updateErrors = [new Unreachable({ message: "down" })];
      const streamer2 = new MockStreamer();
      const update2 = vi.spyOn(streamer2, "update");
      let opens = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => (++opens === 1 ? streamer1 : streamer2),
        { channels: [1] },
        { breaker: FAST_RETRY, onRetry: quiet },
      );
      await hardened.update([1, 2], { waitForAck: true });
      expect(update2).toHaveBeenCalledWith([1, 2], { waitForAck: true });
      hardened.close();
    });

    it("should reconnect when the Core ends the stream during an update", async () => {
      const streamer1 = new MockStreamer();
      streamer1.updateErrors = [new EOF()];
      const streamer2 = new MockStreamer();
      const update2 = vi.spyOn(streamer2, "update");
      let opens = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => (++opens === 1 ? streamer1 : streamer2),
        { channels: [1] },
        { breaker: FAST_RETRY, onRetry: quiet },
      );
      await hardened.update([1, 2], { waitForAck: true });
      expect(update2).toHaveBeenCalledWith([1, 2], { waitForAck: true });
      expect(opens).toBe(2);
      hardened.close();
    });

    it("should not leak a stream when an update races a reconnect", async () => {
      const streamer1 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      streamer1.responses = [
        [fr1, null],
        [fr1, new Unreachable({ message: "down" })],
      ];
      const streamer2 = new MockStreamer();
      streamer2.read = async () => await new Promise<never>(() => {});
      const pendingOpens: ((s: framer.Streamer) => void)[] = [];
      let opens = 0;
      const onDrop = vi.fn();
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          opens++;
          if (opens === 1) return streamer1;
          return await new Promise<framer.Streamer>((resolve) =>
            pendingOpens.push(resolve),
          );
        },
        { channels: [1] },
        { breaker: { maxInterval: TimeSpan.milliseconds(5), jitter: 0 }, onDrop },
      );
      expect(await hardened.read()).toEqual(fr1);
      // Age the stream past stableAfter so the reconnect skips the backoff sleep.
      await sleep.sleep(TimeSpan.milliseconds(10));
      void hardened.read().catch(() => {});
      await expect.poll(() => opens).toBe(2);
      const updateP = hardened.update([1, 2]);
      await sleep.sleep(TimeSpan.milliseconds(20));
      expect(opens).toBe(2);
      pendingOpens[0](streamer2);
      await updateP;
      expect(streamer2.updateMock).toHaveBeenCalledWith([1, 2]);
      expect(opens).toBe(2);
      expect(onDrop).toHaveBeenCalledTimes(1);
      hardened.close();
      expect(streamer1.closeMock).toHaveBeenCalled();
      expect(streamer2.closeMock).toHaveBeenCalled();
    });

    it("should reject every joiner when a shared reconnect fails", async () => {
      const streamer1 = new MockStreamer();
      const fr1 = new framer.Frame({ 1: new Series([1]) });
      streamer1.responses = [
        [fr1, null],
        [fr1, new Unreachable({ message: "down" })],
      ];
      const pendingOpens: ((e: Error) => void)[] = [];
      let opens = 0;
      const hardened = await framer.HardenedStreamer.open(
        async () => {
          opens++;
          if (opens === 1) return streamer1;
          return await new Promise<framer.Streamer>((_, reject) =>
            pendingOpens.push(reject),
          );
        },
        { channels: [1] },
        { breaker: { maxInterval: TimeSpan.milliseconds(5), jitter: 0 } },
      );
      expect(await hardened.read()).toEqual(fr1);
      await sleep.sleep(TimeSpan.milliseconds(10));
      const readP = hardened.read().catch((e: unknown) => e);
      await expect.poll(() => opens).toBe(2);
      const updateP = hardened.update([1, 2]).catch((e: unknown) => e);
      const denied = new AccessDeniedError("no permission to stream");
      pendingOpens[0](denied);
      expect(await readP).toBe(denied);
      expect(await updateP).toBe(denied);
      hardened.close();
    });
  });

  describe("observable", () => {
    it("should notify observers when frames are received", async () => {
      const mockStreamer = new MockStreamer();
      const frame1 = new framer.Frame({ 1: new Series([1, 2, 3]) });
      const frame2 = new framer.Frame({ 1: new Series([4, 5, 6]) });

      mockStreamer.responses = [
        [frame1, null],
        [frame2, null],
      ];
      mockStreamer.keys = [1];

      const observable = new framer.ObservableStreamer(mockStreamer);

      const receivedFrames: framer.Frame[] = [];
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
      const frame1 = new framer.Frame({ 1: new Series([1, 2, 3]) });
      const frame2 = new framer.Frame({ 1: new Series([4, 5, 6]) });

      mockStreamer.responses = [
        [frame1, null],
        [frame2, null],
      ];
      mockStreamer.keys = [1];

      const transform = (frame: framer.Frame): [number, true] | [null, false] => {
        try {
          const data = Array.from(frame.get(1));
          const firstValue = data[0] as number;
          return [firstValue, true];
        } catch {
          return [null, false];
        }
      };

      const observable = new framer.ObservableStreamer(mockStreamer, transform);

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
      const frame1 = new framer.Frame({ 1: new Series([10, 20]) });

      mockStreamer.responses = [[frame1, null]];
      mockStreamer.keys = [1];

      const observable = new framer.ObservableStreamer(mockStreamer);

      const observer1Results: framer.Frame[] = [];
      const observer2Results: framer.Frame[] = [];

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

      const observable = new framer.ObservableStreamer(mockStreamer);

      await observable.update([3, 4, 5]);

      expect(mockStreamer.updateMock).toHaveBeenCalledWith([3, 4, 5]);

      await observable.close();
    });

    test("should handle empty frame stream gracefully", async () => {
      const mockStreamer = new MockStreamer();
      mockStreamer.keys = [1];

      const observable = new framer.ObservableStreamer(mockStreamer);

      const receivedFrames: framer.Frame[] = [];
      observable.onChange((frame) => {
        receivedFrames.push(frame);
      });

      await expect.poll(() => receivedFrames.length).toBe(0);
      expect(receivedFrames).toHaveLength(0);

      await observable.close();
    });

    test("should properly close and cleanup resources", async () => {
      const mockStreamer = new MockStreamer();
      const frame1 = new framer.Frame({ 1: new Series([1]) });

      mockStreamer.responses = [[frame1, null]];
      mockStreamer.keys = [1];

      const observable = new framer.ObservableStreamer(mockStreamer);

      const receivedFrames: framer.Frame[] = [];
      observable.onChange((frame) => {
        receivedFrames.push(frame);
      });

      await observable.close();

      expect(mockStreamer.closeMock).toHaveBeenCalled();
    });

    test("should report the failure that ended the stream", async () => {
      const mockStreamer = new MockStreamer();
      mockStreamer.responses = [
        [new framer.Frame(), new AccessDeniedError("access revoked")],
      ];
      const onDead = vi.fn();
      const observable = new framer.ObservableStreamer(mockStreamer, undefined, onDead);

      await expect.poll(() => onDead.mock.calls.length).toBe(1);
      expect(AccessDeniedError.matches(onDead.mock.calls[0][0])).toBe(true);

      await observable.close();
    });

    test("should not report a stream the caller closed", async () => {
      const mockStreamer = new MockStreamer();
      const onDead = vi.fn();
      const observable = new framer.ObservableStreamer(mockStreamer, undefined, onDead);

      await observable.close();

      expect(onDead).not.toHaveBeenCalled();
    });
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, Frame, type framer } from "@synnaxlabs/client";
import { type MultiSeries, Series, sleep, TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Cache } from "@/telem/client/cache/cache";
import { MockRetriever } from "@/telem/client/reader.spec";
import { Streamer } from "@/telem/client/streamer";

class MockStreamer implements framer.Streamer {
  private keysI: channel.Params[];
  readonly updateVi = vi.fn();
  readonly closeVi = vi.fn();
  readonly iteratorVi = vi.fn();
  readonly nextVi = vi.fn();
  readonly reads?: framer.Frame[];
  readonly nextFn?: () => Promise<IteratorResult<framer.Frame>>;

  constructor(
    keys: channel.Key[],
    nextFn?: () => Promise<IteratorResult<framer.Frame>>,
    reads?: framer.Frame[],
  ) {
    this.keysI = [keys];
    this.reads = reads;
    this.nextFn = nextFn;
  }

  get keys(): channel.Key[] {
    return this.keysI.at(-1) as channel.Key[];
  }

  update(keys: channel.Params): Promise<void> {
    this.keysI.push(keys);
    this.updateVi();
    return Promise.resolve();
  }

  close(): void {
    this.closeVi();
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    if (this.reads == null && this.nextFn == null)
      throw new Error("no next function or reads provided");
    if (this.nextFn != null) return await this.nextFn();
    const fr = this.reads?.shift();
    this.nextVi(fr);
    if (fr == null) return { done: true, value: undefined };
    return { done: false, value: fr };
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new Error("no more frames");
    return res.value;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    this.iteratorVi();
    return this;
  }
}

const createStreamOpener =
  (streamers: MockStreamer[]): framer.StreamOpener =>
  async () => {
    const streamer = streamers.shift();
    if (streamer == null) throw new Error("no streamers left");
    return streamer;
  };

describe("Streamer", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  describe("construction", () => {
    it("should correctly construct a new streamer that operates", async () => {
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: createStreamOpener([new MockStreamer([])]),
      });
      expect(streamer).toBeDefined();
    });
  });

  describe("basic operation", () => {
    it("should allow the caller to subscribe to changes from a channel", async () => {
      const retriever = new MockRetriever();
      let i = 0;
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: retriever }),
        openStreamer: createStreamOpener([
          new MockStreamer([1], async () => {
            await sleep.sleep(TimeSpan.milliseconds(5));
            i++;
            return {
              done: false,
              value: new Frame({
                1: new Series({
                  data: new Float32Array([1]),
                  alignment: BigInt(i),
                }),
              }),
            };
          }),
        ]),
      });

      const responses: Map<channel.Key, MultiSeries>[] = [];
      const disconnect = await streamer.stream((d) => responses.push(d), [1]);
      // Advance past the 100ms debounce plus enough 5ms read cycles to get well over 5
      // responses.
      await vi.advanceTimersByTimeAsync(200);
      disconnect();

      expect(responses.length).toBeGreaterThan(5);
      // We should only ever get data for that particular channel.
      expect(responses.filter((r) => r.get(1)?.series.length === 0)).toHaveLength(
        responses.length - 1,
      );
      // The first response should have no data, as it's just pulling initial relevant
      // values from the cache.
      expect(responses[0].get(1)?.series.length).toEqual(0);
      // We should only every has a single response that has data, as its the only
      // buffer we're allocating and subsequent calls just tell the handler to re-read
      // the buffer.
      expect(responses.filter((r) => r.get(1)?.series.length === 1)).toHaveLength(1);
    });
  });

  describe("updateStreamer race", () => {
    it("should not open two concurrent streamers when stream() is called twice during a slow openStreamer", async () => {
      const ms1 = new MockStreamer([1], async () => {
        await sleep.sleep(TimeSpan.milliseconds(20));
        return { done: true, value: undefined };
      });
      const ms2 = new MockStreamer([1, 2], async () => {
        await sleep.sleep(TimeSpan.milliseconds(20));
        return { done: true, value: undefined };
      });
      const queue = [ms1, ms2];
      let openCalls = 0;
      const slowOpener: framer.StreamOpener = async () => {
        openCalls++;
        // Long enough that the next debounced updateStreamer fire happens while this
        // call is still suspended.
        await sleep.sleep(TimeSpan.milliseconds(300));
        const next = queue.shift();
        if (next == null) throw new Error("no streamers left");
        return next;
      };

      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: slowOpener,
      });

      // First subscription schedules debouncedUpdateStreamer at T+100ms.
      const disconnect1 = await streamer.stream(() => {}, [1]);
      // Advance to T=150 so updateStreamer #1 is mid-openStreamer (sleep ends at
      // T=400).
      await vi.advanceTimersByTimeAsync(150);
      const disconnect2Promise = streamer.stream(() => {}, [2]);
      // Advance enough for any racing opens (300ms each) and run loops to complete.
      await vi.advanceTimersByTimeAsync(700);
      const disconnect2 = await disconnect2Promise;

      expect(openCalls).toBe(1);
      // The orphaned second streamer's run loop should never have started.
      expect(ms2.iteratorVi).not.toHaveBeenCalled();
      // The single streamer should have been updated with the second subscriber's key —
      // otherwise key 2 would be silently dropped.
      expect(ms1.updateVi).toHaveBeenCalled();
      expect(ms1.keys.slice().sort()).toEqual([1, 2]);

      disconnect1();
      disconnect2();
    });
  });

  describe("closed guard", () => {
    it("should not open a streamer if close() is called before the debounce fires", async () => {
      let openCalls = 0;
      const opener: framer.StreamOpener = async () => {
        openCalls++;
        return new MockStreamer([1]);
      };
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: opener,
      });

      await streamer.stream(() => {}, [1]);
      // Close before the 100ms debounce window elapses.
      await streamer.close();
      // Advance well past the debounce — the queued updateStreamer must observe
      // this.closed inside the mutex and bail out.
      await vi.advanceTimersByTimeAsync(500);

      expect(openCalls).toBe(0);
    });

    it("should close a freshly-opened streamer when close() runs during a slow openStreamer", async () => {
      const ms1 = new MockStreamer([1], async () => {
        await sleep.sleep(TimeSpan.milliseconds(10));
        return { done: true, value: undefined };
      });
      let openCalls = 0;
      const slowOpener: framer.StreamOpener = async () => {
        openCalls++;
        await sleep.sleep(TimeSpan.milliseconds(300));
        return ms1;
      };
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: slowOpener,
      });

      await streamer.stream(() => {}, [1]);
      // Advance into the openStreamer suspension window.
      await vi.advanceTimersByTimeAsync(150);
      // close() must wait for the in-flight updateStreamer to finish so that it can
      // observe the newly-assigned streamer and tear it down.
      const closePromise = streamer.close();
      await vi.advanceTimersByTimeAsync(500);
      await closePromise;

      expect(openCalls).toBe(1);
      expect(ms1.closeVi).toHaveBeenCalled();
    });

    it("should close the underlying streamer when the last listener disconnects", async () => {
      const ms1 = new MockStreamer([1], async () => {
        await sleep.sleep(TimeSpan.milliseconds(10));
        return { done: true, value: undefined };
      });
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: createStreamOpener([ms1]),
        streamUpdateDelay: TimeSpan.milliseconds(50),
      });

      const disconnect = await streamer.stream(() => {}, [1]);
      // Advance past debounce + the single nextFn cycle so the run loop ends.
      await vi.advanceTimersByTimeAsync(200);
      expect(ms1.closeVi).not.toHaveBeenCalled();

      disconnect();
      // streamUpdateDelay (50ms) + debounce (100ms) + slack so the no-keys branch in
      // updateStreamer fires and tears down the streamer.
      await vi.advanceTimersByTimeAsync(300);

      expect(ms1.closeVi).toHaveBeenCalled();
    });

    it("should be a no-op when stream() is called after close()", async () => {
      let openCalls = 0;
      const opener: framer.StreamOpener = async () => {
        openCalls++;
        return new MockStreamer([1]);
      };
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: opener,
      });

      await streamer.close();
      const disconnect = await streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(500);

      expect(openCalls).toBe(0);
      disconnect();
    });
  });

  describe("update path", () => {
    it("should not call update again when the merged key set is unchanged", async () => {
      const ms1 = new MockStreamer([1], async () => {
        await sleep.sleep(TimeSpan.milliseconds(10));
        return { done: true, value: undefined };
      });
      const ins = new alamos.Instrumentation({
        key: "test",
        logger: new alamos.Logger(),
      });
      const debugSpy = vi.spyOn(ins.L, "debug").mockImplementation(() => {});
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: createStreamOpener([ms1]),
        instrumentation: ins,
      });

      const d1 = await streamer.stream(() => {}, [1]);
      // Initial open path also issues a single update([1]).
      await vi.advanceTimersByTimeAsync(200);
      expect(ms1.updateVi).toHaveBeenCalledTimes(1);

      // Second subscriber on the same key — the merged set is still [1], so the
      // valuesEqual branch must short-circuit and log "streamer keys unchanged".
      const d2 = await streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);

      expect(ms1.updateVi).toHaveBeenCalledTimes(1);
      expect(debugSpy).toHaveBeenCalledWith("streamer keys unchanged", {
        keys: [1],
      });

      d1();
      d2();
    });

    it("should log an error when streamer.update rejects", async () => {
      const ms1 = new MockStreamer([1], async () => {
        await sleep.sleep(TimeSpan.milliseconds(10));
        return { done: true, value: undefined };
      });
      const updateErr = new Error("update boom");
      ms1.update = vi.fn(async () => {
        throw updateErr;
      });
      const ins = new alamos.Instrumentation({
        key: "test",
        logger: new alamos.Logger(),
      });
      const errorSpy = vi.spyOn(ins.L, "error").mockImplementation(() => {});
      // updateStreamer is fired-and-forgotten via the debounce, so a rethrow would
      // surface as an unhandled rejection. The error must be logged and swallowed.
      const rejections: unknown[] = [];
      const onRejection = (reason: unknown) => rejections.push(reason);
      process.on("unhandledRejection", onRejection);

      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: createStreamOpener([ms1]),
        instrumentation: ins,
      });

      try {
        await streamer.stream(() => {}, [1]);
        await vi.advanceTimersByTimeAsync(300);
      } finally {
        process.off("unhandledRejection", onRejection);
      }

      expect(errorSpy).toHaveBeenCalledWith("failed to update streamer", {
        error: updateErr,
      });
      expect(rejections).toHaveLength(0);
    });
  });

  describe("liveness", () => {
    it("should not block new subscriptions while a streamer teardown is stuck draining", async () => {
      let reads = 0;
      const stuck = new MockStreamer([1], async () => {
        reads++;
        if (reads === 1) {
          await sleep.sleep(TimeSpan.milliseconds(5));
          return {
            done: false,
            value: new Frame({ 1: new Series(new Float32Array([1])) }),
          };
        }
        // Simulate a streamer whose read never returns (e.g. a HardenedStreamer stuck
        // reconnecting): the run loop can never drain, so the teardown await hangs.
        return await new Promise<IteratorResult<framer.Frame>>(() => {});
      });
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: createStreamOpener([stuck, new MockStreamer([2])]),
        streamUpdateDelay: TimeSpan.milliseconds(50),
      });

      const disconnect1 = await streamer.stream(() => {}, [1]);
      // Open the streamer and let the run loop read once, then block on the second read.
      await vi.advanceTimersByTimeAsync(200);

      // Last listener leaves: the no-keys teardown closes the streamer and awaits the
      // stuck run loop, holding the connection lock indefinitely.
      disconnect1();
      await vi.advanceTimersByTimeAsync(200);

      // A fresh subscription must still resolve even though the connection lock is held
      // by the stuck teardown. Under a single shared lock this await would hang forever.
      const disconnect2 = await streamer.stream(() => {}, [2]);
      expect(disconnect2).toBeTypeOf("function");

      disconnect2();
    });

    it("should not block a new subscription while a streamer is mid-open", async () => {
      let opens = 0;
      const slowOpener: framer.StreamOpener = async () => {
        opens++;
        await sleep.sleep(TimeSpan.milliseconds(300));
        return new MockStreamer([1], async () => {
          await sleep.sleep(TimeSpan.milliseconds(10));
          return { done: true, value: undefined };
        });
      };
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: slowOpener,
      });

      await streamer.stream(() => {}, [1]);
      // Advance into the open's suspension window so updateStreamer holds the connection
      // lock across the network round-trip.
      await vi.advanceTimersByTimeAsync(150);
      expect(opens).toBe(1);

      // The open is still suspended (its 300ms sleep hasn't elapsed). A concurrent
      // subscription must not wait on it. Under a single shared lock this would hang.
      const disconnect = await streamer.stream(() => {}, [2]);
      expect(disconnect).toBeTypeOf("function");

      disconnect();
    });
  });

  describe("close path", () => {
    it("should log an error when streamer.close throws and still mark the streamer closed", async () => {
      const ms1 = new MockStreamer([1], async () => {
        await sleep.sleep(TimeSpan.milliseconds(10));
        return { done: true, value: undefined };
      });
      const closeErr = new Error("close boom");
      ms1.closeVi.mockImplementation(() => {
        throw closeErr;
      });
      const ins = new alamos.Instrumentation({
        key: "test",
        logger: new alamos.Logger(),
      });
      const errorSpy = vi.spyOn(ins.L, "error").mockImplementation(() => {});

      let openCalls = 0;
      const opener: framer.StreamOpener = async () => {
        openCalls++;
        return ms1;
      };
      const streamer = new Streamer({
        cache: new Cache({ channelRetriever: new MockRetriever() }),
        openStreamer: opener,
        instrumentation: ins,
      });

      await streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);
      expect(openCalls).toBe(1);

      await streamer.close();

      expect(errorSpy).toHaveBeenCalledWith("failed to close streamer", {
        error: closeErr,
      });
      // close() should still flip this.closed even when the underlying
      // streamer.close throws — subsequent stream() calls must be no-ops.
      const disconnect = await streamer.stream(() => {}, [2]);
      await vi.advanceTimersByTimeAsync(200);
      expect(openCalls).toBe(1);
      disconnect();
    });
  });
});

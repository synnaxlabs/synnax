// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type MultiSeries, Series, sleep, TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type framer } from "@/framer";
import { Cache } from "@/framer/cache/cache";
import { Streamer, type StreamHooks } from "@/framer/cache/streamer";
import { Frame } from "@/framer/frame";

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
    this.updateVi(keys);
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

/** A streamer whose read blocks until close(), which ends iteration like a real
 * streamer reporting EOF. */
const pendingStreamer = (keys: channel.Key[]): MockStreamer => {
  let closed = false;
  let resolve: ((r: IteratorResult<framer.Frame>) => void) | null = null;
  const ms = new MockStreamer(keys, async () => {
    if (closed) return { done: true, value: undefined };
    return await new Promise<IteratorResult<framer.Frame>>((r) => (resolve = r));
  });
  const origClose = ms.close.bind(ms);
  ms.close = () => {
    closed = true;
    resolve?.({ done: true, value: undefined });
    origClose();
  };
  return ms;
};

const createStreamOpener =
  (
    streamers: MockStreamer[],
    hooksOut?: StreamHooks[],
  ): ((
    config: framer.StreamerConfig,
    hooks: StreamHooks,
  ) => Promise<framer.Streamer>) =>
  async (_, hooks) => {
    hooksOut?.push(hooks);
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
    it("should correctly construct a new streamer that operates", () => {
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([new MockStreamer([])]),
      });
      expect(streamer).toBeDefined();
    });
  });

  describe("basic operation", () => {
    it("should allow the caller to subscribe to changes from a channel", async () => {
      let i = 0;
      const streamer = new Streamer({
        cache: new Cache(),
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
      const sub = streamer.stream((d) => responses.push(d), [1]);
      // The initial buffers land synchronously, before any stream is opened.
      expect(responses).toHaveLength(1);
      expect(responses[0].get(1)?.series.length).toEqual(0);
      expect(sub.status(1).variant).toEqual("loading");
      // Advance past the 100ms reconcile delay plus enough 5ms read cycles to get
      // well over 5 responses.
      await vi.advanceTimersByTimeAsync(200);
      sub.close();

      expect(responses.length).toBeGreaterThan(5);
      // We should only ever get data for that particular channel.
      expect(responses.filter((r) => r.get(1)?.series.length === 0)).toHaveLength(
        responses.length - 1,
      );
      // Only one response carries the allocated buffer; subsequent frames land in it
      // and the handler re-reads it.
      expect(responses.filter((r) => r.get(1)?.series.length === 1)).toHaveLength(1);
      expect(sub.status(1).variant).toEqual("success");
    });
  });

  describe("reconcile race", () => {
    it("should not open two concurrent streamers when stream() is called twice during a slow open", async () => {
      const ms1 = pendingStreamer([1]);
      const ms2 = pendingStreamer([1, 2]);
      const queue = [ms1, ms2];
      let openCalls = 0;
      const slowOpener = async (): Promise<framer.Streamer> => {
        openCalls++;
        await sleep.sleep(TimeSpan.milliseconds(300));
        const next = queue.shift();
        if (next == null) throw new Error("no streamers left");
        return next;
      };

      const streamer = new Streamer({ cache: new Cache(), openStreamer: slowOpener });

      const sub1 = streamer.stream(() => {}, [1]);
      // Advance so reconcile #1 is mid-open (sleep ends at T=400).
      await vi.advanceTimersByTimeAsync(150);
      const sub2 = streamer.stream(() => {}, [2]);
      await vi.advanceTimersByTimeAsync(700);

      expect(openCalls).toBe(1);
      // The orphaned second streamer's run loop should never have started.
      expect(ms2.iteratorVi).not.toHaveBeenCalled();
      // The single streamer must carry the second subscriber's key.
      expect(ms1.updateVi).toHaveBeenCalled();
      expect(ms1.keys.slice().sort()).toEqual([1, 2]);

      sub1.close();
      sub2.close();
    });
  });

  describe("closed guard", () => {
    it("should not open a streamer if close() is called before the reconcile fires", async () => {
      let openCalls = 0;
      const opener = async (): Promise<framer.Streamer> => {
        openCalls++;
        return pendingStreamer([1]);
      };
      const streamer = new Streamer({ cache: new Cache(), openStreamer: opener });

      streamer.stream(() => {}, [1]);
      await streamer.close();
      await vi.advanceTimersByTimeAsync(500);

      expect(openCalls).toBe(0);
    });

    it("should close a freshly-opened streamer when close() runs during a slow open", async () => {
      const ms1 = pendingStreamer([1]);
      let openCalls = 0;
      const slowOpener = async (): Promise<framer.Streamer> => {
        openCalls++;
        await sleep.sleep(TimeSpan.milliseconds(300));
        return ms1;
      };
      const streamer = new Streamer({ cache: new Cache(), openStreamer: slowOpener });

      streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(150);
      // close() must wait for the in-flight reconcile to finish so that it can
      // observe the newly-assigned streamer and tear it down.
      const closePromise = streamer.close();
      await vi.advanceTimersByTimeAsync(500);
      await closePromise;

      expect(openCalls).toBe(1);
      expect(ms1.closeVi).toHaveBeenCalled();
    });

    it("should close the underlying streamer when the last listener disconnects", async () => {
      const ms1 = pendingStreamer([1]);
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([ms1]),
        removalDelay: TimeSpan.milliseconds(50),
      });

      const sub = streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);
      expect(ms1.closeVi).not.toHaveBeenCalled();

      sub.close();
      // removalDelay (50ms) + reconcile delay (100ms) + slack so the no-keys branch
      // tears down the streamer.
      await vi.advanceTimersByTimeAsync(300);

      expect(ms1.closeVi).toHaveBeenCalled();
    });

    it("should throw when stream() is called after close()", async () => {
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([]),
      });
      await streamer.close();
      expect(() => streamer.stream(() => {}, [1])).toThrow(UnexpectedError);
    });
  });

  describe("registration atomicity", () => {
    it("should register no demand when the handler throws on its initial call", async () => {
      const openedWith: unknown[] = [];
      const opener = async (
        config: framer.StreamerConfig,
      ): Promise<framer.Streamer> => {
        if (typeof config === "object" && "channels" in config)
          openedWith.push(config.channels);
        return pendingStreamer([2]);
      };
      const streamer = new Streamer({ cache: new Cache(), openStreamer: opener });

      expect(() =>
        streamer.stream(() => {
          throw new Error("broken handler");
        }, [1]),
      ).toThrow("broken handler");
      await vi.advanceTimersByTimeAsync(500);
      expect(openedWith).toHaveLength(0);

      // A later subscriber must not drag the failed registration's keys onto the
      // stream.
      const responses: Map<channel.Key, MultiSeries>[] = [];
      streamer.stream((d) => responses.push(d), [2]);
      await vi.advanceTimersByTimeAsync(200);
      expect(openedWith).toEqual([[2]]);
      expect(responses).toHaveLength(1);
    });
  });

  describe("update path", () => {
    it("should not call update again when the merged key set is unchanged", async () => {
      const ms1 = pendingStreamer([1]);
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([ms1]),
      });

      const sub1 = streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);
      expect(ms1.updateVi).not.toHaveBeenCalled();

      // Second subscriber on the same key: the merged set is still [1], so no
      // update round trip is issued.
      const sub2 = streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);

      expect(ms1.updateVi).not.toHaveBeenCalled();
      sub1.close();
      sub2.close();
    });

    it("should retry a failed update until it succeeds", async () => {
      const ms1 = pendingStreamer([1]);
      const updateErr = new Error("update boom");
      let updateCalls = 0;
      ms1.update = vi.fn(async (keys: channel.Params) => {
        updateCalls++;
        if (updateCalls === 1) throw updateErr;
        ms1.updateVi(keys);
      });
      const ins = new alamos.Instrumentation({
        key: "test",
        logger: new alamos.Logger(),
      });
      const errorSpy = vi.spyOn(ins.L, "error").mockImplementation(() => {});
      const rejections: unknown[] = [];
      const onRejection = (reason: unknown) => rejections.push(reason);
      process.on("unhandledRejection", onRejection);

      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([ms1]),
        instrumentation: ins,
      });

      try {
        const sub1 = streamer.stream(() => {}, [1]);
        await vi.advanceTimersByTimeAsync(200);
        const sub2 = streamer.stream(() => {}, [2]);
        // First update throws; the streamer must back off and retry on its own.
        await vi.advanceTimersByTimeAsync(2000);
        expect(updateCalls).toBeGreaterThanOrEqual(2);
        expect(ms1.updateVi).toHaveBeenCalledWith([1, 2]);
        expect(sub2.status(2).variant).toEqual("success");
        sub1.close();
        sub2.close();
      } finally {
        process.off("unhandledRejection", onRejection);
      }

      expect(errorSpy).toHaveBeenCalledWith("failed to update streamer", {
        error: updateErr,
      });
      expect(rejections).toHaveLength(0);
    });
  });

  describe("repair", () => {
    it("should reopen the stream when the run loop ends while demand remains", async () => {
      const frame = new Frame({ 1: new Series(new Float32Array([1])) });
      // The first streamer delivers one frame and then reports done, simulating a
      // stream that dies. The streamer must detect the dead loop and reopen.
      const dying = new MockStreamer([1], undefined, [frame]);
      const replacement = pendingStreamer([1]);
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([dying, replacement]),
      });

      const sub = streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(1000);

      expect(replacement.iteratorVi).toHaveBeenCalled();
      expect(sub.status(1).variant).toEqual("success");
      sub.close();
    });
  });

  describe("subscriber isolation", () => {
    it("should keep the stream alive when a handler throws on live delivery", async () => {
      let i = 0;
      let openCalls = 0;
      const opener = async (): Promise<framer.Streamer> => {
        openCalls++;
        return new MockStreamer([1], async () => {
          await sleep.sleep(TimeSpan.milliseconds(5));
          i++;
          return {
            done: false,
            value: new Frame({
              1: new Series({ data: new Float32Array([1]), alignment: BigInt(i) }),
            }),
          };
        });
      };
      const streamer = new Streamer({ cache: new Cache(), openStreamer: opener });

      let brokenCalls = 0;
      const broken = streamer.stream(() => {
        brokenCalls++;
        // Survive the initial synchronous call so registration completes, then
        // throw on every live delivery.
        if (brokenCalls > 1) throw new Error("broken live handler");
      }, [1]);
      const responses: Map<channel.Key, MultiSeries>[] = [];
      const good = streamer.stream((d) => responses.push(d), [1]);
      await vi.advanceTimersByTimeAsync(300);
      broken.close();
      good.close();

      expect(brokenCalls).toBeGreaterThan(5);
      expect(responses.length).toBeGreaterThan(5);
      expect(openCalls).toBe(1);
    });

    it("should notify remaining status handlers when one throws", async () => {
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([pendingStreamer([1])]),
      });

      const broken = streamer.stream(() => {}, [1]);
      broken.onStatusChange(() => {
        throw new Error("broken status handler");
      });
      const good = streamer.stream(() => {}, [1]);
      const statuses: string[] = [];
      good.onStatusChange((_, s) => statuses.push(s.variant));
      await vi.advanceTimersByTimeAsync(300);
      broken.close();
      good.close();

      expect(statuses).toContain("success");
    });
  });

  describe("per-key isolation", () => {
    it("should keep streaming healthy channels when one channel's write fails", async () => {
      const cache = new Cache();
      let emitted = false;
      const ms = new MockStreamer([1, 2], async () => {
        if (emitted) return await new Promise(() => {});
        emitted = true;
        return {
          done: false,
          value: new Frame({
            1: new Series(new Float32Array([1])),
            2: new Series(new Float32Array([2])),
          }),
        };
      });
      const streamer = new Streamer({ cache, openStreamer: createStreamOpener([ms]) });

      const received: Map<channel.Key, MultiSeries>[] = [];
      const sub = streamer.stream((d) => received.push(d), [1, 2]);
      // Kill channel 1's cache entry so its write throws.
      cache.get(1).close();
      await vi.advanceTimersByTimeAsync(300);

      const withData = received.filter((r) => r.size > 0 && r.has(2));
      expect(withData.length).toBeGreaterThan(0);
      expect(sub.status(1).variant).toEqual("error");
      expect(sub.status(2).variant).toEqual("success");
      sub.close();
    });
  });

  describe("connection health", () => {
    it("should report reconnecting on drop and streaming on reopen", async () => {
      const hooks: StreamHooks[] = [];
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([pendingStreamer([1])], hooks),
      });
      const transitions: string[] = [];
      const sub = streamer.stream(() => {}, [1]);
      sub.onStatusChange((_, s) => transitions.push(s.variant));
      await vi.advanceTimersByTimeAsync(200);
      expect(sub.status(1).variant).toEqual("success");

      hooks[0].onDrop(new Error("conn lost"));
      expect(sub.status(1).variant).toEqual("warning");
      hooks[0].onReopen();
      expect(sub.status(1).variant).toEqual("success");
      expect(transitions).toEqual(["success", "warning", "success"]);
      sub.close();
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
        // A streamer whose read never returns: the run loop can never drain, so the
        // teardown await hangs.
        return await new Promise<IteratorResult<framer.Frame>>(() => {});
      });
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: createStreamOpener([stuck, pendingStreamer([2])]),
        removalDelay: TimeSpan.milliseconds(50),
      });

      const sub1 = streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);

      // Last listener leaves: the no-keys teardown closes the streamer and awaits
      // the stuck run loop, holding the connection lock indefinitely.
      sub1.close();
      await vi.advanceTimersByTimeAsync(200);

      // A fresh subscription must still register even though the connection lock is
      // held by the stuck teardown.
      const sub2 = streamer.stream(() => {}, [2]);
      expect(sub2.status(2).variant).toEqual("loading");
      sub2.close();
    });
  });

  describe("close path", () => {
    it("should log an error when streamer.close throws and still mark the streamer closed", async () => {
      const ms1 = pendingStreamer([1]);
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
      const opener = async (): Promise<framer.Streamer> => {
        openCalls++;
        return ms1;
      };
      const streamer = new Streamer({
        cache: new Cache(),
        openStreamer: opener,
        instrumentation: ins,
      });

      streamer.stream(() => {}, [1]);
      await vi.advanceTimersByTimeAsync(200);
      expect(openCalls).toBe(1);

      await streamer.close();

      expect(errorSpy).toHaveBeenCalledWith("failed to close streamer", {
        error: closeErr,
      });
      expect(() => streamer.stream(() => {}, [2])).toThrow(UnexpectedError);
    });
  });
});

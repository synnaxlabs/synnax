// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { DataType, Series, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { type channel } from "@/channel";
import { framer } from "@/framer";
import { Joins } from "@/framer/cache/join";
import { Store } from "@/framer/cache/store";
import { MultiplexedStreamer, type StreamHooks } from "@/framer/cache/streamer";
import { NotFoundError, UnexpectedError } from "@/index";

// Returns the stores a streamer writes into, with no join that can wait.
const createCache = (store: Store = new Store()) => ({
  store,
  joins: new Joins(store),
  onWait: () => {},
});

const createErrorSpy = () => {
  const instrumentation = new alamos.Instrumentation({
    key: "test",
    logger: new alamos.Logger({ filters: [alamos.logThresholdFilter("error")] }),
  });
  const errorSpy = vi.spyOn(instrumentation.L, "error").mockImplementation(() => {});
  return { instrumentation, errorSpy };
};

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

/** A streamer that delivers one frame, then blocks until close() like
 * {@link pendingStreamer}. */
const oneFrameStreamer = (keys: channel.Key[], frame: framer.Frame): MockStreamer => {
  let sent = false;
  let closed = false;
  let resolve: ((r: IteratorResult<framer.Frame>) => void) | null = null;
  const ms = new MockStreamer(keys, async () => {
    if (!sent) {
      sent = true;
      return { done: false, value: frame };
    }
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

describe("MultiplexedStreamer", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  describe("key rejections", () => {
    const BAD = 3;
    const channelsOf = (config: framer.StreamerConfig): channel.Key[] =>
      (typeof config === "object" && "channels" in config
        ? config.channels
        : config) as channel.Key[];
    const rejectingStreamer = (keys: channel.Key[]): MockStreamer => {
      const ms = pendingStreamer(keys);
      ms.update = async (next) => {
        if ((next as channel.Key[]).includes(BAD))
          throw new NotFoundError(`channel ${BAD} not found`);
        ms.updateVi(next);
      };
      return ms;
    };

    it("should stream every other key when the open rejects one", async () => {
      const opened: channel.Key[][] = [];
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: async (config) => {
          const keys = channelsOf(config);
          opened.push(keys);
          if (keys.includes(BAD)) throw new NotFoundError(`channel ${BAD} not found`);
          return rejectingStreamer(keys);
        },
        instrumentation: createErrorSpy().instrumentation,
      });
      const sub = streamer.demand([1, 2, BAD], () => {});
      await vi.advanceTimersByTimeAsync(1000);
      expect(sub.status(1).variant).toEqual("success");
      expect(sub.status(2).variant).toEqual("success");
      expect(sub.status(BAD).variant).toEqual("error");
      const opens = opened.length;
      await vi.advanceTimersByTimeAsync(10_000);
      expect(opened).toHaveLength(opens);
      await streamer.close();
    });

    it("should stream a key opened after a rejected one", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: async (config) => {
          const keys = channelsOf(config);
          if (keys.includes(BAD)) throw new NotFoundError(`channel ${BAD} not found`);
          return rejectingStreamer(keys);
        },
        instrumentation: createErrorSpy().instrumentation,
      });
      const sub = streamer.demand([BAD, 1], () => {});
      await vi.advanceTimersByTimeAsync(1000);
      expect(sub.status(1).variant).toEqual("success");
      expect(sub.status(BAD).variant).toEqual("error");
      await streamer.close();
    });

    it("should keep a rejected key's error when no other key is demanded", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: async () => {
          throw new NotFoundError(`channel ${BAD} not found`);
        },
        instrumentation: createErrorSpy().instrumentation,
      });
      const sub = streamer.demand([BAD], () => {});
      await vi.advanceTimersByTimeAsync(1000);
      expect(sub.status(BAD).variant).toEqual("error");
      await streamer.close();
    });

    it("should give each opened stream its own abort signal", async () => {
      const hooks: StreamHooks[] = [];
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener(
          [pendingStreamer([1]), pendingStreamer([1])],
          hooks,
        ),
        removalDelay: TimeSpan.milliseconds(10),
      });
      const first = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      first.close();
      await vi.advanceTimersByTimeAsync(500);
      streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(hooks).toHaveLength(2);
      expect(hooks[0].signal).not.toBe(hooks[1].signal);
      await streamer.close();
      expect(hooks[0].signal.aborted).toBe(false);
      expect(hooks[1].signal.aborted).toBe(true);
    });

    it("should keep streaming when an update adds a rejected key", async () => {
      const ms = rejectingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms]),
        instrumentation: createErrorSpy().instrumentation,
      });
      const first = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      const second = streamer.demand([2, BAD], () => {});
      await vi.advanceTimersByTimeAsync(1000);
      expect(first.status(1).variant).toEqual("success");
      expect(second.status(2).variant).toEqual("success");
      expect(second.status(BAD).variant).toEqual("error");
      expect(ms.updateVi).toHaveBeenLastCalledWith([1, 2]);
      await streamer.close();
    });

    it("should narrow a rejected key without waiting between attempts", async () => {
      const ms = rejectingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms]),
        instrumentation: createErrorSpy().instrumentation,
      });
      streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(1);
      const sub = streamer.demand([2, 4, 5, BAD], () => {});
      await vi.advanceTimersByTimeAsync(100);
      expect(sub.status(2).variant).toEqual("success");
      expect(sub.status(4).variant).toEqual("success");
      expect(sub.status(5).variant).toEqual("success");
      expect(sub.status(BAD).variant).toEqual("error");
      await streamer.close();
    });

    it("should retry a rejected key once its demand returns", async () => {
      let attempts = 0;
      const ms = rejectingStreamer([1]);
      const update = ms.update.bind(ms);
      ms.update = async (next) => {
        if ((next as channel.Key[]).includes(BAD)) attempts++;
        await update(next);
      };
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms]),
        removalDelay: TimeSpan.milliseconds(10),
        instrumentation: createErrorSpy().instrumentation,
      });
      streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      const bad = streamer.demand([BAD], () => {});
      await vi.advanceTimersByTimeAsync(1000);
      expect(attempts).toBe(1);
      bad.close();
      await vi.advanceTimersByTimeAsync(1000);
      const again = streamer.demand([BAD], () => {});
      await vi.advanceTimersByTimeAsync(1000);
      expect(attempts).toBe(2);
      expect(again.status(BAD).variant).toEqual("error");
      await streamer.close();
    });
  });

  describe("construction", () => {
    it("should correctly construct a new streamer that operates", () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([new MockStreamer([])]),
      });
      expect(streamer).toBeDefined();
    });
  });

  describe("basic operation", () => {
    it("should allow the caller to subscribe to changes from a channel", async () => {
      let i = 0;
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([
          new MockStreamer([1], async () => {
            await sleep.sleep(TimeSpan.milliseconds(5));
            i++;
            return {
              done: false,
              value: new framer.Frame({
                1: new Series({
                  data: new Float32Array([1]),
                  alignment: BigInt(i),
                }),
              }),
            };
          }),
        ]),
      });

      const frames: Map<channel.Key, Series[]>[] = [];
      const sub = streamer.demand([1], (frame) => frames.push(frame));
      expect(sub.status(1).variant).toEqual("loading");
      // Advance through enough 5ms read cycles to get well over 5 frames.
      await vi.advanceTimersByTimeAsync(200);
      sub.close();

      expect(frames.length).toBeGreaterThan(5);
      frames.forEach((f) => expect([...f.keys()]).toEqual([1]));
      expect(sub.status(1).variant).toEqual("success");
    });

    it("should deliver the series of a key a frame repeats together", async () => {
      const frame = new framer.Frame(
        [1, 1],
        [
          new Series({ data: new Float32Array([1]), alignment: 0n }),
          new Series({ data: new Float32Array([2]), alignment: 1n }),
        ],
      );
      let sent = false;
      let closed = false;
      let resolve: ((r: IteratorResult<framer.Frame>) => void) | null = null;
      const ms = new MockStreamer([1], async () => {
        if (!sent) {
          sent = true;
          return { done: false, value: frame };
        }
        if (closed) return { done: true, value: undefined };
        return await new Promise<IteratorResult<framer.Frame>>((r) => (resolve = r));
      });
      const origClose = ms.close.bind(ms);
      ms.close = () => {
        closed = true;
        resolve?.({ done: true, value: undefined });
        origClose();
      };
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms]),
      });
      const frames: Map<channel.Key, Series[]>[] = [];
      streamer.demand([1], (f) => frames.push(f));
      await vi.advanceTimersByTimeAsync(200);
      expect(frames).toHaveLength(1);
      expect(frames[0].get(1)).toEqual(frame.series);
      await streamer.close();
    });
  });

  describe("reconcile race", () => {
    it("should not open two concurrent streamers when demand() is called twice during a slow open", async () => {
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

      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: slowOpener,
      });

      const sub1 = streamer.demand([1], () => {});
      // Advance so reconcile #1 is mid-open (sleep ends at T=300).
      await vi.advanceTimersByTimeAsync(150);
      const sub2 = streamer.demand([2], () => {});
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

  describe("update timing", () => {
    it("should open the stream on the next task after the first subscription", async () => {
      const open = vi.fn(createStreamOpener([pendingStreamer([1])]));
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: open,
      });
      streamer.demand([1], () => {});
      expect(open).not.toHaveBeenCalled();
      await vi.advanceTimersByTimeAsync(1);
      expect(open).toHaveBeenCalledOnce();
      await streamer.close();
    });

    it("should open one stream with every key subscribed in the same task", async () => {
      const open = vi.fn(createStreamOpener([pendingStreamer([1, 2, 3])]));
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: open,
      });
      streamer.demand([1], () => {});
      streamer.demand([2], () => {});
      streamer.demand([3], () => {});
      await vi.advanceTimersByTimeAsync(1);
      expect(open).toHaveBeenCalledOnce();
      expect(open).toHaveBeenCalledWith(
        expect.objectContaining({ channels: [1, 2, 3] }),
        expect.anything(),
      );
      await streamer.close();
    });

    it("should update the stream at most once per 100 ms while demand keeps changing", async () => {
      const ms = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms]),
      });
      streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(1);
      for (let key = 2; key <= 31; key++) {
        streamer.demand([key], () => {});
        await vi.advanceTimersByTimeAsync(10);
      }
      await vi.advanceTimersByTimeAsync(100);
      expect(ms.updateVi).toHaveBeenCalledTimes(3);
      expect(ms.updateVi).toHaveBeenLastCalledWith(
        Array.from({ length: 31 }, (_, i) => i + 1),
      );
      await streamer.close();
    });
  });

  describe("closed guard", () => {
    it("should not open a streamer if close() is called before the reconcile fires", async () => {
      let openCalls = 0;
      const opener = async (): Promise<framer.Streamer> => {
        openCalls++;
        return pendingStreamer([1]);
      };
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: opener,
      });

      streamer.demand([1], () => {});
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
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: slowOpener,
      });

      streamer.demand([1], () => {});
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
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms1]),
        removalDelay: TimeSpan.milliseconds(50),
      });

      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(ms1.closeVi).not.toHaveBeenCalled();

      sub.close();
      // removalDelay (50ms) + slack so the no-keys branch tears down the streamer.
      await vi.advanceTimersByTimeAsync(300);

      expect(ms1.closeVi).toHaveBeenCalled();
    });

    it("should release a subscription closed twice", async () => {
      const ms1 = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms1]),
        removalDelay: TimeSpan.milliseconds(50),
      });

      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      sub.close();
      sub.close();
      await vi.advanceTimersByTimeAsync(300);

      expect(ms1.closeVi).toHaveBeenCalled();
      await streamer.close();
    });

    it("should throw when demand() is called after close()", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([]),
      });
      await streamer.close();
      expect(() => streamer.demand([1], () => {})).toThrow(UnexpectedError);
    });
  });

  describe("update path", () => {
    it("should not call update again when the merged key set is unchanged", async () => {
      const ms1 = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms1]),
      });

      const sub1 = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(ms1.updateVi).not.toHaveBeenCalled();

      // Second subscriber on the same key: the merged set is still [1], so no
      // update round trip is issued.
      const sub2 = streamer.demand([1], () => {});
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
      const { instrumentation, errorSpy } = createErrorSpy();
      const rejections: unknown[] = [];
      const onRejection = (reason: unknown) => rejections.push(reason);
      process.on("unhandledRejection", onRejection);

      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms1]),
        instrumentation,
      });

      try {
        const sub1 = streamer.demand([1], () => {});
        await vi.advanceTimersByTimeAsync(200);
        const sub2 = streamer.demand([2], () => {});
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
      const frame = new framer.Frame({ 1: new Series(new Float32Array([1])) });
      // The first streamer delivers one frame and then reports done, simulating a
      // stream that dies. The streamer must detect the dead loop and reopen.
      const dying = new MockStreamer([1], undefined, [frame]);
      const replacement = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([dying, replacement]),
      });

      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(1000);

      expect(replacement.iteratorVi).toHaveBeenCalled();
      expect(sub.status(1).variant).toEqual("success");
      sub.close();
    });

    it("should reopen when the loop ends during a slow update", async () => {
      const dying = pendingStreamer([1]);
      const replacement = pendingStreamer([1, 2]);
      const { promise: updateGate, resolve: releaseUpdate } =
        Promise.withResolvers<void>();
      dying.update = async () => await updateGate;
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([dying, replacement]),
      });

      const sub1 = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      const sub2 = streamer.demand([2], () => {});
      await vi.advanceTimersByTimeAsync(200);

      // The loop dies mid-update, so its repair clears the sent keys before the
      // update that is still in flight writes the new demand back over them.
      dying.close();
      await vi.advanceTimersByTimeAsync(0);
      releaseUpdate();
      await vi.advanceTimersByTimeAsync(500);

      expect(replacement.iteratorVi).toHaveBeenCalled();
      sub1.close();
      sub2.close();
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
            value: new framer.Frame({
              1: new Series({ data: new Float32Array([1]), alignment: BigInt(i) }),
            }),
          };
        });
      };
      const { instrumentation, errorSpy } = createErrorSpy();
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: opener,
        instrumentation,
      });

      let brokenCalls = 0;
      const broken = streamer.demand([1], () => {
        brokenCalls++;
        throw new Error("broken live handler");
      });
      const frames: Map<channel.Key, Series[]>[] = [];
      const good = streamer.demand([1], (f) => frames.push(f));
      await vi.advanceTimersByTimeAsync(300);
      broken.close();
      good.close();

      expect(brokenCalls).toBeGreaterThan(5);
      expect(frames.length).toBeGreaterThan(5);
      expect(openCalls).toBe(1);
      expect(errorSpy).toHaveBeenCalledWith(
        "stream handler failed",
        { error: new Error("broken live handler") },
        true,
      );
    });

    it("should notify remaining status handlers when one throws", async () => {
      const { instrumentation, errorSpy } = createErrorSpy();
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])]),
        instrumentation,
      });

      const broken = streamer.demand([1], () => {});
      broken.onStatusChange(() => {
        throw new Error("broken status handler");
      });
      const good = streamer.demand([1], () => {});
      const statuses: string[] = [];
      good.onStatusChange((_, s) => statuses.push(s.variant));
      await vi.advanceTimersByTimeAsync(300);
      broken.close();
      good.close();

      expect(statuses).toContain("success");
      expect(errorSpy).toHaveBeenCalledWith(
        "status handler failed",
        { error: new Error("broken status handler") },
        true,
      );
    });
  });

  describe("connection health", () => {
    it("should report reconnecting on drop and streaming on reopen", async () => {
      const hooks: StreamHooks[] = [];
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])], hooks),
      });
      const transitions: string[] = [];
      const sub = streamer.demand([1], () => {});
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

    it("should run the drop handlers of open demands when the stream drops", async () => {
      const hooks: StreamHooks[] = [];
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])], hooks),
      });
      const sub = streamer.demand([1], () => {});
      let drops = 0;
      sub.onDrop(() => drops++);
      await vi.advanceTimersByTimeAsync(200);
      hooks[0].onDrop(new Error("conn lost"));
      expect(drops).toBe(1);
      sub.close();
      hooks[0].onDrop(new Error("conn lost"));
      expect(drops).toBe(1);
      await streamer.close();
    });

    it("should run the drop handlers when the run loop dies", async () => {
      const frame = new framer.Frame({ 1: new Series(new Float32Array([1])) });
      const dying = new MockStreamer([1], undefined, [frame]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([dying, pendingStreamer([1])]),
      });
      const sub = streamer.demand([1], () => {});
      let drops = 0;
      sub.onDrop(() => drops++);
      await vi.advanceTimersByTimeAsync(1000);
      expect(drops).toBe(1);
      sub.close();
      await streamer.close();
    });
  });

  describe("acks", () => {
    it("should resolve a wait once the stream acks every key", async () => {
      const ms = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms]),
      });
      const first = streamer.demand([1], () => {});
      const waited = vi.fn();
      void streamer.waitAcked([1, 2]).then(waited);
      await vi.advanceTimersByTimeAsync(200);
      expect(waited).not.toHaveBeenCalled();
      const second = streamer.demand([2], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(ms.updateVi).toHaveBeenCalledWith([1, 2]);
      expect(waited).toHaveBeenCalled();
      first.close();
      second.close();
      await streamer.close();
    });

    it("should wait again after a drop until the stream reopens", async () => {
      const hooks: StreamHooks[] = [];
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])], hooks),
      });
      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      hooks[0].onDrop(new Error("conn lost"));
      const waited = vi.fn();
      void streamer.waitAcked([1]).then(waited);
      await vi.advanceTimersByTimeAsync(200);
      expect(waited).not.toHaveBeenCalled();
      hooks[0].onReopen();
      await vi.advanceTimersByTimeAsync(0);
      expect(waited).toHaveBeenCalled();
      sub.close();
      await streamer.close();
    });

    it("should reject a wait with the Core's rejection of a key", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: async () => {
          throw new NotFoundError("channel 3 not found");
        },
        instrumentation: createErrorSpy().instrumentation,
      });
      const sub = streamer.demand([3], () => {});
      const waited = streamer.waitAcked([3]);
      const rejected = expect(waited).rejects.toThrow("channel 3 not found");
      await vi.advanceTimersByTimeAsync(200);
      await rejected;
      sub.close();
      await streamer.close();
    });

    it("should remove a wait's abort listener when the wait settles", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])]),
      });
      const sub = streamer.demand([1], () => {});
      const { signal } = new AbortController();
      const added = vi.spyOn(signal, "addEventListener");
      const removed = vi.spyOn(signal, "removeEventListener");
      const waited = streamer.waitAcked([1], signal);
      await vi.advanceTimersByTimeAsync(200);
      await waited;
      expect(removed).toHaveBeenCalledWith("abort", added.mock.calls[0][1]);
      sub.close();
      await streamer.close();
    });

    it("should reject a wait when its signal aborts or the streamer closes", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])]),
      });
      const abort = new AbortController();
      const aborted = streamer.waitAcked([1], abort.signal);
      abort.abort(new UnexpectedError("join left"));
      await expect(aborted).rejects.toThrow("join left");
      const closed = streamer.waitAcked([1]);
      await streamer.close();
      await expect(closed).rejects.toThrow(
        new UnexpectedError("telemetry streamer is closed"),
      );
    });
  });

  describe("buffer flushing", () => {
    const TIMES = [10, 11, 12].map((s) => TimeStamp.seconds(s).valueOf());

    const createTimestamps = (): Series =>
      new Series({
        data: new BigInt64Array(TIMES),
        dataType: DataType.TIMESTAMP,
        alignment: 0n,
        timeRange: TimeStamp.seconds(10).range(TimeStamp.seconds(13)),
      });

    // Returns the stores a streamer writes into, with a caught-up join of each index
    // channel, so the stream writes their samples into open blocks.
    const createJoined = (...indexes: channel.Key[]) => {
      const cache = createCache();
      indexes.forEach((k) => {
        cache.joins.join(k, k);
        cache.joins.landed(k, cache.joins.waiting(k));
      });
      return cache;
    };

    it("should close a channel's open block when its demand ends", async () => {
      const cache = createJoined(1);
      const { store } = cache;
      const streamer = new MultiplexedStreamer({
        ...cache,
        openStreamer: createStreamOpener([
          oneFrameStreamer([1], new framer.Frame({ 1: createTimestamps() })),
        ]),
        removalDelay: TimeSpan.milliseconds(50),
      });
      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(store.open(1)).not.toBeNull();

      sub.close();
      await vi.advanceTimersByTimeAsync(300);

      expect(store.open(1)).toBeNull();
      const block = store.raw(1, 0);
      expect(block.buffer).toBeNull();
      expect(block.fetched.flatMap((s) => Array.from(s))).toEqual(TIMES);
      await streamer.close();
    });

    it("should close only the blocks of the keys that left the stream", async () => {
      const cache = createJoined(1, 2);
      const { store } = cache;
      const streamer = new MultiplexedStreamer({
        ...cache,
        openStreamer: createStreamOpener([
          oneFrameStreamer(
            [1, 2],
            new framer.Frame([1, 2], [createTimestamps(), createTimestamps()]),
          ),
        ]),
        removalDelay: TimeSpan.milliseconds(50),
      });
      const sub1 = streamer.demand([1], () => {});
      const sub2 = streamer.demand([2], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(store.open(1)).not.toBeNull();
      expect(store.open(2)).not.toBeNull();

      sub2.close();
      await vi.advanceTimersByTimeAsync(300);

      expect(store.open(1)).not.toBeNull();
      expect(store.open(2)).toBeNull();
      sub1.close();
      await streamer.close();
    });

    it("should close open blocks when the stream drops", async () => {
      const cache = createJoined(1);
      const { store } = cache;
      const hooks: StreamHooks[] = [];
      const streamer = new MultiplexedStreamer({
        ...cache,
        openStreamer: createStreamOpener(
          [oneFrameStreamer([1], new framer.Frame({ 1: createTimestamps() }))],
          hooks,
        ),
      });
      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(store.open(1)).not.toBeNull();

      hooks[0].onDrop(new Error("conn lost"));

      expect(store.open(1)).toBeNull();
      sub.close();
      await streamer.close();
    });

    it("should close open blocks when the run loop dies", async () => {
      const cache = createJoined(1);
      const { store } = cache;
      const dying = new MockStreamer([1], undefined, [
        new framer.Frame({ 1: createTimestamps() }),
      ]);
      const replacement = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...cache,
        openStreamer: createStreamOpener([dying, replacement]),
      });
      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(1000);

      expect(replacement.iteratorVi).toHaveBeenCalled();
      expect(store.open(1)).toBeNull();
      expect(store.raw(1, 0).buffer).toBeNull();
      sub.close();
      await streamer.close();
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
            value: new framer.Frame({ 1: new Series(new Float32Array([1])) }),
          };
        }
        // A streamer whose read never returns: the run loop can never drain, so the
        // teardown await hangs.
        return await new Promise<IteratorResult<framer.Frame>>(() => {});
      });
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([stuck, pendingStreamer([2])]),
        removalDelay: TimeSpan.milliseconds(50),
      });

      const sub1 = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);

      // Last listener leaves: the no-keys teardown closes the streamer and awaits
      // the stuck run loop, holding the connection lock indefinitely.
      sub1.close();
      await vi.advanceTimersByTimeAsync(200);

      // A fresh subscription must still register even though the connection lock is
      // held by the stuck teardown.
      const sub2 = streamer.demand([2], () => {});
      expect(sub2.status(2).variant).toEqual("loading");
      sub2.close();
    });
  });

  describe("close path", () => {
    it("should cancel a pending removal timer on close", async () => {
      const ms1 = pendingStreamer([1]);
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([ms1]),
        removalDelay: TimeSpan.seconds(5),
      });
      const baseline = vi.getTimerCount();
      const sub = streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      sub.close();
      expect(vi.getTimerCount()).toBeGreaterThan(baseline);
      await streamer.close();
      expect(vi.getTimerCount()).toEqual(baseline);
    });

    it("should cancel a pending stream update on close", async () => {
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: createStreamOpener([pendingStreamer([1])]),
      });
      const baseline = vi.getTimerCount();
      streamer.demand([1], () => {});
      expect(vi.getTimerCount()).toBeGreaterThan(baseline);
      await streamer.close();
      expect(vi.getTimerCount()).toEqual(baseline);
    });

    it("should log an error when streamer.close throws and still mark the streamer closed", async () => {
      const ms1 = pendingStreamer([1]);
      const closeErr = new Error("close boom");
      ms1.closeVi.mockImplementation(() => {
        throw closeErr;
      });
      const { instrumentation, errorSpy } = createErrorSpy();

      let openCalls = 0;
      const opener = async (): Promise<framer.Streamer> => {
        openCalls++;
        return ms1;
      };
      const streamer = new MultiplexedStreamer({
        ...createCache(),
        openStreamer: opener,
        instrumentation,
      });

      streamer.demand([1], () => {});
      await vi.advanceTimersByTimeAsync(200);
      expect(openCalls).toBe(1);

      await streamer.close();

      expect(errorSpy).toHaveBeenCalledWith("failed to close streamer", {
        error: closeErr,
      });
      expect(() => streamer.demand([2], () => {})).toThrow(UnexpectedError);
    });
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import { type record, Series } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import z from "zod";

import { Engine } from "@/cache/engine";
import { type UnaryStore } from "@/cache/store";
import { type channel } from "@/channel";
import { type framer } from "@/framer";
import { Frame } from "@/framer/frame";

interface Doc extends record.Keyed<string> {
  key: string;
  name: string;
}

class MockStreamer implements framer.Streamer {
  keys: channel.Key[] = [];
  private readonly frames: framer.Frame[];
  private done = false;

  constructor(frames: framer.Frame[] = []) {
    this.frames = [...frames];
  }

  async update(): Promise<void> {}

  close(): void {
    this.done = true;
  }

  async next(): Promise<IteratorResult<framer.Frame>> {
    const fr = this.frames.shift();
    if (fr != null) return { done: false, value: fr };
    // Block until closed so the read loop stays alive like a real stream.
    while (!this.done) await new Promise((resolve) => setTimeout(resolve, 5));
    return { done: true, value: undefined };
  }

  async read(): Promise<framer.Frame> {
    const res = await this.next();
    if (res.done) throw new EOF();
    return res.value;
  }

  [Symbol.asyncIterator](): AsyncIterator<framer.Frame> {
    return this;
  }
}

const noopHandler = vi.fn((excOrFunc: any) => {
  if (typeof excOrFunc === "function")
    void (async () => {
      try {
        await excOrFunc();
      } catch {
        // swallowed
      }
    })();
});
const noopAsyncHandler = vi.fn(async (excOrFunc: any) => {
  if (typeof excOrFunc === "function")
    try {
      await excOrFunc();
    } catch {
      // swallowed
    }
});

const makeEngine = (openStreamer?: framer.StreamOpener) =>
  new Engine({
    openStreamer: openStreamer ?? (async () => new MockStreamer()),
    handleError: noopHandler,
    handleAsyncError: noopAsyncHandler,
  });

describe("Engine", () => {
  describe("getCache", () => {
    it("lazily constructs a cache on first call for a key", () => {
      const engine = makeEngine();
      const cache = engine.getCache<{ k: string }, number>("a");
      expect(cache).toBeDefined();
      expect(cache.get({ k: "x" })).toBeUndefined();
    });

    it("returns the same instance on subsequent calls with the same key", () => {
      const engine = makeEngine();
      const first = engine.getCache<{ k: string }, number>("a");
      const second = engine.getCache<{ k: string }, number>("a");
      expect(first).toBe(second);
    });

    it("returns distinct instances for distinct keys", () => {
      const engine = makeEngine();
      const a = engine.getCache<{ k: string }, number>("a");
      const b = engine.getCache<{ k: string }, number>("b");
      expect(a).not.toBe(b);
    });

    it("isolates entries across keys on the same engine", () => {
      const engine = makeEngine();
      const a = engine.getCache<{ k: string }, number>("a");
      const b = engine.getCache<{ k: string }, number>("b");
      a.set({ k: "x" }, { variant: "success", data: 1 });
      expect(b.get({ k: "x" })).toBeUndefined();
    });

    it("isolates caches across separate Engine instances", () => {
      const e1 = makeEngine();
      const e2 = makeEngine();
      const a1 = e1.getCache<{ k: string }, number>("a");
      const a2 = e2.getCache<{ k: string }, number>("a");
      a1.set({ k: "x" }, { variant: "success", data: 1 });
      expect(a2.get({ k: "x" })).toBeUndefined();
    });
  });

  describe("detached", () => {
    it("auto-registers empty local stores on first access", () => {
      const engine = new Engine({ openStreamer: null });
      expect(engine.detached).toBe(true);
      const store = engine.store<string, number>("docs");
      store.set("a", 1);
      expect(engine.store<string, number>("docs").get("a")).toEqual(1);
    });

    it("keeps ensureStreaming a no-op and stays at epoch 0", async () => {
      const engine = new Engine({ openStreamer: null });
      engine.store<string, number>("docs");
      await engine.ensureStreaming();
      expect(engine.epoch).toEqual(0);
      const registered = () => engine.registerStore("late", { listeners: [] });
      expect(registered).not.toThrow();
      await engine.close();
    });

    it("still throws on unknown keys when a stream source exists", () => {
      const engine = makeEngine();
      expect(engine.detached).toBe(false);
      expect(() => engine.store("nope")).toThrow("not registered");
    });
  });

  describe("registerStore", () => {
    it("registers a store retrievable through store()", () => {
      const engine = makeEngine();
      engine.registerStore("docs", { listeners: [] });
      const store = engine.store<string, Doc>("docs");
      store.set({ key: "k1", name: "a" });
      expect(store.get("k1")).toEqual({ key: "k1", name: "a" });
    });

    it("throws on duplicate registration", () => {
      const engine = makeEngine();
      engine.registerStore("docs", { listeners: [] });
      expect(() => engine.registerStore("docs", { listeners: [] })).toThrow(
        "already registered",
      );
    });

    it("throws when accessing an unregistered store", () => {
      const engine = makeEngine();
      expect(() => engine.store("nope")).toThrow("not registered");
    });

    it("throws when registering after streaming has started", async () => {
      const engine = makeEngine();
      engine.registerStore("docs", { listeners: [] });
      await engine.ensureStreaming();
      expect(() => engine.registerStore("late", { listeners: [] })).toThrow(
        "after streaming",
      );
      await engine.close();
    });

    it("shares data across scopes while suppressing same-scope listeners", () => {
      const engine = makeEngine();
      engine.registerStore("docs", { listeners: [] });
      const a = engine.store<string, Doc>("docs", "scopeA");
      const b = engine.store<string, Doc>("docs", "scopeB");
      const aListener = vi.fn();
      const bListener = vi.fn();
      a.onSet(aListener);
      b.onSet(bListener);
      a.set({ key: "k1", name: "a" });
      expect(aListener).not.toHaveBeenCalled();
      expect(bListener).toHaveBeenCalledTimes(1);
      expect(b.get("k1")).toEqual({ key: "k1", name: "a" });
    });
  });

  describe("lazy streaming and epochs", () => {
    it("does not open the stream until ensureStreaming is called", async () => {
      const opener = vi.fn(async () => new MockStreamer());
      const engine = makeEngine(opener);
      engine.registerStore("docs", { listeners: [] });
      expect(opener).not.toHaveBeenCalled();
      expect(engine.epoch).toBe(0);
      await engine.ensureStreaming();
      expect(opener).toHaveBeenCalledTimes(1);
      expect(engine.epoch).toBe(1);
      await engine.close();
    });

    it("opens the stream once across concurrent ensureStreaming calls", async () => {
      const opener = vi.fn(async () => new MockStreamer());
      const engine = makeEngine(opener);
      await Promise.all([
        engine.ensureStreaming(),
        engine.ensureStreaming(),
        engine.ensureStreaming(),
      ]);
      expect(opener).toHaveBeenCalledTimes(1);
      expect(engine.epoch).toBe(1);
      await engine.close();
    });

    it("notifies epoch listeners on first open", async () => {
      const engine = makeEngine();
      const epochs: number[] = [];
      engine.onEpoch((epoch) => epochs.push(epoch));
      await engine.ensureStreaming();
      expect(epochs).toEqual([1]);
      await engine.close();
    });

    it("routes streamed changes into the registered store", async () => {
      const schema = z.object({ key: z.string(), name: z.string() });
      const opener = async () =>
        new MockStreamer([
          new Frame({ docs_set: new Series([{ key: "k1", name: "remote" }]) }),
        ]);
      const engine = makeEngine(opener);
      engine.registerStore("docs", {
        listeners: [
          {
            channel: "docs_set",
            schema,
            onChange: ({ changed, store }) => {
              (store.docs as UnaryStore<string, Doc>).set(changed as Doc);
            },
          },
        ],
      });
      await engine.ensureStreaming();
      const store = engine.store<string, Doc>("docs");
      await expect.poll(() => store.get("k1")).toEqual({ key: "k1", name: "remote" });
      await engine.close();
    });

    it("bumps the epoch and reconciles after a stream reopen", async () => {
      let opens = 0;
      const refetch = vi.fn(async (keys: string[]) =>
        keys.filter((k) => k !== "gone").map((k) => ({ key: k, name: `${k}-fresh` })),
      );
      // Fails on first read with a non-EOF error, forcing a reopen.
      const failing: framer.Streamer = {
        keys: [],
        update: async () => {},
        close: () => {},
        next: async () => {
          throw new Error("conn dropped");
        },
        read: async () => {
          throw new Error("conn dropped");
        },
        [Symbol.asyncIterator]() {
          return this;
        },
      };
      const opener = async (): Promise<framer.Streamer> => {
        opens++;
        if (opens === 1) return failing;
        return new MockStreamer();
      };
      const engine = makeEngine(opener);
      engine.registerStore("docs", { listeners: [], refetch });
      const store = engine.store<string, Doc>("docs");
      store.set([
        { key: "kept", name: "stale" },
        { key: "gone", name: "deleted-on-server" },
      ]);
      const epochs: number[] = [];
      engine.onEpoch((epoch) => epochs.push(epoch));
      await engine.ensureStreaming();
      await expect.poll(() => engine.epoch, { timeout: 5000 }).toBe(2);
      expect(epochs).toEqual([1, 2]);
      await expect.poll(() => refetch.mock.calls.length).toBeGreaterThan(0);
      expect(refetch).toHaveBeenCalledWith(["kept", "gone"]);
      await expect.poll(() => store.status("gone")).toBe("tombstoned");
      expect(store.getTombstone("gone")?.corpse).toEqual({
        key: "gone",
        name: "deleted-on-server",
      });
      expect(store.get("kept")).toEqual({ key: "kept", name: "kept-fresh" });
      await engine.close();
    });
  });

  describe("reconcile", () => {
    it("skips stores without a refetch", async () => {
      const engine = makeEngine();
      engine.registerStore("docs", { listeners: [] });
      engine.store<string, Doc>("docs").set({ key: "k1", name: "a" });
      await engine.reconcile();
      expect(engine.store<string, Doc>("docs").get("k1")).toEqual({
        key: "k1",
        name: "a",
      });
    });

    it("skips refetch when the store is empty", async () => {
      const refetch = vi.fn(async () => []);
      const engine = makeEngine();
      engine.registerStore("docs", { listeners: [], refetch });
      await engine.reconcile();
      expect(refetch).not.toHaveBeenCalled();
    });

    it("continues reconciling other stores when one refetch fails", async () => {
      const engine = makeEngine();
      const goodRefetch = vi.fn(async (keys: string[]) =>
        keys.map((k) => ({ key: k, name: "fresh" })),
      );
      engine.registerStore("bad", {
        listeners: [],
        refetch: async () => {
          throw new Error("network");
        },
      });
      engine.registerStore("good", { listeners: [], refetch: goodRefetch });
      engine.store<string, Doc>("bad").set({ key: "b1", name: "a" });
      engine.store<string, Doc>("good").set({ key: "g1", name: "stale" });
      await engine.reconcile();
      expect(engine.store<string, Doc>("good").get("g1")).toEqual({
        key: "g1",
        name: "fresh",
      });
      expect(engine.store<string, Doc>("bad").get("b1")).toEqual({
        key: "b1",
        name: "a",
      });
    });
  });
});

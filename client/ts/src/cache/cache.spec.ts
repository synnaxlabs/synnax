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

import { cache } from "@/cache";
import { type channel } from "@/channel";
import { framer } from "@/framer";

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

const wrapOpener =
  (opener: framer.StreamOpener): cache.StreamOpener =>
  async (channels, { onOpen, onReopen }) => {
    const hardened = await framer.HardenedStreamer.open(
      opener,
      channels,
      undefined,
      onReopen,
    );
    onOpen?.();
    return new framer.ObservableStreamer(hardened);
  };

const makeEngine = (openStreamer?: framer.StreamOpener) =>
  new cache.Cache({
    openStreamer: wrapOpener(openStreamer ?? (async () => new MockStreamer())),
    onInternalError: vi.fn(),
  });

describe("Cache", () => {
  describe("detached", () => {
    it("auto-registers empty local tables on first access", () => {
      const engine = new cache.Cache({ openStreamer: null });
      expect(engine.detached).toBe(true);
      const table = engine.table<string, number>("docs");
      table.set("a", 1);
      expect(engine.table<string, number>("docs").get("a")).toEqual(1);
    });

    it("keeps ensureStreaming a no-op and stays at epoch 0", async () => {
      const engine = new cache.Cache({ openStreamer: null });
      engine.table<string, number>("docs");
      await engine.ensureStreaming();
      expect(engine.epoch).toEqual(0);
      const registered = () => engine.registerTable("late", { listeners: [] });
      expect(registered).not.toThrow();
      await engine.close();
    });

    it("still throws on unknown keys when a stream source exists", () => {
      const engine = makeEngine();
      expect(engine.detached).toBe(false);
      expect(() => engine.table("nope")).toThrow("not registered");
    });
  });

  describe("registerTable", () => {
    it("registers a table retrievable through table()", () => {
      const engine = makeEngine();
      engine.registerTable("docs", { listeners: [] });
      const table = engine.table<string, Doc>("docs");
      table.set("k1", { key: "k1", name: "a" });
      expect(table.get("k1")).toEqual({ key: "k1", name: "a" });
    });

    it("throws on duplicate registration", () => {
      const engine = makeEngine();
      engine.registerTable("docs", { listeners: [] });
      expect(() => engine.registerTable("docs", { listeners: [] })).toThrow(
        "already registered",
      );
    });

    it("throws when accessing an unregistered table", () => {
      const engine = makeEngine();
      expect(() => engine.table("nope")).toThrow("not registered");
    });

    it("throws when registering after streaming has started", async () => {
      const engine = makeEngine();
      engine.registerTable("docs", { listeners: [] });
      await engine.ensureStreaming();
      expect(() => engine.registerTable("late", { listeners: [] })).toThrow(
        "after streaming",
      );
      await engine.close();
    });

    it("silences sets whose value equals the cached row", () => {
      const engine = makeEngine();
      engine.registerTable("docs", { listeners: [] });
      const table = engine.table<string, Doc>("docs");
      const listener = vi.fn();
      table.subscribe(listener);
      table.set("k1", { key: "k1", name: "a" });
      expect(listener).toHaveBeenCalledTimes(1);
      table.set("k1", { key: "k1", name: "a" });
      expect(listener).toHaveBeenCalledTimes(1);
      table.set("k1", { key: "k1", name: "b" });
      expect(listener).toHaveBeenCalledTimes(2);
    });

    it("respects a custom equal override from the table config", () => {
      const engine = makeEngine();
      engine.registerTable<string, Doc>("docs", {
        listeners: [],
        equal: (a, b) => a.name.toLowerCase() === b.name.toLowerCase(),
      });
      const table = engine.table<string, Doc>("docs");
      const listener = vi.fn();
      table.subscribe(listener);
      table.set("k1", { key: "k1", name: "A" });
      table.set("k1", { key: "k1", name: "a" });
      expect(listener).toHaveBeenCalledTimes(1);
    });
  });

  describe("lazy streaming and epochs", () => {
    it("does not open the stream until ensureStreaming is called", async () => {
      const opener = vi.fn(async () => new MockStreamer());
      const engine = makeEngine(opener);
      engine.registerTable("docs", { listeners: [] });
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

    it("routes streamed changes into the registered table", async () => {
      const schema = z.object({ key: z.string(), name: z.string() });
      const opener = async () =>
        new MockStreamer([
          new framer.Frame({ docs_set: new Series([{ key: "k1", name: "remote" }]) }),
        ]);
      const engine = makeEngine(opener);
      engine.registerTable("docs", {
        listeners: [
          {
            channel: "docs_set",
            schema,
            onChange: ({ changed, store }) => {
              const doc = changed as Doc;
              (store.docs as cache.Table<string, Doc>).set(doc.key, doc);
            },
          },
        ],
      });
      await engine.ensureStreaming();
      const table = engine.table<string, Doc>("docs");
      await expect.poll(() => table.get("k1")).toEqual({ key: "k1", name: "remote" });
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
      engine.registerTable("docs", { listeners: [], refetch });
      const table = engine.table<string, Doc>("docs");
      table.setMany([
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
      await expect.poll(() => table.status("gone")).toBe("tombstoned");
      expect(table.getTombstone("gone")?.corpse).toEqual({
        key: "gone",
        name: "deleted-on-server",
      });
      expect(table.get("kept")).toEqual({ key: "kept", name: "kept-fresh" });
      await engine.close();
    });
  });

  describe("reconcile", () => {
    it("skips tables without a refetch", async () => {
      const engine = makeEngine();
      engine.registerTable("docs", { listeners: [] });
      engine.table<string, Doc>("docs").set("k1", { key: "k1", name: "a" });
      await engine.reconcile();
      expect(engine.table<string, Doc>("docs").get("k1")).toEqual({
        key: "k1",
        name: "a",
      });
    });

    it("skips refetch when the table is empty", async () => {
      const refetch = vi.fn(async () => []);
      const engine = makeEngine();
      engine.registerTable("docs", { listeners: [], refetch });
      await engine.reconcile();
      expect(refetch).not.toHaveBeenCalled();
    });

    it("continues reconciling other tables when one refetch fails", async () => {
      const engine = makeEngine();
      const goodRefetch = vi.fn(async (keys: string[]) =>
        keys.map((k) => ({ key: k, name: "fresh" })),
      );
      engine.registerTable("bad", {
        listeners: [],
        refetch: async () => {
          throw new Error("network");
        },
      });
      engine.registerTable("good", { listeners: [], refetch: goodRefetch });
      engine.table<string, Doc>("bad").set("b1", { key: "b1", name: "a" });
      engine.table<string, Doc>("good").set("g1", { key: "g1", name: "stale" });
      await engine.reconcile();
      expect(engine.table<string, Doc>("good").get("g1")).toEqual({
        key: "g1",
        name: "fresh",
      });
      expect(engine.table<string, Doc>("bad").get("b1")).toEqual({
        key: "b1",
        name: "a",
      });
    });
  });
});

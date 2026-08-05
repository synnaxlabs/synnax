// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { EOF } from "@synnaxlabs/freighter";
import { type record, Series, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";
import z from "zod";

import { type channel } from "@/channel";
import { NotFoundError } from "@/errors";
import { framer } from "@/framer";
import { query } from "@/query";

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
  (opener: framer.StreamOpener): query.StreamOpener =>
  async (channels, { onOpen, onReopen }) => {
    const hardened = await framer.HardenedStreamer.open(
      opener,
      channels,
      // near-zero backoff so forced reopens land within poll timeouts
      { baseInterval: TimeSpan.milliseconds(1) },
      onReopen,
    );
    onOpen?.();
    return new framer.ObservableStreamer(hardened);
  };

const makeEngine = (openStreamer?: framer.StreamOpener) =>
  new query.Cache({
    openStreamer: wrapOpener(openStreamer ?? (async () => new MockStreamer())),
    onError: vi.fn(),
  });

// Fails on every read with a non-EOF error, forcing a reopen.
const failingStreamer = (): framer.Streamer => ({
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
    return this as AsyncIterator<framer.Frame>;
  },
});

/** Delivers one final frame only after close is requested, then ends. */
const lateFrameStreamer = (frame: framer.Frame): framer.Streamer => {
  let closed = false;
  let emitted = false;
  const next = async (): Promise<IteratorResult<framer.Frame>> => {
    while (!closed) await new Promise((resolve) => setTimeout(resolve, 2));
    if (emitted) return { done: true, value: undefined };
    emitted = true;
    return { done: false, value: frame };
  };
  return {
    keys: [],
    update: async () => {},
    close: () => {
      closed = true;
    },
    next,
    read: async () => {
      const res = await next();
      if (res.done) throw new EOF();
      return res.value;
    },
    [Symbol.asyncIterator]() {
      return this as AsyncIterator<framer.Frame>;
    },
  };
};

/** An engine whose first stream drops immediately, forcing one reopen. */
const makeReopeningEngine = () => {
  let opens = 0;
  return makeEngine(async (): Promise<framer.Streamer> => {
    opens++;
    if (opens === 1) return failingStreamer();
    return new MockStreamer();
  });
};

describe("Cache", () => {
  describe("detached (null openStreamer)", () => {
    it("creates purely local tables", () => {
      const cache = new query.Cache({ openStreamer: null });
      const table = cache.createTable<string, number>({ name: "docs" });
      table.set("a", 1);
      expect(table.get("a")).toEqual(1);
    });

    it("keeps ensureStreaming a no-op and stays at epoch 0", async () => {
      const cache = new query.Cache({ openStreamer: null });
      cache.createTable<string, number>({ name: "docs" });
      await cache.ensureStreaming();
      expect(cache.epoch).toEqual(0);
      const late = () => cache.createTable({ name: "late" });
      expect(late).not.toThrow();
      await cache.close();
    });
  });

  describe("createTable", () => {
    it("returns a live table owned by the cache", () => {
      const cache = makeEngine();
      const table = cache.createTable<string, Doc>({ name: "docs" });
      table.set("k1", { key: "k1", name: "a" });
      expect(table.get("k1")).toEqual({ key: "k1", name: "a" });
    });

    it("throws when creating a table after streaming has started", async () => {
      const cache = makeEngine();
      cache.createTable({ name: "docs" });
      await cache.ensureStreaming();
      expect(() => cache.createTable({ name: "late" })).toThrow("after streaming");
      await cache.close();
    });

    it("throws when registering a reaction after streaming has started", async () => {
      const cache = makeEngine();
      cache.createTable({ name: "docs" });
      await cache.ensureStreaming();
      expect(() =>
        cache.listen({ channel: "late", schema: z.string(), onChange: vi.fn() }),
      ).toThrow("after streaming");
      await cache.close();
    });

    it("silences sets whose value equals the cached row", () => {
      const cache = makeEngine();
      const table = cache.createTable<string, Doc>({ name: "docs" });
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
      const cache = makeEngine();
      const table = cache.createTable<string, Doc>({
        name: "docs",
        equal: (a, b) => a.name.toLowerCase() === b.name.toLowerCase(),
      });
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
      const cache = makeEngine(opener);
      cache.createTable({ name: "docs" });
      expect(opener).not.toHaveBeenCalled();
      expect(cache.epoch).toBe(0);
      await cache.ensureStreaming();
      expect(opener).toHaveBeenCalledTimes(1);
      expect(cache.epoch).toBe(1);
      await cache.close();
    });

    it("opens the stream once across concurrent ensureStreaming calls", async () => {
      const opener = vi.fn(async () => new MockStreamer());
      const cache = makeEngine(opener);
      await Promise.all([
        cache.ensureStreaming(),
        cache.ensureStreaming(),
        cache.ensureStreaming(),
      ]);
      expect(opener).toHaveBeenCalledTimes(1);
      expect(cache.epoch).toBe(1);
      await cache.close();
    });

    it("notifies epoch listeners on first open", async () => {
      const cache = makeEngine();
      const epochs: number[] = [];
      cache.onEpoch((epoch) => epochs.push(epoch));
      await cache.ensureStreaming();
      expect(epochs).toEqual([1]);
      await cache.close();
    });

    it("routes streamed changes into the table via a declared set listener", async () => {
      const schema = z.object({ key: z.string(), name: z.string() });
      const opener = async () =>
        new MockStreamer([
          new framer.Frame({ docs_set: new Series([{ key: "k1", name: "remote" }]) }),
        ]);
      const cache = makeEngine(opener);
      const table = cache.createTable<string, Doc>({
        name: "docs",
        listen: [query.createSetListener("docs_set", schema)],
      });
      await cache.ensureStreaming();
      await expect.poll(() => table.get("k1")).toEqual({ key: "k1", name: "remote" });
      await cache.close();
    });

    it("routes streamed deletions via a declared delete listener", async () => {
      const opener = async () =>
        new MockStreamer([new framer.Frame({ docs_delete: new Series(["k1"]) })]);
      const cache = makeEngine(opener);
      const table = cache.createTable<string, Doc>({
        name: "docs",
        listen: [query.createDeleteListener("docs_delete", z.string())],
      });
      table.set("k1", { key: "k1", name: "a" });
      await cache.ensureStreaming();
      await expect.poll(() => table.get("k1")).toBeUndefined();
      await cache.close();
    });

    it("refetches announced keys via a declared fetch listener", async () => {
      const fetch = vi.fn(async (keys: string[]) =>
        keys.map((k) => ({ key: k, name: `${k}-fresh` })),
      );
      const opener = async () =>
        new MockStreamer([new framer.Frame({ docs_set: new Series(["k1"]) })]);
      const cache = makeEngine(opener);
      const table = cache.createTable<string, Doc>({
        name: "docs",
        fetch,
        listen: [query.createFetchListener("docs_set", z.string())],
      });
      table.set("k1", { key: "k1", name: "stale" });
      await cache.ensureStreaming();
      await expect.poll(() => table.get("k1")).toEqual({ key: "k1", name: "k1-fresh" });
      expect(fetch).toHaveBeenCalledWith(["k1"]);
      await cache.close();
    });

    it("delivers channel reactions registered via listen", async () => {
      const opener = async () =>
        new MockStreamer([new framer.Frame({ commands: new Series(["start"]) })]);
      const cache = makeEngine(opener);
      const onChange = vi.fn();
      cache.listen({ channel: "commands", schema: z.string(), onChange });
      await cache.ensureStreaming();
      await expect.poll(() => onChange.mock.calls.length).toBe(1);
      expect(onChange).toHaveBeenCalledWith("start");
      await cache.close();
    });

    it("bumps the epoch and reconciles after a stream reopen", async () => {
      const fetch = vi.fn(async (keys: string[]) =>
        keys.filter((k) => k !== "gone").map((k) => ({ key: k, name: `${k}-fresh` })),
      );
      const cache = makeReopeningEngine();
      const table = cache.createTable<string, Doc>({ name: "docs", fetch });
      table.set([
        { key: "kept", name: "stale" },
        { key: "gone", name: "deleted-on-server" },
      ]);
      const epochs: number[] = [];
      cache.onEpoch((epoch) => epochs.push(epoch));
      await cache.ensureStreaming();
      await expect.poll(() => cache.epoch).toBe(2);
      expect(epochs).toEqual([1, 2]);
      await expect.poll(() => fetch.mock.calls.length).toBeGreaterThan(0);
      expect(fetch).toHaveBeenCalledWith(["kept", "gone"]);
      await expect.poll(() => table.status("gone")).toBe("tombstoned");
      expect(table.getTombstone("gone")?.corpse).toEqual({
        key: "gone",
        name: "deleted-on-server",
      });
      expect(table.get("kept")).toEqual({ key: "kept", name: "kept-fresh" });
      await cache.close();
    });
  });

  describe("reconciliation after reopen", () => {
    it("leaves tables without a fetch untouched", async () => {
      const cache = makeReopeningEngine();
      const table = cache.createTable<string, Doc>({ name: "docs" });
      table.set("k1", { key: "k1", name: "a" });
      await cache.ensureStreaming();
      await expect.poll(() => cache.epoch).toBe(2);
      expect(table.get("k1")).toEqual({ key: "k1", name: "a" });
      await cache.close();
    });

    it("skips the fetch when the table is empty", async () => {
      const fetch = vi.fn(async () => []);
      const cache = makeReopeningEngine();
      cache.createTable({ name: "docs", fetch });
      await cache.ensureStreaming();
      await expect.poll(() => cache.epoch).toBe(2);
      expect(fetch).not.toHaveBeenCalled();
      await cache.close();
    });

    it("probes keys individually when a strict fetch rejects the batch", async () => {
      const fetch = vi.fn(async (keys: string[]) => {
        if (keys.includes("gone"))
          throw new NotFoundError("Docs with keys [gone] not found");
        return keys.map((k) => ({ key: k, name: `${k}-fresh` }));
      });
      const cache = makeReopeningEngine();
      const table = cache.createTable<string, Doc>({ name: "docs", fetch });
      table.set([
        { key: "kept", name: "stale" },
        { key: "gone", name: "deleted-on-server" },
      ]);
      await cache.ensureStreaming();
      await expect.poll(() => table.status("gone")).toBe("tombstoned");
      expect(table.get("kept")).toEqual({ key: "kept", name: "kept-fresh" });
      await cache.close();
    });

    it("tombstones every key when the whole batch has vanished", async () => {
      const onError = vi.fn();
      let opens = 0;
      const cache = new query.Cache({
        openStreamer: wrapOpener(async (): Promise<framer.Streamer> => {
          opens++;
          if (opens === 1) return failingStreamer();
          return new MockStreamer();
        }),
        onError,
      });
      const fetch = vi.fn(async (keys: string[]): Promise<Doc[]> => {
        throw new NotFoundError(`Docs with keys [${keys.join(",")}] not found`);
      });
      const table = cache.createTable<string, Doc>({ name: "docs", fetch });
      table.set([
        { key: "a", name: "one" },
        { key: "b", name: "two" },
      ]);
      await cache.ensureStreaming();
      await expect.poll(() => table.status("a")).toBe("tombstoned");
      expect(table.status("b")).toBe("tombstoned");
      expect(onError).not.toHaveBeenCalled();
      await cache.close();
    });

    it("reports the failure when a probe rejects with a non-NotFound error", async () => {
      const onError = vi.fn();
      let opens = 0;
      const cache = new query.Cache({
        openStreamer: wrapOpener(async (): Promise<framer.Streamer> => {
          opens++;
          if (opens === 1) return failingStreamer();
          return new MockStreamer();
        }),
        onError,
      });
      const fetch = vi.fn(async (keys: string[]): Promise<Doc[]> => {
        if (keys.length > 1)
          throw new NotFoundError(`Docs with keys [${keys.join(",")}] not found`);
        if (keys[0] === "b") throw new Error("network");
        return keys.map((k) => ({ key: k, name: `${k}-fresh` }));
      });
      const table = cache.createTable<string, Doc>({ name: "docs", fetch });
      table.set([
        { key: "a", name: "one" },
        { key: "b", name: "two" },
      ]);
      await cache.ensureStreaming();
      await expect.poll(() => onError.mock.calls.length).toBe(1);
      const error = onError.mock.calls[0][0] as Error;
      expect(error.message).toContain("failed to reconcile docs cache");
      expect(table.get("a")).toEqual({ key: "a", name: "one" });
      expect(table.get("b")).toEqual({ key: "b", name: "two" });
      await cache.close();
    });

    it("continues reconciling other tables when one fetch fails", async () => {
      const cache = makeReopeningEngine();
      const goodFetch = vi.fn(async (keys: string[]) =>
        keys.map((k) => ({ key: k, name: "fresh" })),
      );
      const bad = cache.createTable<string, Doc>({
        name: "bad",
        fetch: async () => {
          throw new Error("network");
        },
      });
      const good = cache.createTable<string, Doc>({
        name: "good",
        fetch: goodFetch,
      });
      bad.set("b1", { key: "b1", name: "a" });
      good.set("g1", { key: "g1", name: "stale" });
      await cache.ensureStreaming();
      await expect.poll(() => good.get("g1")).toEqual({ key: "g1", name: "fresh" });
      expect(bad.get("b1")).toEqual({ key: "b1", name: "a" });
      await cache.close();
    });

    it("discards a reconcile that completes after a reset", async () => {
      let release: (docs: Doc[]) => void = () => {};
      const gate = new Promise<Doc[]>((resolve) => {
        release = resolve;
      });
      const fetch = vi.fn(async () => await gate);
      const cache = makeReopeningEngine();
      const table = cache.createTable<string, Doc>({ name: "docs", fetch });
      table.set([{ key: "kept", name: "stale" }]);
      await cache.ensureStreaming();
      await expect.poll(() => fetch.mock.calls.length).toBe(1);
      await cache.reset();
      release([{ key: "kept", name: "old-cluster" }]);
      await gate;
      await new Promise((resolve) => setTimeout(resolve, 5));
      expect(table.get("kept")).toBeUndefined();
      expect(table.keys()).toEqual([]);
      await cache.close();
    });
  });

  describe("reset", () => {
    it("clears every table and returns the epoch to 0", async () => {
      const cache = makeEngine();
      const table = cache.createTable<string, Doc>({ name: "docs" });
      table.set("k1", { key: "k1", name: "a" });
      table.set("k2", { key: "k2", name: "b" });
      table.delete("k2");
      await cache.ensureStreaming();
      expect(cache.epoch).toBe(1);
      const epochs: number[] = [];
      cache.onEpoch((epoch) => epochs.push(epoch));
      await cache.reset();
      expect(cache.epoch).toBe(0);
      expect(epochs).toEqual([0]);
      expect(table.get()).toEqual([]);
      expect(table.status("k1")).toEqual("unknown");
      expect(table.status("k2")).toEqual("unknown");
      await cache.close();
    });

    it("reopens the stream on the next demand and returns to epoch 1", async () => {
      const cache = makeEngine();
      cache.createTable<string, Doc>({ name: "docs" });
      await cache.ensureStreaming();
      expect(cache.epoch).toBe(1);
      await cache.reset();
      await cache.ensureStreaming();
      expect(cache.epoch).toBe(1);
      await cache.close();
    });

    it("resets answer spaces so subscribers refetch", async () => {
      const cache = makeEngine();
      const table = cache.createTable<string, Doc>({ name: "docs" });
      const space = cache.queries<string, Doc, string, Doc>({
        name: "docs",
        table,
        fetch: async (key) => {
          table.set(key, { key, name: "fetched" });
          return [key];
        },
        compose: (records) => records[0],
        keyOf: (query) => query,
        single: true,
      });
      await space.retrieve("k1");
      expect(space.getCached("k1")).toEqual({ key: "k1", name: "fetched" });
      const results: Array<query.Cached<Doc> | undefined> = [];
      const detach = space.onChange("k1", (result) => results.push(result));
      await cache.reset();
      expect(results).toEqual([undefined]);
      expect(space.getCached("k1")).toBeUndefined();
      await expect(space.retrieve("k1")).resolves.toEqual({
        key: "k1",
        name: "fetched",
      });
      detach();
      await cache.close();
    });

    it("ignores open callbacks from a streamer retired mid-open", async () => {
      let release: () => void = () => {};
      const gate = new Promise<void>((resolve) => {
        release = resolve;
      });
      const cache = makeEngine(async () => {
        await gate;
        return new MockStreamer();
      });
      cache.createTable<string, Doc>({ name: "docs" });
      const epochs: number[] = [];
      cache.onEpoch((epoch) => epochs.push(epoch));
      const demanded = cache.ensureStreaming();
      const resetting = cache.reset();
      release();
      await Promise.all([demanded, resetting]);
      expect(cache.epoch).toBe(0);
      expect(epochs).toEqual([0]);
      await cache.ensureStreaming();
      expect(cache.epoch).toBe(1);
      await cache.close();
    });

    it("drains a late frame from the retired stream before clearing", async () => {
      const schema = z.object({ key: z.string(), name: z.string() });
      const delivered = vi.fn();
      const cache = makeEngine(async () =>
        lateFrameStreamer(
          new framer.Frame({ docs_set: new Series([{ key: "k9", name: "late" }]) }),
        ),
      );
      const table = cache.createTable<string, Doc>({
        name: "docs",
        listen: [query.createSetListener("docs_set", schema)],
      });
      cache.listen({ channel: "docs_set", schema, onChange: delivered });
      await cache.ensureStreaming();
      await cache.reset();
      await expect.poll(() => delivered.mock.calls.length).toBe(1);
      expect(table.get()).toEqual([]);
      await cache.close();
    });

    it("completes the reset when closing the retired stream fails", async () => {
      const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
      try {
        const cache = new query.Cache({
          openStreamer: async (_channels, { onOpen }) => {
            onOpen?.();
            return {
              onChange: () => {},
              close: async () => {
                throw new Error("socket already dead");
              },
            };
          },
          onError: vi.fn(),
        });
        const table = cache.createTable<string, Doc>({ name: "docs" });
        table.set("k1", { key: "k1", name: "a" });
        await cache.ensureStreaming();
        expect(cache.epoch).toBe(1);
        await cache.reset();
        expect(cache.epoch).toBe(0);
        expect(table.get()).toEqual([]);
        expect(consoleError).toHaveBeenCalled();
      } finally {
        consoleError.mockRestore();
      }
    });
  });
});

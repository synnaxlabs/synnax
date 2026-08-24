// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => {
  interface Store {
    path: string;
    options: unknown;
    entries: Map<string, unknown>;
    saves: number;
  }
  const ref: { engine: "web" | "tauri"; stores: Store[] } = {
    engine: "web",
    stores: [],
  };
  return ref;
});

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

// The desktop store is a file behind tauri IPC, so the plugin and the path helpers
// stand in for it. Everything above them is the real TauriKV.
vi.mock("@tauri-apps/api/path", () => ({
  appLocalDataDir: async () => "/local",
  join: async (...parts: string[]) => parts.join("/"),
}));

vi.mock("@tauri-apps/plugin-store", () => ({
  LazyStore: class {
    private readonly entries = new Map<string, unknown>();

    constructor(path: string, options: unknown) {
      mocks.stores.push({ path, options, entries: this.entries, saves: 0 });
    }

    private get record() {
      return mocks.stores[mocks.stores.length - 1];
    }

    async get(key: string) {
      return this.entries.get(key);
    }

    async set(key: string, value: unknown) {
      this.entries.set(key, value);
    }

    async delete(key: string) {
      this.entries.delete(key);
    }

    async keys() {
      return Array.from(this.entries.keys());
    }

    async length() {
      return this.entries.size;
    }

    async clear() {
      this.entries.clear();
    }

    async save() {
      this.record.saves += 1;
    }
  },
}));

import { Persist } from "@/session/persist";

// Runtime.ENGINE resolves to "web" under jsdom, so openSugaredKV returns an
// IndexedDB-backed store. The suite exercises that implementation end to end.
describe("openSugaredKV (IndexedDBKV)", () => {
  beforeEach(async () => {
    mocks.engine = "web";
    await Persist.openSugaredKV("base").clear();
  });

  it("should round-trip a value through JSON serialization", async () => {
    const kv = Persist.openSugaredKV("base");
    await kv.set("key", { a: 1, nested: { b: 2 } });
    await expect(kv.get("key")).resolves.toEqual({ a: 1, nested: { b: 2 } });
  });

  it("should return null for a missing key", async () => {
    const kv = Persist.openSugaredKV("base");
    await expect(kv.get("missing")).resolves.toBeNull();
  });

  it("should delete a key", async () => {
    const kv = Persist.openSugaredKV("base");
    await kv.set("key", "value");
    await kv.delete("key");
    await expect(kv.get("key")).resolves.toBeNull();
  });

  it("should count only keys scoped to its base", async () => {
    const kv = Persist.openSugaredKV("base");
    const other = Persist.openSugaredKV("other");
    await kv.set("a", 1);
    await kv.set("b", 2);
    await other.set("c", 3);
    await expect(kv.length()).resolves.toBe(2);
    await expect(other.length()).resolves.toBe(1);
  });

  it("should not resolve clear before its transaction commits", async () => {
    const kv = Persist.openSugaredKV("base");
    await kv.set("key", "value");
    // Both callers of clear reload the page as soon as it resolves, and navigation
    // aborts an uncommitted transaction, so resolving early loses the clear.
    let committed = false;
    // The capture is re-bound through .call, so the unbound-method hazard does not
    // apply.
    // eslint-disable-next-line @typescript-eslint/unbound-method
    const original = IDBDatabase.prototype.transaction;
    const spy = vi
      .spyOn(IDBDatabase.prototype, "transaction")
      .mockImplementation(function (
        this: IDBDatabase,
        name: string | Iterable<string>,
        mode?: IDBTransactionMode,
      ) {
        const tx = original.call(this, name, mode);
        if (mode === "readwrite")
          tx.addEventListener("complete", () => (committed = true));
        return tx;
      });
    try {
      await kv.clear();
      expect(committed).toBe(true);
    } finally {
      spy.mockRestore();
    }
  });

  it("should ask the engine to keep the database", async () => {
    // jsdom ships no Storage Manager, so the standard API is supplied here.
    const persist = vi.fn(async () => true);
    vi.stubGlobal("navigator", { ...navigator, storage: { persist } });
    // Without the request the browser may evict the whole session under pressure.
    await Persist.openSugaredKV("evictable").length();
    expect(persist).toHaveBeenCalled();
    vi.unstubAllGlobals();
  });

  it("should list every key it holds", async () => {
    const kv = Persist.openSugaredKV("base");
    await kv.set("a", 1);
    await kv.set("b", 2);
    await expect(kv.keys().then((keys) => keys.sort())).resolves.toEqual(["a", "b"]);
  });

  it("should clear only keys scoped to its base", async () => {
    const kv = Persist.openSugaredKV("base");
    const other = Persist.openSugaredKV("other");
    await kv.set("a", 1);
    await other.set("c", 3);
    await kv.clear();
    await expect(kv.length()).resolves.toBe(0);
    await expect(other.get("c")).resolves.toBe(3);
  });

  it("should isolate stores that share a key name across different bases", async () => {
    const a = Persist.openSugaredKV("a");
    const b = Persist.openSugaredKV("b");
    await a.set("shared", "from-a");
    await b.set("shared", "from-b");
    await expect(a.get("shared")).resolves.toBe("from-a");
    await expect(b.get("shared")).resolves.toBe("from-b");
  });

  it("should persist across separate handles to the same base", async () => {
    await Persist.openSugaredKV("base").set("key", "value");
    await expect(Persist.openSugaredKV("base").get("key")).resolves.toBe("value");
  });
});

// The desktop store rewrites its whole file on every save, so the count of saves is
// the thing worth pinning: one per batch, never one per key.
describe("openSugaredKV (TauriKV)", () => {
  beforeEach(() => {
    mocks.engine = "tauri";
    mocks.stores.length = 0;
  });

  const openKV = () => Persist.openSugaredKV("session");
  const opened = () => {
    const store = mocks.stores[0];
    if (store == null) throw new Error("no store was opened");
    return store;
  };

  it("should name the file under the local app data directory", async () => {
    await openKV().set("key", "value");
    // A relative path would resolve against the roaming directory instead.
    expect(opened().path).toBe("/local/session.json");
    expect(opened().options).toEqual({ autoSave: false, defaults: {} });
  });

  it("should open the file only once across operations", async () => {
    const kv = openKV();
    await kv.set("key", "value");
    await kv.get("key");
    await kv.keys();
    expect(mocks.stores).toHaveLength(1);
  });

  it("should round-trip a value", async () => {
    const kv = openKV();
    await kv.set("key", { a: 1 });
    await expect(kv.get("key")).resolves.toEqual({ a: 1 });
    await expect(kv.length()).resolves.toBe(1);
    await expect(kv.keys()).resolves.toEqual(["key"]);
  });

  it("should save once for a batch of writes", async () => {
    await openKV().setMany([
      { key: "a", value: 1 },
      { key: "b", value: 2 },
      { key: "c", value: 3 },
    ]);
    expect(opened().entries.size).toBe(3);
    expect(opened().saves).toBe(1);
  });

  it("should save once for a batch of deletes", async () => {
    const kv = openKV();
    await kv.setMany([
      { key: "a", value: 1 },
      { key: "b", value: 2 },
    ]);
    await kv.deleteMany(["a", "b"]);
    expect(opened().entries.size).toBe(0);
    // One save for the writes and one for the deletes.
    expect(opened().saves).toBe(2);
  });

  it("should clear the file", async () => {
    const kv = openKV();
    await kv.set("key", "value");
    await kv.clear();
    await expect(kv.length()).resolves.toBe(0);
  });
});

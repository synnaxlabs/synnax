// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it } from "vitest";

import { Persist } from "@/session/persist";

// Runtime.ENGINE resolves to "web" under jsdom, so openSugaredKV returns a
// localStorage-backed store. The suite exercises that implementation end to end.
describe("openSugaredKV (LocalStorageKV)", () => {
  beforeEach(() => localStorage.clear());

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

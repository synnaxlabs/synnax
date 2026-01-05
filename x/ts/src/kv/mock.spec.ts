// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it } from "vitest";

import { kv } from "@/kv";

describe("MockAsync", () => {
  let db: kv.MockAsync;

  beforeEach(() => {
    db = new kv.MockAsync();
  });

  it("should set and get values", async () => {
    await db.set("key", "value");
    expect(await db.get("key")).toBe("value");
  });

  it("should return null for non-existent keys", async () => {
    expect(await db.get("non-existent")).toBeNull();
  });

  it("should delete values", async () => {
    await db.set("key", "value");
    await db.delete("key");
    expect(await db.get("key")).toBeNull();
  });

  it("should return correct length", async () => {
    expect(await db.length()).toBe(0);
    await db.set("key1", "value1");
    await db.set("key2", "value2");
    expect(await db.length()).toBe(2);
  });

  it("should clear all values", async () => {
    await db.set("key1", "value1");
    await db.set("key2", "value2");
    await db.clear();
    expect(await db.length()).toBe(0);
    expect(await db.get("key1")).toBeNull();
  });

  it("should handle different value types", async () => {
    const number = 42;
    const object = { test: "value" };
    const array = [1, 2, 3];

    await db.set("number", number);
    await db.set("object", object);
    await db.set("array", array);

    expect(await db.get("number")).toBe(number);
    expect(await db.get("object")).toEqual(object);
    expect(await db.get("array")).toEqual(array);
  });
});

describe("MockSync", () => {
  let db: kv.MockSync;

  beforeEach(() => {
    db = new kv.MockSync();
  });

  it("should set and get values", () => {
    db.set("key", "value");
    expect(db.get("key")).toBe("value");
  });

  it("should return null for non-existent keys", () => {
    expect(db.get("non-existent")).toBeNull();
  });

  it("should delete values", () => {
    db.set("key", "value");
    db.delete("key");
    expect(db.get("key")).toBeNull();
  });

  it("should handle different value types", () => {
    const number = 42;
    const object = { test: "value" };
    const array = [1, 2, 3];

    db.set("number", number);
    db.set("object", object);
    db.set("array", array);

    expect(db.get("number")).toBe(number);
    expect(db.get("object")).toEqual(object);
    expect(db.get("array")).toEqual(array);
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { type Form } from "@/form";
import { createLocalStore } from "@/form/store";

interface Values {
  name: string;
  profile: { bio: string };
  channels: { key: string; port: number }[];
  empty?: { key: string }[];
}

const createValues = (): Values => ({
  name: "a",
  profile: { bio: "b" },
  channels: [
    { key: "a", port: 1 },
    { key: "b", port: 2 },
  ],
});

const keys = (store: Form.Store<Values>): string[] =>
  store.get().channels.map(({ key }) => key);

describe("createLocalStore", () => {
  it("should hold a copy of the values", () => {
    const values = createValues();
    const store = createLocalStore(values);
    expect(store.get()).toEqual(values);
    expect(store.get()).not.toBe(values);
  });

  describe("set", () => {
    it("should replace the value at the path", () => {
      const store = createLocalStore(createValues());
      store.set("profile.bio", "c");
      expect(store.get().profile.bio).toBe("c");
    });

    it("should replace the values at the empty path", () => {
      const store = createLocalStore(createValues());
      const next = { ...createValues(), name: "z" };
      store.set("", next);
      expect(store.get()).toBe(next);
    });

    it("should give each ancestor a new identity and keep the rest", () => {
      const store = createLocalStore(createValues());
      const prev = store.get();
      store.set("channels.b.port", 3);
      const next = store.get();
      expect(next).not.toBe(prev);
      expect(next.channels).not.toBe(prev.channels);
      expect(next.channels[1]).not.toBe(prev.channels[1]);
      expect(next.channels[0]).toBe(prev.channels[0]);
      expect(next.profile).toBe(prev.profile);
      expect(prev.channels[1].port).toBe(2);
    });
  });

  describe("remove", () => {
    it("should remove an object field", () => {
      const store = createLocalStore(createValues());
      store.remove("profile.bio");
      expect(store.get().profile).toEqual({});
    });

    it("should remove a list entry by key", () => {
      const store = createLocalStore(createValues());
      store.remove("channels.b");
      expect(keys(store)).toEqual(["a"]);
    });
  });

  describe("insert", () => {
    it("should insert first when after is absent", () => {
      const store = createLocalStore(createValues());
      store.insert("channels", { key: "c", port: 3 });
      expect(keys(store)).toEqual(["c", "a", "b"]);
    });

    it("should insert after the named entry", () => {
      const store = createLocalStore(createValues());
      store.insert("channels", { key: "c", port: 3 }, "a");
      expect(keys(store)).toEqual(["a", "c", "b"]);
    });

    it("should insert last when after names no entry", () => {
      const store = createLocalStore(createValues());
      store.insert("channels", { key: "c", port: 3 }, "z");
      expect(keys(store)).toEqual(["a", "b", "c"]);
    });

    it("should do nothing when the key is in the list", () => {
      const store = createLocalStore(createValues());
      const prev = store.get();
      store.insert("channels", { key: "a", port: 9 });
      expect(store.get()).toBe(prev);
    });

    it("should create an absent list", () => {
      const store = createLocalStore(createValues());
      store.insert("empty", { key: "x" });
      expect(store.get().empty).toEqual([{ key: "x" }]);
    });
  });

  describe("move", () => {
    it("should move an entry first when after is absent", () => {
      const store = createLocalStore(createValues());
      store.move("channels", "b");
      expect(keys(store)).toEqual(["b", "a"]);
    });

    it("should move an entry after the named entry", () => {
      const store = createLocalStore(createValues());
      store.insert("channels", { key: "c", port: 3 }, "b");
      store.move("channels", "a", "b");
      expect(keys(store)).toEqual(["b", "a", "c"]);
    });

    it("should do nothing for a missing entry", () => {
      const store = createLocalStore(createValues());
      const prev = store.get();
      store.move("channels", "z");
      expect(store.get()).toBe(prev);
    });

    it("should do nothing when after names the entry itself", () => {
      const store = createLocalStore(createValues());
      const prev = store.get();
      store.move("channels", "a", "a");
      expect(store.get()).toBe(prev);
    });
  });

  describe("onChange", () => {
    it("should call the callback after each change", () => {
      const store = createLocalStore(createValues());
      const callback = vi.fn();
      store.onChange(callback);
      store.set("name", "b");
      store.remove("channels.a");
      expect(callback).toHaveBeenCalledTimes(2);
    });

    it("should stop calling the callback after the destructor runs", () => {
      const store = createLocalStore(createValues());
      const callback = vi.fn();
      const destroy = store.onChange(callback);
      destroy();
      store.set("name", "b");
      expect(callback).not.toHaveBeenCalled();
    });
  });

  describe("batch", () => {
    it("should call the callback once for all the writes in a batch", () => {
      const store = createLocalStore(createValues());
      const callback = vi.fn();
      store.onChange(callback);
      store.batch(() => {
        store.set("name", "b");
        store.batch(() => store.remove("channels.a"));
        expect(callback).not.toHaveBeenCalled();
      });
      expect(callback).toHaveBeenCalledTimes(1);
      expect(store.get().name).toBe("b");
      expect(keys(store)).toEqual(["b"]);
    });

    it("should not call the callback for a batch without writes", () => {
      const store = createLocalStore(createValues());
      const callback = vi.fn();
      store.onChange(callback);
      store.batch(() => {});
      expect(callback).not.toHaveBeenCalled();
    });

    it("should report the writes of a batch that throws", () => {
      const store = createLocalStore(createValues());
      const callback = vi.fn();
      store.onChange(callback);
      expect(() =>
        store.batch(() => {
          store.set("name", "b");
          throw new Error("failed");
        }),
      ).toThrow("failed");
      expect(callback).toHaveBeenCalledTimes(1);
    });
  });
});

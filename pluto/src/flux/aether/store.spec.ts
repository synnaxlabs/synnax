// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { createStore, UnaryStore } from "@/flux/aether/store";

describe("UnaryStore", () => {
  describe("set and get", () => {
    it("should set and get a value", () => {
      const store = new UnaryStore<string, string>();
      store.set("key1", "value1");
      expect(store.get("key1")).toBe("value1");
    });

    it("should update an existing value", () => {
      const store = new UnaryStore<string, string>();
      store.set("key1", "value1");
      store.set("key1", "value2");
      expect(store.get("key1")).toBe("value2");
    });

    it("should handle setter functions", () => {
      const store = new UnaryStore<string, string>();
      store.set("key1", "initial");
      store.set("key1", (prev) => `${prev}_updated`);
      expect(store.get("key1")).toBe("initial_updated");
    });

    it("should return undefined for non-existent keys", () => {
      const store = new UnaryStore<string, string>();
      expect(store.get("nonexistent")).toBeUndefined();
    });

    it("should get multiple values by keys array", () => {
      const store = new UnaryStore<string, string>();
      store.set("key1", "value1");
      store.set("key2", "value2");
      store.set("key3", "value3");

      const values = store.get(["key1", "key3", "nonexistent"]);
      expect(values).toEqual(["value1", "value3"]);
    });

    it("should filter values using a predicate", () => {
      const store = new UnaryStore<string, number>();
      store.set("a", 1);
      store.set("b", 2);
      store.set("c", 3);
      store.set("d", 4);

      const evenValues = store.get((value) => value % 2 === 0);
      expect(evenValues).toEqual([2, 4]);
    });

    it("should not set null values", () => {
      const store = new UnaryStore<string, string>();
      store.set("key1", "value1");
      store.set("key1", () => null as any);
      expect(store.get("key1")).toBe("value1");
    });
  });

  describe("delete", () => {
    it("should delete an entry", () => {
      const store = new UnaryStore<string, string>();
      store.set("key1", "value1");
      expect(store.get("key1")).toBe("value1");

      store.delete("key1");
      expect(store.get("key1")).toBeUndefined();
    });

    it("should handle deleting non-existent keys", () => {
      const store = new UnaryStore<string, string>();
      expect(() => store.delete("nonexistent")).not.toThrow();
    });
  });

  describe("onSet listeners", () => {
    it("should notify listeners when a value is set", () => {
      const store = new UnaryStore<string, string>();
      const listener = vi.fn();

      store.onSet(listener);
      store.set("key1", "value1");

      expect(listener).toHaveBeenCalledWith("value1");
    });

    it("should notify only for specific key when key filter is provided", () => {
      const store = new UnaryStore<string, string>();
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      store.onSet(listener1, "key1");
      store.onSet(listener2, "key2");

      store.set("key1", "value1");
      expect(listener1).toHaveBeenCalledWith("value1");
      expect(listener2).not.toHaveBeenCalled();

      store.set("key2", "value2");
      expect(listener2).toHaveBeenCalledWith("value2");
      expect(listener1).toHaveBeenCalledTimes(1);
    });

    it("should not notify when notify option is false", () => {
      const store = new UnaryStore<string, string>();
      const listener = vi.fn();

      store.onSet(listener);
      store.set("key1", "value1", { notify: false });

      expect(listener).not.toHaveBeenCalled();
    });

    it("should remove listener when destructor is called", () => {
      const store = new UnaryStore<string, string>();
      const listener = vi.fn();

      const destructor = store.onSet(listener);
      store.set("key1", "value1");
      expect(listener).toHaveBeenCalledTimes(1);

      destructor();
      store.set("key2", "value2");
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should handle multiple listeners", () => {
      const store = new UnaryStore<string, string>();
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      store.onSet(listener1);
      store.onSet(listener2);

      store.set("key1", "value1");

      expect(listener1).toHaveBeenCalledWith("value1");
      expect(listener2).toHaveBeenCalledWith("value1");
    });
  });

  describe("onDelete listeners", () => {
    it("should notify listeners when a value is deleted", () => {
      const store = new UnaryStore<string, string>();
      const listener = vi.fn();

      store.onDelete(listener);
      store.set("key1", "value1");
      store.delete("key1");

      expect(listener).toHaveBeenCalledWith("key1");
    });

    it("should notify only for specific key when key filter is provided", () => {
      const store = new UnaryStore<string, string>();
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      store.onDelete(listener1, "key1");
      store.onDelete(listener2, "key2");

      store.set("key1", "value1");
      store.set("key2", "value2");

      store.delete("key1");
      expect(listener1).toHaveBeenCalledWith("key1");
      expect(listener2).not.toHaveBeenCalled();

      store.delete("key2");
      expect(listener2).toHaveBeenCalledWith("key2");
      expect(listener1).toHaveBeenCalledTimes(1);
    });

    it("should remove listener when destructor is called", () => {
      const store = new UnaryStore<string, string>();
      const listener = vi.fn();

      const destructor = store.onDelete(listener);
      store.set("key1", "value1");
      store.delete("key1");
      expect(listener).toHaveBeenCalledTimes(1);

      destructor();
      store.set("key2", "value2");
      store.delete("key2");
      expect(listener).toHaveBeenCalledTimes(1);
    });
  });

  describe("complex state types", () => {
    interface User {
      id: string;
      name: string;
      age: number;
    }

    it("should handle object state", () => {
      const store = new UnaryStore<string, User>();
      const user: User = { id: "1", name: "John", age: 30 };

      store.set("user1", user);
      expect(store.get("user1")).toEqual(user);
    });

    it("should update nested properties with setter function", () => {
      const store = new UnaryStore<string, User>();
      const user: User = { id: "1", name: "John", age: 30 };

      store.set("user1", user);
      store.set("user1", (prev) => prev && { ...prev, age: 31 });

      const updated = store.get("user1");
      expect(updated?.age).toBe(31);
      expect(updated?.name).toBe("John");
    });
  });
});

describe("createStore", () => {
  it("should create a store with configured keys", () => {
    const config = {
      users: { listeners: [] },
      tasks: { listeners: [] },
      settings: { listeners: [] },
    };

    const store = createStore(config);

    expect(store.users).toBeInstanceOf(UnaryStore);
    expect(store.tasks).toBeInstanceOf(UnaryStore);
    expect(store.settings).toBeInstanceOf(UnaryStore);
  });

  it("should create independent stores for each key", () => {
    const config = {
      store1: { listeners: [] },
      store2: { listeners: [] },
    };

    const store = createStore(config);

    store.store1.set("key1", "value1");
    store.store2.set("key1", "different");

    expect(store.store1.get("key1")).toBe("value1");
    expect(store.store2.get("key1")).toBe("different");
  });

  it("should handle empty configuration", () => {
    const store = createStore({});
    expect(store).toEqual({});
  });
});

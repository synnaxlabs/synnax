// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import { createStore, ScopedUnaryStore, scopeStore } from "@/flux/aether/store";

const basicHandleError = vi.fn((excOrFunc: any, _?: string) => {
  if (typeof excOrFunc === "function") void excOrFunc();
});
const squashError = vi.fn((excOrFunc: any, _?: string) => {
  if (typeof excOrFunc === "function")
    void (async () => {
      try {
        await excOrFunc();
      } catch (_) {
        // Error caught
      }
    })();
});

describe("ScopedUnaryStore", () => {
  beforeEach(() => {
    squashError.mockClear();
  });

  describe("set and get", () => {
    it("should set and get a value", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      store.set("key1", "value1");
      expect(store.get("key1")).toBe("value1");
    });

    it("should update an existing value", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      store.set("key1", "value1");
      store.set("key1", "value2");
      expect(store.get("key1")).toBe("value2");
    });

    it("should handle setter functions", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      store.set("key1", "initial");
      store.set("key1", (prev) => `${prev}_updated`);
      expect(store.get("key1")).toBe("initial_updated");
    });

    it("should return undefined for non-existent keys", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      expect(store.get("nonexistent")).toBeUndefined();
    });

    it("should get multiple values by keys array", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      store.set("key1", "value1");
      store.set("key2", "value2");
      store.set("key3", "value3");

      const values = store.get(["key1", "key3", "nonexistent"]);
      expect(values).toEqual(["value1", "value3"]);
    });

    it("should filter values using a predicate", () => {
      const store = new ScopedUnaryStore<string, number>(basicHandleError).scope(
        "scope",
      );
      store.set("a", 1);
      store.set("b", 2);
      store.set("c", 3);
      store.set("d", 4);

      const evenValues = store.get((value) => value % 2 === 0);
      expect(evenValues).toEqual([2, 4]);
    });

    it("should not set null values", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      store.set("key1", "value1");
      store.set("key1", () => null as any);
      expect(store.get("key1")).toBe("value1");
    });
  });

  describe("delete", () => {
    it("should delete an entry", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      store.set("key1", "value1");
      expect(store.get("key1")).toBe("value1");

      store.delete("key1");
      expect(store.get("key1")).toBeUndefined();
    });

    it("should handle deleting non-existent keys", () => {
      const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
        "scope",
      );
      expect(() => store.delete("nonexistent")).not.toThrow();
    });
  });

  describe("onSet listeners", () => {
    it("should notify listeners from different scopes when a value is set", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener = vi.fn();

      scope2.onSet(listener);
      scope1.set("key1", "value1");

      expect(listener).toHaveBeenCalledWith("value1");
    });

    it("should not notify listeners from the same scope", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const listener = vi.fn();

      scope1.onSet(listener);
      scope1.set("key1", "value1");

      expect(listener).not.toHaveBeenCalled();
    });

    it("should notify only for specific key when key filter is provided", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      scope2.onSet(listener1, "key1");
      scope2.onSet(listener2, "key2");

      scope1.set("key1", "value1");
      expect(listener1).toHaveBeenCalledWith("value1");
      expect(listener2).not.toHaveBeenCalled();

      scope1.set("key2", "value2");
      expect(listener2).toHaveBeenCalledWith("value2");
      expect(listener1).toHaveBeenCalledTimes(1);
    });

    it("should remove listener when destructor is called", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener = vi.fn();

      const destructor = scope2.onSet(listener);
      scope1.set("key1", "value1");
      expect(listener).toHaveBeenCalledTimes(1);

      destructor();
      scope1.set("key2", "value2");
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should handle multiple listeners from different scopes", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const scope3 = baseStore.scope("scope3");
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      scope2.onSet(listener1);
      scope3.onSet(listener2);

      scope1.set("key1", "value1");

      expect(listener1).toHaveBeenCalledWith("value1");
      expect(listener2).toHaveBeenCalledWith("value1");
    });

    it("should handle mixed scope listeners correctly", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn();
      const listener2 = vi.fn();
      const listener3 = vi.fn();

      scope1.onSet(listener1);
      scope2.onSet(listener2);
      scope2.onSet(listener3);

      scope1.set("key1", "value1");
      expect(listener1).not.toHaveBeenCalled();
      expect(listener2).toHaveBeenCalledWith("value1");
      expect(listener3).toHaveBeenCalledWith("value1");

      scope2.set("key2", "value2");
      expect(listener1).toHaveBeenCalledWith("value2");
      expect(listener2).not.toHaveBeenCalledWith("value2");
      expect(listener3).not.toHaveBeenCalledWith("value2");
    });

    it("should call error handler when a listener throws a synchronous error", async () => {
      const errorHandler = vi.fn();
      const baseStore = new ScopedUnaryStore<string, string>(errorHandler);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const error = new Error("Listener error");
      const listener = vi.fn(() => {
        throw error;
      });

      scope2.onSet(listener);
      scope1.set("key1", "value1");

      await vi.waitFor(() => {
        expect(errorHandler).toHaveBeenCalled();
      });

      const errorCall = errorHandler.mock.calls[0];
      expect(errorCall[1]).toBe("Failed to notify set listener");
    });

    it("should call error handler when a listener returns a rejected promise", async () => {
      const errorHandler = vi.fn();
      const baseStore = new ScopedUnaryStore<string, string>(errorHandler);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const error = new Error("Async listener error");
      const listener = vi.fn(async () => {
        throw error;
      });

      scope2.onSet(listener);
      scope1.set("key1", "value1");

      await vi.waitFor(() => {
        expect(errorHandler).toHaveBeenCalled();
      });

      const errorCall = errorHandler.mock.calls[0];
      expect(errorCall[1]).toBe("Failed to notify set listener");
    });

    it("should continue notifying other listeners when one throws an error", async () => {
      const baseStore = new ScopedUnaryStore<string, string>(squashError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn(() => {
        throw new Error("First listener error");
      });
      const listener2 = vi.fn();
      const listener3 = vi.fn();

      scope2.onSet(listener1);
      scope2.onSet(listener2);
      scope2.onSet(listener3);

      scope1.set("key1", "value1");

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(listener1).toHaveBeenCalledWith("value1");
      expect(listener2).toHaveBeenCalledWith("value1");
      expect(listener3).toHaveBeenCalledWith("value1");
      expect(squashError).toHaveBeenCalledTimes(3);
    });

    it("should handle errors from multiple listeners", async () => {
      const baseStore = new ScopedUnaryStore<string, string>(squashError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn(() => {
        throw new Error("First error");
      });
      const listener2 = vi.fn(async () => {
        throw new Error("Second error");
      });
      const listener3 = vi.fn();

      scope2.onSet(listener1);
      scope2.onSet(listener2);
      scope2.onSet(listener3);

      scope1.set("key1", "value1");

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(listener3).toHaveBeenCalledWith("value1");
      expect(squashError).toHaveBeenCalledTimes(3);
    });
  });

  describe("onDelete listeners", () => {
    it("should notify listeners from different scopes when a value is deleted", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener = vi.fn();

      scope2.onDelete(listener);
      scope1.set("key1", "value1");
      scope1.delete("key1");

      expect(listener).toHaveBeenCalledWith("key1");
    });

    it("should not notify listeners from the same scope when deleted", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const listener = vi.fn();

      scope1.onDelete(listener);
      scope1.set("key1", "value1");
      scope1.delete("key1");

      expect(listener).not.toHaveBeenCalled();
    });

    it("should notify only for specific key when key filter is provided", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn();
      const listener2 = vi.fn();

      scope2.onDelete(listener1, "key1");
      scope2.onDelete(listener2, "key2");

      scope1.set("key1", "value1");
      scope1.set("key2", "value2");

      scope1.delete("key1");
      expect(listener1).toHaveBeenCalledWith("key1");
      expect(listener2).not.toHaveBeenCalled();

      scope1.delete("key2");
      expect(listener2).toHaveBeenCalledWith("key2");
      expect(listener1).toHaveBeenCalledTimes(1);
    });

    it("should remove listener when destructor is called", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener = vi.fn();

      const destructor = scope2.onDelete(listener);
      scope1.set("key1", "value1");
      scope1.delete("key1");
      expect(listener).toHaveBeenCalledTimes(1);

      destructor();
      scope1.set("key2", "value2");
      scope1.delete("key2");
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should call error handler when a listener throws a synchronous error", async () => {
      const errorHandler = vi.fn();
      const baseStore = new ScopedUnaryStore<string, string>(errorHandler);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const error = new Error("Delete listener error");
      const listener = vi.fn(() => {
        throw error;
      });

      scope2.onDelete(listener);
      scope1.set("key1", "value1");
      scope1.delete("key1");

      await vi.waitFor(() => {
        expect(errorHandler).toHaveBeenCalled();
      });

      const errorCall = errorHandler.mock.calls[0];
      expect(errorCall[1]).toBe("Failed to notify delete listener");
    });

    it("should call error handler when a listener returns a rejected promise", async () => {
      const errorHandler = vi.fn();
      const baseStore = new ScopedUnaryStore<string, string>(errorHandler);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const error = new Error("Async delete listener error");
      const listener = vi.fn(async () => {
        throw error;
      });

      scope2.onDelete(listener);
      scope1.set("key1", "value1");
      scope1.delete("key1");

      await vi.waitFor(() => {
        expect(errorHandler).toHaveBeenCalled();
      });

      const errorCall = errorHandler.mock.calls[0];
      expect(errorCall[1]).toBe("Failed to notify delete listener");
    });

    it("should continue notifying other listeners when one throws an error", async () => {
      const baseStore = new ScopedUnaryStore<string, string>(squashError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn(() => {
        throw new Error("First delete listener error");
      });
      const listener2 = vi.fn();
      const listener3 = vi.fn();

      scope2.onDelete(listener1);
      scope2.onDelete(listener2);
      scope2.onDelete(listener3);

      scope1.set("key1", "value1");
      scope1.delete("key1");

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(listener1).toHaveBeenCalledWith("key1");
      expect(listener2).toHaveBeenCalledWith("key1");
      expect(listener3).toHaveBeenCalledWith("key1");
      expect(squashError).toHaveBeenCalledTimes(3);
    });

    it("should handle errors from multiple delete listeners", async () => {
      const baseStore = new ScopedUnaryStore<string, string>(squashError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn(() => {
        throw new Error("First delete error");
      });
      const listener2 = vi.fn(async () => {
        throw new Error("Second delete error");
      });
      const listener3 = vi.fn();

      scope2.onDelete(listener1);
      scope2.onDelete(listener2);
      scope2.onDelete(listener3);

      scope1.set("key1", "value1");
      scope1.delete("key1");

      await new Promise((resolve) => setTimeout(resolve, 10));

      expect(listener3).toHaveBeenCalledWith("key1");
      expect(squashError).toHaveBeenCalledTimes(3);
    });

    it("should handle mixed scope delete listeners correctly", () => {
      const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
      const scope1 = baseStore.scope("scope1");
      const scope2 = baseStore.scope("scope2");
      const listener1 = vi.fn();
      const listener2 = vi.fn();
      const listener3 = vi.fn();

      scope1.onDelete(listener1);
      scope2.onDelete(listener2);
      scope2.onDelete(listener3);

      scope1.set("key1", "value1");
      scope1.delete("key1");
      expect(listener1).not.toHaveBeenCalled();
      expect(listener2).toHaveBeenCalledWith("key1");
      expect(listener3).toHaveBeenCalledWith("key1");

      scope2.set("key2", "value2");
      scope2.delete("key2");
      expect(listener1).toHaveBeenCalledWith("key2");
      expect(listener2).not.toHaveBeenCalledWith("key2");
      expect(listener3).not.toHaveBeenCalledWith("key2");
    });
  });

  describe("complex state types", () => {
    interface User {
      id: string;
      name: string;
      age: number;
    }

    it("should handle object state", () => {
      const store = new ScopedUnaryStore<string, User>(basicHandleError).scope("scope");
      const user: User = { id: "1", name: "John", age: 30 };

      store.set("user1", user);
      expect(store.get("user1")).toEqual(user);
    });

    it("should update nested properties with setter function", () => {
      const store = new ScopedUnaryStore<string, User>(basicHandleError).scope("scope");
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
  it("should create independent stores for each key", () => {
    const config = {
      store1: { listeners: [] },
      store2: { listeners: [] },
    };

    const store = scopeStore(createStore(config, basicHandleError), "scope");

    store.store1.set("key1", "value1");
    store.store2.set("key1", "different");

    expect(store.store1.get("key1")).toBe("value1");
    expect(store.store2.get("key1")).toBe("different");
  });

  it("should handle empty configuration", () => {
    const store = createStore({}, basicHandleError);
    expect(store).toEqual({});
  });
});

describe("Scoping exclusion behavior", () => {
  it("should allow multiple scopes from the same base store", () => {
    const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
    const scope1 = baseStore.scope("scope1");
    const scope2 = baseStore.scope("scope2");
    const scope3 = baseStore.scope("scope3");

    scope1.set("key1", "value1");
    scope2.set("key2", "value2");
    scope3.set("key3", "value3");

    expect(scope1.get("key1")).toBe("value1");
    expect(scope1.get("key2")).toBe("value2");
    expect(scope1.get("key3")).toBe("value3");
  });

  it("should properly handle cross-scope listener exclusion for sets", () => {
    const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
    const scope1 = baseStore.scope("scope1");
    const scope2 = baseStore.scope("scope2");
    const scope3 = baseStore.scope("scope3");

    const listener1 = vi.fn();
    const listener2 = vi.fn();
    const listener3 = vi.fn();

    scope1.onSet(listener1);
    scope2.onSet(listener2);
    scope3.onSet(listener3);

    scope1.set("key1", "value1");
    expect(listener1).not.toHaveBeenCalled();
    expect(listener2).toHaveBeenCalledWith("value1");
    expect(listener3).toHaveBeenCalledWith("value1");

    scope2.set("key2", "value2");
    expect(listener1).toHaveBeenCalledWith("value2");
    expect(listener2).not.toHaveBeenCalledWith("value2");
    expect(listener3).toHaveBeenCalledWith("value2");

    scope3.set("key3", "value3");
    expect(listener1).toHaveBeenCalledWith("value3");
    expect(listener2).toHaveBeenCalledWith("value3");
    expect(listener3).not.toHaveBeenCalledWith("value3");
  });

  it("should properly handle cross-scope listener exclusion for deletes", () => {
    const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
    const scope1 = baseStore.scope("scope1");
    const scope2 = baseStore.scope("scope2");
    const scope3 = baseStore.scope("scope3");

    const listener1 = vi.fn();
    const listener2 = vi.fn();
    const listener3 = vi.fn();

    scope1.onDelete(listener1);
    scope2.onDelete(listener2);
    scope3.onDelete(listener3);

    scope1.set("key1", "value1");
    scope1.delete("key1");
    expect(listener1).not.toHaveBeenCalled();
    expect(listener2).toHaveBeenCalledWith("key1");
    expect(listener3).toHaveBeenCalledWith("key1");

    scope2.set("key2", "value2");
    scope2.delete("key2");
    expect(listener1).toHaveBeenCalledWith("key2");
    expect(listener2).not.toHaveBeenCalledWith("key2");
    expect(listener3).toHaveBeenCalledWith("key2");

    scope3.set("key3", "value3");
    scope3.delete("key3");
    expect(listener1).toHaveBeenCalledWith("key3");
    expect(listener2).toHaveBeenCalledWith("key3");
    expect(listener3).not.toHaveBeenCalledWith("key3");
  });

  it("should handle key-specific filtering with scoping exclusion", () => {
    const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
    const scope1 = baseStore.scope("scope1");
    const scope2 = baseStore.scope("scope2");

    const listenerAll = vi.fn();
    const listenerKey1 = vi.fn();
    const listenerKey2 = vi.fn();

    scope2.onSet(listenerAll);
    scope2.onSet(listenerKey1, "key1");
    scope2.onSet(listenerKey2, "key2");

    scope1.set("key1", "value1");
    expect(listenerAll).toHaveBeenCalledWith("value1");
    expect(listenerKey1).toHaveBeenCalledWith("value1");
    expect(listenerKey2).not.toHaveBeenCalled();

    scope1.set("key2", "value2");
    expect(listenerAll).toHaveBeenCalledWith("value2");
    expect(listenerKey2).toHaveBeenCalledWith("value2");
    expect(listenerKey1).toHaveBeenCalledTimes(1);
  });

  it("should handle listener cleanup correctly with scoping", () => {
    const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
    const scope1 = baseStore.scope("scope1");
    const scope2 = baseStore.scope("scope2");

    const listener1 = vi.fn();
    const listener2 = vi.fn();

    const cleanup1 = scope2.onSet(listener1);
    const cleanup2 = scope2.onSet(listener2);

    scope1.set("key1", "value1");
    expect(listener1).toHaveBeenCalledWith("value1");
    expect(listener2).toHaveBeenCalledWith("value1");

    cleanup1();

    scope1.set("key2", "value2");
    expect(listener1).toHaveBeenCalledTimes(1);
    expect(listener2).toHaveBeenCalledWith("value2");

    cleanup2();

    scope1.set("key3", "value3");
    expect(listener1).toHaveBeenCalledTimes(1);
    expect(listener2).toHaveBeenCalledTimes(2);
  });
});

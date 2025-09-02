// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type core } from "@/flux/core";
import { createStore, ScopedUnaryStore, scopeStore } from "@/flux/core/store";

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

describe("Core Store", () => {
  beforeEach(() => {
    squashError.mockClear();
  });

  describe("ScopedUnaryStore", () => {
    describe("Basic Operations", () => {
      describe("Set and Get", () => {
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

      describe("Delete", () => {
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

      describe("List", () => {
        it("should return empty array when store is empty", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          expect(store.list()).toEqual([]);
        });

        it("should return all values in the store", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          store.set("key1", "value1");
          store.set("key2", "value2");
          store.set("key3", "value3");

          const values = store.list();
          expect(values).toHaveLength(3);
          expect(values).toContain("value1");
          expect(values).toContain("value2");
          expect(values).toContain("value3");
        });

        it("should return values after deletions", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          store.set("key1", "value1");
          store.set("key2", "value2");
          store.set("key3", "value3");
          store.delete("key2");

          const values = store.list();
          expect(values).toHaveLength(2);
          expect(values).toContain("value1");
          expect(values).toContain("value3");
          expect(values).not.toContain("value2");
        });

        it("should return values after updates", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          store.set("key1", "value1");
          store.set("key2", "value2");
          store.set("key1", "updated1");

          const values = store.list();
          expect(values).toHaveLength(2);
          expect(values).toContain("updated1");
          expect(values).toContain("value2");
          expect(values).not.toContain("value1");
        });

        it("should work with complex object types", () => {
          interface User {
            id: string;
            name: string;
            age: number;
          }

          const store = new ScopedUnaryStore<string, User>(basicHandleError).scope(
            "scope",
          );
          const user1: User = { id: "1", name: "John", age: 30 };
          const user2: User = { id: "2", name: "Jane", age: 25 };
          const user3: User = { id: "3", name: "Bob", age: 35 };

          store.set("user1", user1);
          store.set("user2", user2);
          store.set("user3", user3);

          const users = store.list();
          expect(users).toHaveLength(3);
          expect(users).toContainEqual(user1);
          expect(users).toContainEqual(user2);
          expect(users).toContainEqual(user3);
        });

        it("should return values across different scopes", () => {
          const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
          const scope1 = baseStore.scope("scope1");
          const scope2 = baseStore.scope("scope2");
          const scope3 = baseStore.scope("scope3");

          scope1.set("key1", "value1");
          scope2.set("key2", "value2");
          scope3.set("key3", "value3");

          const values1 = scope1.list();
          const values2 = scope2.list();
          const values3 = scope3.list();

          expect(values1).toHaveLength(3);
          expect(values2).toHaveLength(3);
          expect(values3).toHaveLength(3);

          expect(values1).toContain("value1");
          expect(values1).toContain("value2");
          expect(values1).toContain("value3");
        });

        it("should return values after bulk set operations", () => {
          const store = new ScopedUnaryStore<string, { key: string; value: string }>(
            basicHandleError,
          ).scope("scope");

          const items = [
            { key: "key1", value: "value1" },
            { key: "key2", value: "value2" },
            { key: "key3", value: "value3" },
          ];

          store.set(items);

          const values = store.list();
          expect(values).toHaveLength(3);
          expect(values).toContainEqual({ key: "key1", value: "value1" });
          expect(values).toContainEqual({ key: "key2", value: "value2" });
          expect(values).toContainEqual({ key: "key3", value: "value3" });
        });

        it("should return values after bulk delete operations", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          store.set("key1", "value1");
          store.set("key2", "value2");
          store.set("key3", "value3");
          store.set("key4", "value4");

          store.delete(["key2", "key3"]);

          const values = store.list();
          expect(values).toHaveLength(2);
          expect(values).toContain("value1");
          expect(values).toContain("value4");
        });

        it("should return empty array after clear", () => {
          const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
          const scope = baseStore.scope("scope");

          scope.set("key1", "value1");
          scope.set("key2", "value2");
          scope.set("key3", "value3");

          expect(scope.list()).toHaveLength(3);

          baseStore.clear();

          expect(scope.list()).toEqual([]);
        });

        it("should work with number keys", () => {
          const store = new ScopedUnaryStore<number, string>(basicHandleError).scope(
            "scope",
          );
          store.set(1, "value1");
          store.set(2, "value2");
          store.set(3, "value3");

          const values = store.list();
          expect(values).toHaveLength(3);
          expect(values).toContain("value1");
          expect(values).toContain("value2");
          expect(values).toContain("value3");
        });

        it("should handle mixed operations correctly", () => {
          const store = new ScopedUnaryStore<string, number>(basicHandleError).scope(
            "scope",
          );

          store.set("a", 1);
          store.set("b", 2);
          store.set("c", 3);
          store.delete("b");
          store.set("d", 4);
          store.set("a", 10);

          const values = store.list();
          expect(values).toHaveLength(3);
          expect(values).toContain(10);
          expect(values).toContain(3);
          expect(values).toContain(4);
          expect(values).not.toContain(1);
          expect(values).not.toContain(2);
        });

        it("should return independent arrays on each call", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          store.set("key1", "value1");
          store.set("key2", "value2");

          const list1 = store.list();
          const list2 = store.list();

          expect(list1).not.toBe(list2);
          expect(list1).toEqual(list2);

          list1.push("extra");
          expect(list1).toHaveLength(3);
          expect(list2).toHaveLength(2);
          expect(store.list()).toHaveLength(2);
        });

        it("should preserve values with equal function check", () => {
          const equalFunc = (a: string, b: string) =>
            a.toLowerCase() === b.toLowerCase();
          const store = new ScopedUnaryStore<string, string>(
            basicHandleError,
            equalFunc,
          ).scope("scope");

          store.set("key1", "Value1");
          store.set("key2", "Value2");
          store.set("key1", "VALUE1");

          const values = store.list();
          expect(values).toHaveLength(2);
          expect(values).toContain("Value1");
          expect(values).toContain("Value2");
        });
      });
    });

    describe("Event Listeners", () => {
      describe("OnSet Listeners", () => {
        it("should notify listeners from different scopes when a value is set", () => {
          const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
          const scope1 = baseStore.scope("scope1");
          const scope2 = baseStore.scope("scope2");
          const listener = vi.fn();

          scope2.onSet(listener);
          scope1.set("key1", "value1");

          expect(listener).toHaveBeenCalledWith("value1", undefined);
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
          expect(listener1).toHaveBeenCalledWith("value1", undefined);
          expect(listener2).not.toHaveBeenCalled();

          scope1.set("key2", "value2");
          expect(listener2).toHaveBeenCalledWith("value2", undefined);
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

          expect(listener1).toHaveBeenCalledWith("value1", undefined);
          expect(listener2).toHaveBeenCalledWith("value1", undefined);
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
          expect(listener2).toHaveBeenCalledWith("value1", undefined);
          expect(listener3).toHaveBeenCalledWith("value1", undefined);

          scope2.set("key2", "value2");
          expect(listener1).toHaveBeenCalledWith("value2", undefined);
          expect(listener2).not.toHaveBeenCalledWith("value2", undefined);
          expect(listener3).not.toHaveBeenCalledWith("value2", undefined);
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

          expect(listener1).toHaveBeenCalledWith("value1", undefined);
          expect(listener2).toHaveBeenCalledWith("value1", undefined);
          expect(listener3).toHaveBeenCalledWith("value1", undefined);
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

          expect(listener3).toHaveBeenCalledWith("value1", undefined);
          expect(squashError).toHaveBeenCalledTimes(3);
        });
      });

      describe("OnDelete Listeners", () => {
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
    });

    describe("Advanced Features", () => {
      describe("Complex State Types", () => {
        interface User {
          id: string;
          name: string;
          age: number;
        }

        it("should handle object state", () => {
          const store = new ScopedUnaryStore<string, User>(basicHandleError).scope(
            "scope",
          );
          const user: User = { id: "1", name: "John", age: 30 };

          store.set("user1", user);
          expect(store.get("user1")).toEqual(user);
        });

        it("should update nested properties with setter function", () => {
          const store = new ScopedUnaryStore<string, User>(basicHandleError).scope(
            "scope",
          );
          const user: User = { id: "1", name: "John", age: 30 };

          store.set("user1", user);
          store.set("user1", (prev) => prev && { ...prev, age: 31 });

          const updated = store.get("user1");
          expect(updated?.age).toBe(31);
          expect(updated?.name).toBe("John");
        });
      });

      describe("Extra Arguments", () => {
        it("should allow the caller to pass extra arguments to listeners", () => {
          const base = new ScopedUnaryStore<string, string, "cat">(basicHandleError);
          const scoped1 = base.scope("scope1");
          const scoped2 = base.scope("scope2");
          const listener = vi.fn();
          scoped1.onSet(listener, "key1");
          scoped2.set("key1", "value1", "cat");
          expect(listener).toHaveBeenCalledWith("value1", "cat");
        });

        it("should allow the caller to pass extra arguments on array sets", () => {
          const base = new ScopedUnaryStore<string, record.Keyed<string>, "cat">(
            basicHandleError,
          );
          const scoped1 = base.scope("scope1");
          const scoped2 = base.scope("scope2");
          const listener = vi.fn();
          scoped1.onSet(listener, "key1");
          scoped2.set([{ key: "key1" }, { key: "key2" }], "cat");
          expect(listener).toHaveBeenCalledWith({ key: "key1" }, "cat");
        });
      });

      describe("Scoping and Cross-Scope Behavior", () => {
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
          expect(listener2).toHaveBeenCalledWith("value1", undefined);
          expect(listener3).toHaveBeenCalledWith("value1", undefined);

          scope2.set("key2", "value2");
          expect(listener1).toHaveBeenCalledWith("value2", undefined);
          expect(listener2).not.toHaveBeenCalledWith("value2", undefined);
          expect(listener3).toHaveBeenCalledWith("value2", undefined);

          scope3.set("key3", "value3");
          expect(listener1).toHaveBeenCalledWith("value3", undefined);
          expect(listener2).toHaveBeenCalledWith("value3", undefined);
          expect(listener3).not.toHaveBeenCalledWith("value3", undefined);
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
          expect(listenerAll).toHaveBeenCalledWith("value1", undefined);
          expect(listenerKey1).toHaveBeenCalledWith("value1", undefined);
          expect(listenerKey2).not.toHaveBeenCalled();

          scope1.set("key2", "value2");
          expect(listenerAll).toHaveBeenCalledWith("value2", undefined);
          expect(listenerKey2).toHaveBeenCalledWith("value2", undefined);
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
          expect(listener1).toHaveBeenCalledWith("value1", undefined);
          expect(listener2).toHaveBeenCalledWith("value1", undefined);

          cleanup1();

          scope1.set("key2", "value2");
          expect(listener1).toHaveBeenCalledTimes(1);
          expect(listener2).toHaveBeenCalledWith("value2", undefined);

          cleanup2();

          scope1.set("key3", "value3");
          expect(listener1).toHaveBeenCalledTimes(1);
          expect(listener2).toHaveBeenCalledTimes(2);
        });
      });
    });
  });

  describe("Store Factory Functions", () => {
    describe("createStore", () => {
      it("should create independent stores for each key", () => {
        const config: core.StoreConfig<{
          store1: core.UnaryStore<record.Key, record.Keyed<record.Key>, undefined>;
          store2: core.UnaryStore<record.Key, record.Keyed<record.Key>, undefined>;
        }> = {
          store1: { listeners: [] },
          store2: { listeners: [] },
        };

        const store = scopeStore(createStore(config, basicHandleError), "scope");

        store.store1.set("key1", "value1", undefined);
        store.store2.set("key1", "different", undefined);

        expect(store.store1.get("key1")).toBe("value1");
        expect(store.store2.get("key1")).toBe("different");
      });

      it("should handle empty configuration", () => {
        const store = createStore({}, basicHandleError);
        expect(store).toEqual({});
      });
    });
  });
});

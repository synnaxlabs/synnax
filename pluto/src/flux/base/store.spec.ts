// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { type base } from "@/flux/base";
import { createStore, ScopedUnaryStore, scopeStore } from "@/flux/base/store";

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

describe("Base Store", () => {
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

        it("should set a single value with key property", () => {
          interface KeyedValue extends record.Keyed<string> {
            key: string;
            value: string;
          }
          const store = new ScopedUnaryStore<string, KeyedValue>(
            basicHandleError,
          ).scope("scope");

          const item: KeyedValue = { key: "key1", value: "value1" };
          store.set(item);

          expect(store.get("key1")).toEqual({ key: "key1", value: "value1" });
        });

        it("should set multiple values with key property using array", () => {
          interface KeyedValue extends record.Keyed<string> {
            key: string;
            value: string;
          }
          const store = new ScopedUnaryStore<string, KeyedValue>(
            basicHandleError,
          ).scope("scope");

          const items: KeyedValue[] = [
            { key: "key1", value: "value1" },
            { key: "key2", value: "value2" },
            { key: "key3", value: "value3" },
          ];
          store.set(items);

          expect(store.get("key1")).toEqual({ key: "key1", value: "value1" });
          expect(store.get("key2")).toEqual({ key: "key2", value: "value2" });
          expect(store.get("key3")).toEqual({ key: "key3", value: "value3" });
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

        it("should distinguish between key and keyed object types", () => {
          // This test validates that when we pass different numbers of arguments,
          // the store correctly interprets them
          interface ComplexValue extends record.Keyed<string> {
            key: string;
            data: number;
          }

          const store = new ScopedUnaryStore<string, ComplexValue>(
            basicHandleError,
          ).scope("scope");

          // Case 1: Setting with explicit key and value (2 arguments)
          // The first arg is the key, second is the value
          store.set("explicitKey", { key: "valueKey", data: 100 });
          expect(store.get("explicitKey")).toEqual({ key: "valueKey", data: 100 });

          // Case 2: Setting with single keyed object (1 argument)
          // Should use the object's key property as the key
          const singleObj: ComplexValue = { key: "derivedKey", data: 200 };
          store.set(singleObj);
          expect(store.get("derivedKey")).toEqual({ key: "derivedKey", data: 200 });

          // Case 3: Setting with array of keyed objects
          // Each object's key property is used as its key
          store.set([
            { key: "arrayKey1", data: 300 },
            { key: "arrayKey2", data: 400 },
          ]);
          expect(store.get("arrayKey1")).toEqual({ key: "arrayKey1", data: 300 });
          expect(store.get("arrayKey2")).toEqual({ key: "arrayKey2", data: 400 });
        });

        it("should handle mixed set operations", () => {
          interface KeyedData extends record.Keyed<string> {
            key: string;
            value: number;
          }

          const store = new ScopedUnaryStore<string, KeyedData>(basicHandleError).scope(
            "scope",
          );

          // Set using key-value pair
          store.set("key1", { key: "key1", value: 100 });

          // Set using single keyed object
          store.set({ key: "key2", value: 200 });

          // Set using array of keyed objects
          store.set([
            { key: "key3", value: 300 },
            { key: "key4", value: 400 },
          ]);

          expect(store.get("key1")).toEqual({ key: "key1", value: 100 });
          expect(store.get("key2")).toEqual({ key: "key2", value: 200 });
          expect(store.get("key3")).toEqual({ key: "key3", value: 300 });
          expect(store.get("key4")).toEqual({ key: "key4", value: 400 });
        });
      });

      describe("Rollback Functionality", () => {
        describe("Set Rollback", () => {
          it("should rollback a set operation for new entry", () => {
            const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
              "scope",
            );
            const rollback = store.set("key1", "value1");
            expect(store.get("key1")).toBe("value1");

            rollback();
            expect(store.get("key1")).toBeUndefined();
          });

          it("should rollback a set operation for existing entry", () => {
            const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
              "scope",
            );
            store.set("key1", "initial");
            const rollback = store.set("key1", "updated");
            expect(store.get("key1")).toBe("updated");

            rollback();
            expect(store.get("key1")).toBe("initial");
          });

          it("should rollback multiple set operations", () => {
            interface KeyedString {
              key: string;
              value: string;
            }
            const store = new ScopedUnaryStore<string, KeyedString>(
              basicHandleError,
            ).scope("scope");
            const rollback = store.set([
              { key: "key1", value: "value1" },
              { key: "key2", value: "value2" },
              { key: "key3", value: "value3" },
            ]);

            expect(store.get("key1")).toEqual({ key: "key1", value: "value1" });
            expect(store.get("key2")).toEqual({ key: "key2", value: "value2" });
            expect(store.get("key3")).toEqual({ key: "key3", value: "value3" });

            rollback();
            expect(store.get("key1")).toBeUndefined();
            expect(store.get("key2")).toBeUndefined();
            expect(store.get("key3")).toBeUndefined();
          });

          it("should rollback a single keyed value set operation", () => {
            interface KeyedString extends record.Keyed<string> {
              key: string;
              value: string;
            }
            const store = new ScopedUnaryStore<string, KeyedString>(
              basicHandleError,
            ).scope("scope");

            const item: KeyedString = { key: "key1", value: "value1" };
            const rollback = store.set(item);

            expect(store.get("key1")).toEqual({ key: "key1", value: "value1" });

            rollback();
            expect(store.get("key1")).toBeUndefined();
          });

          it("should rollback update of existing entry using keyed value", () => {
            interface KeyedString extends record.Keyed<string> {
              key: string;
              value: string;
            }
            const store = new ScopedUnaryStore<string, KeyedString>(
              basicHandleError,
            ).scope("scope");

            store.set({ key: "key1", value: "initial" });
            const rollback = store.set({ key: "key1", value: "updated" });

            expect(store.get("key1")).toEqual({ key: "key1", value: "updated" });

            rollback();
            expect(store.get("key1")).toEqual({ key: "key1", value: "initial" });
          });

          it("should notify delete listeners when rolling back new entry", () => {
            const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
            const scope1 = baseStore.scope("scope1");
            const scope2 = baseStore.scope("scope2");
            const deleteListener = vi.fn();

            scope2.onDelete(deleteListener);
            const rollback = scope1.set("key1", "value1");
            expect(deleteListener).not.toHaveBeenCalled();

            rollback();
            expect(deleteListener).toHaveBeenCalledWith("key1");
          });

          it("should notify set listeners when rolling back updated entry", () => {
            const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
            const scope1 = baseStore.scope("scope1");
            const scope2 = baseStore.scope("scope2");
            const setListener = vi.fn();

            scope1.set("key1", "initial");
            scope2.onSet(setListener);
            const rollback = scope1.set("key1", "updated");
            expect(setListener).toHaveBeenCalledWith("updated", undefined);

            setListener.mockClear();
            rollback();
            expect(setListener).toHaveBeenCalledWith("initial", undefined);
          });
        });

        describe("Delete Rollback", () => {
          it("should rollback a delete operation", () => {
            const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
              "scope",
            );
            store.set("key1", "value1");
            const rollback = store.delete("key1");
            expect(store.get("key1")).toBeUndefined();

            rollback();
            expect(store.get("key1")).toBe("value1");
          });

          it("should rollback multiple delete operations", () => {
            const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
              "scope",
            );
            store.set("key1", "value1");
            store.set("key2", "value2");
            store.set("key3", "value3");

            const rollback = store.delete(["key1", "key3"]);
            expect(store.get("key1")).toBeUndefined();
            expect(store.get("key2")).toBe("value2");
            expect(store.get("key3")).toBeUndefined();

            rollback();
            expect(store.get("key1")).toBe("value1");
            expect(store.get("key2")).toBe("value2");
            expect(store.get("key3")).toBe("value3");
          });

          it("should rollback filter-based delete", () => {
            const store = new ScopedUnaryStore<string, number>(basicHandleError).scope(
              "scope",
            );
            store.set("a", 1);
            store.set("b", 2);
            store.set("c", 3);
            store.set("d", 4);

            const rollback = store.delete((value) => value % 2 === 0);
            expect(store.get("a")).toBe(1);
            expect(store.get("b")).toBeUndefined();
            expect(store.get("c")).toBe(3);
            expect(store.get("d")).toBeUndefined();

            rollback();
            expect(store.get("a")).toBe(1);
            expect(store.get("b")).toBe(2);
            expect(store.get("c")).toBe(3);
            expect(store.get("d")).toBe(4);
          });

          it("should notify set listeners when rolling back delete", () => {
            const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
            const scope1 = baseStore.scope("scope1");
            const scope2 = baseStore.scope("scope2");
            const setListener = vi.fn();

            scope1.set("key1", "value1");
            scope2.onSet(setListener);
            const rollback = scope1.delete("key1");
            expect(setListener).not.toHaveBeenCalled();

            rollback();
            expect(setListener).toHaveBeenCalledWith("value1", undefined);
          });

          it("should preserve variant during delete rollback", () => {
            const baseStore = new ScopedUnaryStore<string, string, "variant">(
              basicHandleError,
            );
            const scope1 = baseStore.scope("scope1");
            const scope2 = baseStore.scope("scope2");
            const setListener = vi.fn();

            scope1.set("key1", "value1", "variant");
            scope2.onSet(setListener);
            const rollback = scope1.delete("key1", "variant");

            setListener.mockClear();
            rollback();
            expect(setListener).toHaveBeenCalledWith("value1", "variant");
          });
        });

        describe("Complex Rollback Scenarios", () => {
          it("should handle nested rollbacks", () => {
            const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
              "scope",
            );
            const rollback1 = store.set("key1", "value1");
            const rollback2 = store.set("key1", "value2");
            const rollback3 = store.delete("key1");

            expect(store.get("key1")).toBeUndefined();

            rollback3();
            expect(store.get("key1")).toBe("value2");

            rollback2();
            expect(store.get("key1")).toBe("value1");

            rollback1();
            expect(store.get("key1")).toBeUndefined();
          });

          it("should handle rollback of no-op operations", () => {
            const store = new ScopedUnaryStore<string, string>(
              basicHandleError,
              (a, b) => a === b,
            ).scope("scope");

            store.set("key1", "value1");
            const rollback = store.set("key1", "value1");

            expect(() => rollback()).not.toThrow();
            expect(store.get("key1")).toBe("value1");
          });

          it("should handle rollback of delete on non-existent keys", () => {
            const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
              "scope",
            );
            const rollback = store.delete("nonexistent");

            expect(() => rollback()).not.toThrow();
            expect(store.get("nonexistent")).toBeUndefined();
          });
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

        it("should delete entries using a filter function", () => {
          const store = new ScopedUnaryStore<string, number>(basicHandleError).scope(
            "scope",
          );
          store.set("a", 1);
          store.set("b", 2);
          store.set("c", 3);
          store.set("d", 4);
          store.set("e", 5);

          store.delete((value) => value % 2 === 0);

          expect(store.get("a")).toBe(1);
          expect(store.get("b")).toBeUndefined();
          expect(store.get("c")).toBe(3);
          expect(store.get("d")).toBeUndefined();
          expect(store.get("e")).toBe(5);
        });

        it("should delete entries using a filter with key parameter", () => {
          const store = new ScopedUnaryStore<string, string>(basicHandleError).scope(
            "scope",
          );
          store.set("key1", "value1");
          store.set("key2", "value2");
          store.set("test1", "test1");
          store.set("test2", "test2");

          store.delete((_, key) => key.startsWith("key"));

          expect(store.get("key1")).toBeUndefined();
          expect(store.get("key2")).toBeUndefined();
          expect(store.get("test1")).toBe("test1");
          expect(store.get("test2")).toBe("test2");
        });

        it("should delete complex objects using filter", () => {
          interface User {
            id: string;
            name: string;
            age: number;
          }

          const store = new ScopedUnaryStore<string, User>(basicHandleError).scope(
            "scope",
          );

          store.set("user1", { id: "1", name: "Alice", age: 25 });
          store.set("user2", { id: "2", name: "Bob", age: 35 });
          store.set("user3", { id: "3", name: "Charlie", age: 30 });
          store.set("user4", { id: "4", name: "David", age: 40 });

          store.delete((user) => user.age >= 35);

          expect(store.get("user1")).toEqual({ id: "1", name: "Alice", age: 25 });
          expect(store.get("user2")).toBeUndefined();
          expect(store.get("user3")).toEqual({ id: "3", name: "Charlie", age: 30 });
          expect(store.get("user4")).toBeUndefined();
        });

        it("should delete nothing when filter matches no entries", () => {
          const store = new ScopedUnaryStore<string, number>(basicHandleError).scope(
            "scope",
          );
          store.set("a", 1);
          store.set("b", 2);
          store.set("c", 3);

          store.delete((value) => value > 10);

          expect(store.get("a")).toBe(1);
          expect(store.get("b")).toBe(2);
          expect(store.get("c")).toBe(3);
        });

        it("should delete all entries when filter matches all", () => {
          const store = new ScopedUnaryStore<string, number>(basicHandleError).scope(
            "scope",
          );
          store.set("a", 1);
          store.set("b", 2);
          store.set("c", 3);

          store.delete(() => true);

          expect(store.get("a")).toBeUndefined();
          expect(store.get("b")).toBeUndefined();
          expect(store.get("c")).toBeUndefined();
          expect(store.list()).toEqual([]);
        });

        it("should combine filter with value and key checks", () => {
          const store = new ScopedUnaryStore<
            string,
            { value: number; active: boolean }
          >(basicHandleError).scope("scope");

          store.set("item1", { value: 10, active: true });
          store.set("item2", { value: 20, active: false });
          store.set("special1", { value: 30, active: true });
          store.set("special2", { value: 40, active: false });

          store.delete((obj, key) => key.startsWith("special") && !obj.active);

          expect(store.get("item1")).toEqual({ value: 10, active: true });
          expect(store.get("item2")).toEqual({ value: 20, active: false });
          expect(store.get("special1")).toEqual({ value: 30, active: true });
          expect(store.get("special2")).toBeUndefined();
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

        it("should notify listeners for each item deleted by filter", () => {
          const baseStore = new ScopedUnaryStore<string, number>(basicHandleError);
          const scope1 = baseStore.scope("scope1");
          const scope2 = baseStore.scope("scope2");
          const listener = vi.fn();

          scope2.onDelete(listener);

          scope1.set("a", 1);
          scope1.set("b", 2);
          scope1.set("c", 3);
          scope1.set("d", 4);

          scope1.delete((value) => value % 2 === 0);

          expect(listener).toHaveBeenCalledTimes(2);
          expect(listener).toHaveBeenCalledWith("b");
          expect(listener).toHaveBeenCalledWith("d");
        });

        it("should notify key-specific listeners only for matching filtered deletes", () => {
          const baseStore = new ScopedUnaryStore<string, number>(basicHandleError);
          const scope1 = baseStore.scope("scope1");
          const scope2 = baseStore.scope("scope2");
          const listenerA = vi.fn();
          const listenerB = vi.fn();
          const listenerC = vi.fn();

          scope2.onDelete(listenerA, "a");
          scope2.onDelete(listenerB, "b");
          scope2.onDelete(listenerC, "c");

          scope1.set("a", 1);
          scope1.set("b", 2);
          scope1.set("c", 3);

          scope1.delete((value) => value === 2);

          expect(listenerA).not.toHaveBeenCalled();
          expect(listenerB).toHaveBeenCalledWith("b");
          expect(listenerC).not.toHaveBeenCalled();
        });

        it("should not notify any listeners when filter matches nothing", () => {
          const baseStore = new ScopedUnaryStore<string, number>(basicHandleError);
          const scope1 = baseStore.scope("scope1");
          const scope2 = baseStore.scope("scope2");
          const listener = vi.fn();

          scope2.onDelete(listener);

          scope1.set("a", 1);
          scope1.set("b", 2);
          scope1.set("c", 3);

          scope1.delete((value) => value > 10);

          expect(listener).not.toHaveBeenCalled();
        });

        it("should handle filter delete with mixed listener types", () => {
          const baseStore = new ScopedUnaryStore<string, string>(basicHandleError);
          const scope1 = baseStore.scope("scope1");
          const scope2 = baseStore.scope("scope2");
          const globalListener = vi.fn();
          const specificListener = vi.fn();

          scope2.onDelete(globalListener);
          scope2.onDelete(specificListener, "key2");

          scope1.set("key1", "value1");
          scope1.set("key2", "value2");
          scope1.set("key3", "value3");

          scope1.delete((_, key) => key !== "key1");

          expect(globalListener).toHaveBeenCalledTimes(2);
          expect(globalListener).toHaveBeenCalledWith("key2");
          expect(globalListener).toHaveBeenCalledWith("key3");
          expect(specificListener).toHaveBeenCalledTimes(1);
          expect(specificListener).toHaveBeenCalledWith("key2");
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
        const config: base.StoreConfig<{
          store1: base.UnaryStore<record.Key, record.Keyed<record.Key>, undefined>;
          store2: base.UnaryStore<record.Key, record.Keyed<record.Key>, undefined>;
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

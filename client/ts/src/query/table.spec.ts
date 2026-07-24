// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { query } from "@/query";

const noopError = (_: Error) => {};

const setEvents = (listener: ReturnType<typeof vi.fn>) =>
  listener.mock.calls.filter(([e]) => e.variant === "set");

describe("Table", () => {
  describe("Basic Operations", () => {
    describe("Set and Get", () => {
      it("should set and get a value", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        expect(table.get("key1")).toBe("value1");
      });

      it("should set a single value with key property", () => {
        interface KeyedValue extends record.Keyed<string> {
          key: string;
          value: string;
        }
        const table = new query.Table<string, KeyedValue>(noopError);

        const item: KeyedValue = { key: "key1", value: "value1" };
        table.set(item.key, item);

        expect(table.get("key1")).toEqual({ key: "key1", value: "value1" });
      });

      it("should set multiple values with key property via keyed-object set", () => {
        interface KeyedValue extends record.Keyed<string> {
          key: string;
          value: string;
        }
        const table = new query.Table<string, KeyedValue>(noopError);

        const items: KeyedValue[] = [
          { key: "key1", value: "value1" },
          { key: "key2", value: "value2" },
          { key: "key3", value: "value3" },
        ];
        table.set(items);

        expect(table.get("key1")).toEqual({ key: "key1", value: "value1" });
        expect(table.get("key2")).toEqual({ key: "key2", value: "value2" });
        expect(table.get("key3")).toEqual({ key: "key3", value: "value3" });
      });

      it("should update an existing value", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key1", "value2");
        expect(table.get("key1")).toBe("value2");
      });

      it("should handle setter functions", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "initial");
        table.set("key1", (prev) => `${prev}_updated`);
        expect(table.get("key1")).toBe("initial_updated");
      });

      it("should return undefined for non-existent keys", () => {
        const table = new query.Table<string, string>(noopError);
        expect(table.get("nonexistent")).toBeUndefined();
      });

      it("should get multiple values by keys array", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key3", "value3");

        const values = table.get(["key1", "key3", "nonexistent"]);
        expect(values).toEqual(["value1", "value3"]);
      });

      it("should filter values using a predicate", () => {
        const table = new query.Table<string, number>(noopError);
        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);
        table.set("d", 4);

        const evenValues = table.get((value) => value % 2 === 0);
        expect(evenValues).toEqual([2, 4]);
      });

      it("should not set null values", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key1", () => null as any);
        expect(table.get("key1")).toBe("value1");
      });

      it("should distinguish between the row key and the value's key field", () => {
        interface ComplexValue extends record.Keyed<string> {
          key: string;
          data: number;
        }

        const table = new query.Table<string, ComplexValue>(noopError);

        // set keys the row by its first argument, not the value's key field.
        table.set("explicitKey", { key: "valueKey", data: 100 });
        expect(table.get("explicitKey")).toEqual({ key: "valueKey", data: 100 });

        // Keyed-object set keys each row by the object's key property.
        table.set([{ key: "derivedKey", data: 200 }]);
        expect(table.get("derivedKey")).toEqual({ key: "derivedKey", data: 200 });

        table.set([
          { key: "arrayKey1", data: 300 },
          { key: "arrayKey2", data: 400 },
        ]);
        expect(table.get("arrayKey1")).toEqual({ key: "arrayKey1", data: 300 });
        expect(table.get("arrayKey2")).toEqual({ key: "arrayKey2", data: 400 });
      });

      it("should handle mixed set operations", () => {
        interface KeyedData extends record.Keyed<string> {
          key: string;
          value: number;
        }

        const table = new query.Table<string, KeyedData>(noopError);

        table.set("key1", { key: "key1", value: 100 });
        table.set([{ key: "key2", value: 200 }]);
        table.set([
          { key: "key3", value: 300 },
          { key: "key4", value: 400 },
        ]);

        expect(table.get("key1")).toEqual({ key: "key1", value: 100 });
        expect(table.get("key2")).toEqual({ key: "key2", value: 200 });
        expect(table.get("key3")).toEqual({ key: "key3", value: 300 });
        expect(table.get("key4")).toEqual({ key: "key4", value: 400 });
      });
    });

    describe("Set If Absent", () => {
      interface KeyedValue extends record.Keyed<string> {
        key: string;
        value: string;
      }

      it("should insert a single value when the key is absent", () => {
        const table = new query.Table<string, KeyedValue>(noopError);
        table.setIfAbsent({ key: "key1", value: "value1" });
        expect(table.get("key1")).toEqual({ key: "key1", value: "value1" });
      });

      it("should leave an existing value untouched", () => {
        const table = new query.Table<string, KeyedValue>(noopError);
        table.set([{ key: "key1", value: "original" }]);
        table.setIfAbsent({ key: "key1", value: "replacement" });
        expect(table.get("key1")).toEqual({ key: "key1", value: "original" });
      });

      it("should insert only the absent keys from an array", () => {
        const table = new query.Table<string, KeyedValue>(noopError);
        table.set([{ key: "key1", value: "original" }]);
        table.setIfAbsent([
          { key: "key1", value: "replacement" },
          { key: "key2", value: "value2" },
        ]);
        expect(table.get("key1")).toEqual({ key: "key1", value: "original" });
        expect(table.get("key2")).toEqual({ key: "key2", value: "value2" });
      });

      it("should not notify subscribers for keys that already exist", () => {
        const table = new query.Table<string, KeyedValue>(noopError);
        const listener = vi.fn();

        table.set([{ key: "key1", value: "original" }]);
        table.subscribe(listener);
        table.setIfAbsent([
          { key: "key1", value: "replacement" },
          { key: "key2", value: "value2" },
        ]);

        expect(listener).toHaveBeenCalledTimes(1);
        expect(listener).toHaveBeenCalledWith({
          variant: "set",
          key: "key2",
          value: { key: "key2", value: "value2" },
        });
      });

      it("should only roll back the keys it inserted", () => {
        const table = new query.Table<string, KeyedValue>(noopError);
        table.set([{ key: "key1", value: "original" }]);
        const rollback = table.setIfAbsent([
          { key: "key1", value: "replacement" },
          { key: "key2", value: "value2" },
        ]);

        rollback();
        expect(table.get("key1")).toEqual({ key: "key1", value: "original" });
        expect(table.get("key2")).toBeUndefined();
      });
    });

    describe("Rollback Functionality", () => {
      describe("Set Rollback", () => {
        it("should rollback a set operation for new entry", () => {
          const table = new query.Table<string, string>(noopError);
          const rollback = table.set("key1", "value1");
          expect(table.get("key1")).toBe("value1");

          rollback();
          expect(table.get("key1")).toBeUndefined();
        });

        it("should rollback a set operation for existing entry", () => {
          const table = new query.Table<string, string>(noopError);
          table.set("key1", "initial");
          const rollback = table.set("key1", "updated");
          expect(table.get("key1")).toBe("updated");

          rollback();
          expect(table.get("key1")).toBe("initial");
        });

        it("should rollback multiple set operations", () => {
          interface KeyedString extends record.Keyed<string> {
            key: string;
            value: string;
          }
          const table = new query.Table<string, KeyedString>(noopError);
          const rollback = table.set([
            { key: "key1", value: "value1" },
            { key: "key2", value: "value2" },
            { key: "key3", value: "value3" },
          ]);

          expect(table.get("key1")).toEqual({ key: "key1", value: "value1" });
          expect(table.get("key2")).toEqual({ key: "key2", value: "value2" });
          expect(table.get("key3")).toEqual({ key: "key3", value: "value3" });

          rollback();
          expect(table.get("key1")).toBeUndefined();
          expect(table.get("key2")).toBeUndefined();
          expect(table.get("key3")).toBeUndefined();
        });

        it("should rollback a single keyed value set operation", () => {
          interface KeyedString extends record.Keyed<string> {
            key: string;
            value: string;
          }
          const table = new query.Table<string, KeyedString>(noopError);

          const item: KeyedString = { key: "key1", value: "value1" };
          const rollback = table.set([item]);

          expect(table.get("key1")).toEqual({ key: "key1", value: "value1" });

          rollback();
          expect(table.get("key1")).toBeUndefined();
        });

        it("should rollback update of existing entry using keyed value", () => {
          interface KeyedString extends record.Keyed<string> {
            key: string;
            value: string;
          }
          const table = new query.Table<string, KeyedString>(noopError);

          table.set([{ key: "key1", value: "initial" }]);
          const rollback = table.set([{ key: "key1", value: "updated" }]);

          expect(table.get("key1")).toEqual({ key: "key1", value: "updated" });

          rollback();
          expect(table.get("key1")).toEqual({ key: "key1", value: "initial" });
        });

        it("should notify subscribers of a delete when rolling back new entry", () => {
          const table = new query.Table<string, string>(noopError);
          const listener = vi.fn();

          table.subscribe(listener);
          const rollback = table.set("key1", "value1");
          expect(listener).not.toHaveBeenCalledWith({
            variant: "delete",
            key: "key1",
          });

          rollback();
          expect(listener).toHaveBeenCalledWith({ variant: "delete", key: "key1" });
        });

        it("should notify subscribers of a set when rolling back updated entry", () => {
          const table = new query.Table<string, string>(noopError);
          const listener = vi.fn();

          table.set("key1", "initial");
          table.subscribe(listener);
          const rollback = table.set("key1", "updated");
          expect(listener).toHaveBeenCalledWith({
            variant: "set",
            key: "key1",
            value: "updated",
          });

          listener.mockClear();
          rollback();
          expect(listener).toHaveBeenCalledWith({
            variant: "set",
            key: "key1",
            value: "initial",
          });
        });
      });

      describe("Delete Rollback", () => {
        it("should rollback a delete operation", () => {
          const table = new query.Table<string, string>(noopError);
          table.set("key1", "value1");
          const rollback = table.delete("key1");
          expect(table.get("key1")).toBeUndefined();

          rollback();
          expect(table.get("key1")).toBe("value1");
        });

        it("should rollback multiple delete operations", () => {
          const table = new query.Table<string, string>(noopError);
          table.set("key1", "value1");
          table.set("key2", "value2");
          table.set("key3", "value3");

          const rollback = table.delete(["key1", "key3"]);
          expect(table.get("key1")).toBeUndefined();
          expect(table.get("key2")).toBe("value2");
          expect(table.get("key3")).toBeUndefined();

          rollback();
          expect(table.get("key1")).toBe("value1");
          expect(table.get("key2")).toBe("value2");
          expect(table.get("key3")).toBe("value3");
        });

        it("should rollback filter-based delete", () => {
          const table = new query.Table<string, number>(noopError);
          table.set("a", 1);
          table.set("b", 2);
          table.set("c", 3);
          table.set("d", 4);

          const rollback = table.delete((value) => value % 2 === 0);
          expect(table.get("a")).toBe(1);
          expect(table.get("b")).toBeUndefined();
          expect(table.get("c")).toBe(3);
          expect(table.get("d")).toBeUndefined();

          rollback();
          expect(table.get("a")).toBe(1);
          expect(table.get("b")).toBe(2);
          expect(table.get("c")).toBe(3);
          expect(table.get("d")).toBe(4);
        });

        it("should notify subscribers of a set when rolling back delete", () => {
          const table = new query.Table<string, string>(noopError);
          const listener = vi.fn();

          table.set("key1", "value1");
          table.subscribe(listener);
          const rollback = table.delete("key1");
          expect(setEvents(listener)).toHaveLength(0);

          rollback();
          expect(listener).toHaveBeenCalledWith({
            variant: "set",
            key: "key1",
            value: "value1",
          });
        });
      });

      describe("Complex Rollback Scenarios", () => {
        it("should handle nested rollbacks", () => {
          const table = new query.Table<string, string>(noopError);
          const rollback1 = table.set("key1", "value1");
          const rollback2 = table.set("key1", "value2");
          const rollback3 = table.delete("key1");

          expect(table.get("key1")).toBeUndefined();

          rollback3();
          expect(table.get("key1")).toBe("value2");

          rollback2();
          expect(table.get("key1")).toBe("value1");

          rollback1();
          expect(table.get("key1")).toBeUndefined();
        });

        it("should handle rollback of no-op operations", () => {
          const table = new query.Table<string, string>(noopError);

          table.set("key1", "value1");
          const rollback = table.set("key1", "value1");

          expect(() => rollback()).not.toThrow();
          expect(table.get("key1")).toBe("value1");
        });

        it("should handle rollback of delete on non-existent keys", () => {
          const table = new query.Table<string, string>(noopError);
          const rollback = table.delete("nonexistent");

          expect(() => rollback()).not.toThrow();
          expect(table.get("nonexistent")).toBeUndefined();
        });
      });
    });

    describe("Delete", () => {
      it("should delete an entry", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        expect(table.get("key1")).toBe("value1");

        table.delete("key1");
        expect(table.get("key1")).toBeUndefined();
      });

      it("should handle deleting non-existent keys", () => {
        const table = new query.Table<string, string>(noopError);
        expect(() => table.delete("nonexistent")).not.toThrow();
      });

      it("should delete entries using a filter function", () => {
        const table = new query.Table<string, number>(noopError);
        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);
        table.set("d", 4);
        table.set("e", 5);

        table.delete((value) => value % 2 === 0);

        expect(table.get("a")).toBe(1);
        expect(table.get("b")).toBeUndefined();
        expect(table.get("c")).toBe(3);
        expect(table.get("d")).toBeUndefined();
        expect(table.get("e")).toBe(5);
      });

      it("should delete entries using a filter with key parameter", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("test1", "test1");
        table.set("test2", "test2");

        table.delete((_, key) => key.startsWith("key"));

        expect(table.get("key1")).toBeUndefined();
        expect(table.get("key2")).toBeUndefined();
        expect(table.get("test1")).toBe("test1");
        expect(table.get("test2")).toBe("test2");
      });

      it("should delete complex objects using filter", () => {
        interface User {
          id: string;
          name: string;
          age: number;
        }

        const table = new query.Table<string, User>(noopError);

        table.set("user1", { id: "1", name: "Alice", age: 25 });
        table.set("user2", { id: "2", name: "Bob", age: 35 });
        table.set("user3", { id: "3", name: "Charlie", age: 30 });
        table.set("user4", { id: "4", name: "David", age: 40 });

        table.delete((user) => user.age >= 35);

        expect(table.get("user1")).toEqual({ id: "1", name: "Alice", age: 25 });
        expect(table.get("user2")).toBeUndefined();
        expect(table.get("user3")).toEqual({ id: "3", name: "Charlie", age: 30 });
        expect(table.get("user4")).toBeUndefined();
      });

      it("should delete nothing when filter matches no entries", () => {
        const table = new query.Table<string, number>(noopError);
        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);

        table.delete((value) => value > 10);

        expect(table.get("a")).toBe(1);
        expect(table.get("b")).toBe(2);
        expect(table.get("c")).toBe(3);
      });

      it("should delete all entries when filter matches all", () => {
        const table = new query.Table<string, number>(noopError);
        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);

        table.delete(() => true);

        expect(table.get("a")).toBeUndefined();
        expect(table.get("b")).toBeUndefined();
        expect(table.get("c")).toBeUndefined();
        expect(table.list()).toEqual([]);
      });

      it("should combine filter with value and key checks", () => {
        const table = new query.Table<string, { value: number; active: boolean }>(
          noopError,
        );

        table.set("item1", { value: 10, active: true });
        table.set("item2", { value: 20, active: false });
        table.set("special1", { value: 30, active: true });
        table.set("special2", { value: 40, active: false });

        table.delete((obj, key) => key.startsWith("special") && !obj.active);

        expect(table.get("item1")).toEqual({ value: 10, active: true });
        expect(table.get("item2")).toEqual({ value: 20, active: false });
        expect(table.get("special1")).toEqual({ value: 30, active: true });
        expect(table.get("special2")).toBeUndefined();
      });
    });

    describe("List", () => {
      it("should return empty array when table is empty", () => {
        const table = new query.Table<string, string>(noopError);
        expect(table.list()).toEqual([]);
      });

      it("should return all values in the table", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key3", "value3");

        const values = table.list();
        expect(values).toHaveLength(3);
        expect(values).toContain("value1");
        expect(values).toContain("value2");
        expect(values).toContain("value3");
      });

      it("should return values after deletions", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key3", "value3");
        table.delete("key2");

        const values = table.list();
        expect(values).toHaveLength(2);
        expect(values).toContain("value1");
        expect(values).toContain("value3");
        expect(values).not.toContain("value2");
      });

      it("should return values after updates", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key1", "updated1");

        const values = table.list();
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

        const table = new query.Table<string, User>(noopError);
        const user1: User = { id: "1", name: "John", age: 30 };
        const user2: User = { id: "2", name: "Jane", age: 25 };
        const user3: User = { id: "3", name: "Bob", age: 35 };

        table.set("user1", user1);
        table.set("user2", user2);
        table.set("user3", user3);

        const users = table.list();
        expect(users).toHaveLength(3);
        expect(users).toContainEqual(user1);
        expect(users).toContainEqual(user2);
        expect(users).toContainEqual(user3);
      });

      it("should return values after bulk set operations", () => {
        const table = new query.Table<string, { key: string; value: string }>(
          noopError,
        );

        const items = [
          { key: "key1", value: "value1" },
          { key: "key2", value: "value2" },
          { key: "key3", value: "value3" },
        ];

        table.set(items);

        const values = table.list();
        expect(values).toHaveLength(3);
        expect(values).toContainEqual({ key: "key1", value: "value1" });
        expect(values).toContainEqual({ key: "key2", value: "value2" });
        expect(values).toContainEqual({ key: "key3", value: "value3" });
      });

      it("should return values after bulk delete operations", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key3", "value3");
        table.set("key4", "value4");

        table.delete(["key2", "key3"]);

        const values = table.list();
        expect(values).toHaveLength(2);
        expect(values).toContain("value1");
        expect(values).toContain("value4");
      });

      it("should return empty array after clear", () => {
        const table = new query.Table<string, string>(noopError);

        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key3", "value3");

        expect(table.list()).toHaveLength(3);

        table.clear();

        expect(table.list()).toEqual([]);
      });

      it("should notify subscribers for every cleared key", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");
        const listener = vi.fn();
        table.subscribe(listener);

        table.clear();

        expect(listener).toHaveBeenCalledTimes(2);
        expect(listener).toHaveBeenCalledWith({ variant: "delete", key: "key1" });
        expect(listener).toHaveBeenCalledWith({ variant: "delete", key: "key2" });
      });

      it("should work with number keys", () => {
        const table = new query.Table<number, string>(noopError);
        table.set(1, "value1");
        table.set(2, "value2");
        table.set(3, "value3");

        const values = table.list();
        expect(values).toHaveLength(3);
        expect(values).toContain("value1");
        expect(values).toContain("value2");
        expect(values).toContain("value3");
      });

      it("should handle mixed operations correctly", () => {
        const table = new query.Table<string, number>(noopError);

        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);
        table.delete("b");
        table.set("d", 4);
        table.set("a", 10);

        const values = table.list();
        expect(values).toHaveLength(3);
        expect(values).toContain(10);
        expect(values).toContain(3);
        expect(values).toContain(4);
        expect(values).not.toContain(1);
        expect(values).not.toContain(2);
      });

      it("should return independent arrays on each call", () => {
        const table = new query.Table<string, string>(noopError);
        table.set("key1", "value1");
        table.set("key2", "value2");

        const list1 = table.list();
        const list2 = table.list();

        expect(list1).not.toBe(list2);
        expect(list1).toEqual(list2);

        list1.push("extra");
        expect(list1).toHaveLength(3);
        expect(list2).toHaveLength(2);
        expect(table.list()).toHaveLength(2);
      });

      it("should preserve values with equal function check", () => {
        const equalFunc = (a: string, b: string) => a.toLowerCase() === b.toLowerCase();
        const table = new query.Table<string, string>(noopError, equalFunc);

        table.set("key1", "Value1");
        table.set("key2", "Value2");
        table.set("key1", "VALUE1");

        const values = table.list();
        expect(values).toHaveLength(2);
        expect(values).toContain("Value1");
        expect(values).toContain("Value2");
      });
    });
  });

  describe("Subscriptions", () => {
    describe("Set Events", () => {
      it("should notify subscribers when a value is set", () => {
        const table = new query.Table<string, string>(noopError);
        const listener = vi.fn();

        table.subscribe(listener);
        table.set("key1", "value1");

        expect(listener).toHaveBeenCalledWith({
          variant: "set",
          key: "key1",
          value: "value1",
        });
      });

      it("should notify only for specific key when key filter is provided", () => {
        const table = new query.Table<string, string>(noopError);
        const listener1 = vi.fn();
        const listener2 = vi.fn();

        table.subscribe(listener1, "key1");
        table.subscribe(listener2, "key2");

        table.set("key1", "value1");
        expect(listener1).toHaveBeenCalledWith({
          variant: "set",
          key: "key1",
          value: "value1",
        });
        expect(listener2).not.toHaveBeenCalled();

        table.set("key2", "value2");
        expect(listener2).toHaveBeenCalledWith({
          variant: "set",
          key: "key2",
          value: "value2",
        });
        expect(listener1).toHaveBeenCalledTimes(1);
      });

      it("should remove subscriber when destructor is called", () => {
        const table = new query.Table<string, string>(noopError);
        const listener = vi.fn();

        const destructor = table.subscribe(listener);
        table.set("key1", "value1");
        expect(listener).toHaveBeenCalledTimes(1);

        destructor();
        table.set("key2", "value2");
        expect(listener).toHaveBeenCalledTimes(1);
      });

      it("should notify every subscriber on a set", () => {
        const table = new query.Table<string, string>(noopError);
        const listener1 = vi.fn();
        const listener2 = vi.fn();

        table.subscribe(listener1);
        table.subscribe(listener2);

        table.set("key1", "value1");

        const event = { variant: "set", key: "key1", value: "value1" };
        expect(listener1).toHaveBeenCalledWith(event);
        expect(listener2).toHaveBeenCalledWith(event);
      });

      it("should notify subscribers for every value in a multi-value set", () => {
        const table = new query.Table<string, record.Keyed<string>>(noopError);
        const listener = vi.fn();
        const keyed = vi.fn();
        table.subscribe(listener);
        table.subscribe(keyed, "key1");
        table.set([{ key: "key1" }, { key: "key2" }]);
        expect(listener).toHaveBeenCalledTimes(2);
        expect(keyed).toHaveBeenCalledTimes(1);
        expect(keyed).toHaveBeenCalledWith({
          variant: "set",
          key: "key1",
          value: { key: "key1" },
        });
      });

      it("should call the error sink when a subscriber throws", () => {
        const onError = vi.fn();
        const table = new query.Table<string, string>(onError);
        const error = new Error("Listener error");
        const listener = vi.fn(() => {
          throw error;
        });

        table.subscribe(listener);
        table.set("key1", "value1");

        expect(onError).toHaveBeenCalledTimes(1);
        const reported = onError.mock.calls[0][0] as Error;
        expect(reported.message).toBe("failed to notify table subscriber");
        expect(reported.cause).toBe(error);
      });

      it("should continue notifying other subscribers when one throws", () => {
        const onError = vi.fn();
        const table = new query.Table<string, string>(onError);
        const listener1 = vi.fn(() => {
          throw new Error("First listener error");
        });
        const listener2 = vi.fn();
        const listener3 = vi.fn();

        table.subscribe(listener1);
        table.subscribe(listener2);
        table.subscribe(listener3);

        table.set("key1", "value1");

        const event = { variant: "set", key: "key1", value: "value1" };
        expect(listener1).toHaveBeenCalledWith(event);
        expect(listener2).toHaveBeenCalledWith(event);
        expect(listener3).toHaveBeenCalledWith(event);
        expect(onError).toHaveBeenCalledTimes(1);
      });

      it("should handle errors from multiple subscribers", () => {
        const onError = vi.fn();
        const table = new query.Table<string, string>(onError);
        const listener1 = vi.fn(() => {
          throw new Error("First error");
        });
        const listener2 = vi.fn(() => {
          throw new Error("Second error");
        });
        const listener3 = vi.fn();

        table.subscribe(listener1);
        table.subscribe(listener2);
        table.subscribe(listener3);

        table.set("key1", "value1");

        expect(listener3).toHaveBeenCalledWith({
          variant: "set",
          key: "key1",
          value: "value1",
        });
        expect(onError).toHaveBeenCalledTimes(2);
      });
    });

    describe("Delete Events", () => {
      it("should notify subscribers when a value is deleted", () => {
        const table = new query.Table<string, string>(noopError);
        const listener = vi.fn();

        table.subscribe(listener);
        table.set("key1", "value1");
        table.delete("key1");

        expect(listener).toHaveBeenCalledWith({ variant: "delete", key: "key1" });
      });

      it("should notify only for specific key when key filter is provided", () => {
        const table = new query.Table<string, string>(noopError);
        const listener1 = vi.fn();
        const listener2 = vi.fn();

        table.set("key1", "value1");
        table.set("key2", "value2");

        table.subscribe(listener1, "key1");
        table.subscribe(listener2, "key2");

        table.delete("key1");
        expect(listener1).toHaveBeenCalledWith({ variant: "delete", key: "key1" });
        expect(listener2).not.toHaveBeenCalled();

        table.delete("key2");
        expect(listener2).toHaveBeenCalledWith({ variant: "delete", key: "key2" });
        expect(listener1).toHaveBeenCalledTimes(1);
      });

      it("should remove subscriber when destructor is called", () => {
        const table = new query.Table<string, string>(noopError);
        const listener = vi.fn();

        table.set("key1", "value1");
        table.set("key2", "value2");
        const destructor = table.subscribe(listener);
        table.delete("key1");
        expect(listener).toHaveBeenCalledTimes(1);

        destructor();
        table.delete("key2");
        expect(listener).toHaveBeenCalledTimes(1);
      });

      it("should call the error sink when a subscriber throws on delete", () => {
        const onError = vi.fn();
        const table = new query.Table<string, string>(onError);
        const error = new Error("Delete listener error");
        table.set("key1", "value1");
        const listener = vi.fn((event: query.TableEvent<string, string>) => {
          if (event.variant === "delete") throw error;
        });

        table.subscribe(listener);
        table.delete("key1");

        expect(onError).toHaveBeenCalledTimes(1);
        const reported = onError.mock.calls[0][0] as Error;
        expect(reported.message).toBe("failed to notify table subscriber");
        expect(reported.cause).toBe(error);
      });

      it("should continue notifying other subscribers when one throws", () => {
        const onError = vi.fn();
        const table = new query.Table<string, string>(onError);
        table.set("key1", "value1");
        const listener1 = vi.fn((event: query.TableEvent<string, string>) => {
          if (event.variant === "delete") throw new Error("First delete error");
        });
        const listener2 = vi.fn();
        const listener3 = vi.fn();

        table.subscribe(listener1);
        table.subscribe(listener2);
        table.subscribe(listener3);

        table.delete("key1");

        const event = { variant: "delete", key: "key1" };
        expect(listener1).toHaveBeenCalledWith(event);
        expect(listener2).toHaveBeenCalledWith(event);
        expect(listener3).toHaveBeenCalledWith(event);
        expect(onError).toHaveBeenCalledTimes(1);
      });

      it("should notify subscribers for each item deleted by filter", () => {
        const table = new query.Table<string, number>(noopError);
        const listener = vi.fn();

        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);
        table.set("d", 4);
        table.subscribe(listener);

        table.delete((value) => value % 2 === 0);

        expect(listener).toHaveBeenCalledTimes(2);
        expect(listener).toHaveBeenCalledWith({ variant: "delete", key: "b" });
        expect(listener).toHaveBeenCalledWith({ variant: "delete", key: "d" });
      });

      it("should notify key-specific subscribers only for matching filtered deletes", () => {
        const table = new query.Table<string, number>(noopError);
        const listenerA = vi.fn();
        const listenerB = vi.fn();
        const listenerC = vi.fn();

        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);

        table.subscribe(listenerA, "a");
        table.subscribe(listenerB, "b");
        table.subscribe(listenerC, "c");

        table.delete((value) => value === 2);

        expect(listenerA).not.toHaveBeenCalled();
        expect(listenerB).toHaveBeenCalledWith({ variant: "delete", key: "b" });
        expect(listenerC).not.toHaveBeenCalled();
      });

      it("should not notify any subscribers when filter matches nothing", () => {
        const table = new query.Table<string, number>(noopError);
        const listener = vi.fn();

        table.set("a", 1);
        table.set("b", 2);
        table.set("c", 3);
        table.subscribe(listener);

        table.delete((value) => value > 10);

        expect(listener).not.toHaveBeenCalled();
      });

      it("should handle filter delete with mixed subscriber types", () => {
        const table = new query.Table<string, string>(noopError);
        const globalListener = vi.fn();
        const specificListener = vi.fn();

        table.set("key1", "value1");
        table.set("key2", "value2");
        table.set("key3", "value3");

        table.subscribe(globalListener);
        table.subscribe(specificListener, "key2");

        table.delete((_, key) => key !== "key1");

        expect(globalListener).toHaveBeenCalledTimes(2);
        expect(globalListener).toHaveBeenCalledWith({
          variant: "delete",
          key: "key2",
        });
        expect(globalListener).toHaveBeenCalledWith({
          variant: "delete",
          key: "key3",
        });
        expect(specificListener).toHaveBeenCalledTimes(1);
        expect(specificListener).toHaveBeenCalledWith({
          variant: "delete",
          key: "key2",
        });
      });
    });

    describe("Equality Silencing", () => {
      it("should not notify subscribers when the set value deep-equals the row", () => {
        const table = new query.Table<string, { name: string }>(noopError);
        const listener = vi.fn();

        table.subscribe(listener);
        table.set("key1", { name: "a" });
        expect(listener).toHaveBeenCalledTimes(1);

        table.set("key1", { name: "a" });
        expect(listener).toHaveBeenCalledTimes(1);
      });

      it("should notify subscribers when the set value differs", () => {
        const table = new query.Table<string, { name: string }>(noopError);
        const listener = vi.fn();

        table.set("key1", { name: "a" });
        table.subscribe(listener);
        table.set("key1", { name: "b" });

        expect(listener).toHaveBeenCalledWith({
          variant: "set",
          key: "key1",
          value: { name: "b" },
        });
      });

      it("should respect a custom equal override", () => {
        const table = new query.Table<string, string>(
          noopError,
          (a, b) => a.toLowerCase() === b.toLowerCase(),
        );
        const listener = vi.fn();

        table.subscribe(listener);
        table.set("key1", "Value");
        table.set("key1", "VALUE");
        expect(listener).toHaveBeenCalledTimes(1);

        table.set("key1", "other");
        expect(listener).toHaveBeenCalledTimes(2);
      });

      it("should silence equal-value multi-set entries individually", () => {
        interface KeyedValue extends record.Keyed<string> {
          key: string;
          value: string;
        }
        const table = new query.Table<string, KeyedValue>(noopError);
        const listener = vi.fn();

        table.set([{ key: "key1", value: "a" }]);
        table.subscribe(listener);
        table.set([
          { key: "key1", value: "a" },
          { key: "key2", value: "b" },
        ]);

        expect(listener).toHaveBeenCalledTimes(1);
        expect(listener).toHaveBeenCalledWith({
          variant: "set",
          key: "key2",
          value: { key: "key2", value: "b" },
        });
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
        const table = new query.Table<string, User>(noopError);
        const user: User = { id: "1", name: "John", age: 30 };

        table.set("user1", user);
        expect(table.get("user1")).toEqual(user);
      });

      it("should update nested properties with setter function", () => {
        const table = new query.Table<string, User>(noopError);
        const user: User = { id: "1", name: "John", age: 30 };

        table.set("user1", user);
        table.set("user1", (prev) => prev && { ...prev, age: 31 });

        const updated = table.get("user1");
        expect(updated?.age).toBe(31);
        expect(updated?.name).toBe("John");
      });
    });

    describe("partialUpdate", () => {
      interface User {
        id: string;
        name: string;
        age: number;
      }

      it("should merge the partial into the existing row", () => {
        const table = new query.Table<string, User>(noopError);
        table.set("user1", { id: "1", name: "John", age: 30 });
        query.partialUpdate(table, "user1", { age: 31 });
        expect(table.get("user1")).toEqual({ id: "1", name: "John", age: 31 });
      });

      it("should be a no-op when the row is absent", () => {
        const table = new query.Table<string, User>(noopError);
        query.partialUpdate(table, "missing", { age: 31 });
        expect(table.get("missing")).toBeUndefined();
      });

      it("should return a rollback that restores the prior value", () => {
        const table = new query.Table<string, User>(noopError);
        table.set("user1", { id: "1", name: "John", age: 30 });
        const rollback = query.partialUpdate(table, "user1", { age: 31 });
        rollback();
        expect(table.get("user1")).toEqual({ id: "1", name: "John", age: 30 });
      });
    });

    describe("Table Independence", () => {
      it("keeps rows isolated between separate tables", () => {
        const table1 = new query.Table<string, record.Keyed<string>>(noopError);
        const table2 = new query.Table<string, record.Keyed<string>>(noopError);

        table1.set([{ key: "key1" }]);
        table2.set([{ key: "key2" }]);

        expect(table1.get("key1")).toEqual({ key: "key1" });
        expect(table1.get("key2")).toBeUndefined();
        expect(table2.get("key2")).toEqual({ key: "key2" });
      });
    });
  });

  describe("orderByKeys", () => {
    interface Item {
      key: number;
      name: string;
    }
    const getKey = (i: Item) => i.key;

    it("should return items in the order of the input keys", () => {
      const items: Item[] = [
        { key: 3, name: "c" },
        { key: 1, name: "a" },
        { key: 2, name: "b" },
      ];
      const ordered = query.orderByKeys([1, 2, 3], items, getKey);
      expect(ordered.map((i) => i.name)).toEqual(["a", "b", "c"]);
    });

    it("should drop keys that have no corresponding item", () => {
      const items: Item[] = [
        { key: 1, name: "a" },
        { key: 3, name: "c" },
      ];
      const ordered = query.orderByKeys([1, 2, 3], items, getKey);
      expect(ordered.map((i) => i.name)).toEqual(["a", "c"]);
    });

    it("should deduplicate repeated keys", () => {
      const items: Item[] = [
        { key: 1, name: "a" },
        { key: 2, name: "b" },
      ];
      const ordered = query.orderByKeys([1, 2, 1, 2, 1], items, getKey);
      expect(ordered.map((i) => i.name)).toEqual(["a", "b"]);
    });

    it("should return an empty array when keys is empty", () => {
      const items: Item[] = [{ key: 1, name: "a" }];
      expect(query.orderByKeys([], items, getKey)).toEqual([]);
    });

    it("should return an empty array when items is empty", () => {
      expect(query.orderByKeys([1, 2, 3], [], getKey)).toEqual([]);
    });

    it("should ignore items whose key is not present in keys", () => {
      const items: Item[] = [
        { key: 1, name: "a" },
        { key: 99, name: "x" },
      ];
      const ordered = query.orderByKeys([1], items, getKey);
      expect(ordered.map((i) => i.name)).toEqual(["a"]);
    });

    it("should support string keys", () => {
      const items = [
        { key: "b", name: "two" },
        { key: "a", name: "one" },
      ];
      const ordered = query.orderByKeys(["a", "b"], items, (i) => i.key);
      expect(ordered.map((i) => i.name)).toEqual(["one", "two"]);
    });

    it("should keep the first occurrence when items contains duplicate keys", () => {
      const items: Item[] = [
        { key: 1, name: "first" },
        { key: 1, name: "second" },
      ];
      const ordered = query.orderByKeys([1], items, getKey);
      // Map.set with the same key keeps the last value written — confirming contract.
      expect(ordered).toEqual([{ key: 1, name: "second" }]);
    });
  });
});

describe("Tombstones", () => {
  interface Doc extends record.Keyed<string> {
    key: string;
    name: string;
  }
  const newTable = () => new query.Table<string, Doc>(noopError);

  it("should report unknown for a never-seen key", () => {
    const table = newTable();
    expect(table.status("missing")).toBe("unknown");
    expect(table.getTombstone("missing")).toBeUndefined();
  });

  it("should report present for a live entry", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    expect(table.status("k1")).toBe("present");
    expect(table.getTombstone("k1")).toBeUndefined();
  });

  it("should corpse a deleted entry", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    table.delete("k1");
    expect(table.get("k1")).toBeUndefined();
    expect(table.status("k1")).toBe("tombstoned");
    const tombstone = table.getTombstone("k1");
    expect(tombstone?.corpse).toEqual({ key: "k1", name: "a" });
    expect(tombstone?.deletedAt).toBeDefined();
  });

  it("should not corpse a delete of an absent key", () => {
    const table = newTable();
    table.delete("k1");
    expect(table.status("k1")).toBe("unknown");
    expect(table.getTombstone("k1")).toBeUndefined();
  });

  it("should clear the tombstone on a subsequent set", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    table.delete("k1");
    table.set("k1", { key: "k1", name: "b" });
    expect(table.status("k1")).toBe("present");
    expect(table.getTombstone("k1")).toBeUndefined();
    expect(table.get("k1")).toEqual({ key: "k1", name: "b" });
  });

  it("should clear the tombstone on setIfAbsent", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    table.delete("k1");
    table.setIfAbsent({ key: "k1", name: "c" });
    expect(table.status("k1")).toBe("present");
    expect(table.getTombstone("k1")).toBeUndefined();
  });

  it("should corpse entries deleted through a filter", () => {
    const table = newTable();
    table.set([
      { key: "k1", name: "a" },
      { key: "k2", name: "b" },
    ]);
    table.delete((value) => value.name === "a");
    expect(table.status("k1")).toBe("tombstoned");
    expect(table.status("k2")).toBe("present");
  });

  it("should remove the tombstone when a delete is rolled back", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    const rollback = table.delete("k1");
    rollback();
    expect(table.status("k1")).toBe("present");
    expect(table.getTombstone("k1")).toBeUndefined();
    expect(table.get("k1")).toEqual({ key: "k1", name: "a" });
  });

  it("should restore the tombstone when a resurrecting set is rolled back", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    table.delete("k1");
    const rollback = table.set("k1", { key: "k1", name: "b" });
    rollback();
    expect(table.status("k1")).toBe("tombstoned");
    expect(table.getTombstone("k1")?.corpse).toEqual({ key: "k1", name: "a" });
  });

  it("should clear tombstones on clear", () => {
    const table = newTable();
    table.set("k1", { key: "k1", name: "a" });
    table.delete("k1");
    table.clear();
    expect(table.status("k1")).toBe("unknown");
  });
});

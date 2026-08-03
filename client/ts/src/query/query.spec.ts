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

import { NotFoundError } from "@/errors";
import { query } from "@/query";
import { Deleted } from "@/query/deleted";
import { type AnswersHooks, Queries } from "@/query/query";

const wait = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

const expectDeleted = <D extends query.Data>(
  value: query.Cached<D> | undefined,
): Deleted<D> => {
  if (!Deleted.matches<D>(value)) throw new Error("expected a deleted answer");
  return value;
};

interface Rec extends record.Keyed<string> {
  key: string;
  value: number;
}
const rec = (key: string, value: number): Rec => ({ key, value });

type Q = { k: string };
const qA: Q = { k: "a" };
const qB: Q = { k: "b" };

const newTable = () => new query.Table<string, Rec>({ onError: () => {} });

/** Rule-1 space: the query addresses one record by key; answers compose to
 *  that record's value. */
const singleSpace = (
  table: query.Table<string, Rec>,
  fetch: (query: Q) => Promise<string[]>,
  hooks?: AnswersHooks,
) =>
  new Queries<Q, number, string, Rec>(
    {
      name: "thing",
      table,
      fetch,
      compose: (records) => records[0]?.value ?? 0,
      keyOf: (query) => query.k,
      single: true,
    },
    hooks,
  );

describe("hash", () => {
  it("collapses key orderings to the same hash", () => {
    expect(query.hash({ a: 1, b: 2 })).toEqual(query.hash({ b: 2, a: 1 }));
  });

  it("preserves array order", () => {
    expect(query.hash([1, 2, 3])).not.toEqual(query.hash([3, 2, 1]));
  });

  it("hashes nested objects recursively", () => {
    expect(query.hash({ a: { x: 1, y: 2 } })).toEqual(
      query.hash({ a: { y: 2, x: 1 } }),
    );
  });

  it("hashes null and primitives", () => {
    expect(query.hash(null)).toEqual("null");
    expect(query.hash(undefined)).toEqual("undefined");
    expect(query.hash(42)).toEqual("42");
    expect(query.hash("x")).toEqual('"x"');
  });

  it("disambiguates null from undefined at the top level", () => {
    expect(query.hash(null)).not.toEqual(query.hash(undefined));
  });

  it("keeps null fields distinct from absent while folding undefined", () => {
    const nestedNull = query.hash({ a: null });
    const nestedUndef = query.hash({ a: undefined });
    const absent = query.hash({});
    expect(nestedNull).not.toEqual(absent);
    expect(nestedUndef).toEqual(absent);
  });

  it("hashes bigints without throwing and disambiguates from same-valued numbers", () => {
    expect(query.hash(42n)).toEqual("42n");
    expect(query.hash(42n)).not.toEqual(query.hash(42));
    expect(query.hash({ k: 42n })).toEqual('{"k":42n}');
    expect(() => query.hash({ k: 9007199254740993n })).not.toThrow();
  });

  it("delegates to primitive.Hashable.hash() for class instances", () => {
    class TaggedID {
      constructor(private readonly v: string) {}
      hash(): string {
        return `tag:${this.v}`;
      }
    }
    expect(query.hash({ id: new TaggedID("abc") })).toEqual('{"id":tag:abc}');
    expect(query.hash(new TaggedID("xyz"))).toEqual("tag:xyz");
  });

  it("produces stable hashes across instances representing the same value", () => {
    class Wrapper {
      constructor(private readonly v: number) {}
      hash(): string {
        return this.v.toString();
      }
    }
    expect(query.hash({ k: new Wrapper(7) })).toEqual(
      query.hash({ k: new Wrapper(7) }),
    );
  });
});

describe("Answers", () => {
  describe("retrieve", () => {
    it("fetches on the first read and serves the cache while subscribed", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 7));
        return ["a"];
      });
      const answers = singleSpace(table, fetch);
      answers.onChange(qA, vi.fn());
      expect(await answers.retrieve(qA)).toEqual(7);
      expect(await answers.retrieve(qA)).toEqual(7);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("refetches when nothing subscribes to the query", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 7));
        return ["a"];
      });
      const answers = singleSpace(table, fetch);
      expect(await answers.retrieve(qA)).toEqual(7);
      expect(await answers.retrieve(qA)).toEqual(7);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("dedupes concurrent reads of the same query into one fetch", async () => {
      const table = newTable();
      let resolve: (keys: string[]) => void = () => {};
      const fetch = vi.fn(
        () =>
          new Promise<string[]>((r) => {
            resolve = r;
          }),
      );
      const answers = singleSpace(table, fetch);
      const [a, b] = [answers.retrieve(qA), answers.retrieve(qA)];
      table.set("a", rec("a", 3));
      resolve(["a"]);
      expect(await a).toEqual(3);
      expect(await b).toEqual(3);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("fetches separately for distinct queries", async () => {
      const table = newTable();
      const fetch = vi.fn(async ({ k }: Q) => {
        table.set(k, rec(k, k === "a" ? 1 : 2));
        return [k];
      });
      const answers = singleSpace(table, fetch);
      expect(await answers.retrieve(qA)).toEqual(1);
      expect(await answers.retrieve(qB)).toEqual(2);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("treats equivalent queries with key reorderings as the same entry", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 7));
        return ["a"];
      });
      const answers = new Queries<Record<string, number>, number, string, Rec>({
        name: "thing",
        table,
        fetch,
        compose: (records) => records[0]?.value ?? 0,
        keyOf: () => "a",
      });
      answers.onChange({ a: 1, b: 2 }, vi.fn());
      await answers.retrieve({ a: 1, b: 2 });
      await answers.retrieve({ b: 2, a: 1 });
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("treats distinct Hashable instances of the same value as the same entry", async () => {
      class Wrapper {
        constructor(private readonly v: number) {}
        hash(): string {
          return this.v.toString();
        }
      }
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 7));
        return ["a"];
      });
      const answers = new Queries<{ k: Wrapper }, number, string, Rec>({
        name: "thing",
        table,
        fetch,
        compose: (records) => records[0]?.value ?? 0,
        keyOf: () => "a",
      });
      answers.onChange({ k: new Wrapper(42) }, vi.fn());
      await answers.retrieve({ k: new Wrapper(42) });
      await answers.retrieve({ k: new Wrapper(42) });
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("isolates entries across separate instances", async () => {
      const tableA = newTable();
      const tableB = newTable();
      const a = singleSpace(tableA, async () => {
        tableA.set("a", rec("a", 1));
        return ["a"];
      });
      const b = singleSpace(tableB, async () => {
        tableB.set("a", rec("a", 2));
        return ["a"];
      });
      await a.retrieve(qA);
      expect(await b.retrieve(qA)).toEqual(2);
    });

    it("rejects with the fetch error", async () => {
      const answers = singleSpace(newTable(), async () => {
        throw new Error("boom");
      });
      await expect(answers.retrieve(qA)).rejects.toThrow("boom");
    });

    it("retries after a failed fetch instead of caching the error", async () => {
      const table = newTable();
      let calls = 0;
      const answers = singleSpace(table, async () => {
        calls++;
        if (calls === 1) throw new Error("boom");
        table.set("a", rec("a", 9));
        return ["a"];
      });
      await expect(answers.retrieve(qA)).rejects.toThrow("boom");
      expect(await answers.retrieve(qA)).toEqual(9);
    });

    it("retries a failed fetch on the next read while subscribed", async () => {
      const table = newTable();
      let calls = 0;
      const answers = singleSpace(table, async () => {
        calls++;
        if (calls === 1) throw new Error("boom");
        table.set("a", rec("a", 9));
        return ["a"];
      });
      answers.onChange(qA, vi.fn());
      await expect(answers.retrieve(qA)).rejects.toThrow("boom");
      expect(await answers.retrieve(qA)).toEqual(9);
    });

    it("starts streaming without letting it block the fetch", async () => {
      const table = newTable();
      const ensureStreaming = vi.fn(() => new Promise<void>(() => {}));
      const answers = singleSpace(
        table,
        async () => {
          table.set("a", rec("a", 1));
          return ["a"];
        },
        { ensureStreaming },
      );
      expect(await answers.retrieve(qA)).toEqual(1);
      expect(ensureStreaming).toHaveBeenCalledTimes(1);
    });
  });

  describe("getCached", () => {
    it("returns undefined for unfetched, in-flight, and failed queries", async () => {
      const table = newTable();
      let resolve: (keys: string[]) => void = () => {};
      const answers = singleSpace(
        table,
        () =>
          new Promise<string[]>((r) => {
            resolve = r;
          }),
      );
      expect(answers.getCached(qA)).toBeUndefined();
      const promise = answers.retrieve(qA);
      expect(answers.getCached(qA)).toBeUndefined();
      table.set("a", rec("a", 4));
      resolve(["a"]);
      await promise;
      expect(answers.getCached(qA)).toEqual(4);
    });

    it("returns a referentially stable answer between changes", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const off = answers.onChange(qA, () => {});
      await answers.retrieve(qA);
      const first = answers.getCached(qA);
      expect(answers.getCached(qA)).toBe(first);
      off();
    });

    it("resolves an exact-key query straight from the table without an entry", () => {
      const table = newTable();
      table.set("a", rec("a", 9));
      const fetch = vi.fn(async () => ["a"]);
      const answers = singleSpace(table, fetch);
      expect(answers.getCached(qA)).toEqual(9);
      expect(fetch).not.toHaveBeenCalled();
    });

    it("resolves an exact-key query to deleted from a tombstone without an entry", () => {
      const table = newTable();
      table.set("a", rec("a", 9));
      table.delete("a");
      const answers = singleSpace(table, async () => ["a"]);
      expect(expectDeleted(answers.getCached(qA)).corpse).toEqual(9);
    });

    it("produces a new answer object after a change", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      answers.onChange(qA, () => {});
      await answers.retrieve(qA);
      const first = answers.getCached(qA);
      table.set("a", rec("a", 2));
      const second = answers.getCached(qA);
      expect(second).not.toBe(first);
      expect(second).toEqual(2);
    });
  });

  describe("onChange", () => {
    it("fires with the new answer when the tracked row changes", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      table.set("a", rec("a", 2));
      expect(handler).toHaveBeenCalledWith(2);
      expect(answers.getCached(qA)).toEqual(2);
    });

    it("composes setter updates against the previous row", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 10));
        return ["a"];
      });
      answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      table.set("a", (prev) => prev && { ...prev, value: prev.value + 5 });
      expect(answers.getCached(qA)).toEqual(15);
    });

    it("scopes notifications by query", async () => {
      const table = newTable();
      const answers = singleSpace(table, async ({ k }) => {
        table.set(k, rec(k, 1));
        return [k];
      });
      const aHandler = vi.fn();
      const bHandler = vi.fn();
      answers.onChange(qA, aHandler);
      answers.onChange(qB, bHandler);
      table.set("a", rec("a", 2));
      expect(aHandler).toHaveBeenCalledTimes(1);
      expect(bHandler).not.toHaveBeenCalled();
    });

    it("returns a destructor that removes the handler", () => {
      const table = newTable();
      const answers = singleSpace(table, async () => ["a"]);
      const handler = vi.fn();
      const other = vi.fn();
      answers.onChange(qA, other);
      const unsubscribe = answers.onChange(qA, handler);
      unsubscribe();
      table.set("a", rec("a", 2));
      expect(handler).not.toHaveBeenCalled();
      expect(other).toHaveBeenCalledTimes(1);
    });

    it("fires when a fetch settles the answer", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 8));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      expect(handler).toHaveBeenCalledWith(8);
    });

    it("does not fire on fetch failure", async () => {
      const answers = singleSpace(newTable(), async () => {
        throw new Error("boom");
      });
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await expect(answers.retrieve(qA)).rejects.toThrow("boom");
      expect(handler).not.toHaveBeenCalled();
    });

    it("seeds an exact-key answer from a row already in the table", async () => {
      const table = newTable();
      table.set("a", rec("a", 4));
      const fetch = vi.fn(async () => ["a"]);
      const answers = singleSpace(table, fetch);
      const handler = vi.fn();
      answers.onChange(qA, handler);
      expect(handler).toHaveBeenCalledWith(4);
      expect(answers.getCached(qA)).toEqual(4);
      expect(await answers.retrieve(qA)).toEqual(4);
      expect(fetch).not.toHaveBeenCalled();
    });

    it("seeds an exact-key answer as deleted from a tombstoned row", () => {
      const table = newTable();
      table.set("a", rec("a", 4));
      table.delete("a");
      const answers = singleSpace(table, async () => ["a"]);
      const handler = vi.fn();
      answers.onChange(qA, handler);
      expect(expectDeleted(handler.mock.lastCall?.[0]).corpse).toEqual(4);
      expect(expectDeleted(answers.getCached(qA)).corpse).toEqual(4);
    });

    it("does not seed when the table has no row for the key", () => {
      const table = newTable();
      const answers = singleSpace(table, async () => ["a"]);
      const handler = vi.fn();
      answers.onChange(qA, handler);
      expect(handler).not.toHaveBeenCalled();
      expect(answers.getCached(qA)).toBeUndefined();
    });
  });

  describe("maintenance lifecycle", () => {
    it("maintains the answer until the last subscriber leaves", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = singleSpace(table, fetch);
      const offA = answers.onChange(qA, vi.fn());
      const offB = answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      expect(fetch).toHaveBeenCalledTimes(1);
      offA();
      await answers.retrieve(qA);
      expect(fetch).toHaveBeenCalledTimes(1);
      offB();
      await answers.retrieve(qA);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("resumes maintenance after a full unsubscribe and resubscribe", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = singleSpace(table, fetch);
      const off = answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      off();
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      table.set("a", rec("a", 2));
      expect(handler).toHaveBeenCalledWith(2);
    });
  });

  describe("deletion", () => {
    it("delivers the corpse through onChange and getCached", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 6));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      table.delete("a");
      expect(expectDeleted(handler.mock.lastCall?.[0]).corpse).toEqual(6);
      expect(expectDeleted(answers.getCached(qA)).corpse).toEqual(6);
    });

    it("flips back to changed when the record is re-set", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 6));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      table.delete("a");
      table.set("a", rec("a", 7));
      expect(handler).toHaveBeenLastCalledWith(7);
      expect(answers.getCached(qA)).toEqual(7);
    });

    it("invalidates when there is no corpse to retain", () => {
      const table = newTable();
      const answers = singleSpace(table, async () => ["a"]);
      const handler = vi.fn();
      answers.onChange(qA, handler);
      table.delete("a");
      expect(handler).toHaveBeenCalledWith(undefined);
      expect(answers.getCached(qA)).toBeUndefined();
    });

    it("rejects retrieve with NotFoundError for a deleted answer", async () => {
      const table = newTable();
      const answers = singleSpace(table, async () => {
        table.set("a", rec("a", 6));
        return ["a"];
      });
      answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      table.delete("a");
      await expect(answers.retrieve(qA)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  describe("invalidation", () => {
    /** Rule-2 single space: matches admits records with value < 100. */
    const evictingSpace = (
      table: query.Table<string, Rec>,
      fetch: (query: Q) => Promise<string[]>,
    ) =>
      new Queries<Q, number, string, Rec>({
        name: "thing",
        table,
        fetch,
        compose: (records) => records[0]?.value ?? 0,
        matches: (r) => r.value < 100,
        single: true,
      });

    it("invalidates, notifies with undefined, and refetches on next read", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = evictingSpace(table, fetch);
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      // The row stops matching; single-space eviction with a live row
      // invalidates instead of composing an empty answer.
      table.set("a", rec("a", 100));
      expect(handler).toHaveBeenLastCalledWith(undefined);
      expect(answers.getCached(qA)).toBeUndefined();
      await answers.retrieve(qA);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("flips to deleted when the last member's row is tombstoned", async () => {
      const table = newTable();
      const answers = evictingSpace(table, async () => {
        table.set("a", rec("a", 6));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange(qA, handler);
      await answers.retrieve(qA);
      table.delete("a");
      expect(expectDeleted(handler.mock.lastCall?.[0]).corpse).toEqual(6);
    });

    it("does not notify while the entry is unfetched", () => {
      const table = newTable();
      const answers = evictingSpace(table, async () => []);
      const handler = vi.fn();
      answers.onChange(qA, handler);
      table.set("z", rec("z", 1));
      expect(handler).not.toHaveBeenCalled();
    });
  });

  describe("membership (rule 2)", () => {
    type ListQ = { min: number };
    const listSpace = (
      table: query.Table<string, Rec>,
      fetch: (query: ListQ) => Promise<string[]>,
    ) =>
      new Queries<ListQ, Rec[], string, Rec>({
        name: "things",
        table,
        fetch,
        compose: (records) => records,
        matches: (r, q) => r.value >= q.min,
      });

    it("admits a record that starts matching the query", async () => {
      const table = newTable();
      const answers = listSpace(table, async () => {
        table.set("a", rec("a", 5));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange({ min: 3 }, handler);
      await answers.retrieve({ min: 3 });
      table.set("b", rec("b", 4));
      expect(handler).toHaveBeenLastCalledWith([rec("a", 5), rec("b", 4)]);
    });

    it("evicts a record that stops matching the query", async () => {
      const table = newTable();
      const answers = listSpace(table, async () => {
        table.set([rec("a", 5), rec("b", 4)]);
        return ["a", "b"];
      });
      const handler = vi.fn();
      answers.onChange({ min: 3 }, handler);
      await answers.retrieve({ min: 3 });
      table.set("a", rec("a", 1));
      expect(handler).toHaveBeenLastCalledWith([rec("b", 4)]);
    });

    it("notifies when a member's content changes without membership change", async () => {
      const table = newTable();
      const answers = listSpace(table, async () => {
        table.set("a", rec("a", 5));
        return ["a"];
      });
      const handler = vi.fn();
      answers.onChange({ min: 3 }, handler);
      await answers.retrieve({ min: 3 });
      table.set("a", rec("a", 6));
      expect(handler).toHaveBeenLastCalledWith([rec("a", 6)]);
    });

    it("appends admitted members in event order", async () => {
      const table = newTable();
      const answers = listSpace(table, async () => {
        table.set("b", rec("b", 2));
        return ["b"];
      });
      const handler = vi.fn();
      answers.onChange({ min: 0 }, handler);
      await answers.retrieve({ min: 0 });
      table.set("a", rec("a", 3));
      table.set("c", rec("c", 1));
      expect(handler).toHaveBeenLastCalledWith([rec("b", 2), rec("a", 3), rec("c", 1)]);
    });
  });

  describe("server-computed queries (rule 3)", () => {
    type SearchQ = { searchTerm?: string };
    const searchSpace = (
      table: query.Table<string, Rec>,
      fetch: (query: SearchQ) => Promise<string[]>,
    ) =>
      new Queries<SearchQ, Rec[], string, Rec>({
        name: "things",
        table,
        fetch,
        compose: (records) => records,
        matches: () => true,
        serverFields: ["searchTerm"],
      });

    it("refetches wholesale after a debounced table change", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = searchSpace(table, fetch);
      const handler = vi.fn();
      answers.onChange({ searchTerm: "x" }, handler);
      await answers.retrieve({ searchTerm: "x" });
      expect(fetch).toHaveBeenCalledTimes(1);
      table.set("z", rec("z", 9));
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
    });

    // Regression: only the server recomputes membership, so a change that lands
    // while nothing subscribes reaches the answer through nothing else. Serving
    // the pre-gap answer strands it permanently.
    it("refetches an answer that settled before an unmaintained gap", async () => {
      const table = newTable();
      let members: string[] = [];
      const fetch = vi.fn(async () => {
        members.forEach((key) => table.set(key, rec(key, 1)));
        return [...members];
      });
      const answers = searchSpace(table, fetch);
      const unsubscribe = answers.onChange({ searchTerm: "x" }, vi.fn());
      expect(await answers.retrieve({ searchTerm: "x" })).toEqual([]);
      unsubscribe();
      members = ["a"];
      answers.onChange({ searchTerm: "x" }, vi.fn());
      await expect
        .poll(() => answers.getCached({ searchTerm: "x" }))
        .toEqual([rec("a", 1)]);
    });

    it("does not refetch when a subscriber arrives after the fetch", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => []);
      const answers = searchSpace(table, fetch);
      await answers.retrieve({ searchTerm: "x" });
      expect(fetch).toHaveBeenCalledTimes(1);
      answers.onChange({ searchTerm: "x" }, vi.fn());
      await wait(30);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("keeps a settled answer when a refetch fails", async () => {
      const table = newTable();
      let reachable = true;
      const fetch = vi.fn(async () => {
        if (!reachable) throw new Error("unreachable");
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = searchSpace(table, fetch);
      answers.onChange({ searchTerm: "x" }, vi.fn());
      await answers.retrieve({ searchTerm: "x" });
      reachable = false;
      table.set("z", rec("z", 9));
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
      expect(answers.getCached({ searchTerm: "x" })).toEqual([rec("a", 1)]);
    });

    it("coalesces rapid table changes into a single refetch", async () => {
      const table = newTable();
      const fetch = vi.fn(async () => {
        table.set([rec("a", 1)]);
        return ["a"];
      });
      const answers = searchSpace(table, fetch);
      answers.onChange({ searchTerm: "x" }, vi.fn());
      await answers.retrieve({ searchTerm: "x" });
      table.set("z1", rec("z1", 1));
      table.set("z2", rec("z2", 2));
      table.set("z3", rec("z3", 3));
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
      await wait(30);
      expect(fetch).toHaveBeenCalledTimes(2);
    });
  });

  describe("watches", () => {
    interface Rel extends record.Keyed<string> {
      key: string;
    }

    it("rechecks and backfills keys surfaced by a foreign-table projection", async () => {
      const primary = new query.Table<string, Rec>({
        onError: () => {},
        fetch: async (keys) => keys.map((k) => rec(k, 10)),
      });
      const foreign = new query.Table<string, Rel>({ onError: () => {} });
      foreign.set("a", { key: "a" });
      const answers = new Queries<{ tag: string }, Rec[], string, Rec>({
        name: "things",
        table: primary,
        fetch: async () => {
          primary.set("a", rec("a", 1));
          return ["a"];
        },
        compose: (records) => records,
        matches: (r) => foreign.has(r.key),
        watch: [query.watch(foreign, (event) => [event.key])],
      });
      const handler = vi.fn();
      answers.onChange({ tag: "t" }, handler);
      await answers.retrieve({ tag: "t" });
      foreign.set("b", { key: "b" });
      await expect
        .poll(() => {
          const cached = answers.getCached({ tag: "t" });
          return cached != null && !Deleted.matches(cached) ? cached : [];
        })
        .toEqual([rec("a", 1), rec("b", 10)]);
      foreign.delete("a");
      await expect
        .poll(() => {
          const cached = answers.getCached({ tag: "t" });
          return cached != null && !Deleted.matches(cached) ? cached : [];
        })
        .toEqual([rec("b", 10)]);
    });

    it("schedules a wholesale refetch when a watch projects refetch", async () => {
      const table = newTable();
      const foreign = new query.Table<string, Rel>({ onError: () => {} });
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = new Queries<Q, number, string, Rec>({
        name: "thing",
        table,
        fetch,
        compose: (records) => records[0]?.value ?? 0,
        keyOf: (query) => query.k,
        watch: [query.watch(foreign, () => "refetch")],
      });
      answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      foreign.set("x", { key: "x" });
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
    });
  });

  describe("epochs", () => {
    it("refetches maintained answers on an epoch bump", async () => {
      const table = newTable();
      let bump: (epoch: number) => void = () => {};
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = singleSpace(table, fetch, {
        onEpoch: (callback: (epoch: number) => void) => {
          bump = callback;
          return () => {};
        },
      });
      answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      bump(2);
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
    });

    it("does not refetch unmaintained answers on an epoch bump", async () => {
      const table = newTable();
      let bump: (epoch: number) => void = () => {};
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = singleSpace(table, fetch, {
        onEpoch: (callback: (epoch: number) => void) => {
          bump = callback;
          return () => {};
        },
      });
      await answers.retrieve(qA);
      bump(2);
      await wait(10);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("does not refetch maintained answers when the epoch returns to 0", async () => {
      const table = newTable();
      let bump: (epoch: number) => void = () => {};
      const fetch = vi.fn(async () => {
        table.set("a", rec("a", 1));
        return ["a"];
      });
      const answers = singleSpace(table, fetch, {
        onEpoch: (callback: (epoch: number) => void) => {
          bump = callback;
          return () => {};
        },
      });
      answers.onChange(qA, vi.fn());
      await answers.retrieve(qA);
      bump(0);
      await wait(10);
      expect(fetch).toHaveBeenCalledTimes(1);
    });
  });

  describe("identity-gated settle", () => {
    it("ignores a late resolution that follows a maintenance update", async () => {
      const table = newTable();
      let resolve: (keys: string[]) => void = () => {};
      const answers = singleSpace(
        table,
        () =>
          new Promise<string[]>((r) => {
            resolve = r;
          }),
      );
      answers.onChange(qA, vi.fn());
      const promise = answers.retrieve(qA);
      table.set("a", rec("a", 99));
      resolve(["a"]);
      await promise;
      expect(answers.getCached(qA)).toEqual(99);
    });
  });
});

describe("referential stability", () => {
  interface Doc extends record.Keyed<string> {
    key: string;
    name: string;
  }
  const doc = (key: string, name: string): Doc => ({ key, name });
  const newDocTable = () => new query.Table<string, Doc>({ onError: () => {} });
  /** Identity-compose single space: the answer is the table row itself. */
  const identitySpace = (
    table: query.Table<string, Doc>,
    fetch: (query: Q) => Promise<string[]>,
  ) =>
    new Queries<Q, Doc, string, Doc>({
      name: "doc",
      table,
      fetch,
      compose: (records) => records[0],
      keyOf: (query) => query.k,
      single: true,
    });

  it("returns the table row itself for identity composes", () => {
    const table = newDocTable();
    const row = doc("a", "one");
    table.set("a", row);
    const answers = identitySpace(table, async () => ["a"]);
    expect(answers.getCached(qA)).toBe(row);
  });

  it("returns the same reference across repeated entry-less reads", () => {
    const table = newDocTable();
    table.set("a", doc("a", "one"));
    const answers = identitySpace(table, async () => ["a"]);
    expect(answers.getCached(qA)).toBe(answers.getCached(qA));
  });

  it("returns a new row reference only after the row changes", async () => {
    const table = newDocTable();
    const first = doc("a", "one");
    const answers = identitySpace(table, async () => {
      table.set("a", first);
      return ["a"];
    });
    answers.onChange(qA, vi.fn());
    await answers.retrieve(qA);
    expect(answers.getCached(qA)).toBe(first);
    const second = doc("a", "two");
    table.set("a", second);
    expect(answers.getCached(qA)).toBe(second);
  });

  it("returns the same Deleted instance across repeated reads", () => {
    const table = newDocTable();
    const row = doc("a", "one");
    table.set("a", row);
    table.delete("a");
    const answers = identitySpace(table, async () => ["a"]);
    const first = expectDeleted(answers.getCached(qA));
    expect(answers.getCached(qA)).toBe(first);
    expect(first.corpse).toBe(row);
  });

  it("interns composed corpses so non-identity deleted answers are stable", () => {
    const table = newTable();
    table.set("a", rec("a", 9));
    table.delete("a");
    const answers = singleSpace(table, async () => ["a"]);
    const first = expectDeleted(answers.getCached(qA));
    expect(answers.getCached(qA)).toBe(first);
    expect(first.corpse).toEqual(9);
  });

  it("delivers bare rows and interned Deleted instances through onChange", async () => {
    const table = newDocTable();
    const answers = identitySpace(table, async () => {
      table.set("a", doc("a", "one"));
      return ["a"];
    });
    const handler = vi.fn();
    answers.onChange(qA, handler);
    await answers.retrieve(qA);
    const next = doc("a", "two");
    table.set("a", next);
    expect(handler.mock.lastCall?.[0]).toBe(next);
    table.delete("a");
    expect(handler.mock.lastCall?.[0]).toBe(answers.getCached(qA));
    expect(expectDeleted(handler.mock.lastCall?.[0]).corpse).toBe(next);
  });

  it("brands deleted answers so matches rejects live data", () => {
    const table = newDocTable();
    const row = doc("a", "one");
    table.set("a", row);
    const answers = identitySpace(table, async () => ["a"]);
    expect(Deleted.matches(answers.getCached(qA))).toBe(false);
    table.delete("a");
    expect(Deleted.matches(answers.getCached(qA))).toBe(true);
  });
});

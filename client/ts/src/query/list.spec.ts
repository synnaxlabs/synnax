// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { ValidationError } from "@/errors";
import { query } from "@/query";
import { Lists, MAX_SEARCH_LIMIT } from "@/query/list";
import { DEFAULT_DEBOUNCE, type SpaceHooks } from "@/query/query";

interface Rec {
  key: string;
  name: string;
  value: number;
}
const rec = (key: string, name: string, value = 1): Rec => ({ key, name, value });

type Order = { field: "name"; descending?: boolean };
type Cursor = { key: string; name?: string };
type Q = {
  min?: number;
  limit?: number;
  order?: Order;
  after?: Cursor;
  searchTerm?: string;
};

const cursor = (r: Rec, order?: Order): Cursor =>
  order == null ? { key: r.key } : { key: r.key, name: r.name };

const compareStrings = (a: string, b: string): number => (a < b ? -1 : a > b ? 1 : 0);

const compareCursors = (a: Cursor, b: Cursor, order?: Order): number => {
  if (order == null) return compareStrings(a.key, b.key);
  const c = compareStrings(a.name ?? "", b.name ?? "") || compareStrings(a.key, b.key);
  return order.descending === true ? -c : c;
};

const matches = (r: Rec, q: Q): boolean => r.value >= (q.min ?? 0);

/** Pages the records it holds the way the Core does. */
const createCore = (records: Rec[]) => {
  const core = { records };
  const fetch = vi.fn(async (q: Q): Promise<Rec[]> => {
    if (q.searchTerm != null)
      return core.records
        .filter((r) => r.name.includes(q.searchTerm as string))
        .slice(0, q.limit);
    const sorted = core.records
      .filter((r) => matches(r, q))
      .sort((a, b) => compareCursors(cursor(a, q.order), cursor(b, q.order), q.order));
    const { after } = q;
    const past =
      after == null
        ? sorted
        : sorted.filter((r) => compareCursors(cursor(r, q.order), after, q.order) > 0);
    return past.slice(0, q.limit);
  });
  return { core, fetch };
};

const byName: Order = { field: "name" };

const setup = (records: Rec[], hooks: SpaceHooks = {}) => {
  const table = new query.Table<string, Rec>({ onError: () => {} });
  const { core, fetch } = createCore(records);
  const lists = new Lists<Q, Rec, string, Rec, Cursor, Order>(
    {
      name: "thing",
      table,
      fetch,
      compose: (r) => r,
      matches,
      cursor,
      compare: compareCursors,
    },
    { teardownGrace: TimeSpan.ZERO, ...hooks },
  );
  return { table, core, fetch, lists };
};

const names = (window?: query.Window<Rec, Cursor>) =>
  window?.records.map((r) => r.name);

describe("Lists", () => {
  describe("retrieve", () => {
    it("should fetch the first page in order and end at its last record", async () => {
      const { lists, fetch } = setup([rec("1", "c"), rec("2", "a"), rec("3", "b")]);
      const window = await lists.list({ order: byName, limit: 2 }).retrieve();
      expect(names(window)).toEqual(["a", "b"]);
      expect(window.end).toEqual({ key: "3", name: "b" });
      expect(fetch).toHaveBeenCalledWith({ order: byName, limit: 2 }, undefined);
    });

    it("should end at null when the page holds every match", async () => {
      const { lists } = setup([rec("1", "c"), rec("2", "a")]);
      const window = await lists.list({ order: byName, limit: 5 }).retrieve();
      expect(names(window)).toEqual(["a", "c"]);
      expect(window.end).toBeNull();
    });

    it("should fetch every match when the query sets no limit", async () => {
      const { lists, fetch } = setup([rec("1", "c"), rec("2", "a")]);
      const window = await lists.list({ order: byName }).retrieve();
      expect(window.end).toBeNull();
      expect(fetch).toHaveBeenCalledWith(
        { order: byName, limit: undefined },
        undefined,
      );
    });

    it("should reject a limit that is not positive", () => {
      const { lists } = setup([]);
      expect(() => lists.list({ order: byName, limit: 0 })).toThrow(
        new ValidationError("a thing list needs a positive limit"),
      );
    });

    it("should page in key order when the query sets no order", async () => {
      const { lists } = setup([rec("2", "a"), rec("1", "b"), rec("3", "c")]);
      const window = await lists.list({ limit: 2 }).retrieve();
      expect(window.records.map((r) => r.key)).toEqual(["1", "2"]);
      expect(window.end).toEqual({ key: "2" });
    });

    it("should share one window between limits of one query", async () => {
      const { lists, fetch } = setup([rec("1", "a"), rec("2", "b"), rec("3", "c")]);
      const small = lists.list({ order: byName, limit: 1 });
      small.onChange(vi.fn());
      await small.retrieve();
      const window = await lists.list({ order: byName, limit: 3 }).retrieve();
      expect(names(window)).toEqual(["a", "b", "c"]);
      expect(fetch).toHaveBeenLastCalledWith(
        { order: byName, after: { key: "1", name: "a" }, limit: 2 },
        undefined,
      );
      expect(names(small.getCached())).toEqual(["a", "b", "c"]);
    });

    it("should serve a maintained window without fetching", async () => {
      const { lists, fetch } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName, limit: 1 });
      list.onChange(vi.fn());
      await list.retrieve();
      await list.retrieve();
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should apply a change between its subscribe and its first fetch", async () => {
      const { lists, table } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName });
      list.onChange(vi.fn());
      table.set("2", rec("2", "b"));
      expect(names(await list.retrieve())).toEqual(["a", "b"]);
    });

    it("should fetch again on every read of an unmaintained window", async () => {
      const { lists, fetch } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName, limit: 1 });
      await list.retrieve();
      await list.retrieve();
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should refetch the windows the predicate selects", async () => {
      const { lists, fetch } = setup([rec("a", "a")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      await list.retrieve();
      const before = fetch.mock.calls.length;
      lists.refetchMatching(() => false);
      lists.refetchMatching((q) => q.order?.field === "name");
      await expect.poll(() => fetch.mock.calls.length).toBe(before + 1);
      release();
    });

    it("should return the same list for equal queries", () => {
      const { lists } = setup([]);
      const list = lists.list({ order: byName, limit: 2 });
      expect(lists.list({ limit: 2, order: { field: "name" } })).toBe(list);
      expect(lists.list({ order: byName, limit: 3 })).not.toBe(list);
    });

    it("should return undefined from getCached before a fetch", () => {
      const { lists } = setup([rec("1", "a")]);
      expect(lists.list({ order: byName }).getCached()).toBeUndefined();
    });
  });

  describe("more", () => {
    it("should fetch past the end and add the page", async () => {
      const { lists, fetch } = setup([rec("1", "a"), rec("2", "b"), rec("3", "c")]);
      const list = lists.list({ order: byName, limit: 2 });
      list.onChange(vi.fn());
      await list.retrieve();
      const window = await list.more(2);
      expect(names(window)).toEqual(["a", "b", "c"]);
      expect(window.end).toBeNull();
      expect(fetch).toHaveBeenLastCalledWith(
        { order: byName, after: { key: "2", name: "b" }, limit: 2 },
        undefined,
      );
    });

    it("should fetch nothing when the window holds every match", async () => {
      const { lists, fetch } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName, limit: 2 });
      list.onChange(vi.fn());
      await list.retrieve();
      await list.more(2);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should fetch an unmaintained window again from its start", async () => {
      const { lists, core, table, fetch } = setup([
        rec("1", "a"),
        rec("2", "b"),
        rec("3", "c"),
        rec("4", "d"),
      ]);
      const list = lists.list({ order: byName, limit: 3 });
      await list.retrieve();
      core.records = core.records.filter(({ key }) => key !== "2");
      table.delete("2");
      expect(names(await list.more(1))).toEqual(["a", "c", "d"]);
      expect(fetch).toHaveBeenLastCalledWith({ order: byName, limit: 4 }, undefined);
    });

    it("should fetch nothing for a count that is not positive", async () => {
      const { lists, fetch } = setup([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName, limit: 1 });
      list.onChange(vi.fn());
      await list.retrieve();
      expect(names(await list.more(0))).toEqual(["a"]);
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should keep ties at the page edge", async () => {
      const { lists } = setup([rec("1", "a"), rec("2", "a"), rec("3", "a")]);
      const list = lists.list({ order: byName, limit: 2 });
      await list.retrieve();
      const window = await list.more(2);
      expect(window.records.map((r) => r.key)).toEqual(["1", "2", "3"]);
    });

    it("should apply a change that raced the page", async () => {
      const { lists, table, fetch } = setup([
        rec("1", "a"),
        rec("2", "b"),
        rec("3", "d"),
      ]);
      const list = lists.list({ order: byName, limit: 2 });
      list.onChange(vi.fn());
      await list.retrieve();
      const page = Promise.withResolvers<Rec[]>();
      fetch.mockImplementationOnce(async () => await page.promise);
      const more = list.more(2);
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
      table.set("4", rec("4", "c"));
      page.resolve([rec("3", "d")]);
      expect(names(await more)).toEqual(["a", "b", "c", "d"]);
    });
  });

  describe("changes", () => {
    const maintained = async (records: Rec[], q: Q = { order: byName, limit: 2 }) => {
      const s = setup(records);
      const handler = vi.fn();
      const list = s.lists.list(q);
      list.onChange(handler);
      await list.retrieve();
      return { ...s, list, handler };
    };

    it("should place a match that sorts before the end", async () => {
      const { table, list, handler } = await maintained([
        rec("1", "a"),
        rec("2", "c"),
        rec("3", "d"),
      ]);
      table.set("4", rec("4", "b"));
      expect(names(list.getCached())).toEqual(["a", "b", "c"]);
      expect(names(handler.mock.lastCall?.[0])).toEqual(["a", "b", "c"]);
    });

    it("should leave out a match that sorts past the end", async () => {
      const { table, list, handler } = await maintained([
        rec("1", "a"),
        rec("2", "c"),
        rec("3", "d"),
      ]);
      handler.mockClear();
      table.set("4", rec("4", "e"));
      expect(names(list.getCached())).toEqual(["a", "c"]);
      expect(handler).not.toHaveBeenCalled();
    });

    it("should admit every match when the window ends at null", async () => {
      const { table, list } = await maintained([rec("1", "a")]);
      table.set("4", rec("4", "z"));
      expect(names(list.getCached())).toEqual(["a", "z"]);
    });

    it("should move a renamed member to its new place", async () => {
      const { table, list } = await maintained([rec("1", "a"), rec("2", "b")], {
        order: byName,
      });
      table.set("1", rec("1", "c"));
      expect(names(list.getCached())).toEqual(["b", "c"]);
    });

    it("should move a member renamed past the end out of the window", async () => {
      const { table, list } = await maintained([
        rec("1", "a"),
        rec("2", "b"),
        rec("3", "c"),
      ]);
      table.set("1", rec("1", "z"));
      expect(names(list.getCached())).toEqual(["b"]);
    });

    it("should remove a member that stops matching", async () => {
      const { table, list } = await maintained([rec("1", "a"), rec("2", "b")], {
        order: byName,
        min: 1,
      });
      table.set("1", rec("1", "a", 0));
      expect(names(list.getCached())).toEqual(["b"]);
    });

    it("should remove a deleted member and keep the end", async () => {
      const { table, list } = await maintained([
        rec("1", "a"),
        rec("2", "b"),
        rec("3", "c"),
      ]);
      table.delete("2");
      const window = list.getCached();
      expect(names(window)).toEqual(["a"]);
      expect(window?.end).toEqual({ key: "2", name: "b" });
    });

    it("should place several changes of one batch", async () => {
      const { table, list } = await maintained([rec("1", "b"), rec("2", "d")], {
        order: byName,
      });
      table.set([rec("1", "e"), rec("3", "a"), rec("2", "c")]);
      expect(names(list.getCached())).toEqual(["a", "c", "e"]);
    });

    it("should order a descending list from its largest value", async () => {
      const { table, list } = await maintained([rec("1", "a"), rec("2", "c")], {
        order: { field: "name", descending: true },
      });
      table.set("3", rec("3", "b"));
      expect(names(list.getCached())).toEqual(["c", "b", "a"]);
    });
  });

  describe("search", () => {
    const searching = async (records: Rec[], limit = 2) => {
      const s = setup(records);
      const handler = vi.fn();
      const list = s.lists.list({ searchTerm: "a", limit });
      list.onChange(handler);
      await list.retrieve();
      return { ...s, list, handler };
    };

    it("should keep the order of the Core", async () => {
      const { list } = await searching([rec("2", "ab"), rec("1", "ac")]);
      expect(names(list.getCached())).toEqual(["ab", "ac"]);
    });

    it("should fetch again on a change in the table", async () => {
      const { table, list, fetch, core } = await searching([rec("1", "ab")]);
      core.records.push(rec("2", "ac"));
      table.set("2", rec("2", "ac"));
      await expect.poll(() => names(list.getCached())).toEqual(["ab", "ac"]);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should not fetch again when another list finds new records", async () => {
      const { lists, list, fetch } = await searching([rec("1", "ab"), rec("2", "c")]);
      await lists.list({ order: byName }).retrieve();
      await sleep.sleep(TimeSpan.milliseconds(DEFAULT_DEBOUNCE.milliseconds * 2));
      expect(fetch).toHaveBeenCalledTimes(2);
      expect(names(list.getCached())).toEqual(["ab"]);
    });

    it("should fetch again when another list fetches a changed record", async () => {
      const { lists, list, fetch, core } = await searching([rec("1", "ab")]);
      core.records = [rec("1", "ab", 5), rec("2", "ac")];
      await lists.list({ order: byName }).retrieve();
      await expect.poll(() => names(list.getCached())).toEqual(["ab", "ac"]);
      expect(fetch).toHaveBeenCalledTimes(3);
    });

    it("should fetch again when the stream sets a record a fetch found", async () => {
      const { lists, list, table, fetch, core } = await searching([rec("1", "ab")]);
      core.records.push(rec("2", "ac"));
      await lists.list({ order: byName }).retrieve();
      table.mirrorSet("2", rec("2", "ac"));
      await expect.poll(() => names(list.getCached())).toEqual(["ab", "ac"]);
      await sleep.sleep(TimeSpan.milliseconds(DEFAULT_DEBOUNCE.milliseconds * 2));
      expect(fetch).toHaveBeenCalledTimes(3);
    });

    it("should show a change to a member at once", async () => {
      const { table, handler } = await searching([rec("1", "ab")]);
      table.set("1", rec("1", "ab", 5));
      expect(handler.mock.lastCall?.[0].records).toEqual([rec("1", "ab", 5)]);
    });

    it("should double its limit to grow", async () => {
      const records = Array.from({ length: 10 }, (_, i) => rec(`${i}`, `a${i}`));
      const { list, fetch } = await searching(records);
      const window = await list.more(1);
      expect(window.records).toHaveLength(4);
      expect(fetch).toHaveBeenLastCalledWith({ searchTerm: "a", limit: 4 }, undefined);
    });

    it("should stop growing at the most records a search returns", async () => {
      const records = Array.from({ length: MAX_SEARCH_LIMIT + 1 }, (_, i) =>
        rec(`${i}`, `a${i}`),
      );
      const { list, fetch } = await searching(records, MAX_SEARCH_LIMIT);
      await list.more(10);
      expect(fetch).toHaveBeenCalledTimes(1);
      expect(list.getCached()?.end).not.toBeNull();
    });

    it("should fetch again on a change during its first fetch", async () => {
      const { lists, table, fetch, core } = setup([rec("1", "ab")]);
      const list = lists.list({ searchTerm: "a", limit: 2 });
      list.onChange(vi.fn());
      fetch.mockImplementationOnce(async () => {
        core.records.push(rec("2", "ac"));
        table.set("2", rec("2", "ac"));
        return [rec("1", "ab")];
      });
      await list.retrieve();
      await expect.poll(() => names(list.getCached())).toEqual(["ab", "ac"]);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should treat an empty search term as no search", async () => {
      const { lists, fetch } = setup([rec("1", "b"), rec("2", "a")]);
      const window = await lists.list({ searchTerm: "", order: byName }).retrieve();
      expect(names(window)).toEqual(["a", "b"]);
      expect(fetch).toHaveBeenCalledWith(
        { order: byName, limit: undefined },
        undefined,
      );
    });
  });

  describe("watched tables", () => {
    const setupWatched = (records: Rec[], contentOnly = false) => {
      const statuses = new query.Table<string, { key: string; variant: string }>({
        onError: () => {},
      });
      const table = new query.Table<string, Rec>({ onError: () => {} });
      const { fetch } = createCore(records);
      const lists = new Lists<
        Q,
        Rec & { variant?: string },
        string,
        Rec,
        Cursor,
        Order
      >(
        {
          name: "thing",
          table,
          fetch,
          compose: (r) => ({ ...r, variant: statuses.get(r.key)?.variant }),
          // A pending status leaves the match undecided.
          matches: (r, q) =>
            statuses.get(r.key)?.variant === "pending" ? null : matches(r, q),
          cursor,
          compare: compareCursors,
          triggers: [query.trigger(statuses, ({ key }) => [key], { contentOnly })],
        },
        { teardownGrace: TimeSpan.ZERO },
      );
      return { statuses, table, fetch, lists };
    };

    it("should compose a member again when a watched table changes it", async () => {
      const { statuses, lists } = setupWatched([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName });
      const handler = vi.fn();
      const release = list.onChange(handler);
      await list.retrieve();
      handler.mockClear();
      statuses.set("3", { key: "3", variant: "error" });
      expect(handler).not.toHaveBeenCalled();
      statuses.set("1", { key: "1", variant: "error" });
      expect(list.getCached()?.records.map((r) => r.variant)).toEqual([
        "error",
        undefined,
      ]);
      expect(handler).toHaveBeenCalledTimes(1);
      release();
    });

    it("should keep the composed records of members that a watched change leaves", async () => {
      const { statuses, lists } = setupWatched([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      const [a, b] = (await list.retrieve()).records;
      statuses.set("1", { key: "1", variant: "error" });
      const [nextA, nextB] = list.getCached()?.records ?? [];
      expect(nextA).not.toBe(a);
      expect(nextA.variant).toEqual("error");
      expect(nextB).toBe(b);
      release();
    });

    it("should keep the composed records of members that a table change leaves", async () => {
      const { table, lists } = setupWatched([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      const [a, b] = (await list.retrieve()).records;
      table.set("1", rec("1", "a", 2));
      const [nextA, nextB] = list.getCached()?.records ?? [];
      expect(nextA).not.toBe(a);
      expect(nextA.value).toEqual(2);
      expect(nextB).toBe(b);
      release();
    });

    it("should compose a member again that an undecided match drops and a refetch admits", async () => {
      const { statuses, fetch, lists } = setupWatched([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      await list.retrieve();
      statuses.set("1", { key: "1", variant: "pending" });
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
      expect(list.getCached()?.records.map((r) => r.variant)).toEqual([
        "pending",
        undefined,
      ]);
      release();
    });

    it("should compose every member again after a watched change while unwatched", async () => {
      const { statuses, table, lists } = setupWatched([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      await list.retrieve();
      release();
      statuses.set("1", { key: "1", variant: "error" });
      table.set("2", rec("2", "b", 2));
      expect(list.getCached()?.records.map((r) => [r.variant, r.value])).toEqual([
        ["error", 1],
        [undefined, 2],
      ]);
    });

    it("should compose a member again after a watched change while unwatched", async () => {
      const { statuses, lists } = setupWatched([rec("1", "a")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      await list.retrieve();
      release();
      statuses.set("1", { key: "1", variant: "error" });
      expect(list.getCached()?.records[0].variant).toEqual("error");
    });

    it("should fetch a search again when a watched change can change its matches", async () => {
      const { statuses, fetch, lists } = setupWatched([rec("1", "ab")]);
      const list = lists.list({ searchTerm: "a" });
      list.onChange(vi.fn());
      await list.retrieve();
      statuses.set("1", { key: "1", variant: "error" });
      await list.retrieve();
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should compose a search again without a fetch on a content-only change", async () => {
      const { statuses, fetch, lists } = setupWatched([rec("1", "ab")], true);
      const list = lists.list({ searchTerm: "a" });
      list.onChange(vi.fn());
      await list.retrieve();
      statuses.set("1", { key: "1", variant: "error" });
      expect(list.getCached()?.records[0].variant).toEqual("error");
      await list.retrieve();
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should fetch a window again when a watched table changed while unwatched", async () => {
      const { statuses, fetch, lists } = setupWatched([rec("1", "a")]);
      const list = lists.list({ order: byName });
      const release = list.onChange(() => {});
      await list.retrieve();
      release();
      statuses.set("1", { key: "1", variant: "error" });
      const before = fetch.mock.calls.length;
      const again = list.onChange(() => {});
      await expect.poll(() => fetch.mock.calls.length).toBe(before + 1);
      expect(list.getCached()?.records[0].variant).toEqual("error");
      again();
    });
  });

  describe("maintenance", () => {
    it("should refetch a maintained window at its size on a new epoch", async () => {
      let bump = (_: number): void => {};
      const { lists, fetch } = setup([rec("1", "a"), rec("2", "b"), rec("3", "c")], {
        onEpoch: (callback) => {
          bump = callback;
          return () => {};
        },
      });
      const list = lists.list({ order: byName, limit: 1 });
      list.onChange(vi.fn());
      await list.retrieve();
      await list.more(1);
      bump(2);
      await expect.poll(() => fetch.mock.calls.length).toBe(3);
      expect(fetch).toHaveBeenLastCalledWith({ order: byName, limit: 2 }, undefined);
    });

    it("should size a new epoch's refetch of an unread window to its subscriber", async () => {
      let bump = (_: number): void => {};
      const { lists, fetch } = setup([rec("1", "a"), rec("2", "b"), rec("3", "c")], {
        onEpoch: (callback) => {
          bump = callback;
          return () => {};
        },
      });
      const list = lists.list({ order: byName, limit: 2 });
      list.onChange(vi.fn());
      bump(2);
      await expect.poll(() => names(list.getCached())).toEqual(["a", "b"]);
      expect(fetch).toHaveBeenCalledExactlyOnceWith(
        { order: byName, limit: 2 },
        undefined,
      );
    });

    it("should refetch on resubscribe after a change it missed", async () => {
      const { lists, table, fetch, core } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName });
      list.onChange(vi.fn())();
      await list.retrieve();
      core.records.push(rec("2", "b"));
      table.set("2", rec("2", "b"));
      list.onChange(vi.fn());
      await expect.poll(() => names(list.getCached())).toEqual(["a", "b"]);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should not fetch again on subscribe after an unmaintained fetch", async () => {
      const { lists, fetch } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName });
      await list.retrieve();
      list.onChange(vi.fn());
      await list.retrieve();
      expect(fetch).toHaveBeenCalledTimes(1);
    });

    it("should refetch on subscribe after a change during an unmaintained fetch", async () => {
      const { lists, table, fetch, core } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName });
      fetch.mockImplementationOnce(async () => {
        core.records.push(rec("2", "b"));
        table.set("2", rec("2", "b"));
        return [rec("1", "a")];
      });
      await list.retrieve();
      list.onChange(vi.fn());
      await expect.poll(() => names(list.getCached())).toEqual(["a", "b"]);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it("should drop a record deleted while unmaintained on resubscribe", async () => {
      const { lists, table, core } = setup([rec("1", "a"), rec("2", "b")]);
      const list = lists.list({ order: byName });
      list.onChange(vi.fn())();
      await list.retrieve();
      core.records = core.records.filter(({ key }) => key !== "2");
      table.delete("2");
      list.onChange(vi.fn());
      core.records.push(rec("3", "c"));
      table.set("3", rec("3", "c"));
      expect(names(list.getCached())).toEqual(["a", "c"]);
    });

    it("should release the records it holds on close", async () => {
      const { lists, table } = setup([rec("1", "a")]);
      const list = lists.list({ order: byName });
      list.onChange(vi.fn());
      await list.retrieve();
      table.trim(0);
      expect(table.trim(0)).toEqual([]);
      lists.close();
      expect(table.trim(0)).toEqual(["1"]);
    });

    it("should discard the window on reset and fetch a read in flight again", async () => {
      const { lists, fetch } = setup([rec("1", "a"), rec("2", "b")]);
      const handler = vi.fn();
      const list = lists.list({ order: byName, limit: 1 });
      list.onChange(handler);
      await list.retrieve();
      const page = Promise.withResolvers<Rec[]>();
      fetch.mockImplementationOnce(async () => await page.promise);
      const more = list.more(1);
      await expect.poll(() => fetch.mock.calls.length).toBe(2);
      lists.reset();
      expect(handler).toHaveBeenLastCalledWith(undefined);
      expect(list.getCached()).toBeUndefined();
      page.resolve([rec("2", "b")]);
      expect(names(await more)).toEqual(["a"]);
      expect(fetch).toHaveBeenLastCalledWith({ order: byName, limit: 1 }, undefined);
    });

    it("should report a handler that throws", async () => {
      const onError = vi.fn();
      const { lists, table } = setup([rec("1", "a")], { onError });
      const list = lists.list({ order: byName });
      list.onChange(() => {
        throw new Error("boom");
      });
      await list.retrieve();
      table.set("2", rec("2", "b"));
      expect(onError).toHaveBeenCalledWith(
        new Error("failed to notify thing list subscriber"),
      );
    });
  });
});

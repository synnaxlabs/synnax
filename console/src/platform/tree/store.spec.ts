// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { describe, expect, it, vi } from "vitest";

import { type Source, type Window } from "@/platform/tree/children";
import {
  isMoreKey,
  type Results,
  Store,
  type StoreConfig,
} from "@/platform/tree/store";
import { type Resource } from "@/platform/tree/types";

const ROOT: resource.ID = { type: "builtin", key: "root" };
const ROOT_KEY = resource.idToString(ROOT);

const res = (type: resource.Type, key: string, name: string): Resource => ({
  key: resource.idToString({ type, key }),
  id: { type, key },
  name,
});

interface MemorySource extends Source {
  /** Changes the window, as a change in the cache does. */
  push: (window: Window | undefined) => void;
  subscribers: () => number;
}

// Answers a fetch with the window and each call for more with the next page, and holds
// nothing before the first fetch.
const createMemorySource = (fetched: Window, pages: Window[] = []): MemorySource => {
  let cached: Window | undefined;
  const handlers = new Set<(window: Window | undefined) => void>();
  const push = (window: Window | undefined): void => {
    cached = window;
    handlers.forEach((handler) => handler(window));
  };
  return {
    retrieve: async () => {
      cached ??= fetched;
      return cached;
    },
    more: async () => {
      const page = pages.shift();
      if (page != null) push(page);
    },
    retrieveAll: async () => await Promise.reject(new Error("the store pages lists")),
    getCached: () => cached,
    onChange: (handler) => {
      handlers.add(handler);
      return () => handlers.delete(handler);
    },
    push,
    subscribers: () => handlers.size,
  };
};

const listed = (...resources: Resource[]): Window => ({ resources, more: false });

const createStore = (
  sources: Record<string, Source[]>,
  config: Partial<StoreConfig> = {},
): Store =>
  new Store({
    root: ROOT,
    list: (id) => sources[resource.idToString(id)] ?? [],
    hasChildren: (type) => type === "group",
    isFolder: (type) => type === "group",
    onError: vi.fn(),
    ...config,
  });

const keysOf = (store: Store, key = ROOT_KEY): string[] | undefined =>
  store.getChildren(key);

// Names the show more node, so specs need not know how its key is built.
const MORE = "more";

const markMore = (keys: string[] | undefined): string[] | undefined =>
  keys?.map((key) => (isMoreKey(key) ? MORE : key));

const findMore = async (store: Store): Promise<string> => {
  await expect.poll(() => keysOf(store)?.some(isMoreKey)).toBe(true);
  const key = keysOf(store)?.find(isMoreKey);
  if (key == null) throw new Error("no show more node");
  return key;
};

const GROUP_A = res("group", "a", "a");
const GROUP_B = res("group", "b", "B");

describe("tree store", () => {
  describe("lists", () => {
    it("should show the folders of a node first, then its other children, by name", async () => {
      const store = createStore({
        [ROOT_KEY]: [
          createMemorySource(
            listed(res("channel", "1", "beta"), res("channel", "2", "Alpha")),
          ),
          createMemorySource(listed(GROUP_B, GROUP_A)),
        ],
      });
      store.start();
      await expect
        .poll(() => keysOf(store))
        .toEqual(["group:a", "group:b", "channel:2", "channel:1"]);
    });

    it("should hold back the records past the last record of a list with more", async () => {
      const store = createStore({
        [ROOT_KEY]: [
          createMemorySource({
            resources: [res("channel", "1", "a"), res("channel", "3", "c")],
            more: true,
          }),
          createMemorySource(listed(res("device", "2", "b"), res("device", "4", "d"))),
        ],
      });
      store.start();
      await expect
        .poll(() => markMore(keysOf(store)))
        .toEqual(["channel:1", "device:2", "channel:3", MORE]);
    });

    it("should hide the other records while a folder list has more", async () => {
      const store = createStore({
        [ROOT_KEY]: [
          createMemorySource({ resources: [GROUP_A], more: true }),
          createMemorySource(listed(res("channel", "1", "a"))),
        ],
      });
      store.start();
      await expect.poll(() => markMore(keysOf(store))).toEqual(["group:a", MORE]);
    });

    it("should keep the show more node when deletes empty a list with more", async () => {
      const store = createStore({
        [ROOT_KEY]: [
          createMemorySource({ resources: [], more: true }),
          createMemorySource(listed(res("channel", "1", "a"))),
        ],
      });
      store.start();
      await expect.poll(() => markMore(keysOf(store))).toEqual([MORE]);
    });

    it("should fetch the next page of only the folder lists while one has more", async () => {
      const folders = createMemorySource({ resources: [GROUP_A], more: true });
      const records = createMemorySource({
        resources: [res("channel", "1", "a")],
        more: true,
      });
      const foldersMore = vi.spyOn(folders, "more");
      const recordsMore = vi.spyOn(records, "more");
      const store = createStore({ [ROOT_KEY]: [folders, records] });
      store.start();
      store.more(await findMore(store));
      await expect.poll(() => foldersMore.mock.calls.length).toBe(1);
      expect(recordsMore).not.toHaveBeenCalled();
    });

    it("should fetch the next page of only the lists that hold back more", async () => {
      const paged = createMemorySource(
        { resources: [res("channel", "1", "a"), res("channel", "3", "c")], more: true },
        [
          {
            resources: [
              res("channel", "1", "a"),
              res("channel", "3", "c"),
              res("channel", "5", "e"),
            ],
            more: false,
          },
        ],
      );
      const whole = createMemorySource(
        listed(res("device", "2", "b"), res("device", "4", "d")),
      );
      const wholeMore = vi.spyOn(whole, "more");
      const store = createStore({ [ROOT_KEY]: [paged, whole] });
      store.start();
      store.more(await findMore(store));
      await expect
        .poll(() => keysOf(store))
        .toEqual(["channel:1", "device:2", "channel:3", "device:4", "channel:5"]);
      expect(wholeMore).not.toHaveBeenCalled();
    });

    it("should fetch one page at a time and show it loading meanwhile", async () => {
      let release: () => void = () => {};
      const paged = createMemorySource({ resources: [GROUP_A], more: true });
      const more = vi.spyOn(paged, "more").mockImplementation(
        async () =>
          await new Promise<void>((resolve) => {
            release = resolve;
          }),
      );
      const store = createStore({ [ROOT_KEY]: [paged] });
      store.start();
      const node = await findMore(store);
      store.more(node);
      store.more(node);
      expect(more).toHaveBeenCalledTimes(1);
      expect(store.isLoading(node)).toBe(true);
      release();
      await expect.poll(() => store.isLoading(node)).toBe(false);
    });

    it("should follow a change to a list", async () => {
      const root = createMemorySource(listed(GROUP_A));
      const store = createStore({ [ROOT_KEY]: [root] });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      root.push(listed(GROUP_A, GROUP_B));
      expect(keysOf(store)).toEqual(["group:a", "group:b"]);
    });

    it("should keep the nodes equal when only a resource changes", async () => {
      const root = createMemorySource(listed(GROUP_A));
      const store = createStore({ [ROOT_KEY]: [root] });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      const nodes = store.getNodes();
      const renamed = res("group", "a", "a2");
      const handler = vi.fn();
      store.subscribeKey(handler, renamed.key);
      root.push(listed(renamed));
      expect(store.getNodes()).toBe(nodes);
      expect(store.get(renamed.key)).toBe(renamed);
      expect(handler).toHaveBeenCalledOnce();
    });

    it("should show a key that two open lists hold only at its first place", async () => {
      const store = createStore({
        [ROOT_KEY]: [
          createMemorySource(listed(GROUP_A)),
          createMemorySource(listed(res("channel", "1", "x"))),
        ],
        "group:a": [createMemorySource(listed(res("channel", "1", "x")))],
      });
      store.setExpanded(["group:a"]);
      store.start();
      await expect.poll(() => keysOf(store, "group:a")).toEqual(["channel:1"]);
      expect(keysOf(store)).toEqual(["group:a"]);
      expect(store.getNodes()).toEqual([
        { key: "group:a", children: [{ key: "channel:1" }] },
      ]);
    });

    it("should fetch a list again when the cache drops its window", async () => {
      const root = createMemorySource(listed(GROUP_A));
      const retrieve = vi.spyOn(root, "retrieve");
      const store = createStore({ [ROOT_KEY]: [root] });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      root.push(undefined);
      await expect.poll(() => retrieve.mock.calls.length).toBe(2);
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
    });

    it("should pass each failed fetch to the error handler", async () => {
      const onError = vi.fn();
      const failing: Source = {
        retrieve: async () => await Promise.reject(new Error("unreachable")),
        more: async () => {},
        retrieveAll: async () => await Promise.reject(new Error("unreachable")),
        getCached: () => undefined,
        onChange: () => () => {},
      };
      const store = createStore({ [ROOT_KEY]: [failing] }, { onError });
      store.start();
      await expect.poll(() => onError.mock.calls.length).toBe(1);
      expect(onError.mock.calls[0][0]).toEqual(new Error("unreachable"));
    });
  });

  describe("expansion", () => {
    it("should read the lists of a node only once it expands", async () => {
      const list = vi.fn((id: resource.ID) =>
        resource.idsEqual(id, ROOT)
          ? [createMemorySource(listed(GROUP_A))]
          : [createMemorySource(listed(res("channel", "1", "x")))],
      );
      const store = createStore({}, { list });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      expect(list).toHaveBeenCalledTimes(1);
      store.setExpanded(["group:a"]);
      expect(store.isLoading("group:a")).toBe(true);
      await expect.poll(() => keysOf(store, "group:a")).toEqual(["channel:1"]);
      expect(store.isLoading("group:a")).toBe(false);
      expect(store.getNodes()).toEqual([
        { key: "group:a", children: [{ key: "channel:1" }] },
      ]);
    });

    it("should stop reading the lists of a node that hides, and paint them on return", async () => {
      const child = createMemorySource(listed(res("channel", "1", "x")));
      const store = createStore({
        [ROOT_KEY]: [createMemorySource(listed(GROUP_A))],
        "group:a": [child],
      });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      store.setExpanded(["group:a"]);
      await expect.poll(() => keysOf(store, "group:a")).toEqual(["channel:1"]);
      store.setExpanded([]);
      expect(child.subscribers()).toBe(0);
      expect(keysOf(store, "group:a")).toBeUndefined();
      store.setExpanded(["group:a"]);
      expect(keysOf(store, "group:a")).toEqual(["channel:1"]);
      expect(child.subscribers()).toBe(1);
    });

    it("should tell its subscribers when the expanded keys change", () => {
      const store = createStore({});
      const handler = vi.fn();
      store.subscribe(handler);
      store.setExpanded(["group:a"]);
      expect(handler).toHaveBeenCalled();
      expect(store.getExpanded()).toEqual(["group:a"]);
    });

    it("should stop reading every list when it stops", async () => {
      const root = createMemorySource(listed(GROUP_A));
      const store = createStore({ [ROOT_KEY]: [root] });
      const stop = store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      stop();
      expect(root.subscribers()).toBe(0);
    });
  });

  describe("drafts", () => {
    const DRAFT = res("group", "new", "");

    it("should show a draft first in its parent until it ends", async () => {
      const store = createStore({ [ROOT_KEY]: [createMemorySource(listed(GROUP_A))] });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      const done = store.addDraft({ resource: DRAFT, parent: ROOT_KEY });
      expect(keysOf(store)).toEqual(["group:new", "group:a"]);
      expect(store.isDraft("group:new")).toBe(true);
      expect(store.get("group:new")).toBe(DRAFT);
      done();
      expect(keysOf(store)).toEqual(["group:a"]);
      expect(store.isDraft("group:new")).toBe(false);
    });

    it("should keep an ended draft that the lists hold until its children show", async () => {
      const root = createMemorySource(listed(GROUP_A));
      const created = res("group", "new", "Pumps");
      const store = createStore({
        [ROOT_KEY]: [root],
        "group:new": [createMemorySource(listed(res("channel", "1", "x")))],
      });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      const done = store.addDraft({ resource: DRAFT, parent: ROOT_KEY });
      store.setExpanded(["group:new"]);
      root.push(listed(GROUP_A, created));
      done();
      expect(store.isDraft("group:new")).toBe(true);
      expect(store.get("group:new")).toBe(created);
      await expect.poll(() => store.isDraft("group:new")).toBe(false);
      expect(keysOf(store)).toEqual(["group:a", "group:new"]);
      expect(keysOf(store, "group:new")).toEqual(["channel:1"]);
    });

    it("should end a draft whose own lists fail to fetch", async () => {
      const root = createMemorySource(listed(GROUP_A));
      const failing: Source = {
        retrieve: async () => await Promise.reject(new Error("unreachable")),
        more: async () => {},
        retrieveAll: async () => await Promise.reject(new Error("unreachable")),
        getCached: () => undefined,
        onChange: () => () => {},
      };
      const store = createStore({ [ROOT_KEY]: [root], "group:new": [failing] });
      store.start();
      await expect.poll(() => keysOf(store)).toEqual(["group:a"]);
      const done = store.addDraft({ resource: DRAFT, parent: ROOT_KEY });
      store.setExpanded(["group:new"]);
      root.push(listed(GROUP_A, res("group", "new", "Pumps")));
      done();
      expect(store.isDraft("group:new")).toBe(true);
      await expect.poll(() => store.isDraft("group:new")).toBe(false);
      expect(keysOf(store)).toEqual(["group:a", "group:new"]);
    });

    it("should move the children a draft holds out of their parent", async () => {
      const store = createStore({
        [ROOT_KEY]: [
          createMemorySource(
            listed(GROUP_A, res("channel", "1", "x"), res("channel", "2", "y")),
          ),
        ],
      });
      store.start();
      await expect.poll(() => keysOf(store)?.length).toBe(3);
      store.setExpanded(["group:new"]);
      const done = store.addDraft({
        resource: DRAFT,
        parent: ROOT_KEY,
        children: ["channel:1"],
      });
      expect(keysOf(store)).toEqual(["group:new", "group:a", "channel:2"]);
      expect(keysOf(store, "group:new")).toEqual(["channel:1"]);
      expect(store.getParent("channel:1")).toBe("group:new");
      done();
      expect(keysOf(store)).toEqual(["group:a", "channel:1", "channel:2"]);
    });
  });

  describe("results", () => {
    const results: Results = {
      nodes: [{ key: "group:a", children: [{ key: "channel:1" }] }],
      fixed: new Set(["group:a"]),
      expanded: [],
      paths: new Map(),
      resources: new Map([
        ["group:a", GROUP_A],
        ["channel:1", res("channel", "1", "match")],
      ]),
      ancestors: new Map([["channel:1", new Set(["group:a"])]]),
    };

    it("should show the results in place of the lists of the root and fixed groups", () => {
      const list = vi.fn(() => []);
      const store = createStore({}, { list, results });
      store.start();
      store.setExpanded(["group:a"]);
      expect(keysOf(store)).toEqual(["group:a"]);
      expect(keysOf(store, "group:a")).toEqual(["channel:1"]);
      expect(store.get("channel:1")?.name).toBe("match");
      expect(list).not.toHaveBeenCalled();
    });

    it("should stop reading the lists of a group that the results fix", async () => {
      const child = createMemorySource(listed(res("channel", "2", "every")));
      const store = createStore({
        [ROOT_KEY]: [createMemorySource(listed(GROUP_A))],
        "group:a": [child],
      });
      store.start();
      store.setExpanded(["group:a"]);
      await expect.poll(() => keysOf(store, "group:a")).toEqual(["channel:2"]);
      expect(child.subscribers()).toBe(1);
      store.setResults(results);
      expect(child.subscribers()).toBe(0);
      expect(keysOf(store, "group:a")).toEqual(["channel:1"]);
    });

    it("should expand each folder of the results once", () => {
      const store = createStore({}, { results: { ...results, expanded: ["group:a"] } });
      expect(store.getExpanded()).toEqual(["group:a"]);
      expect(keysOf(store, "group:a")).toEqual(["channel:1"]);
      store.setExpanded([]);
      store.setResults({ ...results, expanded: ["group:a", "group:b"] });
      expect(store.getExpanded()).toEqual(["group:b"]);
    });

    it("should give the ancestors of a match at the top through its path", () => {
      const store = createStore(
        {},
        {
          results: {
            ...results,
            nodes: [{ key: "channel:1" }],
            fixed: new Set(),
            paths: new Map([["channel:1", [GROUP_B.id, GROUP_A.id]]]),
          },
        },
      );
      expect(store.getAncestors("channel:1")).toEqual([GROUP_B.id, GROUP_A.id, ROOT]);
    });

    it("should give the ancestors of a node through the tree", () => {
      const store = createStore({}, { results });
      store.setExpanded(["group:a"]);
      expect(store.getAncestors("channel:1")).toEqual([GROUP_A.id, ROOT]);
    });

    it("should find the keys inside another key in the tree or above a match", () => {
      const flat: Results = {
        ...results,
        nodes: [{ key: "group:a", children: [] }, { key: "channel:1" }],
        fixed: new Set(),
      };
      const store = createStore({}, { results: flat });
      expect(store.getContained(["group:a", "channel:1"])).toEqual(
        new Set(["channel:1"]),
      );
      expect(store.getContained(["channel:1"])).toEqual(new Set());
    });

    it("should read the lists of a group that is not fixed once it expands", async () => {
      const store = createStore(
        { "group:a": [createMemorySource(listed(res("channel", "2", "every")))] },
        { results: { ...results, fixed: new Set() } },
      );
      store.start();
      store.setExpanded(["group:a"]);
      expect(keysOf(store, "group:a")).toEqual(["channel:1"]);
      await expect.poll(() => keysOf(store, "group:a")).toEqual(["channel:2"]);
    });
  });
});

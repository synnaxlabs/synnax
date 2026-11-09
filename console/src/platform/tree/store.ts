// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { type Tree as Base } from "@synnaxlabs/lyra/tree";
import { compare, type destructor } from "@synnaxlabs/x";

import { type Source, type Window } from "@/platform/tree/children";
import { type Resource } from "@/platform/tree/types";

/** The resource types the tree shows before other resources. */
export const FOLDER_TYPES = new Set<resource.Type>(["group", "arc_directory"]);

const isFolder = ({ id }: Resource): boolean => FOLDER_TYPES.has(id.type);

const MORE_PREFIX = "more/";

const moreKey = (folder: string): string => `${MORE_PREFIX}${folder}`;

/** Returns whether the key names the node that loads the next page of a folder. */
export const isMoreKey = (key: string): boolean => key.startsWith(MORE_PREFIX);

const folderOf = (key: string): string => key.slice(MORE_PREFIX.length);

const byName = (a: Resource, b: Resource): number => compare.text(a.name, b.name);

/** A record the user names before the tree creates it. It shows first in its parent. */
export interface Draft {
  resource: Resource;
  /** The key of the node the draft shows under. */
  parent: string;
  /** The keys the draft holds. They leave their parents while it shows. */
  children?: string[];
}

/** The search answer a tree shows in place of its top level. */
export interface Results {
  nodes: Base.Node<string>[];
  /** The groups that show only their matches. */
  fixed: Set<string>;
  resources: Map<string, Resource>;
}

export interface StoreConfig {
  root: resource.ID;
  /** Lists the children of a node. */
  list: (id: resource.ID) => Source[];
  hasChildren: (type: resource.Type) => boolean;
  /** Receives the error of each list fetch that fails. */
  onError: (error: unknown) => void;
  results?: Results;
}

interface Folder {
  sources: Source[];
  windows: Array<Window | undefined>;
  /** The merge of the windows it was built from, reused until a window changes. */
  merged?: { windows: Array<Window | undefined>; value: Merged | undefined };
  /** Ends the subscriptions. Absent while the folder is hidden. */
  release?: destructor.Destructor;
  loading: boolean;
  /** Whether the last fetch failed. The folder then shows what it holds. */
  failed: boolean;
  /** Whether the next page of its lists is on the way. */
  fetchingMore: boolean;
}

interface HeldDraft {
  draft: Draft;
  /**
   * Whether the create ended. The draft then stays until the tree can draw the record.
   */
  done: boolean;
}

interface Segment {
  keys: string[];
  more: boolean;
}

// A record past the least last record of a window with more can sort before a match
// that window has not fetched, so it waits.
const segment = (windows: Window[]): Segment => {
  let cut: string | undefined;
  windows.forEach(({ resources, more }) => {
    const last = resources.at(-1);
    if (more && last != null && (cut == null || compare.text(last.name, cut) < 0))
      cut = last.name;
  });
  const keys = windows
    .flatMap(({ resources }) => resources)
    .filter((r) => cut == null || compare.text(r.name, cut) <= 0)
    .toSorted(byName)
    .map(({ key }) => key);
  return { keys, more: cut != null };
};

interface Merged {
  keys: string[];
  /** The indices of the lists whose next page shows more of the folder. */
  pending: number[];
}

// Folders show first, so a folder list with more hides the other records.
const merge = (windows: Array<Window | undefined>): Merged | undefined => {
  if (!windows.every((w): w is Window => w != null)) return undefined;
  const indices = windows.map((_, i) => i);
  const folders = indices.filter((i) => windows[i].resources.some(isFolder));
  const rest = indices.filter((i) => !folders.includes(i));
  const withMore = (of: number[]) => of.filter((i) => windows[i].more);
  const head = segment(folders.map((i) => windows[i]));
  if (head.more) return { keys: head.keys, pending: withMore(folders) };
  const tail = segment(rest.map((i) => windows[i]));
  return {
    keys: [...head.keys, ...tail.keys],
    pending: tail.more ? withMore(rest) : [],
  };
};

const mapsEqual = (a: Map<string, string[]>, b: Map<string, string[]>): boolean =>
  a.size === b.size &&
  [...a].every(([key, keys]) => {
    const other = b.get(key);
    return other != null && compare.arraysEqual(keys, other);
  });

/**
 * Holds what a resource tree shows: the child lists of each open folder, merged by
 * name, and the drafts. Nodes open as they expand and close as they hide.
 */
export class Store {
  private readonly config: StoreConfig;
  private readonly rootKey: string;
  private readonly folders = new Map<string, Folder>();
  private readonly drafts = new Map<string, HeldDraft>();
  private readonly listeners = new Set<() => void>();
  private readonly keyListeners = new Map<string, Set<() => void>>();
  private results?: Results;
  private resultChildren = new Map<string, string[]>();
  private expanded = new Set<string>();
  private expandedKeys: string[] = [];
  private resources = new Map<string, Resource>();
  private children = new Map<string, string[]>();
  private parents = new Map<string, string>();
  private nodes: Base.Node<string>[] = [];
  private started = false;
  private refreshing = false;
  private stale = false;

  /** Reads the cached child lists of the root. Nothing is fetched until start. */
  constructor(config: StoreConfig) {
    this.config = config;
    this.rootKey = resource.idToString(config.root);
    this.indexResults(config.results);
    if (config.results == null)
      this.folders.set(this.rootKey, this.createFolder(config.root));
    this.refresh();
  }

  /** Starts to read the lists of the open folders. Returns a function that stops. */
  start(): destructor.Destructor {
    this.started = true;
    this.refresh();
    return () => {
      this.started = false;
      this.folders.forEach((folder) => this.pause(folder));
    };
  }

  /** Sets the expanded nodes. An expanded node whose parents show opens. */
  setExpanded(keys: string[]): void {
    this.expandedKeys = keys;
    this.expanded = new Set(keys);
    this.refresh();
    this.listeners.forEach((handler) => handler());
  }

  /** Returns the keys of the expanded nodes. Equal until they change. */
  readonly getExpanded = (): string[] => this.expandedKeys;

  /** Sets the search answer the tree shows. */
  setResults(results: Results): void {
    this.indexResults(results);
    this.refresh();
  }

  /**
   * Shows the draft until the returned function ends it. An ended draft whose record
   * the lists hold stays until the record's own children show.
   */
  addDraft(draft: Draft): destructor.Destructor {
    const held: HeldDraft = { draft, done: false };
    this.drafts.set(draft.resource.key, held);
    this.refresh();
    return () => {
      held.done = true;
      this.refresh();
    };
  }

  /** Returns the nodes from the top of the tree. Equal until the tree changes. */
  readonly getNodes = (): Base.Node<string>[] => this.nodes;

  /** Returns the shown children of an open node, or undefined. */
  getChildren(key: string): string[] | undefined {
    return this.children.get(key);
  }

  /** Returns the key of the node's parent, or undefined for a node not shown. */
  getParent(key: string): string | undefined {
    return this.parents.get(key);
  }

  readonly get = (key: string): Resource | undefined => this.resources.get(key);

  readonly isDraft = (key: string): boolean => this.drafts.has(key);

  /** Returns whether the node waits for its first lists, or for the next page. */
  isLoading(key: string): boolean {
    if (isMoreKey(key)) return this.folders.get(folderOf(key))?.fetchingMore === true;
    return this.folders.get(key)?.loading === true;
  }

  /**
   * Fetches the next page of the folder that the "Show more" node ends. Does nothing
   * while a page is on the way.
   */
  more(key: string): void {
    const folder = this.folders.get(folderOf(key));
    if (folder == null || folder.fetchingMore) return;
    const pending = this.mergedOf(folder)?.pending ?? [];
    if (pending.length === 0) return;
    folder.fetchingMore = true;
    this.notifyKey(key);
    void (async () => {
      try {
        await Promise.all(pending.map(async (i) => await folder.sources[i].more()));
      } catch (error) {
        this.config.onError(error);
      }
      folder.fetchingMore = false;
      this.notifyKey(key);
    })();
  }

  /** Calls the handler each time the nodes or the expanded keys change. */
  readonly subscribe = (handler: () => void): destructor.Destructor => {
    this.listeners.add(handler);
    return () => this.listeners.delete(handler);
  };

  /** Calls the handler each time the resource or the loading of the key changes. */
  readonly subscribeKey = (handler: () => void, key: string): destructor.Destructor => {
    let handlers = this.keyListeners.get(key);
    if (handlers == null) {
      handlers = new Set();
      this.keyListeners.set(key, handlers);
    }
    handlers.add(handler);
    return () => {
      handlers.delete(handler);
      if (handlers.size === 0) this.keyListeners.delete(key);
    };
  };

  private indexResults(results?: Results): void {
    this.results = results;
    this.resultChildren = new Map();
    if (results == null) return;
    const index = (key: string, nodes: Base.Node<string>[]) => {
      this.resultChildren.set(
        key,
        nodes.map((node) => node.key),
      );
      nodes.forEach((node) => {
        if (node.children != null) index(node.key, node.children);
      });
    };
    index(this.rootKey, results.nodes);
  }

  private createFolder(id: resource.ID): Folder {
    const sources = this.config.list(id);
    return {
      sources,
      windows: sources.map((source) => source.getCached()),
      loading: false,
      failed: false,
      fetchingMore: false,
    };
  }

  private notifyKey(key: string): void {
    this.keyListeners.get(key)?.forEach((handler) => handler());
  }

  private refresh(): void {
    if (this.refreshing) {
      this.stale = true;
      return;
    }
    this.refreshing = true;
    try {
      do {
        this.stale = false;
        this.recompute();
      } while (this.stale);
    } finally {
      this.refreshing = false;
    }
  }

  // Results show in place of the lists of the root and of the fixed groups.
  private showsResults(key: string): boolean {
    return (
      this.results != null && (key === this.rootKey || this.results.fixed.has(key))
    );
  }

  private readsLists(key: string): boolean {
    return !this.showsResults(key) && this.drafts.get(key)?.done !== false;
  }

  private childrenOf(key: string, moved: Set<string>): string[] | undefined {
    const held = this.drafts.get(key);
    let listed: string[] | undefined;
    if (held?.done === false) listed = held.draft.children ?? [];
    else if (this.showsResults(key)) listed = this.resultChildren.get(key) ?? [];
    else {
      const folder = this.folders.get(key);
      const merged = folder == null ? undefined : this.mergedOf(folder);
      if (merged == null) listed = this.resultChildren.get(key) ?? held?.draft.children;
      else if (merged.pending.length === 0) listed = merged.keys;
      else listed = [...merged.keys, moreKey(key)];
    }
    const drafted = [...this.drafts.values()]
      .filter(({ draft }) => draft.parent === key)
      .map(({ draft }) => draft.resource.key);
    if (listed == null) return drafted.length > 0 ? drafted : undefined;
    const placed = new Set(drafted);
    const own = new Set(held?.draft.children);
    const kept = listed.filter(
      (child) => !placed.has(child) && (own.has(child) || !moved.has(child)),
    );
    return [...drafted, ...kept];
  }

  private mergedOf(folder: Folder): Merged | undefined {
    const { merged, windows } = folder;
    if (merged != null && compare.arraysEqual(merged.windows, windows))
      return merged.value;
    const value = merge(windows);
    folder.merged = { windows: [...windows], value };
    return value;
  }

  private recompute(): void {
    const moved = new Set(
      [...this.drafts.values()].flatMap(({ draft }) => draft.children ?? []),
    );
    const children = new Map<string, string[]>();
    const parents = new Map<string, string>();
    const reachable = new Set<string>();
    const visit = (key: string): void => {
      if (reachable.has(key)) return;
      reachable.add(key);
      const listed = this.childrenOf(key, moved);
      if (listed == null) return;
      const shown: string[] = [];
      children.set(key, shown);
      listed.forEach((child) => {
        // A key under two shown parents, such as a flat search match that its open
        // group also lists, shows only at its first place.
        if (child === this.rootKey || parents.has(child)) return;
        parents.set(child, key);
        shown.push(child);
        if (this.expanded.has(child)) visit(child);
      });
    };
    visit(this.rootKey);

    const listedKeys = new Set<string>();
    const resources = new Map<string, Resource>(this.results?.resources);
    this.drafts.forEach(({ draft }) =>
      resources.set(draft.resource.key, draft.resource),
    );
    reachable.forEach((key) =>
      this.folders.get(key)?.windows.forEach((window) =>
        window?.resources.forEach((r) => {
          listedKeys.add(r.key);
          resources.set(r.key, r);
        }),
      ),
    );

    const settled = [...this.drafts].filter(
      ([key, { done }]) => done && (!listedKeys.has(key) || this.answered(key)),
    );
    settled.forEach(([key]) => this.drafts.delete(key));

    if (this.started) {
      reachable.forEach((key) => {
        if (!this.readsLists(key)) return;
        const folder = this.folders.get(key);
        if (folder?.release == null) this.open(key, folder);
      });
      this.folders.forEach((folder, key) => {
        if (!reachable.has(key) || !this.readsLists(key)) this.pause(folder);
      });
    }
    // A settled draft or a newly open folder changes what shows.
    if (settled.length > 0) this.stale = true;

    const previous = this.resources;
    this.resources = resources;
    const changedKeys = new Set(
      [...previous.keys(), ...resources.keys()].filter(
        (key) => previous.get(key) !== resources.get(key),
      ),
    );
    this.parents = parents;
    const moves = !mapsEqual(children, this.children);
    if (moves) {
      this.children = children;
      this.nodes = this.build(this.rootKey);
    }
    changedKeys.forEach((key) => this.notifyKey(key));
    if (moves) this.listeners.forEach((handler) => handler());
  }

  // Answered means each of the node's lists has a window or failed, or the node is not
  // expanded.
  private answered(key: string): boolean {
    if (!this.expanded.has(key)) return true;
    const folder = this.folders.get(key);
    if (folder == null) return false;
    return folder.failed || folder.windows.every((w) => w != null);
  }

  private build(key: string): Base.Node<string>[] {
    return (this.children.get(key) ?? []).map((child) => {
      if (isMoreKey(child) || !this.config.hasChildren(resource.parseID(child).type))
        return { key: child };
      return {
        key: child,
        children: this.children.has(child) ? this.build(child) : [],
      };
    });
  }

  private open(key: string, existing?: Folder): void {
    const folder = existing ?? this.createFolder(resource.parseID(key));
    this.folders.set(key, folder);
    // Subscribe before fetching, so a change between the two reaches the window.
    const releases = folder.sources.map((source, i) =>
      source.onChange((window) => {
        folder.windows[i] = window;
        if (window == null && folder.release != null) this.fetch(key, folder, [i]);
        this.refresh();
      }),
    );
    folder.release = () => releases.forEach((release) => release());
    folder.windows = folder.sources.map(
      (source, i) => source.getCached() ?? folder.windows[i],
    );
    this.fetch(
      key,
      folder,
      folder.sources.map((_, i) => i),
    );
    this.stale = true;
  }

  private fetch(key: string, folder: Folder, indices: number[]): void {
    const loading = folder.windows.some((w) => w == null);
    if (loading !== folder.loading) {
      folder.loading = loading;
      this.notifyKey(key);
    }
    // A change can land between the fetch and its answer, so the window comes from the
    // cache.
    void this.settle(key, folder, indices);
  }

  private async settle(key: string, folder: Folder, indices: number[]): Promise<void> {
    folder.failed = false;
    try {
      await Promise.all(
        indices.map(async (i) => {
          const source = folder.sources[i];
          await source.retrieve();
          folder.windows[i] = source.getCached() ?? folder.windows[i];
        }),
      );
    } catch (error) {
      folder.failed = true;
      this.config.onError(error);
    }
    if (folder.loading) {
      folder.loading = false;
      this.notifyKey(key);
    }
    this.refresh();
  }

  private pause(folder: Folder): void {
    folder.release?.();
    folder.release = undefined;
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type destructor, observe, type record, shallow } from "@synnaxlabs/x";

/** An entry of a list that the form edits by key. */
export type Entry = record.Keyed<string>;

/**
 * Holds the values of a form. Paths are dot-separated, and a segment into a list of
 * keyed entries names an entry's key or its index.
 */
export interface Store<V> {
  /** @returns the values. A change gives each object on its path a new identity. */
  get: () => V;
  /** Replaces the value at the path. An empty path replaces the values. */
  set: (path: string, value: unknown) => void;
  /** Removes the object field or the list entry at the path. */
  remove: (path: string) => void;
  /**
   * Inserts an entry into the list at the path, after the entry with the key `after`.
   * An absent `after` inserts first, and one that names no entry inserts last. Does
   * nothing when the list holds an entry with the same key.
   */
  insert: <E extends Entry>(path: string, value: E, after?: string) => void;
  /** Moves the entry with the key after the entry with the key `after`, as insert. */
  move: (path: string, key: string, after?: string) => void;
  /** Runs the writes that the function makes as one change. Nested calls join it. */
  batch: (fn: () => void) => void;
  /** Calls the callback after each change. @returns a destructor that stops it. */
  onChange: (callback: () => void) => destructor.Destructor;
}

/**
 * Copies the root and every container on the way to the path's leaf, walking the path
 * the way `deep.set` and `deep.remove` do. A write at the path then gives each ancestor
 * a new identity and leaves every untouched subtree shared.
 */
const copyPath = <V>(root: V, path: string): V => {
  const next = shallow.copy(root);
  const parts = path.split(deep.SEPARATOR);
  let parent = next as record.Unknown;
  let i = 0;
  while (i < parts.length - 1) {
    const best = deep.findBestKey(parent, parts.slice(i, parts.length - 1));
    if (best == null) break;
    const [key, consumed] = best;
    i += consumed;
    const child = deep.defaultGetter(parent, key);
    const copy = shallow.copy(child) as record.Unknown;
    // An array entry found by its key field sits at an index, not at the key.
    if (parent[key] === child) parent[key] = copy;
    else parent[(parent as unknown as unknown[]).indexOf(child)] = copy;
    parent = copy;
  }
  return next;
};

const position = (list: Entry[], after: string | undefined): number => {
  if (after == null) return 0;
  const i = list.findIndex(({ key }) => key === after);
  return i < 0 ? list.length : i + 1;
};

/** Creates a store that holds a copy of the values in memory. */
export const createLocalStore = <V>(values: V): Store<V> => {
  const observer = new observe.Observer<void>();
  let current = deep.copy(values);
  let depth = 0;
  let changed = false;
  const commit = (next: V) => {
    current = next;
    if (depth > 0) changed = true;
    else observer.notify();
  };
  const write = (path: string, mutate: (next: V) => void) => {
    const next = copyPath(current, path);
    mutate(next);
    commit(next);
  };
  const list = (path: string): Entry[] =>
    deep.get<Entry[], V>(current, path, { optional: true }) ?? [];
  const set = (path: string, value: unknown) => {
    if (path === "") commit(value as V);
    else write(path, (next) => deep.set(next, path, value));
  };
  return {
    get: () => current,
    set,
    remove: (path) => write(path, (next) => deep.remove(next, path)),
    insert: (path, value, after) => {
      const entries = list(path);
      if (entries.some(({ key }) => key === value.key)) return;
      const next = [...entries];
      next.splice(position(next, after), 0, value);
      set(path, next);
    },
    move: (path, key, after) => {
      const entries = list(path);
      const i = entries.findIndex((e) => e.key === key);
      if (i < 0 || after === key) return;
      const next = [...entries];
      const [entry] = next.splice(i, 1);
      next.splice(position(next, after), 0, entry);
      set(path, next);
    },
    batch: (fn) => {
      depth++;
      try {
        fn();
      } finally {
        depth--;
        if (depth === 0 && changed) {
          changed = false;
          observer.notify();
        }
      }
    },
    onChange: (callback) => observer.onChange(callback),
  };
};

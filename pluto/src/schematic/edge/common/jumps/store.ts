// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";

/// @brief the shared, frozen empty result returned for edges with no hops. Returning a
/// stable reference is required by useSyncExternalStore, which loops if getSnapshot
/// yields a fresh value each call.
const EMPTY: xy.XY[] = [];

const pointsEqual = (a: xy.XY[], b: xy.XY[]): boolean => {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++)
    if (a[i].x !== b[i].x || a[i].y !== b[i].y) return false;
  return true;
};

/// @brief a per-edge external store of hop points, suitable for useSyncExternalStore.
/// get returns a reference-stable array per key (the same array until that key's hops
/// actually change), and commit diffs a freshly computed result against the current
/// one, notifying only the listeners of keys whose hops changed.
export interface Store {
  subscribe: (key: string, onChange: () => void) => () => void;
  get: (key: string) => xy.XY[];
  commit: (next: Map<string, xy.XY[]>) => void;
}

export const create = (): Store => {
  let current = new Map<string, xy.XY[]>();
  const listeners = new Map<string, Set<() => void>>();

  const notify = (key: string): void => listeners.get(key)?.forEach((l) => l());

  return {
    subscribe: (key, onChange) => {
      let set = listeners.get(key);
      if (set == null) listeners.set(key, (set = new Set()));
      set.add(onChange);
      return () => {
        const s = listeners.get(key);
        if (s == null) return;
        s.delete(onChange);
        if (s.size === 0) listeners.delete(key);
      };
    },
    get: (key) => current.get(key) ?? EMPTY,
    commit: (next) => {
      const merged = new Map<string, xy.XY[]>();
      const changed: string[] = [];
      for (const [key, points] of next) {
        const prev = current.get(key);
        // Preserve the previous array reference when a key's hops are unchanged so
        // getSnapshot stays stable and the edge does not re-render.
        if (prev != null && pointsEqual(prev, points)) merged.set(key, prev);
        else {
          merged.set(key, points);
          changed.push(key);
        }
      }
      for (const key of current.keys()) if (!next.has(key)) changed.push(key);
      current = merged;
      for (const key of changed) notify(key);
    },
  };
};

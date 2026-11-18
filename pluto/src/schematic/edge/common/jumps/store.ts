// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Path } from "@/schematic/edge/common/path";

const EMPTY: Path.Crossing[] = [];

const crossingsEqual = (a: Path.Crossing[], b: Path.Crossing[]): boolean => {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++)
    if (a[i].x !== b[i].x || a[i].y !== b[i].y || a[i].radius !== b[i].radius)
      return false;
  return true;
};

export interface Store {
  subscribe: (key: string, onChange: () => void) => () => void;
  get: (key: string) => Path.Crossing[];
  commit: (next: Map<string, Path.Crossing[]>) => void;
}

export const createStore = (): Store => {
  let current = new Map<string, Path.Crossing[]>();
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
      const merged = new Map<string, Path.Crossing[]>();
      const changed: string[] = [];
      for (const [key, crossings] of next) {
        const prev = current.get(key);
        // Reuse the previous reference for unchanged keys so getSnapshot stays stable.
        if (prev != null && crossingsEqual(prev, crossings)) merged.set(key, prev);
        else {
          merged.set(key, crossings);
          changed.push(key);
        }
      }
      for (const key of current.keys()) if (!next.has(key)) changed.push(key);
      current = merged;
      for (const key of changed) notify(key);
    },
  };
};

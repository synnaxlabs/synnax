// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";

import { type Store } from "@/platform/tree/store";

/**
 * Returns the IDs above a shown node, nearest first, ending with the root. A search
 * result at the top of the tree sits in groups that the paths give.
 */
export const findAncestors = (
  store: Store,
  key: string,
  root: resource.ID,
  paths?: Map<string, resource.ID[]>,
): resource.ID[] => {
  const rootKey = resource.idToString(root);
  const ancestors: resource.ID[] = [];
  let top = key;
  for (
    let parent = store.getParent(key);
    parent != null && parent !== rootKey;
    parent = store.getParent(parent)
  ) {
    ancestors.push(resource.parseID(parent));
    top = parent;
  }
  ancestors.push(...(paths?.get(top) ?? []), root);
  return ancestors;
};

/**
 * Returns the keys that sit inside another key of the set, in the tree or in the
 * ancestors of a search match. Depth cannot tell: two nodes at different depths are
 * often unrelated.
 */
export const findContainedKeys = (
  store: Store,
  keys: string[],
  ancestors: Map<string, Set<string>> = new Map(),
): Set<string> => {
  const all = new Set(keys);
  const contained = (key: string): boolean => {
    for (
      let parent = store.getParent(key);
      parent != null;
      parent = store.getParent(parent)
    )
      if (all.has(parent)) return true;
    return [...(ancestors.get(key) ?? [])].some((a) => all.has(a));
  };
  return new Set(keys.filter(contained));
};

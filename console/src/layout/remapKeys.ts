// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mosaic } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { type SliceState, type State } from "@/layout/types";

export interface RemapResult {
  slice: SliceState;
  oldKeyForNew: Map<string, string>;
}

/**
 * Assign fresh UUIDs to all layout keys so the same workspace can be
 * imported repeatedly without key collisions on the server ontology.
 */
export const remapKeys = (slice: SliceState): RemapResult => {
  const oldToNew = new Map<string, string>();
  for (const oldKey of Object.keys(slice.layouts))
    if (oldKey !== "main") oldToNew.set(oldKey, uuid.create());
  const remap = (key: string): string => oldToNew.get(key) ?? key;

  const layouts: Record<string, State> = {};
  for (const [oldKey, entry] of Object.entries(slice.layouts)) {
    const newKey = remap(oldKey);
    layouts[newKey] = { ...entry, key: newKey };
  }

  const remapNode = (node: Mosaic.Node): Mosaic.Node => ({
    ...node,
    tabs: node.tabs?.map((t) => ({ ...t, tabKey: remap(t.tabKey) })),
    selected: node.selected != null ? remap(node.selected) : undefined,
    first: node.first != null ? remapNode(node.first) : undefined,
    last: node.last != null ? remapNode(node.last) : undefined,
  });

  const mosaics: typeof slice.mosaics = {};
  for (const [mKey, m] of Object.entries(slice.mosaics))
    mosaics[mKey] = {
      ...m,
      activeTab: m.activeTab != null ? remap(m.activeTab) : null,
      root: remapNode(m.root),
    };

  const oldKeyForNew = new Map<string, string>();
  for (const [oldKey, newKey] of oldToNew) oldKeyForNew.set(newKey, oldKey);

  return { slice: { ...slice, layouts, mosaics }, oldKeyForNew };
};

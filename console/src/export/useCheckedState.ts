// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tree } from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

export interface CheckedState {
  checked: Set<string>;
  toggle: (key: string, nodes: Tree.Node[]) => void;
  isChecked: (key: string) => boolean;
  isIndeterminate: (key: string, nodes: Tree.Node[]) => boolean;
  reconcile: (nodes: Tree.Node[]) => void;
}

const descendantKeys = (node: Tree.Node): string[] =>
  Tree.getDescendants(node)
    .slice(1)
    .map((n) => n.key);

const isLeaf = (node: Tree.Node): boolean =>
  node.children == null || node.children.length === 0;

const walkAncestors = (
  nodes: Tree.Node[],
  key: string,
  checked: Set<string>,
): void => {
  const parent = Tree.findNodeParent({ tree: nodes, key });
  if (parent == null) return;
  const allChecked = (parent.children ?? []).every((c) => checked.has(c.key));
  if (allChecked) checked.add(parent.key);
  else checked.delete(parent.key);
  walkAncestors(nodes, parent.key, checked);
};

export const useCheckedState = (): CheckedState => {
  const [checked, setChecked] = useState<Set<string>>(() => new Set());

  const toggle = useCallback((key: string, nodes: Tree.Node[]) => {
    setChecked((prev) => {
      const next = new Set(prev);
      const node = Tree.findNode({ tree: nodes, key });
      const wasChecked = next.has(key);
      const keys = node != null ? descendantKeys(node) : [];
      if (wasChecked) {
        next.delete(key);
        for (const dk of keys) next.delete(dk);
      } else {
        next.add(key);
        for (const dk of keys) next.add(dk);
      }
      walkAncestors(nodes, key, next);
      return next;
    });
  }, []);

  const isChecked = useCallback((key: string) => checked.has(key), [checked]);

  const isIndeterminate = useCallback(
    (key: string, nodes: Tree.Node[]) => {
      if (checked.has(key)) return false;
      const node = Tree.findNode({ tree: nodes, key });
      if (node == null || isLeaf(node)) return false;
      const leafKeys = Tree.getDescendants(node).filter(isLeaf).map((n) => n.key);
      if (leafKeys.length === 0) return false;
      const checkedCount = leafKeys.filter((k) => checked.has(k)).length;
      return checkedCount > 0 && checkedCount < leafKeys.length;
    },
    [checked],
  );

  const reconcile = useCallback((nodes: Tree.Node[]) => {
    setChecked((prev) => {
      const next = new Set(prev);
      let changed = false;
      for (const key of prev) {
        const node = Tree.findNode({ tree: nodes, key });
        if (node == null) continue;
        for (const dk of descendantKeys(node))
          if (!next.has(dk)) {
            next.add(dk);
            changed = true;
          }
      }
      return changed ? next : prev;
    });
  }, []);

  return { checked, toggle, isChecked, isIndeterminate, reconcile };
};

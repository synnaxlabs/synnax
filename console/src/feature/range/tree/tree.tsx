// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { useInitializerRef, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Store } from "@synnaxlabs/lyra/store";
import { Ranger } from "@synnaxlabs/pluto";
import {
  type PropsWithChildren,
  type ReactElement,
  useEffect,
  useLayoutEffect,
  useState,
  useSyncExternalStore,
} from "react";

/** A range in a tree list, at its depth below the list's roots. */
export interface Node {
  key: ranger.Key;
  depth: number;
  /**
   * The range whose next page of children the node reads once it shows. Only the last
   * child read of a range with more children sets it.
   */
  tail?: ranger.Key;
}

/** The children of a range, in the order given. */
export const childrenQuery = (key: ranger.Key, order: ranger.Order) => ({
  where: { parents: [key] },
  order,
});

export interface UseParams {
  /** The keys at depth zero, in list order. */
  roots: ranger.Key[];
  /** The keys whose children show under them. */
  expanded: ranger.Key[];
  /** The order of the children under each range. They show in the order they ran. */
  order?: ranger.Order;
  /** Ranges whose children are given, not read. */
  fixed?: Map<ranger.Key, ranger.Key[]>;
}

export interface UseReturn {
  nodes: Node[];
  /** Reads the next page of the children of the range. */
  more: (key: ranger.Key) => void;
}

/**
 * Lays out the roots with the children of each expanded range under it, one page at a
 * time. A range shows once, at its first place.
 */
export const use = ({ roots, expanded, order, fixed }: UseParams): UseReturn => {
  const { children: read, more } = Ranger.useChildren({
    parents: expanded.filter((key) => fixed?.has(key) !== true),
    order,
  });
  const children = new Map<ranger.Key, ranger.Key[]>(fixed);
  const incomplete = new Set<ranger.Key>();
  read.forEach(({ keys, complete }, key) => {
    children.set(key, keys);
    if (!complete) incomplete.add(key);
  });
  const open = new Set(expanded);
  const seen = new Set<ranger.Key>();
  const nodes: Node[] = [];
  const visit = (key: ranger.Key, depth: number, tail?: ranger.Key): void => {
    if (seen.has(key)) return;
    seen.add(key);
    nodes.push({ key, depth, tail });
    if (!open.has(key)) return;
    const keys = children.get(key) ?? [];
    const last = incomplete.has(key) ? keys.at(-1) : undefined;
    keys.forEach((child) => visit(child, depth + 1, child === last ? key : undefined));
  };
  roots.forEach((key) => visit(key, 0));
  return { nodes, more };
};

/** @returns the expanded keys and a toggle for one key. */
export const useExpanded = (
  initial: ranger.Key[] = [],
): [ranger.Key[], (key: ranger.Key, expanded?: boolean) => void] => {
  const [expanded, setExpanded] = useState(initial);
  const toggle = (key: ranger.Key, next?: boolean) =>
    setExpanded((prev) => {
      const open = next ?? !prev.includes(key);
      if (open) return prev.includes(key) ? prev : [...prev, key];
      return prev.filter((k) => k !== key);
    });
  return [expanded, toggle];
};

/** The place of one range in a tree, as its item reads it. */
export interface NodeState {
  depth: number;
  expanded: boolean;
  /** See {@link Node.tail}. */
  tail?: ranger.Key;
}

const ROOT: NodeState = { depth: 0, expanded: false };

interface NodeStore {
  get: (key: ranger.Key) => NodeState;
  subscribe: (listener: () => void, key: ranger.Key) => () => void;
  onExpand: (key: ranger.Key, expanded: boolean) => void;
  more: (key: ranger.Key) => void;
  /** Gives each key new to the tree its state, before its item first renders. */
  add: (nodes: Node[], expanded: ranger.Key[]) => void;
}

const [Context, useContext] = context.create<NodeStore>({
  displayName: "Tree.NodeContext",
  providerName: "Tree.NodeProvider",
});

const sameState = (a: NodeState, b: NodeState): boolean =>
  a.depth === b.depth && a.expanded === b.expanded && a.tail === b.tail;

export interface NodeProviderProps extends PropsWithChildren {
  nodes: Node[];
  expanded: ranger.Key[];
  onExpand: (key: ranger.Key, expanded: boolean) => void;
  more: (key: ranger.Key) => void;
}

/**
 * Gives each item of a tree its depth and whether it is open. A change to the tree
 * renders only the items whose place changed.
 */
export const NodeProvider = ({
  nodes,
  expanded,
  onExpand,
  more,
  children,
}: NodeProviderProps): ReactElement => {
  const { subscribe, notifyListeners } = Store.useKeyedListeners<ranger.Key>();
  const statesRef = useInitializerRef(() => new Map<ranger.Key, NodeState>());
  const actionsRef = useSyncedRef({ onExpand, more });
  const [store] = useState<NodeStore>(() => ({
    get: (key) => statesRef.current.get(key) ?? ROOT,
    subscribe,
    onExpand: (key, next) => actionsRef.current.onExpand(key, next),
    more: (key) => actionsRef.current.more(key),
    add: (nodes, expanded) => {
      const open = new Set(expanded);
      nodes.forEach(({ key, depth, tail }) => {
        if (!statesRef.current.has(key))
          statesRef.current.set(key, { depth, expanded: open.has(key), tail });
      });
    },
  }));
  // A new item reads its state as it mounts, so a key starts with its state when it
  // first shows. A change to a known key waits for the commit, which tells its item.
  store.add(nodes, expanded);
  useLayoutEffect(() => {
    const open = new Set(expanded);
    const prev = statesRef.current;
    const next = new Map<ranger.Key, NodeState>();
    const changed: ranger.Key[] = [];
    nodes.forEach(({ key, depth, tail }) => {
      const state = { depth, expanded: open.has(key), tail };
      const last = prev.get(key);
      if (last != null && sameState(last, state)) next.set(key, last);
      else {
        next.set(key, state);
        changed.push(key);
      }
    });
    prev.forEach((_, key) => {
      if (!next.has(key)) changed.push(key);
    });
    statesRef.current = next;
    if (changed.length > 0) notifyListeners(changed);
  }, [nodes, expanded]);
  return <Context value={store}>{children}</Context>;
};

export interface UseNodeReturn extends NodeState {
  onExpand: (key: ranger.Key, expanded: boolean) => void;
}

/** @returns the place of the range in the tree. Renders again only when it changes. */
export const useNode = (key: ranger.Key): UseNodeReturn => {
  const store = useContext("Tree.useNode");
  const state = useSyncExternalStore(
    (listener) => store.subscribe(listener, key),
    () => store.get(key),
  );
  const { more } = store;
  useEffect(() => {
    if (state.tail != null) more(state.tail);
  }, [state.tail, more]);
  return { ...state, onExpand: store.onExpand };
};

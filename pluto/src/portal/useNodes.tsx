// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, type RefObject, useRef } from "react";

import { useSyncedRef } from "@/hooks";
import { In, Node } from "@/portal/Portal";

export interface UseNodesProps {
  /**
   * keys enumerates the portals to keep alive. Each key gets one stable
   * {@link Node} that survives re-renders and re-parenting; nodes for keys that
   * disappear are released.
   */
  keys: string[];
  /**
   * attrs are set as attributes on each node's element when it is created.
   * Changing them later has no effect on existing nodes.
   */
  attrs?: Record<string, string>;
  /**
   * onClick is invoked with a node's key when its element is clicked. Clicks
   * inside a portaled element do not bubble to the React tree that renders the
   * hosting Out, so the handler is bound on the element itself.
   */
  onClick?: (key: string) => void;
  /** children renders the content portaled into the node with the given key. */
  children: (key: string) => ReactNode;
}

export type UseNodesReturn = [RefObject<Map<string, Node>>, ReactElement[]];

/**
 * useNodes maintains one stable portal {@link Node} per key. The returned
 * elements render children into their nodes and must be mounted somewhere in the
 * tree; the returned ref maps each key to its node for hosting via Out. Because
 * a node's element is moved rather than recreated when its Out re-parents, the
 * portaled content keeps its state (DOM, WebGL contexts) across moves.
 */
export const useNodes = ({
  keys,
  attrs,
  onClick,
  children,
}: UseNodesProps): UseNodesReturn => {
  const ref = useRef<Map<string, Node>>(new Map());
  const onClickRef = useSyncedRef(onClick);
  const nodes = keys.map((key) => {
    let node = ref.current.get(key);
    if (node == null) {
      node = new Node(attrs);
      node.el.addEventListener("click", () => onClickRef.current?.(key));
      ref.current.set(key, node);
    }
    return (
      <In key={key} node={node}>
        {children(key)}
      </In>
    );
  });
  const alive = new Set(keys);
  ref.current.forEach((_, key) => {
    if (!alive.has(key)) ref.current.delete(key);
  });
  return [ref, nodes];
};

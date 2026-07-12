// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, type ReactNode, useLayoutEffect, useRef } from "react";
import { createPortal } from "react-dom";

import { type Node } from "@/portal/Node";

export interface OutProps {
  /** node is the portal node whose element is hosted where this Out renders. */
  node: Node;
}

/**
 * Out hosts the given node's element at its own position in the DOM. This is the
 * node-handle predecessor of the key-addressed {@link Out}; the composed Mosaic
 * that owns the node registry is the last consumer of it.
 */
export const Out = ({ node }: OutProps): ReactElement => {
  const stub = useRef<HTMLDivElement>(null);
  const portal = useRef<Node>(node);
  useLayoutEffect(() => {
    const placeholder = stub.current;
    if (placeholder == null) return;
    const parent = placeholder.parentNode;
    if (parent == null) return;
    node.mount(parent, placeholder);
    return () => {
      if (stub.current != null) portal.current.unmount(stub.current);
    };
  }, []);
  useLayoutEffect(() => {
    if (portal.current != null && node !== portal.current) {
      portal.current.unmount(stub.current);
      portal.current = node;
    }
    const placeholder = stub.current;
    if (placeholder == null) return;
    const parent = placeholder.parentNode;
    if (parent == null) return;
    node.mount(parent, placeholder);
  }, [node]);
  return <div ref={stub} />;
};

export interface InProps extends OutProps {
  /** children renders into the node's detached element. */
  children: ReactNode;
}

/**
 * In renders children into the given node's detached element. This is the
 * node-handle predecessor of the key-addressed {@link In}.
 */
export const In = ({ node, children }: InProps): ReactElement =>
  createPortal(children, node.el);

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createRef, useEffect, useRef } from "react";
import { createPortal } from "react-dom";

type NodeProps = Record<string, string>;

export class Node {
  parent: ParentNode | undefined;
  stub: HTMLElement | undefined;
  el: HTMLElement;

  constructor(props: NodeProps = {}) {
    this.el = document.createElement("div");
    Object.entries(props).forEach(([k, v]) => this.el.setAttribute(k, v));
  }

  mount(to: ParentNode, stub: HTMLElement): void {
    if (stub === this.stub) return;
    this.unmount(null);
    to.replaceChild(this.el, stub);
    this.parent = to;
    this.stub = stub;
  }

  unmount(stub: HTMLElement | null): void {
    // Skip unmounts for placeholders that aren't currently mounted
    // They will have been automatically unmounted already by a subsequent mount()
    if (
      (stub != null && stub !== this.stub) ||
      this.parent == null ||
      this.stub == null
    )
      return;
    this.parent.replaceChild(this.stub, this.el);
    this.parent = undefined;
    this.stub = undefined;
  }
}

export interface OutProps {
  node: Node;
}

export const Out = ({ node }: OutProps): React.ReactElement => {
  const stub = createRef<HTMLDivElement>();
  const portal = useRef<Node>(node);
  useEffect(() => {
    const placeholder = stub.current;
    if (placeholder == null) return;
    const parent = placeholder.parentNode;
    if (parent == null) return;
    node.mount(parent, placeholder);
    return () => {
      if (stub.current != null) node.unmount(stub.current);
    };
  }, []);
  useEffect(() => {
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

interface InProps extends OutProps {
  children: React.ReactNode;
}

export const In = ({ node, children }: InProps): React.ReactElement =>
  createPortal(children, node.el);

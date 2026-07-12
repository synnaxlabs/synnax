// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

type NodeProps = Record<string, string>;

/**
 * Node owns a persistent detached element that the legacy In renders content
 * into and the legacy Out hosts in the DOM. Because the element is moved
 * between hosts rather than recreated, its contents (DOM state, WebGL
 * contexts) survive re-parenting.
 */
export class Node {
  /** parent is the element currently hosting el, if any. */
  parent: ParentNode | undefined;
  /** stub is the placeholder el replaced in its current host, if any. */
  stub: HTMLElement | undefined;
  /** el is the persistent element content renders into. */
  el: HTMLElement;

  constructor(props: NodeProps = {}) {
    this.el = document.createElement("div");
    Object.entries(props).forEach(([k, v]) => this.el.setAttribute(k, v));
  }

  /**
   * mount swaps el in for stub under to. Mounting onto the stub el already
   * occupies is a no-op; mounting onto a new stub releases the previous host
   * first.
   */
  mount(to: ParentNode, stub: HTMLElement): void {
    if (stub === this.stub) return;
    this.unmount(null);
    to.replaceChild(this.el, stub);
    this.parent = to;
    this.stub = stub;
  }

  /**
   * unmount restores stub in el's place, releasing el from its host. Pass null
   * to release unconditionally.
   */
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

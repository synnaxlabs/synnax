// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { describe, expect, it } from "vitest";

import { Portal } from "@/portal";

const newStub = (parent: HTMLElement): HTMLElement => {
  const stub = document.createElement("div");
  parent.appendChild(stub);
  return stub;
};

describe("Portal", () => {
  describe("Node", () => {
    it("should apply constructor props as attributes on its element", () => {
      const node = new Portal.Node({ style: "width: 100%;", role: "presentation" });
      expect(node.el.getAttribute("style")).toEqual("width: 100%;");
      expect(node.el.getAttribute("role")).toEqual("presentation");
    });

    it("should replace the stub with its element on mount", () => {
      const parent = document.createElement("div");
      const stub = newStub(parent);
      const node = new Portal.Node();

      node.mount(parent, stub);

      expect(parent.contains(node.el)).toBe(true);
      expect(parent.contains(stub)).toBe(false);
    });

    it("should be a no-op when mounted onto the same stub twice", () => {
      const parent = document.createElement("div");
      const stub = newStub(parent);
      const node = new Portal.Node();

      node.mount(parent, stub);
      node.mount(parent, stub);

      expect(parent.children).toHaveLength(1);
      expect(parent.contains(node.el)).toBe(true);
    });

    it("should restore the previous stub when mounted onto a new one", () => {
      const parentA = document.createElement("div");
      const parentB = document.createElement("div");
      const stubA = newStub(parentA);
      const stubB = newStub(parentB);
      const node = new Portal.Node();

      node.mount(parentA, stubA);
      node.mount(parentB, stubB);

      expect(parentB.contains(node.el)).toBe(true);
      expect(parentA.contains(stubA)).toBe(true);
      expect(parentA.contains(node.el)).toBe(false);
    });

    it("should restore the stub in its element's place on unmount", () => {
      const parent = document.createElement("div");
      const stub = newStub(parent);
      const node = new Portal.Node();

      node.mount(parent, stub);
      node.unmount(stub);

      expect(parent.contains(stub)).toBe(true);
      expect(parent.contains(node.el)).toBe(false);
    });

    it("should ignore an unmount from a stub it is not mounted on", () => {
      const parent = document.createElement("div");
      const stub = newStub(parent);
      const stale = document.createElement("div");
      const node = new Portal.Node();

      node.mount(parent, stub);
      node.unmount(stale);

      expect(parent.contains(node.el)).toBe(true);
    });

    it("should ignore an unmount when not mounted", () => {
      const node = new Portal.Node();
      expect(() => node.unmount(null)).not.toThrow();
    });
  });

  describe("Out", () => {
    it("should host the node's element where it renders", () => {
      const node = new Portal.Node();
      const { container } = render(<Portal.Out node={node} />);
      expect(container.contains(node.el)).toBe(true);
    });

    it("should release the node's element when unmounted", () => {
      const node = new Portal.Node();
      const { container, unmount } = render(<Portal.Out node={node} />);
      unmount();
      expect(node.el.parentNode).toBeNull();
      expect(container.childNodes).toHaveLength(0);
    });

    it("should swap elements when the node prop changes", () => {
      const first = new Portal.Node();
      const second = new Portal.Node();
      const { container, rerender } = render(<Portal.Out node={first} />);
      expect(container.contains(first.el)).toBe(true);

      rerender(<Portal.Out node={second} />);

      expect(container.contains(second.el)).toBe(true);
      expect(first.el.parentNode).toBeNull();
    });

    it("should release the current node when unmounted after a swap", () => {
      const first = new Portal.Node();
      const second = new Portal.Node();
      const { container, rerender, unmount } = render(<Portal.Out node={first} />);
      rerender(<Portal.Out node={second} />);

      unmount();

      expect(second.el.parentNode).toBeNull();
      expect(container.childNodes).toHaveLength(0);
    });
  });

  describe("In", () => {
    it("should render children into the node's element while detached", () => {
      const node = new Portal.Node();
      render(<Portal.In node={node}>detached content</Portal.In>);
      expect(node.el.textContent).toEqual("detached content");
      expect(screen.queryByText("detached content")).toBeNull();
    });

    it("should make children visible once an Out hosts the node", () => {
      const node = new Portal.Node();
      render(
        <>
          <Portal.In node={node}>hosted content</Portal.In>
          <Portal.Out node={node} />
        </>,
      );
      expect(screen.getByText("hosted content")).toBeTruthy();
    });
  });

  describe("content lifetime", () => {
    interface LayoutProps {
      slot: "a" | "b";
      node: Portal.Node;
      children: React.ReactNode;
    }

    // Layout mirrors the mosaic's usage: content renders into the node once via
    // In, while the hosting Out moves between regions as the tree restructures.
    const Layout = ({ slot, node, children }: LayoutProps): ReactElement => (
      <>
        <Portal.In node={node}>{children}</Portal.In>
        <section aria-label="region a">
          {slot === "a" && <Portal.Out node={node} />}
        </section>
        <section aria-label="region b">
          {slot === "b" && <Portal.Out node={node} />}
        </section>
      </>
    );

    const Counter = (): ReactElement => {
      const [count, setCount] = useState(0);
      return <button onClick={() => setCount((c) => c + 1)}>{`count:${count}`}</button>;
    };

    it("should move content between hosts without losing React state", () => {
      const node = new Portal.Node();
      const { rerender } = render(
        <Layout slot="a" node={node}>
          <Counter />
        </Layout>,
      );

      fireEvent.click(screen.getByRole("button"));
      expect(screen.getByText("count:1")).toBeTruthy();
      expect(screen.getByRole("region", { name: "region a" }).contains(node.el)).toBe(
        true,
      );

      rerender(
        <Layout slot="b" node={node}>
          <Counter />
        </Layout>,
      );

      expect(screen.getByText("count:1")).toBeTruthy();
      expect(screen.getByRole("region", { name: "region b" }).contains(node.el)).toBe(
        true,
      );
    });

    it("should leave the active host in place when a stale Out unmounts", () => {
      const node = new Portal.Node();
      const Both = ({ withA }: { withA: boolean }): ReactElement => (
        <>
          <section aria-label="region a">{withA && <Portal.Out node={node} />}</section>
          <section aria-label="region b">
            <Portal.Out node={node} />
          </section>
        </>
      );
      // Region b's Out mounts last, so it hosts the node.
      const { rerender } = render(<Both withA />);
      expect(screen.getByRole("region", { name: "region b" }).contains(node.el)).toBe(
        true,
      );

      rerender(<Both withA={false} />);

      expect(screen.getByRole("region", { name: "region b" }).contains(node.el)).toBe(
        true,
      );
    });
  });
});

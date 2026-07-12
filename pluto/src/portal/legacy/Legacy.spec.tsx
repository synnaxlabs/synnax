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

describe("Portal.Legacy", () => {
  describe("Out", () => {
    it("should host the node's element where it renders", () => {
      const node = new Portal.Node();
      const { container } = render(<Portal.Legacy.Out node={node} />);
      expect(container.contains(node.el)).toBe(true);
    });

    it("should release the node's element when unmounted", () => {
      const node = new Portal.Node();
      const { container, unmount } = render(<Portal.Legacy.Out node={node} />);
      unmount();
      expect(node.el.parentNode).toBeNull();
      expect(container.childNodes).toHaveLength(0);
    });

    it("should swap elements when the node prop changes", () => {
      const first = new Portal.Node();
      const second = new Portal.Node();
      const { container, rerender } = render(<Portal.Legacy.Out node={first} />);
      expect(container.contains(first.el)).toBe(true);

      rerender(<Portal.Legacy.Out node={second} />);

      expect(container.contains(second.el)).toBe(true);
      expect(first.el.parentNode).toBeNull();
    });

    it("should release the current node when unmounted after a swap", () => {
      const first = new Portal.Node();
      const second = new Portal.Node();
      const { container, rerender, unmount } = render(
        <Portal.Legacy.Out node={first} />,
      );
      rerender(<Portal.Legacy.Out node={second} />);

      unmount();

      expect(second.el.parentNode).toBeNull();
      expect(container.childNodes).toHaveLength(0);
    });
  });

  describe("In", () => {
    it("should render children into the node's element while detached", () => {
      const node = new Portal.Node();
      render(<Portal.Legacy.In node={node}>detached content</Portal.Legacy.In>);
      expect(node.el.textContent).toEqual("detached content");
      expect(screen.queryByText("detached content")).toBeNull();
    });

    it("should make children visible once an Out hosts the node", () => {
      const node = new Portal.Node();
      render(
        <>
          <Portal.Legacy.In node={node}>hosted content</Portal.Legacy.In>
          <Portal.Legacy.Out node={node} />
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

    const Layout = ({ slot, node, children }: LayoutProps): ReactElement => (
      <>
        <Portal.Legacy.In node={node}>{children}</Portal.Legacy.In>
        <section aria-label="region a">
          {slot === "a" && <Portal.Legacy.Out node={node} />}
        </section>
        <section aria-label="region b">
          {slot === "b" && <Portal.Legacy.Out node={node} />}
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
          <section aria-label="region a">
            {withA && <Portal.Legacy.Out node={node} />}
          </section>
          <section aria-label="region b">
            <Portal.Legacy.Out node={node} />
          </section>
        </>
      );
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

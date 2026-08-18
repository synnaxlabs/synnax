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
import { describe, expect, it, vi } from "vitest";

import { Portal } from "@/portal";

interface HarnessProps {
  keys: string[];
  host?: string | null;
  onClick?: (key: string) => void;
}

const Harness = ({ keys, host, onClick }: HarnessProps): ReactElement => (
  <Portal.Context>
    {keys.map((key) => (
      <Portal.In key={key} itemKey={key}>
        <p>content-{key}</p>
      </Portal.In>
    ))}
    <section aria-label="host">
      {host !== undefined && (
        <Portal.Out itemKey={host} onClick={onClick && (() => onClick(host ?? ""))} />
      )}
    </section>
  </Portal.Context>
);

describe("Portal", () => {
  describe("In", () => {
    it("should render children detached while no Out hosts the key", () => {
      render(<Harness keys={["a"]} />);
      expect(screen.queryByText("content-a")).toBeNull();
    });

    it("should throw when rendered outside a Context", () => {
      expect(() => render(<Portal.In itemKey="a">content</Portal.In>)).toThrow(
        "Portal.In must be used within Portal.Context",
      );
    });
  });

  describe("Out", () => {
    it("should host the content registered under its key", () => {
      render(<Harness keys={["a", "b"]} host="a" />);
      const host = screen.getByRole("region", { name: "host" });
      expect(host.contains(screen.getByText("content-a"))).toBe(true);
      expect(screen.queryByText("content-b")).toBeNull();
    });

    it("should host content regardless of mount order", () => {
      // The Out renders before the In in tree order, so it resolves the node
      // only after the In registers it within the same commit.
      render(
        <Portal.Context>
          <Portal.Out itemKey="a" />
          <Portal.In itemKey="a">late content</Portal.In>
        </Portal.Context>,
      );
      expect(screen.getByText("late content")).toBeTruthy();
    });

    it("should render an empty placeholder when the key is null", () => {
      render(<Harness keys={["a"]} host={null} />);
      const host = screen.getByRole("region", { name: "host" });
      expect(host.textContent).toEqual("");
    });

    it("should render an empty placeholder for an unregistered key", () => {
      render(<Harness keys={["a"]} host="missing" />);
      const host = screen.getByRole("region", { name: "host" });
      expect(host.textContent).toEqual("");
    });

    it("should swap content when the key changes", () => {
      const { rerender } = render(<Harness keys={["a", "b"]} host="a" />);
      expect(screen.getByText("content-a")).toBeTruthy();

      rerender(<Harness keys={["a", "b"]} host="b" />);

      expect(screen.getByText("content-b")).toBeTruthy();
      expect(screen.queryByText("content-a")).toBeNull();
    });

    it("should release the content when unmounted", () => {
      const { rerender } = render(<Harness keys={["a"]} host="a" />);
      rerender(<Harness keys={["a"]} />);
      expect(screen.queryByText("content-a")).toBeNull();
    });

    it("should release the content when its In unmounts while hosted", () => {
      const { rerender } = render(<Harness keys={["a"]} host="a" />);
      expect(screen.getByText("content-a")).toBeTruthy();

      rerender(<Harness keys={[]} host="a" />);

      expect(screen.queryByText("content-a")).toBeNull();
    });

    it("should forward remaining props to its host element", () => {
      render(
        <Portal.Context>
          <Portal.In itemKey="a">content</Portal.In>
          <Portal.Out itemKey="a" role="presentation" style={{ width: "100%" }} />
        </Portal.Context>,
      );
      const host = screen.getByRole("presentation");
      expect(host.style.width).toEqual("100%");
      expect(host.contains(screen.getByText("content"))).toBe(true);
    });

    it("should throw when rendered outside a Context", () => {
      expect(() => render(<Portal.Out itemKey="a" />)).toThrow(
        "Portal.Out must be used within Portal.Context",
      );
    });
  });

  describe("content lifetime", () => {
    interface LayoutProps {
      slot: "a" | "b";
      children: React.ReactNode;
    }

    // Layout mirrors the mosaic's usage: content renders into its key once via
    // In, while the hosting Out moves between regions as the tree restructures.
    const Layout = ({ slot, children }: LayoutProps): ReactElement => (
      <Portal.Context>
        <Portal.In itemKey="tab">{children}</Portal.In>
        <section aria-label="region a">
          {slot === "a" && <Portal.Out itemKey="tab" />}
        </section>
        <section aria-label="region b">
          {slot === "b" && <Portal.Out itemKey="tab" />}
        </section>
      </Portal.Context>
    );

    const Counter = (): ReactElement => {
      const [count, setCount] = useState(0);
      return <button onClick={() => setCount((c) => c + 1)}>{`count:${count}`}</button>;
    };

    it("should move content between hosts without losing React state", () => {
      const { rerender } = render(
        <Layout slot="a">
          <Counter />
        </Layout>,
      );

      fireEvent.click(screen.getByRole("button"));
      const counter = screen.getByText("count:1");
      expect(screen.getByRole("region", { name: "region a" }).contains(counter)).toBe(
        true,
      );

      rerender(
        <Layout slot="b">
          <Counter />
        </Layout>,
      );

      expect(screen.getByText("count:1")).toBe(counter);
      expect(screen.getByRole("region", { name: "region b" }).contains(counter)).toBe(
        true,
      );
    });

    it("should leave the active host in place when a stale Out unmounts", () => {
      const Both = ({ withA }: { withA: boolean }): ReactElement => (
        <Portal.Context>
          <Portal.In itemKey="tab">content</Portal.In>
          <section aria-label="region a">
            {withA && <Portal.Out itemKey="tab" />}
          </section>
          <section aria-label="region b">
            <Portal.Out itemKey="tab" />
          </section>
        </Portal.Context>
      );
      // Region b's Out mounts last, so it hosts the content.
      const { rerender } = render(<Both withA />);
      expect(
        screen
          .getByRole("region", { name: "region b" })
          .contains(screen.getByText("content")),
      ).toBe(true);

      rerender(<Both withA={false} />);

      expect(
        screen
          .getByRole("region", { name: "region b" })
          .contains(screen.getByText("content")),
      ).toBe(true);
    });
  });

  describe("onClick", () => {
    it("should invoke the Out onClick when the hosted content is clicked", () => {
      const onClick = vi.fn();
      render(<Harness keys={["a"]} host="a" onClick={onClick} />);
      fireEvent.click(screen.getByText("content-a"));
      expect(onClick).toHaveBeenCalledTimes(1);
    });

    it("should invoke the latest onClick handler after re-renders", () => {
      const first = vi.fn();
      const second = vi.fn();
      const { rerender } = render(<Harness keys={["a"]} host="a" onClick={first} />);
      rerender(<Harness keys={["a"]} host="a" onClick={second} />);
      fireEvent.click(screen.getByText("content-a"));
      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledTimes(1);
    });

    it("should fire onClickCapture before an inner click handler so the host acts first", () => {
      const order: string[] = [];
      const Inner = (): ReactElement => (
        <Portal.Context>
          <Portal.In itemKey="a">
            <button onClick={() => order.push("navigate")}>go</button>
          </Portal.In>
          <Portal.Out itemKey="a" onClickCapture={() => order.push("select")} />
        </Portal.Context>
      );
      render(<Inner />);
      fireEvent.click(screen.getByRole("button", { name: "go" }));
      expect(order).toEqual(["select", "navigate"]);
    });
  });
});

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

const newStub = (parent: HTMLElement): HTMLElement => {
  const stub = document.createElement("div");
  parent.appendChild(stub);
  return stub;
};

interface HarnessProps {
  keys: string[];
  host?: string | null;
  attrs?: Record<string, string>;
  onClick?: (key: string) => void;
}

const Harness = ({ keys, host, attrs, onClick }: HarnessProps): ReactElement => (
  <Portal.Provider>
    {keys.map((key) => (
      <Portal.In key={key} itemKey={key} attrs={attrs} onClick={onClick}>
        <p>content-{key}</p>
      </Portal.In>
    ))}
    <section aria-label="host">
      {host !== undefined && <Portal.Out itemKey={host} />}
    </section>
  </Portal.Provider>
);

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

  describe("In", () => {
    it("should render children detached while no Out hosts the key", () => {
      render(<Harness keys={["a"]} />);
      expect(screen.queryByText("content-a")).toBeNull();
    });

    it("should apply attrs to the content's element", () => {
      render(<Harness keys={["a"]} host="a" attrs={{ style: "width: 100%;" }} />);
      const el = screen.getByText("content-a").parentElement;
      expect(el?.getAttribute("style")).toEqual("width: 100%;");
    });

    it("should throw when rendered outside a Provider", () => {
      expect(() => render(<Portal.In itemKey="a">content</Portal.In>)).toThrow(
        "Portal.In must be used within Portal.Provider",
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
        <Portal.Provider>
          <Portal.Out itemKey="a" />
          <Portal.In itemKey="a">late content</Portal.In>
        </Portal.Provider>,
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

    it("should throw when rendered outside a Provider", () => {
      expect(() => render(<Portal.Out itemKey="a" />)).toThrow(
        "Portal.Out must be used within Portal.Provider",
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
      <Portal.Provider>
        <Portal.In itemKey="tab">{children}</Portal.In>
        <section aria-label="region a">
          {slot === "a" && <Portal.Out itemKey="tab" />}
        </section>
        <section aria-label="region b">
          {slot === "b" && <Portal.Out itemKey="tab" />}
        </section>
      </Portal.Provider>
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
        <Portal.Provider>
          <Portal.In itemKey="tab">content</Portal.In>
          <section aria-label="region a">
            {withA && <Portal.Out itemKey="tab" />}
          </section>
          <section aria-label="region b">
            <Portal.Out itemKey="tab" />
          </section>
        </Portal.Provider>
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
    it("should invoke onClick with the key when the hosted content is clicked", () => {
      const onClick = vi.fn();
      render(<Harness keys={["a"]} host="a" onClick={onClick} />);
      fireEvent.click(screen.getByText("content-a"));
      expect(onClick).toHaveBeenCalledTimes(1);
      expect(onClick).toHaveBeenCalledWith("a");
    });

    it("should invoke the latest onClick handler after re-renders", () => {
      const first = vi.fn();
      const second = vi.fn();
      const { rerender } = render(<Harness keys={["a"]} host="a" onClick={first} />);
      rerender(<Harness keys={["a"]} host="a" onClick={second} />);
      fireEvent.click(screen.getByText("content-a"));
      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledWith("a");
    });
  });
});

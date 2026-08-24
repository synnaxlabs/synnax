// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { Position as RFPosition, ReactFlowProvider } from "@xyflow/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Handle } from "@/schematic/node/common/handle";

const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <ReactFlowProvider>{children}</ReactFlowProvider>
);

const renderHandle = (props: Partial<Handle.HandleProps> = {}) =>
  render(
    <Handle.Handle
      id="h1"
      orientation="left"
      location="right"
      top={50}
      left={50}
      {...props}
    />,
    { wrapper: Wrapper },
  );

const queryHandle = (container: HTMLElement): HTMLElement | null =>
  container.querySelector(".react-flow__handle");

describe("Handle.Handle", () => {
  describe("position prop wiring", () => {
    it("should derive data-handlepos from location and orientation", () => {
      // smart(location="right", orientation="left") = Right; default swap
      // bypasses the flip, so the rendered handle sits on the right edge.
      const { container } = renderHandle({
        orientation: "left",
        location: "right",
      });
      expect(queryHandle(container)?.getAttribute("data-handlepos")).toBe(
        RFPosition.Right,
      );
    });

    it("should flip the derived position when swap=true", () => {
      const { container } = renderHandle({
        orientation: "left",
        location: "right",
        swap: true,
      });
      expect(queryHandle(container)?.getAttribute("data-handlepos")).toBe(
        RFPosition.Left,
      );
    });

    it("should rotate the handle position with the node orientation", () => {
      const cases: Array<[location.Outer, location.Outer, RFPosition]> = [
        ["left", "left", RFPosition.Left],
        ["right", "left", RFPosition.Right],
        ["top", "left", RFPosition.Top],
        ["bottom", "left", RFPosition.Bottom],
        ["right", "top", RFPosition.Top],
        ["left", "top", RFPosition.Bottom],
        ["right", "right", RFPosition.Left],
        ["left", "right", RFPosition.Right],
      ];
      for (const [loc, orientation, expected] of cases) {
        const { container, unmount } = renderHandle({ location: loc, orientation });
        expect(queryHandle(container)?.getAttribute("data-handlepos")).toBe(expected);
        unmount();
      }
    });

    it("should compose orientation rotation with the swap flip", () => {
      // For orientation="top", smart("right", "top") = Top; swap=true flips
      // it to Bottom. Verifies the two transforms compose, not just one.
      const { container } = renderHandle({
        orientation: "top",
        location: "right",
        swap: true,
      });
      expect(queryHandle(container)?.getAttribute("data-handlepos")).toBe(
        RFPosition.Bottom,
      );
    });
  });

  describe("inline style", () => {
    it("should map top and left into percentage CSS for the left orientation", () => {
      const { container } = renderHandle({
        orientation: "left",
        top: 25,
        left: 75,
      });
      const el = queryHandle(container) as HTMLElement;
      expect(el.style.top).toBe("25%");
      expect(el.style.left).toBe("75%");
    });

    it("should reflect coordinates through (50, 50) for the right orientation", () => {
      const { container } = renderHandle({
        orientation: "right",
        top: 25,
        left: 75,
      });
      const el = queryHandle(container) as HTMLElement;
      expect(el.style.top).toBe("75%");
      expect(el.style.left).toBe("25%");
    });

    it("should bypass orientation adjustment when preventAutoAdjust=true", () => {
      const { container } = renderHandle({
        orientation: "right",
        preventAutoAdjust: true,
        top: 10,
        left: 90,
      });
      const el = queryHandle(container) as HTMLElement;
      expect(el.style.top).toBe("10%");
      expect(el.style.left).toBe("90%");
    });

    it("should let caller-supplied style override the computed positioning", () => {
      const { container } = renderHandle({
        style: { background: "red", top: "999%" },
      });
      const el = queryHandle(container) as HTMLElement;
      // The component spreads the caller's style after { left, top },
      // so a top in the caller's style wins.
      expect(el.style.top).toBe("999%");
      expect(el.style.background).toBe("red");
    });
  });

  describe("identity attributes", () => {
    it("should embed the id in the BEM handle class", () => {
      const { container } = renderHandle({ id: "intake" });
      expect(queryHandle(container)?.className).toContain("pluto-handle__intake");
    });

    it("should propagate the id to data-handleid on the rendered handle", () => {
      const { container } = renderHandle({ id: "intake" });
      expect(queryHandle(container)?.getAttribute("data-handleid")).toBe("intake");
    });

    it("should always render with type=source", () => {
      const { container } = renderHandle();
      // xyflow encodes type into data-id as `<rfId>-<nodeId>-<handleId>-<type>`.
      expect(queryHandle(container)?.getAttribute("data-id")).toContain("-source");
    });

    it("should mark the handle as source via the source CSS class", () => {
      const { container } = renderHandle();
      expect(queryHandle(container)?.className).toContain("source");
    });
  });
});

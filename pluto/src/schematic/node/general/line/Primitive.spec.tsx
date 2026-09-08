// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Line } from "@/schematic/node/general/line/Primitive";

const getRoot = (container: HTMLElement): SVGElement => {
  const el = container.querySelector<SVGElement>("svg");
  if (el == null) throw new Error("expected a line svg");
  return el;
};

const getLines = (container: HTMLElement): SVGLineElement[] =>
  Array.from(container.querySelectorAll<SVGLineElement>("line"));

describe("line symbol", () => {
  describe("color", () => {
    // The stroke is painted from the display var in line.css; jsdom cannot compute it,
    // so we assert the marker class and the source var.
    it("should carry the symbol-colored class and set the source color", () => {
      const { container } = render(<Line color="#ff0000" />);
      const root = getRoot(container);
      expect(root.getAttribute("class")).toContain("pluto-symbol-colored");
      expect(root.style.getPropertyValue("--pluto-symbol-color")).toBe("255, 0, 0, 1");
    });

    it("should carry the alpha channel of a translucent color", () => {
      const { container } = render(<Line color={[255, 0, 0, 0.5]} />);
      expect(getRoot(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
        "255, 0, 0, 0.5",
      );
    });

    it("should leave the source color unset for the ZERO sentinel", () => {
      const { container } = render(<Line color={color.ZERO} />);
      expect(getRoot(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
        "",
      );
    });

    it("should leave the source color unset when no color is given", () => {
      const { container } = render(<Line />);
      expect(getRoot(container).style.getPropertyValue("--pluto-symbol-color")).toBe(
        "",
      );
    });
  });

  describe("class", () => {
    it("should carry the line class", () => {
      const { container } = render(<Line />);
      expect(getRoot(container).getAttribute("class")).toContain("pluto-line");
    });

    it("should merge a given class name", () => {
      const { container } = render(<Line className="custom" />);
      const cls = getRoot(container).getAttribute("class") ?? "";
      expect(cls).toContain("custom");
      expect(cls).toContain("pluto-line");
    });
  });

  describe("size", () => {
    it("should size the svg to the far endpoint", () => {
      const { container } = render(
        <Line start={{ x: 0, y: 30 }} end={{ x: 80, y: 0 }} />,
      );
      const root = getRoot(container);
      expect(root.getAttribute("width")).toBe("80");
      expect(root.getAttribute("height")).toBe("30");
    });

    it("should keep a horizontal line at least 1px tall", () => {
      const { container } = render(
        <Line start={{ x: 0, y: 0 }} end={{ x: 80, y: 0 }} />,
      );
      expect(getRoot(container).getAttribute("height")).toBe("1");
    });

    it("should keep a vertical line at least 1px wide", () => {
      const { container } = render(
        <Line start={{ x: 0, y: 0 }} end={{ x: 0, y: 80 }} />,
      );
      expect(getRoot(container).getAttribute("width")).toBe("1");
    });

    it("should draw a rising preview line without endpoints", () => {
      const { container } = render(<Line />);
      const root = getRoot(container);
      expect(root.getAttribute("width")).toBe("40");
      expect(root.getAttribute("height")).toBe("20");
      const [, visible] = getLines(container);
      expect(visible.getAttribute("y1")).toBe("20");
      expect(visible.getAttribute("y2")).toBe("0");
    });

    it("should size to the far endpoint when both ends are off the origin", () => {
      const { container } = render(
        <Line start={{ x: 30, y: 0 }} end={{ x: 0, y: 50 }} />,
      );
      const root = getRoot(container);
      expect(root.getAttribute("width")).toBe("30");
      expect(root.getAttribute("height")).toBe("50");
    });
  });

  describe("lines", () => {
    it("should draw the hit band and visible line between the same endpoints", () => {
      const { container } = render(
        <Line start={{ x: 0, y: 30 }} end={{ x: 80, y: 0 }} />,
      );
      const [hit, visible] = getLines(container);
      expect(hit.getAttribute("class")).toBe("pluto-line__hit");
      for (const line of [hit, visible]) {
        expect(line.getAttribute("x1")).toBe("0");
        expect(line.getAttribute("y1")).toBe("30");
        expect(line.getAttribute("x2")).toBe("80");
        expect(line.getAttribute("y2")).toBe("0");
      }
    });

    it("should set the stroke width on the visible line only", () => {
      const { container } = render(<Line strokeWidth={5} />);
      const [hit, visible] = getLines(container);
      expect(visible.getAttribute("stroke-width")).toBe("5");
      expect(hit.getAttribute("stroke-width")).toBeNull();
    });

    it("should default the stroke width to 2", () => {
      const { container } = render(<Line />);
      expect(getLines(container)[1].getAttribute("stroke-width")).toBe("2");
    });
  });
});

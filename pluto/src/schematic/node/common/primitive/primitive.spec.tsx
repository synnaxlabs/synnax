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

import { Primitive } from "@/schematic/node/common/primitive";

const renderInSVG = (children: React.ReactNode): HTMLElement =>
  render(<svg>{children}</svg>).container;

describe("Primitive shape wrappers", () => {
  it("should always inject vectorEffect=non-scaling-stroke on Path", () => {
    const c = renderInSVG(<Primitive.Path d="M0 0 L1 1" />);
    expect(c.querySelector("path")?.getAttribute("vector-effect")).toBe(
      "non-scaling-stroke",
    );
  });

  it("should always inject vectorEffect=non-scaling-stroke on Rect", () => {
    const c = renderInSVG(<Primitive.Rect width="10" height="10" />);
    expect(c.querySelector("rect")?.getAttribute("vector-effect")).toBe(
      "non-scaling-stroke",
    );
  });

  it("should always inject vectorEffect=non-scaling-stroke on Circle", () => {
    const c = renderInSVG(<Primitive.Circle r="5" />);
    expect(c.querySelector("circle")?.getAttribute("vector-effect")).toBe(
      "non-scaling-stroke",
    );
  });

  it("should always inject vectorEffect=non-scaling-stroke on Line", () => {
    const c = renderInSVG(<Primitive.Line x1="0" y1="0" x2="10" y2="10" />);
    expect(c.querySelector("line")?.getAttribute("vector-effect")).toBe(
      "non-scaling-stroke",
    );
  });

  it("should let caller-supplied vectorEffect override the injected default", () => {
    // {...props} is spread after vectorEffect, so a caller can intentionally
    // turn off non-scaling-stroke if they want a hairline that scales with the
    // viewport.
    const c = renderInSVG(<Primitive.Path d="M0 0 L1 1" vectorEffect="none" />);
    expect(c.querySelector("path")?.getAttribute("vector-effect")).toBe("none");
  });

  it("should forward arbitrary SVG props through to the underlying element", () => {
    const c = renderInSVG(<Primitive.Rect width="20" height="30" fill="#abcdef" />);
    const rect = c.querySelector("rect");
    expect(rect?.getAttribute("width")).toBe("20");
    expect(rect?.getAttribute("height")).toBe("30");
    expect(rect?.getAttribute("fill")).toBe("#abcdef");
  });
});

describe("Primitive.SVG", () => {
  describe("dimensions and aspect ratio", () => {
    it("should pass dimensions through unchanged for horizontal orientations", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 40, height: 80 }} orientation="left" />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("viewBox")).toBe("0 0 40 80");
      expect(svg.style.aspectRatio).toBe("40 / 80");
    });

    it("should swap width and height for vertical orientations", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 40, height: 80 }} orientation="top" />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("viewBox")).toBe("0 0 80 40");
      expect(svg.style.aspectRatio).toBe("80 / 40");
    });

    it("should swap dimensions for the bottom orientation as well", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 100, height: 30 }} orientation="bottom" />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("viewBox")).toBe("0 0 30 100");
    });

    it("should pass through dimensions for the right orientation (horizontal)", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 40, height: 80 }} orientation="right" />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("viewBox")).toBe("0 0 40 80");
    });
  });

  describe("scaling", () => {
    it("should compute width as scale * BASE_SCALE * dim.width by default", () => {
      // BASE_SCALE = 0.8, default scale = 1, so width is 40 * 0.8 = 32.
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 40, height: 80 }} />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.style.width).toBe("32px");
    });

    it("should multiply width by an explicit scale factor", () => {
      // scale = 2 -> 40 * 0.8 * 2 = 64.
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 40, height: 80 }} scale={2} />,
      );
      expect((container.querySelector("svg") as SVGSVGElement).style.width).toBe(
        "64px",
      );
    });

    it("should scale the post-swap dimension for vertical orientations", () => {
      // orientation=top swaps to (80, 40), then BASE_SCALE * width = 64.
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 40, height: 80 }} orientation="top" />,
      );
      expect((container.querySelector("svg") as SVGSVGElement).style.width).toBe(
        "64px",
      );
    });
  });

  describe("color handling", () => {
    it("should leave fill and stroke unset when no color is provided", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }} />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("fill")).toBeNull();
      expect(svg.getAttribute("stroke")).toBeNull();
    });

    it("should set the symbol-color variable and marker class for a color", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }} color="#ff0000" />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      // Fill and stroke are driven by CSS off --pluto-symbol-color, not attributes.
      expect(svg.style.getPropertyValue("--pluto-symbol-color")).toMatch(
        /255\s*,\s*0\s*,\s*0/,
      );
      expect(svg.getAttribute("class")).toContain("pluto-symbol-colored");
    });

    it("should treat the ZERO sentinel as unset so it falls back to the theme", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }} color={color.ZERO} />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.style.getPropertyValue("--pluto-symbol-color")).toBe("");
      expect(svg.getAttribute("class")).toContain("pluto-symbol-colored");
    });
  });

  describe("class and structure", () => {
    it("should encode the orientation as a location class", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }} orientation="top" />,
      );
      expect(container.querySelector("svg")?.getAttribute("class")).toContain(
        "pluto--location-top",
      );
    });

    it("should preserve user-supplied className alongside the location class", () => {
      const { container } = render(
        <Primitive.SVG
          dimensions={{ width: 10, height: 10 }}
          orientation="left"
          className="user-cls"
        />,
      );
      expect(container.querySelector("svg")?.getAttribute("class")).toContain(
        "user-cls",
      );
    });

    it("should always wrap children in a single <g>", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }}>
          <rect data-testid="child" />
        </Primitive.SVG>,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.children).toHaveLength(1);
      expect(svg.children[0].tagName.toLowerCase()).toBe("g");
      expect(
        svg
          .querySelector('[data-testid="child"]')
          ?.parentElement?.tagName.toLowerCase(),
      ).toBe("g");
    });

    it("should set the SVG xmlns attribute", () => {
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }} />,
      );
      expect(container.querySelector("svg")?.getAttribute("xmlns")).toBe(
        "http://www.w3.org/2000/svg",
      );
    });
  });

  describe("style merging", () => {
    it("should merge user-supplied style under the computed aspect-ratio and width", () => {
      const { container } = render(
        <Primitive.SVG
          dimensions={{ width: 10, height: 10 }}
          style={{ background: "blue" }}
        />,
      );
      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.style.background).toBe("blue");
      expect(svg.style.aspectRatio).toBe("10 / 10");
    });

    it("should let user-supplied style override the computed width", () => {
      // The component spreads { ...style, aspectRatio, width }, so width and
      // aspectRatio overwrite anything the caller set under those keys. This
      // test pins that ordering so a future refactor that flips it is caught.
      const { container } = render(
        <Primitive.SVG dimensions={{ width: 10, height: 10 }} style={{ width: 999 }} />,
      );
      // 10 * 0.8 * 1 = 8
      expect((container.querySelector("svg") as SVGSVGElement).style.width).toBe("8px");
    });
  });
});

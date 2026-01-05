// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { extractRegions } from "@/schematic/symbol/extractRegions";

describe("extractRegions", () => {
  const TRANSPARENT = color.hex(color.ZERO);

  const createSVG = (content: string): SVGElement => {
    const parser = new DOMParser();
    const doc = parser.parseFromString(
      `<svg xmlns="http://www.w3.org/2000/svg">${content}</svg>`,
      "image/svg+xml",
    );
    return doc.documentElement as unknown as SVGElement;
  };

  it("should extract regions from elements with identical colors", () => {
    const svg = createSVG(`
      <rect style="stroke: #ff0000; fill: #00ff00" />
      <circle style="stroke: #ff0000; fill: #00ff00" />
      <path style="stroke: #0000ff; fill: #ffff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(2);

    // Find region with red stroke and green fill
    const redGreenRegion = regions.find(
      (r) => r.strokeColor === "#ff0000" && r.fillColor === "#00ff00",
    );
    expect(redGreenRegion).toBeDefined();
    expect(redGreenRegion?.selectors).toHaveLength(2);
    expect(redGreenRegion?.selectors).toContain("rect:nth-of-type(1)");
    expect(redGreenRegion?.selectors).toContain("circle:nth-of-type(1)");

    // Find region with blue stroke and yellow fill
    const blueYellowRegion = regions.find(
      (r) => r.strokeColor === "#0000ff" && r.fillColor === "#ffff00",
    );
    expect(blueYellowRegion).toBeDefined();
    expect(blueYellowRegion?.selectors).toHaveLength(1);
    expect(blueYellowRegion?.selectors).toContain("path:nth-of-type(1)");
  });

  it("should treat 'none', undefined, and null as the same", () => {
    const svg = createSVG(`
      <rect style="stroke: none; fill: #00ff00" />
      <circle style="fill: #00ff00" />
      <path style="stroke: none; fill: none" />
      <line />
    `);

    const regions = extractRegions(svg);

    // Should have 2 regions:
    // 1. Elements with no stroke and green fill
    // 2. Elements with no stroke and no fill
    expect(regions).toHaveLength(2);

    const greenFillRegion = regions.find((r) => r.fillColor === "#00ff00");
    expect(greenFillRegion).toBeDefined();
    expect(greenFillRegion?.selectors).toHaveLength(2);
    expect(greenFillRegion?.strokeColor).toBe(TRANSPARENT);

    const noColorRegion = regions.find(
      (r) => r.strokeColor === TRANSPARENT && r.fillColor === TRANSPARENT,
    );
    expect(noColorRegion).toBeDefined();
    expect(noColorRegion?.selectors).toHaveLength(2);
  });

  it("should use element IDs when available", () => {
    const svg = createSVG(`
      <rect id="rect1" style="stroke: #ff0000; fill: #00ff00" />
      <circle id="circle1" style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toContain("#rect1");
    expect(regions[0].selectors).toContain("#circle1");
  });

  it("should use class names when available", () => {
    const svg = createSVG(`
      <rect class="red-rect primary" style="stroke: #ff0000; fill: #00ff00" />
      <circle class="red-circle" style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toContain(".red-rect.primary");
    expect(regions[0].selectors).toContain(".red-circle");
  });

  it("should generate nth-of-type selectors for elements without IDs or classes", () => {
    const svg = createSVG(`
      <rect style="stroke: #ff0000; fill: #00ff00" />
      <rect style="stroke: #ff0000; fill: #00ff00" />
      <circle style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toContain("rect:nth-of-type(1)");
    expect(regions[0].selectors).toContain("rect:nth-of-type(2)");
    expect(regions[0].selectors).toContain("circle:nth-of-type(1)");
  });

  it("should handle nested elements", () => {
    const svg = createSVG(`
      <g>
        <rect style="stroke: #ff0000; fill: #00ff00" />
        <g>
          <circle style="stroke: #ff0000; fill: #00ff00" />
        </g>
      </g>
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toHaveLength(2);
    // Selectors should include parent path
    expect(regions[0].selectors[0]).toContain("rect");
    expect(regions[0].selectors[1]).toContain("circle");
  });

  it("should handle direct attributes as fallback to styles", () => {
    const svg = createSVG(`
      <rect stroke="#ff0000" fill="#00ff00" />
      <circle style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toHaveLength(2);
    expect(regions[0].strokeColor).toBe("#ff0000");
    expect(regions[0].fillColor).toBe("#00ff00");
  });

  it("should handle empty SVG", () => {
    const svg = createSVG("");

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(0);
  });

  it("should handle all visual element types", () => {
    const svg = createSVG(`
      <path style="stroke: #ff0000; fill: #00ff00" />
      <rect style="stroke: #ff0000; fill: #00ff00" />
      <circle style="stroke: #ff0000; fill: #00ff00" />
      <ellipse style="stroke: #ff0000; fill: #00ff00" />
      <polygon style="stroke: #ff0000; fill: #00ff00" />
      <polyline style="stroke: #ff0000; fill: #00ff00" />
      <line style="stroke: #ff0000" />
      <text style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    // Line element has no fill, so it's in a different region
    expect(regions).toHaveLength(2);

    const mainRegion = regions.find((r) => r.fillColor === "#00ff00");
    expect(mainRegion?.selectors).toHaveLength(7);

    const lineRegion = regions.find((r) => r.fillColor === TRANSPARENT);
    expect(lineRegion?.selectors).toHaveLength(1);
    expect(lineRegion?.strokeColor).toBe("#ff0000");
  });

  it("should normalize colors to lowercase", () => {
    const svg = createSVG(`
      <rect style="stroke: #FF0000; fill: #00FF00" />
      <circle style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toHaveLength(2);
    expect(regions[0].strokeColor).toBe("#ff0000");
    expect(regions[0].fillColor).toBe("#00ff00");
  });

  it("should convert all colors to hex format", () => {
    const svg = createSVG(`
      <rect style="stroke: red; fill: blue" />
      <circle style="stroke: rgb(255, 0, 0); fill: rgb(0, 0, 255)" />
      <path style="stroke: #f00; fill: #00f" />
      <line style="stroke: RED; fill: BLUE" />
    `);

    const regions = extractRegions(svg);

    regions.forEach((region) => {
      if (region.strokeColor) expect(region.strokeColor).toMatch(/^#[0-9a-f]{6}$/);
      if (region.fillColor) expect(region.fillColor).toMatch(/^#[0-9a-f]{6}$/);
    });

    const redStrokeBlue = regions.find(
      (r) => r.strokeColor === "#ff0000" && r.fillColor === "#0000ff",
    );
    expect(redStrokeBlue).toBeDefined();
    expect(redStrokeBlue?.selectors).toHaveLength(4);
  });

  it("should use data-region-id attributes when available", () => {
    const svg = createSVG(`
      <rect data-region-id="region-123" style="stroke: #ff0000; fill: #00ff00" />
      <circle data-region-id="region-456" style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toContain('[data-region-id="region-123"]');
    expect(regions[0].selectors).toContain('[data-region-id="region-456"]');
  });

  it("should handle complex nested structures", () => {
    const svg = createSVG(`
      <g id="group1">
        <rect class="shape" style="stroke: #ff0000; fill: #00ff00" />
        <g id="group2">
          <circle style="stroke: #ff0000; fill: #00ff00" />
          <path id="path1" style="stroke: #ff0000; fill: #00ff00" />
        </g>
      </g>
      <rect style="stroke: #0000ff; fill: #ffff00" />
    `);

    const regions = extractRegions(svg);

    expect(regions).toHaveLength(2);

    const redGreenRegion = regions.find((r) => r.strokeColor === "#ff0000");
    expect(redGreenRegion?.selectors).toHaveLength(3);
    expect(redGreenRegion?.selectors).toContain(".shape");
    expect(redGreenRegion?.selectors).toContain("#path1");

    const blueYellowRegion = regions.find((r) => r.strokeColor === "#0000ff");
    expect(blueYellowRegion?.selectors).toHaveLength(1);
  });
});

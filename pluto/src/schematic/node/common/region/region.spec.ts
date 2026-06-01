// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ValidationError } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Region } from "@/schematic/node/common/region";

describe("Region.extract", () => {
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

    const regions = Region.extract(svg);

    expect(regions).toHaveLength(2);

    const redGreenRegion = regions.find(
      (r) => r.strokeColor === "#ff0000" && r.fillColor === "#00ff00",
    );
    expect(redGreenRegion).toBeDefined();
    expect(redGreenRegion?.selectors).toHaveLength(2);
    expect(redGreenRegion?.selectors).toContain("rect:nth-of-type(1)");
    expect(redGreenRegion?.selectors).toContain("circle:nth-of-type(1)");

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

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toContain("#rect1");
    expect(regions[0].selectors).toContain("#circle1");
  });

  it("should use class names when available", () => {
    const svg = createSVG(`
      <rect class="red-rect primary" style="stroke: #ff0000; fill: #00ff00" />
      <circle class="red-circle" style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toHaveLength(2);
    expect(regions[0].selectors[0]).toContain("rect");
    expect(regions[0].selectors[1]).toContain("circle");
  });

  it("should handle direct attributes as fallback to styles", () => {
    const svg = createSVG(`
      <rect stroke="#ff0000" fill="#00ff00" />
      <circle style="stroke: #ff0000; fill: #00ff00" />
    `);

    const regions = Region.extract(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].selectors).toHaveLength(2);
    expect(regions[0].strokeColor).toBe("#ff0000");
    expect(regions[0].fillColor).toBe("#00ff00");
  });

  it("should handle empty SVG", () => {
    const svg = createSVG("");

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

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

    const regions = Region.extract(svg);

    expect(regions).toHaveLength(2);

    const redGreenRegion = regions.find((r) => r.strokeColor === "#ff0000");
    expect(redGreenRegion?.selectors).toHaveLength(3);
    expect(redGreenRegion?.selectors).toContain(".shape");
    expect(redGreenRegion?.selectors).toContain("#path1");

    const blueYellowRegion = regions.find((r) => r.strokeColor === "#0000ff");
    expect(blueYellowRegion?.selectors).toHaveLength(1);
  });

  it("should ignore shapes inside non-rendered containers", () => {
    const svg = createSVG(`
      <defs>
        <clipPath id="clip"><rect style="fill: #ff0000" /></clipPath>
        <mask id="m"><circle style="fill: #ff0000" /></mask>
      </defs>
      <rect style="fill: #00ff00" />
    `);

    const regions = Region.extract(svg);

    expect(regions).toHaveLength(1);
    expect(regions[0].fillColor).toBe("#00ff00");
  });
});

const parse = (markup: string): SVGSVGElement => {
  const doc = new DOMParser().parseFromString(
    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">${markup}</svg>`,
    "image/svg+xml",
  );
  return doc.documentElement as unknown as SVGSVGElement;
};

// readResolver mimics the browser cascade engine closely enough for the rewrite logic:
// it reports the element's effective fill/stroke from its inline style, falling back to
// its presentation attribute. This lets the pure transform be exercised without a
// browser; the real getComputedStyle resolver (which also resolves <style> rules and
// inheritance) is covered by the integration suite.
const readResolver: Region.ColorResolver = (el) => {
  const read = (prop: "fill" | "stroke"): string | null =>
    el.style.getPropertyValue(prop) || el.getAttribute(prop) || null;
  return { fill: read("fill"), stroke: read("stroke") };
};

describe("Region.normalizeElement", () => {
  describe("color flattening", () => {
    it("should move an inline-style fill onto the presentation attribute", () => {
      const svg = parse('<rect style="fill:#5a87c5" id="r"/>');
      Region.normalizeElement(svg, () => ({ fill: "#5a87c5", stroke: null }));
      const rect = svg.querySelector("rect")!;
      expect(rect.getAttribute("fill")).toBe("#5a87c5");
      expect(rect.style.getPropertyValue("fill")).toBe("");
    });

    it("should move both fill and stroke and remove the now-empty style attribute", () => {
      const svg = parse('<rect style="fill:#111;stroke:#222" id="r"/>');
      Region.normalizeElement(svg, () => ({ fill: "#111111", stroke: "#222222" }));
      const rect = svg.querySelector("rect")!;
      expect(rect.getAttribute("fill")).toBe("#111111");
      expect(rect.getAttribute("stroke")).toBe("#222222");
      expect(rect.hasAttribute("style")).toBe(false);
    });

    it("should preserve non-color inline style declarations", () => {
      const svg = parse('<rect style="fill:#111;stroke-width:3" id="r"/>');
      Region.normalizeElement(svg, () => ({ fill: "#111111", stroke: null }));
      const rect = svg.querySelector("rect")!;
      expect(rect.getAttribute("fill")).toBe("#111111");
      expect(rect.style.getPropertyValue("stroke-width")).toBe("3");
    });

    it("should write a resolved class-based color (simulating getComputedStyle)", () => {
      const svg = parse('<path class="st0" d="M0 0h10v10h-10z"/>');
      Region.normalizeElement(svg, () => ({ fill: "#06ac38", stroke: null }));
      const path = svg.querySelector("path")!;
      expect(path.getAttribute("fill")).toBe("#06ac38");
    });

    it("should leave an attribute untouched when the resolver returns null", () => {
      const svg = parse('<rect fill="#abcdef" id="r"/>');
      Region.normalizeElement(svg, () => ({ fill: null, stroke: null }));
      expect(svg.querySelector("rect")!.getAttribute("fill")).toBe("#abcdef");
    });

    it("should resolve each shape independently", () => {
      const svg = parse(
        '<rect style="fill:#aaa" id="a"/><circle style="fill:#bbb" id="b"/>',
      );
      Region.normalizeElement(svg, (el) => ({
        fill: el.id === "a" ? "#aaaaaa" : "#bbbbbb",
        stroke: null,
      }));
      expect(svg.querySelector("rect")!.getAttribute("fill")).toBe("#aaaaaa");
      expect(svg.querySelector("circle")!.getAttribute("fill")).toBe("#bbbbbb");
    });
  });

  describe("paint server / keyword preservation", () => {
    it("should preserve a gradient url() reference rather than destroy it", () => {
      const svg = parse('<rect fill="url(#grad)" id="r"/>');
      Region.normalizeElement(svg, () => ({ fill: "url(#grad)", stroke: null }));
      expect(svg.querySelector("rect")!.getAttribute("fill")).toBe("url(#grad)");
    });

    it("should preserve the none keyword", () => {
      const svg = parse('<rect stroke="#111" id="r"/>');
      Region.normalizeElement(svg, () => ({ fill: "none", stroke: "#111111" }));
      const rect = svg.querySelector("rect")!;
      expect(rect.getAttribute("fill")).toBe("none");
      expect(rect.getAttribute("stroke")).toBe("#111111");
    });
  });

  describe("<style> stripping", () => {
    it("should remove <style> blocks so their rules cannot leak globally", () => {
      const svg = parse('<style>.st0{fill:#06ac38}</style><path class="st0"/>');
      Region.normalizeElement(svg, () => ({ fill: "#06ac38", stroke: null }));
      expect(svg.querySelector("style")).toBeNull();
      // the class color survives because it was flattened onto the element
      expect(svg.querySelector("path")!.getAttribute("fill")).toBe("#06ac38");
    });

    it("should retain the class attribute (existing selectors keep matching)", () => {
      const svg = parse('<style>.st0{fill:#06ac38}</style><path class="st0"/>');
      Region.normalizeElement(svg, () => ({ fill: "#06ac38", stroke: null }));
      expect(svg.querySelector("path")!.getAttribute("class")).toBe("st0");
    });
  });

  describe("region id tagging", () => {
    it("should tag shapes lacking an id with a data-region-id", () => {
      const svg = parse('<rect width="1" height="1"/>');
      Region.normalizeElement(svg, readResolver);
      expect(svg.querySelector("rect")!.getAttribute("data-region-id")).toMatch(
        /^region-/,
      );
    });

    it("should not tag shapes that already carry an id", () => {
      const svg = parse('<rect id="keep" width="1" height="1"/>');
      Region.normalizeElement(svg, readResolver);
      expect(svg.querySelector("rect")!.hasAttribute("data-region-id")).toBe(false);
    });

    it("should not overwrite an existing data-region-id", () => {
      const svg = parse('<rect data-region-id="region-fixed" width="1" height="1"/>');
      Region.normalizeElement(svg, readResolver);
      expect(svg.querySelector("rect")!.getAttribute("data-region-id")).toBe(
        "region-fixed",
      );
    });
  });

  describe("non-rendered containers", () => {
    it("should not flatten colors onto shapes inside a clipPath or defs", () => {
      const svg = parse(
        '<defs><clipPath id="clip"><path id="cp" d="M0 0h1v1z"/></clipPath></defs>' +
          '<rect id="vis" width="5" height="5"/>',
      );
      Region.normalizeElement(svg, () => ({ fill: "#123123", stroke: null }));
      expect(svg.querySelector("#cp")!.getAttribute("fill")).toBeNull();
      expect(svg.querySelector("#vis")!.getAttribute("fill")).toBe("#123123");
    });
  });

  describe("idempotency & backwards compatibility", () => {
    it("should be idempotent: re-normalizing a normalized SVG is a no-op", () => {
      const svg = parse('<style>.st0{fill:#06ac38}</style><path class="st0"/>');
      Region.normalizeElement(svg, () => ({ fill: "#06ac38", stroke: null }));
      const first = new XMLSerializer().serializeToString(svg);
      // second pass reads the now-attribute color back, matching the browser resolver's
      // behavior on an already-flattened element.
      Region.normalizeElement(svg, readResolver);
      expect(new XMLSerializer().serializeToString(svg)).toBe(first);
    });

    it("should leave an already-attribute-based SVG visually identical", () => {
      const svg = parse('<rect id="r" fill="#123456" stroke="#654321"/>');
      Region.normalizeElement(svg, readResolver);
      const rect = svg.querySelector("rect")!;
      expect(rect.getAttribute("fill")).toBe("#123456");
      expect(rect.getAttribute("stroke")).toBe("#654321");
      expect(rect.hasAttribute("style")).toBe(false);
    });
  });
});

describe("Region.normalizeSVG", () => {
  it("should throw a ValidationError when the input does not parse as an SVG", () => {
    expect(() => Region.normalizeSVG("<not-svg>nope")).toThrow(ValidationError);
  });

  it("should throw a ValidationError when the input is empty", () => {
    expect(() => Region.normalizeSVG("   ")).toThrow(ValidationError);
  });

  it("should strip <style> and tag shapes when run end-to-end", () => {
    const out = Region.normalizeSVG(
      '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><style>.a{fill:red}</style><rect class="a" width="5" height="5"/></svg>',
    );
    expect(out).not.toContain("<style");
    expect(out).toContain("data-region-id");
  });
});

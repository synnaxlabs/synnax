// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";

const strokeColor = (el: SVGElement): string =>
  el.style.getPropertyValue(Primitive.SYMBOL_COLOR);

const rgba = (hex: string): string | undefined =>
  color.rgbaString(color.construct(hex));

const MAIN_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>';

const createPaint = (
  selectors: string[] | undefined,
  stroke?: string,
  fill?: string,
): Custom.Paint => ({
  selectors,
  stroke: stroke == null ? undefined : color.construct(stroke),
  fill: fill == null ? undefined : color.construct(fill),
});

const OFF = [createPaint([".main"], "#333333", "#cccccc")];
const ON = [createPaint([".main"], "#00ff00", "#00ff0033")];

const renderAttached = (
  params: Custom.UseRenderParams,
  container: HTMLElement | null,
) => {
  const utils = renderHook((props: Custom.UseRenderParams) => Custom.useRender(props), {
    initialProps: params,
  });
  if (container != null) utils.result.current(container);
  return utils;
};

describe("Custom.useRender", () => {
  const createMockSpec = ({
    svg = MAIN_SVG,
    ...overrides
  }: Partial<schematic.symbol.Spec> & {
    svg?: string;
  } = {}): schematic.symbol.Spec => ({
    drawing: { type: "svg", markup: svg },
    parts: [],
    handles: [],
    states: [],
    commands: [],
    looks: [],
    interactions: [],
    variant: "test",
    scale: 1,
    strokeScaled: false,
    previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
    ...overrides,
  });

  describe("early returns", () => {
    it("should not mount anything when spec is null", () => {
      const container = document.createElement("div");
      renderAttached(
        {
          paints: OFF,
          spec: undefined,
        },
        container,
      );
      expect(container.children.length).toBe(0);
    });

    it("should not mount anything when the svg markup is malformed", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ svg: "<svg><rect></svg>" });
      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );
      expect(container.children.length).toBe(0);
    });

    it("should mount valid markup that replaces malformed markup", () => {
      const container = document.createElement("div");
      const { result, rerender } = renderHook(
        (props: Custom.UseRenderParams) => Custom.useRender(props),
        {
          initialProps: {
            paints: OFF,
            spec: createMockSpec({ svg: "<svg><rect></svg>" }),
          },
        },
      );
      result.current(container);
      expect(container.children.length).toBe(0);
      rerender({
        paints: OFF,
        spec: createMockSpec(),
      });
      expect(container.querySelector(".main")).not.toBeNull();
    });

    it("should not mount anything when svg is empty", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ svg: "" });
      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );
      expect(container.children.length).toBe(0);
    });

    it("should return a stable ref callback even when ref never attaches", () => {
      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          paints: OFF,
          spec: createMockSpec(),
        }),
      );
      const first = result.current;
      rerender();
      expect(result.current).toBe(first);
      expect(typeof result.current).toBe("function");
    });
  });

  describe("mounting", () => {
    it("should mount the drawing into the container as one g", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );

      expect(container.children.length).toBe(1);
      const drawing = container.children[0];
      expect(drawing.tagName).toBe("g");
      expect(drawing.querySelector(".main")).toBeTruthy();
    });

    it("should mount a symbol without its scripts or event handlers", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect class="main" onclick="alert(1)" /><script>alert(1)</script></svg>',
      });
      renderAttached({ paints: OFF, spec }, container);
      const drawing = container.querySelector("g");
      expect(drawing?.getAttribute("onload")).toBeNull();
      expect(drawing?.querySelector(".main")?.getAttribute("onclick")).toBeNull();
      expect(drawing?.querySelector("script")).toBeNull();
    });

    it("should mount built-in markup without the sanitizer", () => {
      const container = document.createElement("div");
      const href = "https://example.com/a.svg#a";
      const spec = createMockSpec({
        svg: `<svg xmlns="http://www.w3.org/2000/svg"><use class="main" href="${href}" /></svg>`,
      });
      renderAttached({ paints: OFF, spec, builtIn: true }, container);
      expect(container.querySelector(".main")?.getAttribute("href")).toBe(href);
    });

    it("should call onMount with the drawing", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const onMount = vi.fn();

      renderAttached(
        {
          paints: OFF,
          spec,
          onMount,
        },
        container,
      );

      expect(onMount).toHaveBeenCalledTimes(1);
      expect(onMount).toHaveBeenCalledWith(container.querySelector("g"));
    });

    it("should keep the groups of the markup inside the drawing", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><g><rect width="50" height="50"/></g><circle r="10"/></svg>',
      });
      renderAttached({ paints: OFF, spec }, container);
      const drawing = container.children[0];
      expect(Array.from(drawing.children, (child) => child.tagName)).toEqual([
        "g",
        "circle",
      ]);
    });

    it("should give the drawing the root attributes other than size and paint", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 100 100" width="100" height="100" fill="none" stroke="#000000" stroke-width="2"><rect width="50" height="50"/></svg>',
      });
      renderAttached({ paints: OFF, spec }, container);
      const drawing = container.children[0];
      expect(Array.from(drawing.attributes, ({ name }) => name)).toEqual([
        "stroke-width",
      ]);
      expect(drawing.getAttribute("stroke-width")).toBe("2");
    });
  });

  describe("auto-heal", () => {
    it("should mount the drawing when ref attaches after the resolving render", () => {
      const container = document.createElement("div");
      const initialProps: Custom.UseRenderParams = { paints: OFF };
      const { result, rerender } = renderHook(
        (props: Custom.UseRenderParams) => Custom.useRender(props),
        { initialProps },
      );
      expect(container.children.length).toBe(0);

      rerender({
        paints: OFF,
        spec: createMockSpec(),
      });
      result.current(container);
      expect(container.children.length).toBe(1);
    });

    it("should re-mount the drawing after a detach and a re-attach", () => {
      const spec = createMockSpec();
      const { result } = renderHook(() =>
        Custom.useRender({
          paints: OFF,
          spec,
        }),
      );

      const container1 = document.createElement("div");
      result.current(container1);
      expect(container1.children.length).toBe(1);

      result.current(null);
      expect(container1.children.length).toBe(0);

      const container2 = document.createElement("div");
      result.current(container2);
      expect(container2.children.length).toBe(1);
    });
  });

  describe("paints", () => {
    it("should draw the stroke and fill of each paint", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#333333"));
      expect(rect.getAttribute("fill")).toBe("#cccccc");
    });

    it("should draw other paints in place of the first", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          paints: ON,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#00ff00"));
      expect(rect.getAttribute("fill")).toBe("#00ff0033");
    });

    it("should keep the stroke attribute and store the original fill", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(rect.getAttribute("data-original-fill")).toBe("white");
      expect(rect.getAttribute("fill")).toBe("#cccccc");
    });

    it("should move between paints", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { result, rerender } = renderHook(
        ({ paints }) =>
          Custom.useRender({
            paints,
            spec,
          }),
        {
          initialProps: { paints: OFF },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#333333"));

      rerender({ paints: ON });
      expect(strokeColor(rect)).toBe(rgba("#00ff00"));

      rerender({ paints: OFF });
      expect(strokeColor(rect)).toBe(rgba("#333333"));
    });

    it("should draw a paint on every selector", () => {
      const container = document.createElement("div");
      const off = [createPaint([".main", ".secondary"], "#ff0000")];
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50"/><circle class="secondary" r="10"/></svg>',
      });

      renderAttached(
        {
          paints: off,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      const circle = container.querySelector(".secondary") as SVGCircleElement;
      expect(strokeColor(rect)).toBe(rgba("#ff0000"));
      expect(strokeColor(circle)).toBe(rgba("#ff0000"));
    });

    it("should draw a paint with no selectors on the whole drawing", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg"><rect class="main"/><circle class="other"/></svg>',
      });
      renderAttached({ paints: [createPaint(undefined, "#ff0000")], spec }, container);
      expect(strokeColor(container.querySelector(".main") as SVGElement)).toBe(
        rgba("#ff0000"),
      );
      expect(strokeColor(container.querySelector(".other") as SVGElement)).toBe(
        rgba("#ff0000"),
      );
    });

    it("should let a later paint win on each field that it sets", () => {
      const container = document.createElement("div");
      const paints = [
        createPaint([".main"], "#ff0000", "#00ff00"),
        createPaint([".main"], "#0000ff"),
      ];
      renderAttached({ paints, spec: createMockSpec() }, container);
      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#0000ff"));
      expect(rect.getAttribute("fill")).toBe("#00ff00");
    });

    it("should hide the shapes of a hidden paint and show them again", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const hidden = [{ selectors: [".main"], hidden: true }];
      const { rerender } = renderAttached({ paints: hidden, spec }, container);
      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.style.display).toBe("none");
      rerender({ paints: [], spec });
      expect(rect.style.display).toBe("");
    });
  });

  describe("stroke scaling", () => {
    it("should add non-scaling-stroke when strokeScaled is false", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ strokeScaled: false });

      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );

      const rect = container.querySelector("rect") as SVGRectElement;
      expect(rect.getAttribute("vector-effect")).toBe("non-scaling-stroke");
    });

    it("should not have vector-effect when strokeScaled is true", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ strokeScaled: true });

      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );

      const rect = container.querySelector("rect") as SVGRectElement;
      expect(rect.getAttribute("vector-effect")).toBeNull();
    });

    it("should apply vector-effect to all shape elements", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
          <path d="M10 10"/>
          <circle r="5"/>
          <rect width="10" height="10"/>
          <line x1="0" y1="0" x2="10" y2="10"/>
          <ellipse rx="5" ry="3"/>
          <polygon points="0,0 10,0 10,10"/>
          <polyline points="0,0 10,0 10,10"/>
        </svg>`,
        strokeScaled: false,
      });

      renderAttached(
        {
          paints: OFF,
          spec,
        },
        container,
      );

      const elements = container.querySelectorAll(
        "path, circle, rect, line, ellipse, polygon, polyline",
      );
      elements.forEach((el) => {
        expect(el.getAttribute("vector-effect")).toBe("non-scaling-stroke");
      });
    });
  });

  describe("performance optimizations", () => {
    it("should not recreate SVG if only state changes", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const onMount = vi.fn();

      const { result, rerender } = renderHook(
        ({ paints }) =>
          Custom.useRender({
            paints,
            spec,
            onMount,
          }),
        {
          initialProps: { paints: OFF },
        },
      );
      result.current(container);

      expect(onMount).toHaveBeenCalledTimes(1);
      const before = container.querySelector("g");

      rerender({ paints: ON });

      expect(container.querySelector("g")).toBe(before);
      expect(onMount).toHaveBeenCalledTimes(1);
    });

    it("should not parse unchanged markup again", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const parse = vi.spyOn(DOMParser.prototype, "parseFromString");
      const { result, rerender } = renderHook(
        ({ paints }) => Custom.useRender({ paints, spec }),
        { initialProps: { paints: OFF } },
      );
      result.current(container);
      expect(parse).toHaveBeenCalledTimes(1);
      rerender({ paints: ON });
      expect(parse).toHaveBeenCalledTimes(1);
      parse.mockRestore();
    });

    it("should recreate SVG when svg content changes", () => {
      const container = document.createElement("div");
      const onMount = vi.fn();

      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            paints: OFF,
            spec,
            onMount,
          }),
        {
          initialProps: { spec: createMockSpec() },
        },
      );
      result.current(container);

      expect(onMount).toHaveBeenCalledTimes(1);

      rerender({
        spec: createMockSpec({
          svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 200 200"><circle r="50"/></svg>',
        }),
      });

      expect(onMount).toHaveBeenCalledTimes(2);
      expect(container.querySelector("circle")).toBeTruthy();
    });

    it("should not update when no props change", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          paints: OFF,
          spec,
        }),
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      const strokeBefore = strokeColor(rect);

      rerender();

      const strokeAfter = strokeColor(rect);
      expect(strokeBefore).toBe(strokeAfter);
    });
  });

  describe("cleanup", () => {
    it("should remove old SVG when spec changes", () => {
      const container = document.createElement("div");

      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            paints: OFF,
            spec,
          }),
        {
          initialProps: {
            spec: createMockSpec({
              svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="first"/></svg>',
            }),
          },
        },
      );
      result.current(container);

      expect(container.querySelector(".first")).toBeTruthy();
      expect(container.children.length).toBe(1);

      rerender({
        spec: createMockSpec({
          svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="second"/></svg>',
        }),
      });

      expect(container.querySelector(".first")).toBeFalsy();
      expect(container.querySelector(".second")).toBeTruthy();
      expect(container.children.length).toBe(1);
    });

    it("should remove the SVG from the container on detach", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const { result } = renderHook(() =>
        Custom.useRender({
          paints: OFF,
          spec,
        }),
      );
      result.current(container);
      expect(container.children.length).toBe(1);

      result.current(null);
      expect(container.children.length).toBe(0);
    });
  });

  describe("edge cases", () => {
    it("should handle malformed SVG gracefully", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: "<not-valid-svg>",
      });

      expect(() => {
        renderAttached(
          {
            paints: OFF,
            spec,
          },
          container,
        );
      }).not.toThrow();
    });

    it("should leave the stored colors for a paint with no colors", () => {
      const container = document.createElement("div");
      const off = [createPaint([".main"])];
      const spec = createMockSpec();

      renderAttached(
        {
          paints: off,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(strokeColor(rect)).toBe("");
      expect(rect.getAttribute("fill")).toBe("white");
    });

    it("should draw nothing for a selector that selects no shapes", () => {
      const container = document.createElement("div");
      const off = [createPaint([".does-not-exist"], "#ff0000")];
      const spec = createMockSpec();

      expect(() => {
        renderAttached(
          {
            paints: off,
            spec,
          },
          container,
        );
      }).not.toThrow();
    });

    it("should leave the stored colors with no paints", () => {
      const container = document.createElement("div");
      renderAttached({ paints: [], spec: createMockSpec() }, container);
      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(strokeColor(rect)).toBe("");
      expect(rect.getAttribute("fill")).toBe("white");
    });

    it("should restore the original fill when a paint drops its fill", () => {
      const container = document.createElement("div");
      const [off, on] = [
        [createPaint([".main"])],
        [createPaint([".main"], undefined, "#ff0000")],
      ];
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="blue"/></svg>',
      });

      const { result, rerender } = renderHook(
        ({ paints }) =>
          Custom.useRender({
            paints,
            spec,
          }),
        {
          initialProps: { paints: off },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("fill")).toBe("blue");

      rerender({ paints: on });
      expect(rect.getAttribute("fill")).toBe("#ff0000");

      rerender({ paints: off });
      expect(rect.getAttribute("fill")).toBe("blue");
    });

    it("should restore colors when a paint drops a shape", () => {
      const container = document.createElement("div");
      const [off, on] = [
        [createPaint([".main"], "#333333", "#cccccc")],
        [createPaint([".main", ".secondary"], "#ff0000", "#ff0000")],
      ];
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/><circle class="secondary" r="10" stroke="blue" fill="yellow"/></svg>',
      });

      const { result, rerender } = renderHook(
        ({ paints }) =>
          Custom.useRender({
            paints,
            spec,
          }),
        {
          initialProps: { paints: off },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      const circle = container.querySelector(".secondary") as SVGCircleElement;

      expect(strokeColor(rect)).toBe(rgba("#333333"));
      expect(rect.getAttribute("fill")).toBe("#cccccc");
      expect(circle.getAttribute("stroke")).toBe("blue");
      expect(strokeColor(circle)).toBe("");
      expect(circle.getAttribute("fill")).toBe("yellow");

      rerender({ paints: on });
      expect(strokeColor(rect)).toBe(rgba("#ff0000"));
      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(strokeColor(circle)).toBe(rgba("#ff0000"));
      expect(circle.getAttribute("fill")).toBe("#ff0000");

      rerender({ paints: off });
      expect(strokeColor(rect)).toBe(rgba("#333333"));
      expect(rect.getAttribute("fill")).toBe("#cccccc");
      expect(circle.getAttribute("stroke")).toBe("blue");
      expect(strokeColor(circle)).toBe("");
      expect(circle.getAttribute("fill")).toBe("yellow");
    });

    it("should keep the original fill across paint changes", () => {
      const container = document.createElement("div");
      const [off, on] = [
        [createPaint([".main"], undefined, "#ff0000")],
        [createPaint([".main"], undefined, "#00ff00")],
      ];
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
      });

      const { result, rerender } = renderHook(
        ({ paints }) =>
          Custom.useRender({
            paints,
            spec,
          }),
        {
          initialProps: { paints: off },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;

      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(rect.getAttribute("data-original-fill")).toBe("white");

      rerender({ paints: on });
      expect(rect.getAttribute("fill")).toBe("#00ff00");
      expect(rect.getAttribute("data-original-fill")).toBe("white");

      rerender({ paints: off });
      expect(rect.getAttribute("fill")).toBe("#ff0000");
    });
  });

  describe("spec changes", () => {
    const renderSpec = (spec: schematic.symbol.Spec) => {
      const container = document.createElement("div");
      const utils = renderAttached({ paints: OFF, spec }, container);
      const rerender = (next: schematic.symbol.Spec) =>
        utils.rerender({ paints: OFF, spec: next });
      return { container, rerender };
    };

    it("should toggle stroke scaling when strokeScaled changes", () => {
      const spec = createMockSpec({ strokeScaled: false });
      const { container, rerender } = renderSpec(spec);
      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("vector-effect")).toBe("non-scaling-stroke");
      rerender({ ...spec, strokeScaled: true });
      expect(rect.getAttribute("vector-effect")).toBeNull();
    });
  });

  // When the SVG markup changes the element is re-parsed from scratch, discarding
  // every derived attribute. All of them must be re-applied against the new DOM even
  // when their own inputs (state, strokeScaled) are unchanged.
  describe("re-application after SVG rebuild", () => {
    it("should re-apply region colors to a rebuilt SVG with unchanged state", () => {
      const container = document.createElement("div");
      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            paints: OFF,
            spec,
          }),
        { initialProps: { spec: createMockSpec() } },
      );
      result.current(container);

      expect(strokeColor(container.querySelector(".main") as SVGRectElement)).toBe(
        rgba("#333333"),
      );

      rerender({
        spec: createMockSpec({
          svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="purple" fill="orange"/></svg>',
        }),
      });

      const rebuilt = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rebuilt)).toBe(rgba("#333333"));
      expect(rebuilt.getAttribute("fill")).toBe("#cccccc");
    });

    it("should re-apply stroke scaling to a rebuilt SVG with unchanged strokeScaled", () => {
      const container = document.createElement("div");
      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            paints: OFF,
            spec,
          }),
        { initialProps: { spec: createMockSpec({ strokeScaled: false }) } },
      );
      result.current(container);
      expect(
        (container.querySelector(".main") as SVGRectElement).getAttribute(
          "vector-effect",
        ),
      ).toBe("non-scaling-stroke");

      rerender({
        spec: createMockSpec({
          strokeScaled: false,
          svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
        }),
      });

      expect(
        (container.querySelector(".main") as SVGRectElement).getAttribute(
          "vector-effect",
        ),
      ).toBe("non-scaling-stroke");
    });
  });
});

describe("Custom.isValidSVG", () => {
  it("should accept markup in the SVG namespace", () => {
    expect(
      Custom.isValidSVG('<svg xmlns="http://www.w3.org/2000/svg"><rect /></svg>'),
    ).toBe(true);
  });

  it.each([
    ["empty", ""],
    ["malformed", "<svg><rect></svg>"],
    ["not rooted in svg", "<div><svg /></div>"],
  ])("should reject %s markup", (_, svg) => {
    expect(Custom.isValidSVG(svg)).toBe(false);
  });
});

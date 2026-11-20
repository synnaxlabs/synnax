// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { schematic } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { color, deep } from "@synnaxlabs/x";
import { act, render, renderHook } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Custom } from "@/schematic/node/common/custom";
import { Primitive } from "@/schematic/node/common/primitive";

const strokeColor = (el: SVGElement): string =>
  el.style.getPropertyValue(Primitive.SYMBOL_COLOR);

const rgba = (hex: string): string | undefined =>
  color.rgbaString(color.construct(hex));

const path = deep.path<{ data: schematic.symbol.Spec }>();

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
  const createMockSpec = (
    overrides?: Partial<schematic.symbol.Spec>,
  ): schematic.symbol.Spec => ({
    svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
    states: [
      {
        key: "inactive",
        name: "Inactive",
        regions: [
          {
            key: "main",
            name: "Main",
            selectors: [".main"],
            strokeColor: color.construct("#333333"),
            fillColor: color.construct("#cccccc"),
          },
        ],
      },
      {
        key: "active",
        name: "Active",
        regions: [
          {
            key: "main",
            name: "Main",
            selectors: [".main"],
            strokeColor: color.construct("#00ff00"),
            fillColor: color.construct("#00ff0033"),
          },
        ],
      },
    ],
    variant: "test",
    handles: [],
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
          activeState: "inactive",
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
          activeState: "inactive",
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
            activeState: "inactive",
            spec: createMockSpec({ svg: "<svg><rect></svg>" }),
          },
        },
      );
      result.current(container);
      expect(container.children.length).toBe(0);
      rerender({
        activeState: "inactive",
        spec: createMockSpec(),
      });
      expect(container.querySelector(".main")).not.toBeNull();
    });

    it("should not mount anything when svg is empty", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ svg: "" });
      renderAttached(
        {
          activeState: "inactive",
          spec,
        },
        container,
      );
      expect(container.children.length).toBe(0);
    });

    it("should return a stable ref callback even when ref never attaches", () => {
      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          activeState: "inactive",
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
          activeState: "inactive",
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
      renderAttached({ activeState: "inactive", spec }, container);
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
      renderAttached({ activeState: "inactive", spec, builtIn: true }, container);
      expect(container.querySelector(".main")?.getAttribute("href")).toBe(href);
    });

    it("should call onMount with the drawing", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const onMount = vi.fn();

      renderAttached(
        {
          activeState: "inactive",
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
      renderAttached({ activeState: "inactive", spec }, container);
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
      renderAttached({ activeState: "inactive", spec }, container);
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
      const initialProps: Custom.UseRenderParams = { activeState: "inactive" };
      const { result, rerender } = renderHook(
        (props: Custom.UseRenderParams) => Custom.useRender(props),
        { initialProps },
      );
      expect(container.children.length).toBe(0);

      rerender({
        activeState: "inactive",
        spec: createMockSpec(),
      });
      result.current(container);
      expect(container.children.length).toBe(1);
    });

    it("should re-mount the drawing after a detach and a re-attach", () => {
      const spec = createMockSpec();
      const { result } = renderHook(() =>
        Custom.useRender({
          activeState: "inactive",
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

  describe("state management", () => {
    it("should apply inactive state when activeState is not 'active'", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          activeState: "inactive",
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#333333"));
      expect(rect.getAttribute("fill")).toBe("#cccccc");
    });

    it("should apply active state when activeState is 'active'", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          activeState: "active",
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
          activeState: "inactive",
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(rect.getAttribute("data-original-fill")).toBe("white");
      expect(rect.getAttribute("fill")).toBe("#cccccc");
    });

    it("should transition between states correctly", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            activeState,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#333333"));

      rerender({ activeState: "active" });
      expect(strokeColor(rect)).toBe(rgba("#00ff00"));

      rerender({ activeState: "inactive" });
      expect(strokeColor(rect)).toBe(rgba("#333333"));
    });

    it("should handle multiple selectors in a region", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50"/><circle class="secondary" r="10"/></svg>',
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "all",
                name: "All",
                selectors: [".main", ".secondary"],
                strokeColor: color.construct("#ff0000"),
              },
            ],
          },
          {
            key: "active",
            name: "Active",
            regions: [
              {
                key: "all",
                name: "All",
                selectors: [".main", ".secondary"],
                strokeColor: color.construct("#00ff00"),
              },
            ],
          },
        ],
      });

      renderAttached(
        {
          activeState: "inactive",
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      const circle = container.querySelector(".secondary") as SVGCircleElement;
      expect(strokeColor(rect)).toBe(rgba("#ff0000"));
      expect(strokeColor(circle)).toBe(rgba("#ff0000"));
    });
  });

  describe("stroke scaling", () => {
    it("should add non-scaling-stroke when strokeScaled is false", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ strokeScaled: false });

      renderAttached(
        {
          activeState: "inactive",
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
          activeState: "inactive",
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
          activeState: "inactive",
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
        ({ activeState }) =>
          Custom.useRender({
            activeState,
            spec,
            onMount,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      expect(onMount).toHaveBeenCalledTimes(1);
      const before = container.querySelector("g");

      rerender({ activeState: "active" });

      expect(container.querySelector("g")).toBe(before);
      expect(onMount).toHaveBeenCalledTimes(1);
    });

    it("should not parse unchanged markup again", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const parse = vi.spyOn(DOMParser.prototype, "parseFromString");
      const { result, rerender } = renderHook(
        ({ activeState }) => Custom.useRender({ activeState, spec }),
        { initialProps: { activeState: "inactive" } },
      );
      result.current(container);
      expect(parse).toHaveBeenCalledTimes(1);
      rerender({ activeState: "active" });
      expect(parse).toHaveBeenCalledTimes(1);
      parse.mockRestore();
    });

    it("should recreate SVG when svg content changes", () => {
      const container = document.createElement("div");
      const onMount = vi.fn();

      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            activeState: "inactive",
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
          activeState: "inactive",
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
            activeState: "inactive",
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
          activeState: "inactive",
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
            activeState: "inactive",
            spec,
          },
          container,
        );
      }).not.toThrow();
    });

    it("should handle missing state colors", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
              },
            ],
          },
          {
            key: "active",
            name: "Active",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
              },
            ],
          },
        ],
      });

      renderAttached(
        {
          activeState: "inactive",
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(strokeColor(rect)).toBe("");
      expect(rect.getAttribute("fill")).toBe("white");
    });

    it("should handle selectors that match no elements", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "nonexistent",
                name: "Nonexistent",
                selectors: [".does-not-exist"],
                strokeColor: color.construct("#ff0000"),
              },
            ],
          },
          {
            key: "active",
            name: "Active",
            regions: [
              {
                key: "nonexistent",
                name: "Nonexistent",
                selectors: [".does-not-exist"],
                strokeColor: color.construct("#00ff00"),
              },
            ],
          },
        ],
      });

      expect(() => {
        renderAttached(
          {
            activeState: "inactive",
            spec,
          },
          container,
        );
      }).not.toThrow();
    });

    it("should handle empty states array", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ states: [] });

      expect(() => {
        renderAttached(
          {
            activeState: "inactive",
            spec,
          },
          container,
        );
      }).not.toThrow();
    });

    it("should handle undefined regions in state", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [],
          },
          {
            key: "active",
            name: "Active",
            regions: [],
          },
        ],
      });

      renderAttached(
        {
          activeState: "inactive",
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(strokeColor(rect)).toBe("");
      expect(rect.getAttribute("fill")).toBe("white");
    });

    it("should restore original fill when fillColor is removed from state", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="blue"/></svg>',
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
              },
            ],
          },
          {
            key: "active",
            name: "Active",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
                fillColor: color.construct("#ff0000"),
              },
            ],
          },
        ],
      });

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            activeState,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("fill")).toBe("blue");

      rerender({ activeState: "active" });
      expect(rect.getAttribute("fill")).toBe("#ff0000");

      rerender({ activeState: "inactive" });
      expect(rect.getAttribute("fill")).toBe("blue");
    });

    it("should restore colors when element is removed from region", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/><circle class="secondary" r="10" stroke="blue" fill="yellow"/></svg>',
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
                strokeColor: color.construct("#333333"),
                fillColor: color.construct("#cccccc"),
              },
            ],
          },
          {
            key: "active",
            name: "Active",
            regions: [
              {
                key: "both",
                name: "Both",
                selectors: [".main", ".secondary"],
                strokeColor: color.construct("#ff0000"),
                fillColor: color.construct("#ff0000"),
              },
            ],
          },
        ],
      });

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            activeState,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
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

      rerender({ activeState: "active" });
      expect(strokeColor(rect)).toBe(rgba("#ff0000"));
      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(strokeColor(circle)).toBe(rgba("#ff0000"));
      expect(circle.getAttribute("fill")).toBe("#ff0000");

      rerender({ activeState: "inactive" });
      expect(strokeColor(rect)).toBe(rgba("#333333"));
      expect(rect.getAttribute("fill")).toBe("#cccccc");
      expect(circle.getAttribute("stroke")).toBe("blue");
      expect(strokeColor(circle)).toBe("");
      expect(circle.getAttribute("fill")).toBe("yellow");
    });

    it("should handle fill color changes across multiple state transitions", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
        states: [
          {
            key: "state1",
            name: "State 1",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
                fillColor: color.construct("#ff0000"),
              },
            ],
          },
          {
            key: "state2",
            name: "State 2",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
                fillColor: color.construct("#00ff00"),
              },
            ],
          },
        ],
      });

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            activeState,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;

      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(rect.getAttribute("data-original-fill")).toBe("white");

      rerender({ activeState: "active" });
      expect(rect.getAttribute("fill")).toBe("#00ff00");
      expect(rect.getAttribute("data-original-fill")).toBe("white");

      rerender({ activeState: "inactive" });
      expect(rect.getAttribute("fill")).toBe("#ff0000");
    });
  });

  describe("spec changes", () => {
    const renderSpec = (spec: schematic.symbol.Spec) => {
      const container = document.createElement("div");
      const utils = renderAttached({ activeState: "inactive", spec }, container);
      const rerender = (next: schematic.symbol.Spec) =>
        utils.rerender({
          activeState: "inactive",
          spec: next,
        });
      return { container, rerender };
    };

    it("should apply region color changes when a state changes", () => {
      const spec = createMockSpec();
      const { container, rerender } = renderSpec(spec);
      const rect = container.querySelector(".main") as SVGRectElement;
      expect(strokeColor(rect)).toBe(rgba("#333333"));
      const [inactive, active] = spec.states;
      const region = {
        ...inactive.regions[0],
        strokeColor: color.construct("#aabbcc"),
      };
      rerender({ ...spec, states: [{ ...inactive, regions: [region] }, active] });
      expect(strokeColor(rect)).toBe(rgba("#aabbcc"));
    });

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
            activeState: "inactive",
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
            activeState: "inactive",
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

interface SymbolValues {
  data: schematic.symbol.Spec;
}

const EDITOR_SVG =
  '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect data-region-id="rect-1" width="50" height="50" stroke="black" fill="white"/></svg>';

const editorInitialValues = (): SymbolValues => ({
  data: {
    svg: "",
    states: [{ key: "base", name: "Base", regions: [] }],
    variant: "static",
    handles: [],
    scale: 1,
    strokeScaled: false,
    previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
  },
});

// Mirrors the editor's Preview: reads the live form spec and drives useRender with the
// currently-selected state key as activeState.
const PreviewLike = ({ activeState }: { activeState: string }): ReactElement => {
  const spec = Form.useFieldValue(path.data);
  const setContainer = Custom.useRender({
    activeState,
    spec,
  });
  return <div data-testid="container" ref={setContainer} />;
};

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

describe("custom symbol editor preview", () => {
  it("should recolor a region when its color is edited through the form", () => {
    let form!: Form.ContextValue<any>;
    const Harness = (): ReactElement => {
      const values = editorInitialValues();
      form = Form.use<any>({ values });
      return (
        <Form.Form {...form}>
          <PreviewLike activeState="base" />
        </Form.Form>
      );
    };

    const { container } = render(<Harness />);

    // Mimic the import flow: set the SVG first, then assign regions to the state by
    // key in a separate commit, exactly as Preview.handleContentsChange does.
    act(() => {
      form.set(path.data.svg, EDITOR_SVG);
    });
    const region: schematic.symbol.Region = {
      key: "region-1",
      name: "Region 1",
      selectors: ['[data-region-id="rect-1"]'],
      strokeColor: color.construct("#000000"),
      fillColor: color.construct("#ffffff"),
    };
    act(() => {
      form.set(path.data.states.base.regions, [region]);
    });

    const rect = container.querySelector("rect") as SVGRectElement;
    expect(rect).toBeTruthy();
    expect(strokeColor(rect)).toBe(rgba("#000000"));

    // Editing the color the way RegionList does: set the keyed region path.
    act(() => {
      form.set(
        path.data.states.base.regions["region-1"].strokeColor,
        color.construct("#ff0000"),
      );
    });

    expect(strokeColor(rect)).toBe(rgba("#ff0000"));
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";
import { act, render, renderHook } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Form } from "@/form";
import { Custom } from "@/schematic/node/common/custom";

const renderAttached = (
  args: Custom.UseRenderParams,
  container: HTMLElement | null,
) => {
  const utils = renderHook((props: Custom.UseRenderParams) => Custom.useRender(props), {
    initialProps: args,
  });
  if (container != null) utils.result.current(container);
  return utils;
};

describe("Custom.useRender", () => {
  const createMockSpec = (
    overrides?: Partial<schematic.symbol.Spec>,
  ): schematic.symbol.Spec => ({
    svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
    states: [
      {
        key: "inactive",
        name: "Inactive",
        regions: [
          {
            key: "main",
            name: "Main",
            selectors: [".main"],
            strokeColor: "#333",
            fillColor: "#ccc",
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
            strokeColor: "#0f0",
            fillColor: "#0f03",
          },
        ],
      },
    ],
    variant: "test",
    handles: [],
    scale: 1,
    scaleStroke: false,
    previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
    ...overrides,
  });

  describe("early returns", () => {
    it("should not mount anything when spec is null", () => {
      const container = document.createElement("div");
      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec: undefined,
        },
        container,
      );
      expect(container.children.length).toBe(0);
    });

    it("should not mount anything when svg is empty", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ svg: "" });
      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );
      expect(container.children.length).toBe(0);
    });

    it("should return a stable ref callback even when ref never attaches", () => {
      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec: createMockSpec(),
        }),
      );
      const first = result.current;
      rerender();
      expect(result.current).toBe(first);
      expect(typeof result.current).toBe("function");
    });
  });

  describe("SVG mounting", () => {
    it("should mount SVG to container", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      expect(container.children.length).toBe(1);
      const svg = container.children[0] as SVGSVGElement;
      expect(svg.tagName.toLowerCase()).toBe("svg");
      expect(svg.querySelector(".main")).toBeTruthy();
    });

    it("should call onMount callback with SVG element", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const onMount = vi.fn();

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
          onMount,
        },
        container,
      );

      expect(onMount).toHaveBeenCalledTimes(1);
      const svgElement = container.querySelector("svg");
      expect(onMount).toHaveBeenCalledWith(svgElement);
    });

    it("should wrap SVG content in g element if not present", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 100 100"><rect width="50" height="50"/><circle r="10"/></svg>',
      });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg");
      const g = svg?.querySelector("g");
      expect(g).toBeTruthy();
      expect(g?.children.length).toBe(2);
    });

    it("should not wrap if g element already exists", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 100 100"><g><rect width="50" height="50"/></g></svg>',
      });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg");
      const gElements = svg?.querySelectorAll("g");
      expect(gElements?.length).toBe(1);
    });
  });

  describe("auto-heal", () => {
    it("should mount SVG when ref attaches after the resolving render", () => {
      const container = document.createElement("div");
      const { result, rerender } = renderHook(
        (props: Custom.UseRenderParams) => Custom.useRender(props),
        {
          initialProps: {
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec: undefined,
          },
        },
      );
      expect(container.children.length).toBe(0);

      rerender({
        orientation: "top",
        activeState: "inactive",
        externalScale: 1,
        spec: createMockSpec(),
      });
      result.current(container);
      expect(container.children.length).toBe(1);
    });

    it("should re-mount SVG after detach and re-attach with the same spec", () => {
      const spec = createMockSpec();
      const { result } = renderHook(() =>
        Custom.useRender({
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
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
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("#333");
      expect(rect.getAttribute("fill")).toBe("#ccc");
    });

    it("should apply active state when activeState is 'active'", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          orientation: "top",
          activeState: "active",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("#0f0");
      expect(rect.getAttribute("fill")).toBe("#0f03");
    });

    it("should store original attributes before applying state", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("data-original-stroke")).toBe("black");
      expect(rect.getAttribute("fill")).toBe("#ccc");
    });

    it("should transition between states correctly", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("#333");

      rerender({ activeState: "active" });
      expect(rect.getAttribute("stroke")).toBe("#0f0");

      rerender({ activeState: "inactive" });
      expect(rect.getAttribute("stroke")).toBe("#333");
    });

    it("should handle multiple selectors in a region", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50"/><circle class="secondary" r="10"/></svg>',
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "all",
                name: "All",
                selectors: [".main", ".secondary"],
                strokeColor: "#f00",
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
                strokeColor: "#0f0",
              },
            ],
          },
        ],
      });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      const circle = container.querySelector(".secondary") as SVGCircleElement;
      expect(rect.getAttribute("stroke")).toBe("#f00");
      expect(circle.getAttribute("stroke")).toBe("#f00");
    });
  });

  describe("scaling", () => {
    it("should apply internal scale", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scale: 2 });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("200");
      expect(svg.getAttribute("height")).toBe("200");
    });

    it("should apply external scale", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 3,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("300");
      expect(svg.getAttribute("height")).toBe("300");
    });

    it("should combine internal and external scale", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scale: 2 });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1.5,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("300");
      expect(svg.getAttribute("height")).toBe("300");
    });

    it("should update dimensions when scale changes", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { result, rerender } = renderHook(
        ({ externalScale }) =>
          Custom.useRender({
            orientation: "top",
            activeState: "inactive",
            externalScale,
            spec,
          }),
        {
          initialProps: { externalScale: 1 },
        },
      );
      result.current(container);

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");

      rerender({ externalScale: 2 });
      expect(svg.getAttribute("width")).toBe("200");
    });
  });

  describe("orientation", () => {
    it("should maintain dimensions for horizontal orientations", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("100");
      expect(svg.getAttribute("viewBox")).toBe("0 0 100 100");
    });

    it("should swap dimensions for vertical orientations", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 200 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
      });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("200");
      expect(svg.getAttribute("viewBox")).toBe("0 0 100 200");
    });

    it("should handle orientation changes", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { result, rerender } = renderHook(
        ({ orientation }) =>
          Custom.useRender({
            orientation,
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { orientation: "left" as location.Outer },
        },
      );
      result.current(container);

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("100");

      rerender({ orientation: "top" as location.Outer });
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("100");
      expect(svg.getAttribute("viewBox")).toBe("0 0 100 100");
    });
  });

  describe("stroke scaling", () => {
    it("should add non-scaling-stroke when scaleStroke is false", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scaleStroke: false });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector("rect") as SVGRectElement;
      expect(rect.getAttribute("vector-effect")).toBe("non-scaling-stroke");
    });

    it("should not have vector-effect when scaleStroke is true", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scaleStroke: true });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
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
        svg: `<svg viewBox="0 0 100 100">
          <path d="M10 10"/>
          <circle r="5"/>
          <rect width="10" height="10"/>
          <line x1="0" y1="0" x2="10" y2="10"/>
          <ellipse rx="5" ry="3"/>
          <polygon points="0,0 10,0 10,10"/>
          <polyline points="0,0 10,0 10,10"/>
        </svg>`,
        scaleStroke: false,
      });

      renderAttached(
        {
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
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
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
            onMount,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      expect(onMount).toHaveBeenCalledTimes(1);
      const svgBefore = container.querySelector("svg");

      rerender({ activeState: "active" });
      const svgAfter = container.querySelector("svg");

      expect(svgBefore).toBe(svgAfter);
      expect(onMount).toHaveBeenCalledTimes(1);
    });

    it("should recreate SVG when svg content changes", () => {
      const container = document.createElement("div");
      const onMount = vi.fn();

      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
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
          svg: '<svg viewBox="0 0 200 200"><circle r="50"/></svg>',
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
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      const strokeBefore = rect.getAttribute("stroke");

      rerender();

      const strokeAfter = rect.getAttribute("stroke");
      expect(strokeBefore).toBe(strokeAfter);
    });
  });

  describe("cleanup", () => {
    it("should remove old SVG when spec changes", () => {
      const container = document.createElement("div");

      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
        {
          initialProps: {
            spec: createMockSpec({
              svg: '<svg viewBox="0 0 100 100"><rect class="first"/></svg>',
            }),
          },
        },
      );
      result.current(container);

      expect(container.querySelector(".first")).toBeTruthy();
      expect(container.children.length).toBe(1);

      rerender({
        spec: createMockSpec({
          svg: '<svg viewBox="0 0 100 100"><rect class="second"/></svg>',
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
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
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
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
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
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
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
                strokeColor: "#f00",
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
                strokeColor: "#0f0",
              },
            ],
          },
        ],
      });

      expect(() => {
        renderAttached(
          {
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
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
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
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
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        },
        container,
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("black");
      expect(rect.getAttribute("fill")).toBe("white");
    });

    it("should restore original fill when fillColor is removed from state", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="blue"/></svg>',
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
                fillColor: "#ff0000",
              },
            ],
          },
        ],
      });

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            orientation: "top",
            activeState,
            externalScale: 1,
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
        svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/><circle class="secondary" r="10" stroke="blue" fill="yellow"/></svg>',
        states: [
          {
            key: "inactive",
            name: "Inactive",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
                strokeColor: "#333",
                fillColor: "#ccc",
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
                strokeColor: "#f00",
                fillColor: "#ff0000",
              },
            ],
          },
        ],
      });

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      const circle = container.querySelector(".secondary") as SVGCircleElement;

      expect(rect.getAttribute("stroke")).toBe("#333");
      expect(rect.getAttribute("fill")).toBe("#ccc");
      expect(circle.getAttribute("stroke")).toBe("blue");
      expect(circle.getAttribute("fill")).toBe("yellow");

      rerender({ activeState: "active" });
      expect(rect.getAttribute("stroke")).toBe("#f00");
      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(circle.getAttribute("stroke")).toBe("#f00");
      expect(circle.getAttribute("fill")).toBe("#ff0000");

      rerender({ activeState: "inactive" });
      expect(rect.getAttribute("stroke")).toBe("#333");
      expect(rect.getAttribute("fill")).toBe("#ccc");
      expect(circle.getAttribute("stroke")).toBe("blue");
      expect(circle.getAttribute("fill")).toBe("yellow");
    });

    it("should handle fill color changes across multiple state transitions", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
        states: [
          {
            key: "state1",
            name: "State 1",
            regions: [
              {
                key: "main",
                name: "Main",
                selectors: [".main"],
                fillColor: "#ff0000",
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
                fillColor: "#00ff00",
              },
            ],
          },
        ],
      });

      const { result, rerender } = renderHook(
        ({ activeState }) =>
          Custom.useRender({
            orientation: "top",
            activeState,
            externalScale: 1,
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

  // The symbol editor's form mutates its value object in place rather than
  // producing a new reference on every change, so the same spec object is
  // passed across rerenders. The diff must detect changes by value, not by
  // object identity.
  describe("in-place spec mutation", () => {
    it("should update dimensions when the internal scale is mutated in place", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );
      result.current(container);

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");

      spec.scale = 2;
      rerender();
      expect(svg.getAttribute("width")).toBe("200");
      expect(svg.getAttribute("height")).toBe("200");
    });

    it("should apply region color changes when a state is mutated in place", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("#333");

      spec.states[0].regions[0].strokeColor = "#abc";
      rerender();
      expect(rect.getAttribute("stroke")).toBe("#abc");
    });

    it("should toggle stroke scaling when scaleStroke is mutated in place", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scaleStroke: false });
      const { result, rerender } = renderHook(() =>
        Custom.useRender({
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );
      result.current(container);

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("vector-effect")).toBe("non-scaling-stroke");

      spec.scaleStroke = true;
      rerender();
      expect(rect.getAttribute("vector-effect")).toBeNull();
    });
  });

  // When the SVG markup changes the element is re-parsed from scratch, discarding
  // every derived attribute. All of them must be re-applied against the new DOM even
  // when their own inputs (state, scaleStroke) are unchanged.
  describe("re-application after SVG rebuild", () => {
    it("should re-apply region colors to a rebuilt SVG with unchanged state", () => {
      const container = document.createElement("div");
      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
        { initialProps: { spec: createMockSpec() } },
      );
      result.current(container);

      expect(
        (container.querySelector(".main") as SVGRectElement).getAttribute("stroke"),
      ).toBe("#333");

      rerender({
        spec: createMockSpec({
          svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="purple" fill="orange"/></svg>',
        }),
      });

      const rebuilt = container.querySelector(".main") as SVGRectElement;
      expect(rebuilt.getAttribute("stroke")).toBe("#333");
      expect(rebuilt.getAttribute("fill")).toBe("#ccc");
    });

    it("should re-apply stroke scaling to a rebuilt SVG with unchanged scaleStroke", () => {
      const container = document.createElement("div");
      const { result, rerender } = renderHook(
        ({ spec }) =>
          Custom.useRender({
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
        { initialProps: { spec: createMockSpec({ scaleStroke: false }) } },
      );
      result.current(container);
      expect(
        (container.querySelector(".main") as SVGRectElement).getAttribute(
          "vector-effect",
        ),
      ).toBe("non-scaling-stroke");

      rerender({
        spec: createMockSpec({
          scaleStroke: false,
          svg: '<svg viewBox="0 0 100 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
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
  '<svg viewBox="0 0 100 100"><rect data-region-id="rect-1" width="50" height="50" stroke="black" fill="white"/></svg>';

const editorInitialValues = (): SymbolValues => ({
  data: {
    svg: "",
    states: [{ key: "base", name: "Base", regions: [] }],
    variant: "static",
    handles: [],
    scale: 1,
    scaleStroke: false,
    previewViewport: { zoom: 1, position: { x: 0, y: 0 } },
  },
});

// Mirrors the editor's Preview: reads the live form spec and drives useRender with the
// currently-selected state key as activeState. The editor's form mutates its value
// object in place, so this exercises useRender through the real Form rather than a
// hand-mutated spec.
const PreviewLike = ({ activeState }: { activeState: string }): ReactElement => {
  const spec = Form.useFieldValue<schematic.symbol.Spec>("data");
  const setContainer = Custom.useRender({
    orientation: "left",
    activeState,
    externalScale: 1,
    spec,
  });
  return <div data-testid="container" ref={setContainer} />;
};

describe("custom symbol editor preview", () => {
  it("should recolor a region when its color is edited through the form", () => {
    let form!: Form.ContextValue<any>;
    const Harness = (): ReactElement => {
      form = Form.use<any>({ values: editorInitialValues() });
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
      form.set("data.svg", EDITOR_SVG);
    });
    const region: schematic.symbol.Region = {
      key: "region-1",
      name: "Region 1",
      selectors: ['[data-region-id="rect-1"]'],
      strokeColor: "#000000",
      fillColor: "#ffffff",
    };
    act(() => {
      form.set("data.states.base.regions", [region]);
    });

    const rect = container.querySelector("rect") as SVGRectElement;
    expect(rect).toBeTruthy();
    expect(rect.getAttribute("stroke")).toBe("#000000");

    // Editing the color the way RegionList does: set the keyed region path.
    act(() => {
      form.set("data.states.base.regions.region-1.strokeColor", "#ff0000");
    });

    expect(rect.getAttribute("stroke")).toBe("#ff0000");
  });
});

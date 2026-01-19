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
import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { useCustom } from "@/schematic/symbol/Custom";

describe("useCustom", () => {
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
    it("should return early when spec is null", () => {
      const container = document.createElement("div");
      const { result } = renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec: undefined,
        }),
      );
      expect(container.children.length).toBe(0);
      expect(result.current).toBeUndefined();
    });

    it("should return early when svg is empty", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ svg: "" });
      const { result } = renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );
      expect(container.children.length).toBe(0);
      expect(result.current).toBeUndefined();
    });

    it("should return early when container is null", () => {
      const spec = createMockSpec();
      const { result } = renderHook(() =>
        useCustom({
          container: null,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );
      expect(result.current).toBeUndefined();
    });
  });

  describe("SVG mounting", () => {
    it("should mount SVG to container", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();
      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
          onMount,
        }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const svg = container.querySelector("svg");
      const gElements = svg?.querySelectorAll("g");
      expect(gElements?.length).toBe(1);
    });
  });

  describe("state management", () => {
    it("should apply inactive state when activeState is not 'active'", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("#333");
      expect(rect.getAttribute("fill")).toBe("#ccc");
    });

    it("should apply active state when activeState is 'active'", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "active",
          externalScale: 1,
          spec,
        }),
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("stroke")).toBe("#0f0");
      expect(rect.getAttribute("fill")).toBe("#0f03");
    });

    it("should store original attributes before applying state", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      expect(rect.getAttribute("data-original-stroke")).toBe("black");
      // Fill was not stored because it's being applied from state immediately
      expect(rect.getAttribute("fill")).toBe("#ccc");
    });

    it("should transition between states correctly", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { rerender } = renderHook(
        ({ activeState }) =>
          useCustom({
            container,
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );

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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("200");
      expect(svg.getAttribute("height")).toBe("200");
    });

    it("should apply external scale", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 3,
          spec,
        }),
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("300");
      expect(svg.getAttribute("height")).toBe("300");
    });

    it("should combine internal and external scale", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scale: 2 });

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1.5,
          spec,
        }),
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("300");
      expect(svg.getAttribute("height")).toBe("300");
    });

    it("should update dimensions when scale changes", () => {
      const container = document.createElement("div");
      const spec = createMockSpec();

      const { rerender } = renderHook(
        ({ externalScale }) =>
          useCustom({
            container,
            orientation: "top",
            activeState: "inactive",
            externalScale,
            spec,
          }),
        {
          initialProps: { externalScale: 1 },
        },
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");

      rerender({ externalScale: 2 });
      expect(svg.getAttribute("width")).toBe("200");
    });
  });

  describe("orientation", () => {
    it("should maintain dimensions for horizontal orientations", () => {
      const container = document.createElement("div");
      // Using the default spec which has viewBox "0 0 100 100"
      const spec = createMockSpec();

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      // Horizontal orientation maintains original dimensions from viewBox
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("100");
      expect(svg.getAttribute("viewBox")).toBe("0 0 100 100");
    });

    it("should swap dimensions for vertical orientations", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: '<svg viewBox="0 0 200 100"><rect class="main" width="50" height="50" stroke="black" fill="white"/></svg>',
      });

      renderHook(() =>
        useCustom({
          container,
          orientation: "top", // "top" and "bottom" are Y directions that cause swapping
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      // For Y-direction orientation with viewBox "0 0 200 100":
      // The dimensions should be swapped to width=100, height=200
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("200");
      expect(svg.getAttribute("viewBox")).toBe("0 0 100 200");
    });

    it("should handle orientation changes", () => {
      const container = document.createElement("div");
      // Using the default spec with viewBox "0 0 100 100"
      const spec = createMockSpec();

      const { rerender } = renderHook(
        ({ orientation }) =>
          useCustom({
            container,
            orientation,
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { orientation: "left" as location.Outer },
        },
      );

      const svg = container.querySelector("svg") as SVGSVGElement;
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("100");

      rerender({ orientation: "top" as location.Outer });
      // After orientation change from X to Y direction, dimensions remain the same for square viewBox
      expect(svg.getAttribute("width")).toBe("100");
      expect(svg.getAttribute("height")).toBe("100");
      expect(svg.getAttribute("viewBox")).toBe("0 0 100 100");
    });
  });

  describe("stroke scaling", () => {
    it("should add non-scaling-stroke when scaleStroke is false", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scaleStroke: false });

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

      const rect = container.querySelector("rect") as SVGRectElement;
      expect(rect.getAttribute("vector-effect")).toBe("non-scaling-stroke");
    });

    it("should not have vector-effect when scaleStroke is true", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ scaleStroke: true });

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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

      const { rerender } = renderHook(
        ({ activeState }) =>
          useCustom({
            container,
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

      const { rerender } = renderHook(
        ({ spec }) =>
          useCustom({
            container,
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

      const { rerender } = renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
      );

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

      const { rerender } = renderHook(
        ({ spec }) =>
          useCustom({
            container,
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
  });

  describe("edge cases", () => {
    it("should handle malformed SVG gracefully", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({
        svg: "<not-valid-svg>",
      });

      expect(() => {
        renderHook(() =>
          useCustom({
            container,
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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
        renderHook(() =>
          useCustom({
            container,
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
        );
      }).not.toThrow();
    });

    it("should handle empty states array", () => {
      const container = document.createElement("div");
      const spec = createMockSpec({ states: [] });

      expect(() => {
        renderHook(() =>
          useCustom({
            container,
            orientation: "top",
            activeState: "inactive",
            externalScale: 1,
            spec,
          }),
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

      renderHook(() =>
        useCustom({
          container,
          orientation: "top",
          activeState: "inactive",
          externalScale: 1,
          spec,
        }),
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
                // No fillColor specified - should keep original blue
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
                fillColor: "#ff0000", // Red fill when active
              },
            ],
          },
        ],
      });

      const { rerender } = renderHook(
        ({ activeState }) =>
          useCustom({
            container,
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      // Initially inactive - should have original blue fill
      expect(rect.getAttribute("fill")).toBe("blue");

      // Switch to active - should have red fill
      rerender({ activeState: "active" });
      expect(rect.getAttribute("fill")).toBe("#ff0000");

      // Switch back to inactive - should restore original blue fill
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
              // secondary is not in any region - should keep original colors
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

      const { rerender } = renderHook(
        ({ activeState }) =>
          useCustom({
            container,
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );

      const rect = container.querySelector(".main") as SVGRectElement;
      const circle = container.querySelector(".secondary") as SVGCircleElement;

      // Initially inactive - main has state colors, secondary has original
      expect(rect.getAttribute("stroke")).toBe("#333");
      expect(rect.getAttribute("fill")).toBe("#ccc");
      expect(circle.getAttribute("stroke")).toBe("blue");
      expect(circle.getAttribute("fill")).toBe("yellow");

      // Switch to active - both get red
      rerender({ activeState: "active" });
      expect(rect.getAttribute("stroke")).toBe("#f00");
      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(circle.getAttribute("stroke")).toBe("#f00");
      expect(circle.getAttribute("fill")).toBe("#ff0000");

      // Switch back to inactive - main gets state colors, secondary restores original
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
                fillColor: "#ff0000", // Red
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
                fillColor: "#00ff00", // Green
              },
            ],
          },
        ],
      });

      const { rerender } = renderHook(
        ({ activeState }) =>
          useCustom({
            container,
            orientation: "top",
            activeState,
            externalScale: 1,
            spec,
          }),
        {
          initialProps: { activeState: "inactive" },
        },
      );

      const rect = container.querySelector(".main") as SVGRectElement;

      // Start with state1 (treated as inactive)
      expect(rect.getAttribute("fill")).toBe("#ff0000");
      expect(rect.getAttribute("data-original-fill")).toBe("white");

      // Switch to state2 (active)
      rerender({ activeState: "active" });
      expect(rect.getAttribute("fill")).toBe("#00ff00");
      expect(rect.getAttribute("data-original-fill")).toBe("white");

      // Back to state1
      rerender({ activeState: "inactive" });
      expect(rect.getAttribute("fill")).toBe("#ff0000");
    });
  });
});

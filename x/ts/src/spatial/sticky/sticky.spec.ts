// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { box } from "@/spatial/box";
import { type location } from "@/spatial/location";
import { sticky } from "@/spatial/sticky";

describe("sticky", () => {
  describe("toCSS", () => {
    interface Spec {
      pos: sticky.XY;
      expected: Partial<Record<location.Outer, string>>;
    }
    const SPECS: Spec[] = [
      {
        pos: { x: 10, y: 20 },
        expected: { left: "1000%", top: "2000%" },
      },
      {
        pos: { x: 10, y: 20, units: { x: "px", y: "px" } },
        expected: { left: "10px", top: "20px" },
      },
      {
        pos: { x: 10, y: 20, units: { x: "decimal", y: "decimal" } },
        expected: { left: "1000%", top: "2000%" },
      },
      {
        pos: { x: 10, y: 20, units: { x: "px", y: "decimal" } },
        expected: { left: "10px", top: "2000%" },
      },
      {
        pos: { x: 10, y: 20, units: { x: "decimal", y: "px" } },
        expected: { left: "1000%", top: "20px" },
      },
      {
        pos: { x: 10, y: 20, root: { x: "right", y: "bottom" } },
        expected: { right: "1000%", bottom: "2000%" },
      },
      {
        pos: {
          x: 10,
          y: 20,
          root: { x: "right", y: "bottom" },
          units: { x: "px", y: "px" },
        },
        expected: { right: "10px", bottom: "20px" },
      },
      {
        pos: {
          x: 10,
          y: 20,
          root: { x: "left", y: "bottom" },
          units: { x: "decimal", y: "px" },
        },
        expected: { left: "1000%", bottom: "20px" },
      },
      {
        pos: {
          x: 10,
          y: 20,
          root: { x: "right", y: "top" },
          units: { x: "px", y: "decimal" },
        },
        expected: { right: "10px", top: "2000%" },
      },
      {
        pos: { x: 0, y: 0 },
        expected: { left: "0%", top: "0%" },
      },
      {
        pos: { x: 0.5, y: 0.5 },
        expected: { left: "50%", top: "50%" },
      },
      {
        pos: { x: 0.5, y: 0.5, units: { x: "decimal", y: "decimal" } },
        expected: { left: "50%", top: "50%" },
      },
      {
        pos: { x: 0.1, y: 0.9, root: { x: "right", y: "bottom" } },
        expected: { right: "10%", bottom: "90%" },
      },
    ];
    SPECS.forEach(({ pos, expected }, i) => {
      test(`toCSS ${i}`, () => {
        expect(sticky.toCSS(pos)).toEqual(expected);
      });
    });
  });
  describe("xy schema", () => {
    interface Spec {
      value: unknown;
      valid: boolean;
    }
    const SPECS: Spec[] = [
      { value: { x: 10, y: 20 }, valid: true },
      { value: { x: 0, y: 0 }, valid: true },
      { value: { x: -10, y: -20 }, valid: true },
      { value: { x: 10, y: 20, root: { x: "left", y: "top" } }, valid: true },
      { value: { x: 10, y: 20, root: { x: "right", y: "bottom" } }, valid: true },
      {
        value: { x: 10, y: 20, units: { x: "px", y: "px" } },
        valid: true,
      },
      {
        value: { x: 10, y: 20, units: { x: "decimal", y: "decimal" } },
        valid: true,
      },
      {
        value: {
          x: 10,
          y: 20,
          root: { x: "left", y: "top" },
          units: { x: "px", y: "decimal" },
        },
        valid: true,
      },
      { value: { x: 10 }, valid: false },
      { value: { y: 20 }, valid: false },
      { value: {}, valid: false },
      { value: { x: "10", y: 20 }, valid: false },
      { value: { x: 10, y: "20" }, valid: false },
      { value: { x: 10, y: 20, root: { x: "invalid", y: "top" } }, valid: false },
      { value: { x: 10, y: 20, root: { x: "left", y: "invalid" } }, valid: false },
      { value: { x: 10, y: 20, units: { x: "invalid", y: "px" } }, valid: false },
      { value: { x: 10, y: 20, units: { x: "px", y: "invalid" } }, valid: false },
    ];
    SPECS.forEach(({ value, valid }, i) => {
      test(`xy schema ${i}`, () => {
        const result = sticky.xy.safeParse(value);
        expect(result.success).toBe(valid);
      });
    });
  });
  describe("completeXY schema", () => {
    interface Spec {
      value: unknown;
      valid: boolean;
    }
    const SPECS: Spec[] = [
      {
        value: {
          x: 10,
          y: 20,
          root: { x: "left", y: "top" },
          units: { x: "px", y: "px" },
        },
        valid: true,
      },
      {
        value: {
          x: 10,
          y: 20,
          root: { x: "right", y: "bottom" },
          units: { x: "decimal", y: "decimal" },
        },
        valid: true,
      },
      {
        value: {
          x: 0,
          y: 0,
          root: { x: "left", y: "top" },
          units: { x: "px", y: "px" },
        },
        valid: true,
      },
      { value: { x: 10, y: 20 }, valid: false },
      { value: { x: 10, y: 20, root: { x: "left", y: "top" } }, valid: false },
      { value: { x: 10, y: 20, units: { x: "px", y: "px" } }, valid: false },
      {
        value: {
          x: 10,
          y: 20,
          root: { x: "invalid", y: "top" },
          units: { x: "px", y: "px" },
        },
        valid: false,
      },
      {
        value: {
          x: 10,
          y: 20,
          root: { x: "left", y: "top" },
          units: { x: "invalid", y: "px" },
        },
        valid: false,
      },
    ];
    SPECS.forEach(({ value, valid }, i) => {
      test(`completeXY schema ${i}`, () => {
        const result = sticky.completeXY.safeParse(value);
        expect(result.success).toBe(valid);
      });
    });
  });
  describe("toDecimal", () => {
    interface Spec {
      pos: sticky.XY;
      element: box.Box;
      container: box.Box;
      expected: { x: number; y: number };
    }
    const SPECS: Spec[] = [
      {
        pos: { x: 100, y: 200, units: { x: "px", y: "px" } },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: { x: 0.1, y: 0.2 },
      },
      {
        pos: {
          x: 100,
          y: 200,
          root: { x: "right", y: "bottom" },
          units: { x: "px", y: "px" },
        },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: { x: 0.85, y: 0.75 },
      },
      {
        pos: { x: 0.5, y: 0.25, units: { x: "decimal", y: "decimal" } },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: { x: 0.5, y: 0.25 },
      },
      {
        pos: {
          x: 0.3,
          y: 0.7,
          root: { x: "right", y: "bottom" },
          units: { x: "decimal", y: "decimal" },
        },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: { x: 0.7, y: 0.3 },
      },
      {
        pos: {
          x: 50,
          y: 100,
          root: { x: "left", y: "top" },
          units: { x: "px", y: "px" },
        },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 500, 500),
        expected: { x: 0.1, y: 0.2 },
      },
      {
        pos: { x: 0, y: 0, units: { x: "px", y: "px" } },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: { x: 0, y: 0 },
      },
      {
        pos: {
          x: 0,
          y: 0,
          root: { x: "right", y: "bottom" },
          units: { x: "px", y: "px" },
        },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: { x: 0.95, y: 0.95 },
      },
      {
        pos: {
          x: 200,
          y: 150,
          root: { x: "left", y: "bottom" },
          units: { x: "px", y: "px" },
        },
        element: box.construct(0, 0, 60, 40),
        container: box.construct(0, 0, 800, 600),
        expected: { x: 0.25, y: 0.68333 },
      },
      {
        pos: {
          x: 200,
          y: 150,
          root: { x: "right", y: "top" },
          units: { x: "px", y: "px" },
        },
        element: box.construct(0, 0, 60, 40),
        container: box.construct(0, 0, 800, 600),
        expected: { x: 0.675, y: 0.25 },
      },
    ];
    SPECS.forEach(({ pos, element, container, expected }, i) => {
      test(`toDecimal ${i}`, () => {
        const result = sticky.toDecimal(pos, element, container);
        expect(result.x).toBeCloseTo(expected.x, 4);
        expect(result.y).toBeCloseTo(expected.y, 4);
      });
    });
  });
  describe("calculate", () => {
    interface Spec {
      pos: sticky.XY;
      element: box.Box;
      container: box.Box;
      expected: {
        x: number;
        y: number;
        root: { x: "left" | "right"; y: "top" | "bottom" };
        units: { x: "px" | "decimal"; y: "px" | "decimal" };
      };
    }
    const SPECS: Spec[] = [
      {
        pos: { x: 0.1, y: 0.1 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: {
          x: 100,
          y: 100,
          root: { x: "left", y: "top" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.9, y: 0.9 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: {
          x: 50,
          y: 50,
          root: { x: "right", y: "bottom" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.5, y: 0.5 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: {
          x: 0.5,
          y: 0.5,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0.05, y: 0.95 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 500, 500),
        expected: {
          x: 25,
          y: -75,
          root: { x: "left", y: "bottom" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.95, y: 0.15 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 500, 500),
        expected: {
          x: -75,
          y: 75,
          root: { x: "right", y: "top" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.3, y: 0.7 },
        element: box.construct(0, 0, 80, 60),
        container: box.construct(0, 0, 800, 600),
        expected: {
          x: 0.3,
          y: 0.7,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0, y: 0 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: {
          x: 0,
          y: 0,
          root: { x: "left", y: "top" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 1, y: 1 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        expected: {
          x: -50,
          y: -50,
          root: { x: "right", y: "bottom" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.2, y: 0.2 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 500, 500),
        expected: {
          x: 0.2,
          y: 0.2,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0.8, y: 0.8 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 500, 500),
        expected: {
          x: 0.8,
          y: 0.8,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
    ];
    SPECS.forEach(({ pos, element, container, expected }, i) => {
      test(`calculate ${i}`, () => {
        const result = sticky.calculate(pos, element, container);
        expect(result).not.toBeNull();
        if (result == null) return;
        expect(result.x).toBeCloseTo(expected.x, 2);
        expect(result.y).toBeCloseTo(expected.y, 2);
        expect(result.root).toEqual(expected.root);
        expect(result.units).toEqual(expected.units);
      });
    });
  });
  describe("calculate with custom thresholds", () => {
    interface Spec {
      pos: sticky.XY;
      element: box.Box;
      container: box.Box;
      options: sticky.CalculateOptions;
      expected: {
        x: number;
        y: number;
        root: { x: "left" | "right"; y: "top" | "bottom" };
        units: { x: "px" | "decimal"; y: "px" | "decimal" };
      };
    }
    const SPECS: Spec[] = [
      {
        pos: { x: 0.3, y: 0.3 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        options: { lowerThreshold: 0.4, upperThreshold: 0.6 },
        expected: {
          x: 0.3,
          y: 0.3,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0.3, y: 0.3 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        options: { lowerThreshold: 0.1, upperThreshold: 0.9 },
        expected: {
          x: 0.3,
          y: 0.3,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0.25, y: 0.75 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        options: { lowerThreshold: 0.3, upperThreshold: 0.7 },
        expected: {
          x: 250,
          y: 250,
          root: { x: "left", y: "bottom" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.5, y: 0.5 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 500, 500),
        options: { lowerThreshold: 0.5, upperThreshold: 0.5 },
        expected: {
          x: 0.5,
          y: 0.5,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0.6, y: 0.4 },
        element: box.construct(0, 0, 50, 50),
        container: box.construct(0, 0, 1000, 1000),
        options: { lowerThreshold: 0.3, upperThreshold: 0.5 },
        expected: {
          x: 350,
          y: 0.4,
          root: { x: "right", y: "top" },
          units: { x: "px", y: "decimal" },
        },
      },
      {
        pos: { x: 0.1, y: 0.95 },
        element: box.construct(0, 0, 80, 60),
        container: box.construct(0, 0, 800, 600),
        options: { lowerThreshold: 0.15, upperThreshold: 0.9 },
        expected: {
          x: 80,
          y: -30,
          root: { x: "left", y: "bottom" },
          units: { x: "px", y: "px" },
        },
      },
      {
        pos: { x: 0.4, y: 0.6 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 1000, 1000),
        options: { lowerThreshold: 0, upperThreshold: 1 },
        expected: {
          x: 0.4,
          y: 0.6,
          root: { x: "left", y: "top" },
          units: { x: "decimal", y: "decimal" },
        },
      },
      {
        pos: { x: 0.4, y: 0.6 },
        element: box.construct(0, 0, 100, 100),
        container: box.construct(0, 0, 1000, 1000),
        options: { lowerThreshold: 0.5, upperThreshold: 0.5 },
        expected: {
          x: 400,
          y: -400,
          root: { x: "left", y: "bottom" },
          units: { x: "px", y: "px" },
        },
      },
    ];
    SPECS.forEach(({ pos, element, container, options, expected }, i) => {
      test(`calculate with custom thresholds ${i}`, () => {
        const result = sticky.calculate(pos, element, container, options);
        expect(result).not.toBeNull();
        if (result == null) return;
        expect(result.x).toBeCloseTo(expected.x, 2);
        expect(result.y).toBeCloseTo(expected.y, 2);
        expect(result.root).toEqual(expected.root);
        expect(result.units).toEqual(expected.units);
      });
    });
  });
});

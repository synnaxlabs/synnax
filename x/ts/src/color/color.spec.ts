// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { color } from "@/color";

describe("color.Color", () => {
  describe("constructor", () => {
    test("from hex", () => {
      const c = color.construct("#7a2c26");
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
    });
    test("from hex with alpha", () => {
      const c = color.construct("#7a2c26", 128);
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
      expect(color.aValue(c)).toEqual(128);
    });

    describe("from eight digit hex", () => {
      test("case 1", () => {
        const c = color.construct("#7a2c26ff");
        expect(color.rValue(c)).toEqual(122);
        expect(color.gValue(c)).toEqual(44);
        expect(color.bValue(c)).toEqual(38);
        expect(color.aValue(c)).toEqual(255);
      });
      test("case 2", () => {
        const c = color.construct("#7a2c2605");
        expect(color.rValue(c)).toEqual(122);
        expect(color.gValue(c)).toEqual(44);
        expect(color.bValue(c)).toEqual(38);
        expect(color.aValue(c)).toEqual(5);
      });
    });

    test("from rgb", () => {
      const c = color.construct([122, 44, 38]);
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
    });
    test("from rgba", () => {
      const c = color.construct([122, 44, 38, 128]);
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
      expect(color.aValue(c)).toEqual(128);
    });
    test("from c", () => {
      const c = color.construct(color.construct("#7a2c26"));
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
    });

    test("from legacy object", () => {
      const c = color.construct({ rgba255: [122, 44, 38, 128] });
      expect(color.rValue(c)).toEqual(122);
      expect(color.gValue(c)).toEqual(44);
      expect(color.bValue(c)).toEqual(38);
      expect(color.aValue(c)).toEqual(128);
    });
  });

  describe("to hex", () => {
    test("without alpha", () => {
      const c = color.construct("#7a2c26");
      expect(color.hex(c)).toEqual("#7a2c26");
    });
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 128);
      expect(color.hex(c)).toEqual("#7a2c2680");
    });
  });

  describe("to RGBA255", () => {
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 128);
      const expected = [122, 44, 38, 128];
      expect(color.construct(c)).toEqual(expected);
    });
    test("without alpha", () => {
      const c = color.construct("#7a2c26");
      const expected = [122, 44, 38, 255];
      expect(color.construct(c)).toEqual(expected);
    });
  });

  describe("to RGBA1", () => {
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 128);
      const expected = [122 / 255, 44 / 255, 38 / 255, 128];
      expected.forEach((v, i) => {
        expect(color.rgba1(c)[i]).toBeCloseTo(v);
      });
    });
    test("without alpha", () => {
      const c = color.construct("#7a2c26");
      const expected = [122 / 255, 44 / 255, 38 / 255, 255];
      expected.forEach((v, i) => {
        expect(color.rgba1(c)[i]).toBeCloseTo(v);
      });
    });
  });

  describe("luminance", () => {
    const tests: Array<[string, number]> = [
      ["#000000", 0],
      ["#ffffff", 1],
    ];
    tests.forEach(([hex, expected]) => {
      test(hex, () => {
        const c = color.construct(hex);
        expect(color.luminance(c)).toBeCloseTo(expected);
      });
    });
  });

  describe("contrast", () => {
    const tests: Array<[string, string, number]> = [
      ["#000000", "#ffffff", 3],
      ["#ffffff", "#000000", 3],
      ["#000000", "#000000", 1],
      ["#ffffff", "#ffffff", 1],
    ];
    tests.forEach(([hex1, hex2, expected]) => {
      test(`${hex1} ${hex2}`, () => {
        const c1 = color.construct(hex1);
        const c2 = color.construct(hex2);
        expect(color.contrast(c1, c2)).toBeCloseTo(expected);
      });
    });
    test("pick c with highest contrast", () => {
      const c = color.construct("#000000");
      const c1 = color.construct("#ffffff");
      const c2 = color.construct("#0000ff");
      expect(color.pickByContrast(c, c1, c2)).toEqual(c1);
    });
  });

  describe("isDark", () => {
    test("black is dark", () => {
      expect(color.isDark("#000000")).toBe(true);
    });

    test("white is not dark", () => {
      expect(color.isDark("#ffffff")).toBe(false);
    });

    test("mid gray is not dark", () => {
      expect(color.isDark("#808080")).toBe(true);
    });

    test("dark red is dark", () => {
      expect(color.isDark("#800000")).toBe(true);
    });

    test("light blue is not dark", () => {
      expect(color.isDark("#add8e6")).toBe(false);
    });

    test("handles RGB array input", () => {
      expect(color.isDark([0, 0, 0])).toBe(true);
      expect(color.isDark([255, 255, 255])).toBe(false);
    });

    test("handles RGBA array input", () => {
      expect(color.isDark([0, 0, 0, 128])).toBe(true);
      expect(color.isDark([255, 255, 255, 128])).toBe(false);
    });

    test("handles Color object input", () => {
      const darkColor = color.construct("#000000");
      const lightColor = color.construct("#ffffff");
      expect(color.isDark(darkColor)).toBe(true);
      expect(color.isDark(lightColor)).toBe(false);
    });
  });

  describe("isLight", () => {
    test("white is light", () => {
      expect(color.isLight("#ffffff")).toBe(true);
    });

    test("black is not light", () => {
      expect(color.isLight("#000000")).toBe(false);
    });

    test("mid gray is not light", () => {
      expect(color.isLight("#808080")).toBe(false);
    });

    test("dark red is not light", () => {
      expect(color.isLight("#800000")).toBe(false);
    });

    test("light blue is light", () => {
      expect(color.isLight("#add8e6")).toBe(true);
    });

    test("handles RGB array input", () => {
      expect(color.isLight([255, 255, 255])).toBe(true);
      expect(color.isLight([0, 0, 0])).toBe(false);
    });

    test("handles RGBA array input", () => {
      expect(color.isLight([255, 255, 255, 128])).toBe(true);
      expect(color.isLight([0, 0, 0, 128])).toBe(false);
    });

    test("handles Color object input", () => {
      const lightColor = color.construct("#ffffff");
      const darkColor = color.construct("#000000");
      expect(color.isLight(lightColor)).toBe(true);
      expect(color.isLight(darkColor)).toBe(false);
    });
  });

  describe("grayness", () => {
    const tests: Array<[string, number]> = [
      ["#000000", 1],
      ["#ffffff", 1],
      ["#0000ff", 0],
      ["#00ff00", 0],
      ["#ff0000", 0],
      ["#ffff00", 0],
      ["#fefed4", 0.834],
      ["#5c6670", 0.92],
      ["#d3c5c5", 0.945],
    ];
    tests.forEach(([hex, expected]) => {
      test(hex, () => {
        const c = color.construct(hex);
        expect(color.grayness(c)).toBeCloseTo(expected);
      });
    });
  });

  describe("fromHSLA", () => {
    const tests: Array<
      [string, [number, number, number, number], [number, number, number, number]]
    > = [
      ["Red", [0, 100, 50, 255], [255, 0, 0, 255]],
      ["Green", [120, 100, 50, 255], [0, 255, 0, 255]],
      ["Blue", [240, 100, 50, 255], [0, 0, 255, 255]],
      ["Yellow", [60, 100, 50, 255], [255, 255, 0, 255]],
      ["Cyan", [180, 100, 50, 255], [0, 255, 255, 255]],
      ["Magenta", [300, 100, 50, 255], [255, 0, 255, 255]],
      ["Black", [0, 0, 0, 255], [0, 0, 0, 255]],
      ["White", [0, 0, 100, 255], [255, 255, 255, 255]],
      ["Mid Gray", [0, 0, 50, 255], [128, 128, 128, 255]],
      ["Semi-transparent Red", [0, 100, 50, 128], [255, 0, 0, 128]],
    ];

    tests.forEach(([name, hsla, expected]) => {
      test(name, () => {
        const result = color.fromHSLA(hsla);
        expect(result).toEqual(expected);
      });
    });

    test("converts HSLA to RGBA", () => {
      const hsla: color.HSLA = [0, 100, 50, 255]; // Red in HSLA
      const expected: color.RGBA = [255, 0, 0, 255]; // Red in RGBA
      expect(color.fromHSLA(hsla)).toEqual(expected);
    });

    test("handles hue wrapping properly", () => {
      const hsla1: color.HSLA = [0, 100, 50, 255]; // 0 degrees = red
      const hsla2: color.HSLA = [360, 100, 50, 255]; // 360 degrees = red
      expect(color.fromHSLA(hsla1)).toEqual(color.fromHSLA(hsla2));
    });

    test("handles zero saturation (grayscale)", () => {
      // For zero saturation, hue doesn't matter, only lightness
      const white: color.HSLA = [0, 0, 100, 255];
      const black: color.HSLA = [0, 0, 0, 255];
      const gray: color.HSLA = [0, 0, 50, 255];

      expect(color.fromHSLA(white)).toEqual([255, 255, 255, 255]);
      expect(color.fromHSLA(black)).toEqual([0, 0, 0, 255]);
      expect(color.fromHSLA(gray)).toEqual([128, 128, 128, 255]);
    });

    test("preserves alpha value", () => {
      const transparent: color.HSLA = [0, 100, 50, 0]; // Transparent red
      const semiTransparent: color.HSLA = [0, 100, 50, 128]; // Semi-transparent red

      expect(color.fromHSLA(transparent)[3]).toEqual(0);
      expect(color.fromHSLA(semiTransparent)[3]).toEqual(128);
    });
  });

  describe("hsla", () => {
    const tests: Array<
      [string, [number, number, number, number], [number, number, number, number]]
    > = [
      // Test primary colors
      ["Red", [255, 0, 0, 255], [0, 100, 50, 255]],
      ["Green", [0, 255, 0, 255], [120, 100, 50, 255]],
      ["Blue", [0, 0, 255, 255], [240, 100, 50, 255]],

      // Test secondary colors
      ["Yellow", [255, 255, 0, 255], [60, 100, 50, 255]],
      ["Cyan", [0, 255, 255, 255], [180, 100, 50, 255]],
      ["Magenta", [255, 0, 255, 255], [300, 100, 50, 255]],

      // Test shades of gray
      ["Black", [0, 0, 0, 255], [0, 0, 0, 255]],
      ["White", [255, 255, 255, 255], [0, 0, 100, 255]],
      ["Mid Gray", [128, 128, 128, 255], [0, 0, 50, 255]],

      // Test different alpha values
      ["Transparent Red", [255, 0, 0, 0], [0, 100, 50, 0]],
      ["Semi-transparent Blue", [0, 0, 255, 128], [240, 100, 50, 128]],

      // Test different lightness levels
      ["Dark Red", [128, 0, 0, 255], [0, 100, 25, 255]],
      ["Light Blue", [128, 128, 255, 255], [240, 100, 75, 255]],

      // Test different saturation levels
      ["Desaturated Red", [191, 64, 64, 255], [0, 50, 50, 255]],
      ["Slightly Saturated Green", [96, 159, 96, 255], [120, 25, 50, 255]],
    ];

    tests.forEach(([name, rgba, expected]) => {
      test(name, () => {
        const result = color.hsla(rgba);
        // Note: Due to potential rounding differences in conversion,
        // we use toBeCloseTo for HSL values with precision 0
        for (let i = 0; i < 3; i++) expect(result[i]).toBeCloseTo(expected[i], 0);

        // Alpha should match exactly
        expect(result[3]).toEqual(expected[3]);
      });
    });

    test("handles hex color input", () => {
      // Red in hex
      const hexColor = "#ff0000";
      const expected = [0, 100, 50, 255];
      const result = color.hsla(hexColor);

      for (let i = 0; i < 3; i++) expect(result[i]).toBeCloseTo(expected[i], 0);

      expect(result[3]).toEqual(expected[3]);
    });

    test("handles RGB array input", () => {
      // Green as RGB array
      const rgbColor: color.RGB = [0, 255, 0];
      const expected = [120, 100, 50, 255];
      const result = color.hsla(rgbColor);

      for (let i = 0; i < 3; i++) expect(result[i]).toBeCloseTo(expected[i], 0);

      expect(result[3]).toEqual(255); // Default alpha
    });

    test("preserves original color after round-trip conversion", () => {
      const originalColors: color.RGBA[] = [
        [255, 0, 0, 255], // Red
        [0, 255, 0, 255], // Green
        [0, 0, 255, 255], // Blue
        [255, 255, 0, 255], // Yellow
        [0, 255, 255, 255], // Cyan
        [255, 0, 255, 255], // Magenta
        [128, 128, 128, 255], // Gray
        [255, 255, 255, 128], // Semi-transparent white
      ];

      for (const original of originalColors) {
        const hsla = color.hsla(original);
        const converted = color.fromHSLA(hsla);

        // Compare RGB values with some tolerance for rounding
        for (let i = 0; i < 3; i++) expect(converted[i]).toBeCloseTo(original[i], 0);

        // Alpha should match exactly
        expect(converted[3]).toEqual(original[3]);
      }
    });

    test("handles achromatic colors correctly", () => {
      // For achromatic colors (black, white, gray),
      // hue is 0 and saturation is 0
      const gray: color.RGBA = [100, 100, 100, 255];
      const result = color.hsla(gray);

      expect(result[0]).toEqual(0); // Hue
      expect(result[1]).toEqual(0); // Saturation
      expect(result[2]).toBeCloseTo(39, 0); // Lightness ~39%
      expect(result[3]).toEqual(255); // Alpha
    });
  });

  describe("setAlpha", () => {
    test("sets alpha on RGB color", () => {
      const rgb: color.RGB = [255, 0, 0];
      const result = color.setAlpha(rgb, 128);

      expect(result).toEqual([255, 0, 0, 128]);
    });

    test("sets alpha on RGBA color", () => {
      const rgba: color.RGBA = [0, 255, 0, 255];
      const result = color.setAlpha(rgba, 77);

      expect(result).toEqual([0, 255, 0, 77]);
    });

    test("sets alpha on hex color", () => {
      const hex = "#0000ff";
      const result = color.setAlpha(hex, 179);

      expect(result).toEqual([0, 0, 255, 179]);
    });

    test("overrides existing alpha in RGBA color", () => {
      const rgba: color.RGBA = [128, 128, 128, 51];
      const result = color.setAlpha(rgba, 204);

      expect(result).toEqual([128, 128, 128, 204]);
    });

    test("handles alpha value of 0", () => {
      const color1 = color.construct("#ff0000");
      const result = color.setAlpha(color1, 0);

      expect(result[3]).toEqual(0);
    });

    test("handles alpha value of 255", () => {
      const color1 = color.construct("#ff0000", 128);
      const result = color.setAlpha(color1, 255);

      expect(result[3]).toEqual(255);
    });

    test("accepts alpha values directly in 0-255 range", () => {
      const color1 = color.construct("#ff0000");
      const result = color.setAlpha(color1, 128);

      expect(result[3]).toEqual(128);
    });

    test("throws error for alpha values > 255", () => {
      const color1 = color.construct("#ff0000");
      expect(() => color.setAlpha(color1, 256)).toThrow();
    });

    test("preserves RGB values when setting alpha", () => {
      const originalColor: color.RGBA = [123, 45, 67, 204];
      const result = color.setAlpha(originalColor, 102);

      expect(result[0]).toEqual(123);
      expect(result[1]).toEqual(45);
      expect(result[2]).toEqual(67);
    });

    test("return type is Color (RGBA format)", () => {
      const hex = "#123456";
      const result = color.setAlpha(hex, 128);

      expect(Array.isArray(result)).toBe(true);
      expect(result.length).toEqual(4);
      expect(typeof result[3]).toBe("number");
    });
  });

  describe("rgbaCSS", () => {
    test("converts RGB array to CSS rgba string", () => {
      const rgb: color.RGB = [255, 0, 0];
      expect(color.rgbaCSS(rgb)).toEqual("rgba(255, 0, 0, 255)");
    });

    test("converts RGBA array to CSS rgba string", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.rgbaCSS(rgba)).toEqual("rgba(0, 255, 0, 128)");
    });

    test("converts hex to CSS rgba string", () => {
      const hex = "#0000ff";
      expect(color.rgbaCSS(hex)).toEqual("rgba(0, 0, 255, 255)");
    });

    test("converts hex with alpha to CSS rgba string", () => {
      const hex = "#ff000080";
      expect(color.rgbaCSS(hex)).toEqual("rgba(255, 0, 0, 128)");
    });

    test("handles Color object", () => {
      const c = color.construct([128, 128, 128, 179]);
      expect(color.rgbaCSS(c)).toEqual("rgba(128, 128, 128, 179)");
    });
  });

  describe("rgbCSS", () => {
    test("converts RGB array to CSS rgb string", () => {
      const rgb: color.RGB = [255, 0, 0];
      expect(color.rgbCSS(rgb)).toEqual("rgb(255, 0, 0)");
    });

    test("converts RGBA array to CSS rgb string (ignores alpha)", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.rgbCSS(rgba)).toEqual("rgb(0, 255, 0)");
    });

    test("converts hex to CSS rgb string", () => {
      const hex = "#0000ff";
      expect(color.rgbCSS(hex)).toEqual("rgb(0, 0, 255)");
    });

    test("handles Color object", () => {
      const c = color.construct([128, 128, 128, 179]);
      expect(color.rgbCSS(c)).toEqual("rgb(128, 128, 128)");
    });
  });

  describe("rgbString", () => {
    test("converts RGB array to comma-separated string", () => {
      const rgb: color.RGB = [255, 0, 0];
      expect(color.rgbString(rgb)).toEqual("255, 0, 0");
    });

    test("converts RGBA array to comma-separated string (ignores alpha)", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.rgbString(rgba)).toEqual("0, 255, 0");
    });

    test("converts hex to comma-separated string", () => {
      const hex = "#0000ff";
      expect(color.rgbString(hex)).toEqual("0, 0, 255");
    });

    test("handles Color object", () => {
      const c = color.construct([128, 128, 128, 179]);
      expect(color.rgbString(c)).toEqual("128, 128, 128");
    });
  });

  describe("equals", () => {
    test("same RGB values are equal", () => {
      const c1: color.RGB = [255, 0, 0];
      const c2: color.RGB = [255, 0, 0];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("different RGB values are not equal", () => {
      const c1: color.RGB = [255, 0, 0];
      const c2: color.RGB = [0, 255, 0];
      expect(color.equals(c1, c2)).toBe(false);
    });

    test("RGBA and RGB with default alpha are equal", () => {
      const c1: color.RGB = [255, 0, 0];
      const c2: color.RGBA = [255, 0, 0, 255];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("different alpha values are not equal", () => {
      const c1: color.RGBA = [255, 0, 0, 255];
      const c2: color.RGBA = [255, 0, 0, 128];
      expect(color.equals(c1, c2)).toBe(false);
    });

    test("hex and RGB with same values are equal", () => {
      const c1 = "#ff0000";
      const c2: color.RGB = [255, 0, 0];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("eight-digit hex and RGBA with same values are equal", () => {
      const c1 = "#ff000080";
      const c2: color.RGBA = [255, 0, 0, 128];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("comparing to undefined", () => {
      const c1: color.RGB = [255, 0, 0];
      expect(color.equals(c1, undefined)).toBe(false);
      expect(color.equals(undefined, c1)).toBe(false);
      expect(color.equals(undefined, undefined)).toBe(true);
    });

    test("comparing color with itself returns true", () => {
      const c = color.construct("#ff0000");
      expect(color.equals(c, c)).toBe(true);
    });
  });

  describe("cssString", () => {
    test("returns undefined for undefined input", () => {
      expect(color.cssString(undefined)).toBeUndefined();
    });

    test("returns undefined for null input", () => {
      expect(color.cssString(null as any)).toBeUndefined();
    });

    test("converts RGB array to rgba CSS string", () => {
      const rgb: color.RGB = [255, 0, 0];
      expect(color.cssString(rgb)).toEqual("rgba(255, 0, 0, 255)");
    });

    test("converts RGBA array to rgba CSS string", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.cssString(rgba)).toEqual("rgba(0, 255, 0, 128)");
    });

    test("converts hex to rgba CSS string", () => {
      const hex = "#0000ff";
      expect(color.cssString(hex)).toEqual("rgba(0, 0, 255, 255)");
    });

    test("returns CSS variables as-is", () => {
      const cssVar = "var(--primary-color)";
      expect(color.cssString(cssVar)).toEqual(cssVar);
    });

    test("handles eight-digit hex with alpha", () => {
      const hex = "#00ff0080";
      expect(color.cssString(hex)).toEqual("rgba(0, 255, 0, 128)");
    });

    test("handles Color object", () => {
      const c = color.construct([128, 128, 128, 179]);
      expect(color.cssString(c)).toEqual("rgba(128, 128, 128, 179)");
    });

    test("throws error for invalid color format", () => {
      expect(() => color.cssString({} as any)).toThrow();
      expect(() => color.cssString([1, 2] as any)).toThrow();
      expect(() => color.cssString([300, 0, 0] as any)).toThrow();
    });
  });

  describe("isCrude", () => {
    test("RGB array is a crude color", () => {
      const rgb: color.RGB = [255, 0, 0];
      expect(color.isCrude(rgb)).toBe(true);
    });

    test("RGBA array is a crude color", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.isCrude(rgba)).toBe(true);
    });

    test("hex string is a crude color", () => {
      expect(color.isCrude("#ff0000")).toBe(true);
    });

    test("hex string with hash is a crude color", () => {
      expect(color.isCrude("#00ff00")).toBe(true);
    });

    test("eight-digit hex is a crude color", () => {
      expect(color.isCrude("#0000ff80")).toBe(true);
    });

    test("Color object is a crude color", () => {
      const c = color.construct("#ff0000");
      expect(color.isCrude(c)).toBe(true);
    });

    test("rejects invalid hex strings", () => {
      expect(color.isCrude("#xyz")).toBe(false);
      expect(color.isCrude("#12345")).toBe(false);
      expect(color.isCrude("#1234567")).toBe(false);
      expect(color.isCrude("#123456789")).toBe(false);
    });

    test("rejects invalid RGB arrays", () => {
      expect(color.isCrude([255])).toBe(false);
      expect(color.isCrude([255, 0])).toBe(false);
      expect(color.isCrude([255, 0, 0, 128, 1])).toBe(false);
      expect(color.isCrude([-1, 0, 0])).toBe(false);
      expect(color.isCrude([0, 256, 0])).toBe(false);
    });

    test("rejects invalid RGBA arrays", () => {
      expect(color.isCrude([255, 0, 0, 256])).toBe(false);
      expect(color.isCrude([255, 0, 0, -1])).toBe(false);
    });

    test("rejects non-color values", () => {
      expect(color.isCrude(null)).toBe(false);
      expect(color.isCrude(undefined)).toBe(false);
      expect(color.isCrude({})).toBe(false);
      expect(color.isCrude("not a color")).toBe(false);
      expect(color.isCrude(123)).toBe(false);
      expect(color.isCrude(true)).toBe(false);
    });
  });

  describe("isColor", () => {
    test("valid RGBA array is a Color", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.isColor(rgba)).toBe(true);
    });

    test("RGB array with 3 elements is not a Color", () => {
      const rgb: color.RGB = [255, 0, 0];
      expect(color.isColor(rgb)).toBe(false);
    });

    test("constructed Color is a Color", () => {
      const c = color.construct("#ff0000");
      expect(color.isColor(c)).toBe(true);
    });

    test("hex string is not a Color", () => {
      expect(color.isColor("#ff0000")).toBe(false);
    });

    test("rejects invalid RGBA arrays", () => {
      expect(color.isColor([255, 0, 0])).toBe(false); // Missing alpha
      expect(color.isColor([255, 0, 0, 0, 0])).toBe(false); // Too many elements
      expect(color.isColor([255, 0, -1, 255])).toBe(false); // Negative value
      expect(color.isColor([255, 0, 256, 255])).toBe(false); // Value > 255
      expect(color.isColor([255, 0, 0, 256])).toBe(false); // Alpha > 255
      expect(color.isColor([255, 0, 0, -1])).toBe(false); // Alpha < 0
    });

    test("rejects non-array values", () => {
      expect(color.isColor(null)).toBe(false);
      expect(color.isColor(undefined)).toBe(false);
      expect(color.isColor({})).toBe(false);
      expect(color.isColor("rgba(255,0,0,1)")).toBe(false);
      expect(color.isColor(123)).toBe(false);
    });

    test("validates RGB values are within range", () => {
      expect(color.isColor([0, 0, 0, 0])).toBe(true);
      expect(color.isColor([255, 255, 255, 255])).toBe(true);
      expect(color.isColor([300, 0, 0, 255])).toBe(false);
      expect(color.isColor([0, -10, 0, 255])).toBe(false);
    });

    test("validates alpha is within 0-255 range", () => {
      expect(color.isColor([0, 0, 0, 0])).toBe(true);
      expect(color.isColor([0, 0, 0, 255])).toBe(true);
      expect(color.isColor([0, 0, 0, 128])).toBe(true);
      expect(color.isColor([0, 0, 0, -1])).toBe(false);
      expect(color.isColor([0, 0, 0, 256])).toBe(false);
    });
  });

  describe("fromCSS", () => {
    test("parses hex colors", () => {
      expect(color.fromCSS("#ff0000")).toEqual([255, 0, 0, 255]);
      expect(color.fromCSS("#00ff00")).toEqual([0, 255, 0, 255]);
      expect(color.fromCSS("#0000ff")).toEqual([0, 0, 255, 255]);
      expect(color.fromCSS("#f00")).toEqual([255, 0, 0, 255]);
      expect(color.fromCSS("#0f0")).toEqual([0, 255, 0, 255]);
      expect(color.fromCSS("#00f")).toEqual([0, 0, 255, 255]);
    });

    test("parses rgb/rgba colors", () => {
      expect(color.fromCSS("rgb(255, 0, 0)")).toEqual([255, 0, 0, 255]);
      expect(color.fromCSS("rgb(0, 255, 0)")).toEqual([0, 255, 0, 255]);
      expect(color.fromCSS("rgba(0, 0, 255, 0.5)")).toEqual([0, 0, 255, 128]);
      expect(color.fromCSS("rgba(128, 128, 128, 1)")).toEqual([128, 128, 128, 255]);
    });

    test("parses named colors", () => {
      expect(color.fromCSS("red")).toEqual([255, 0, 0, 255]);
      expect(color.fromCSS("green")).toEqual([0, 128, 0, 255]);
      expect(color.fromCSS("blue")).toEqual([0, 0, 255, 255]);
      expect(color.fromCSS("black")).toEqual([0, 0, 0, 255]);
      expect(color.fromCSS("white")).toEqual([255, 255, 255, 255]);
    });

    test("handles case insensitive input", () => {
      expect(color.fromCSS("RED")).toEqual([255, 0, 0, 255]);
      expect(color.fromCSS("Green")).toEqual([0, 128, 0, 255]);
      expect(color.fromCSS("BLUE")).toEqual([0, 0, 255, 255]);
    });

    test("returns undefined for invalid input", () => {
      expect(color.fromCSS("")).toBeUndefined();
      expect(color.fromCSS("none")).toBeUndefined();
      expect(color.fromCSS("transparent")).toBeUndefined();
      expect(color.fromCSS("invalid")).toBeUndefined();
      expect(color.fromCSS("#gggggg")).toBeUndefined();
    });
  });

  describe("isZero", () => {
    test("returns true for zero color", () => {
      expect(color.isZero([0, 0, 0, 0])).toBe(true);
    });

    test("returns false for non-zero RGB values", () => {
      expect(color.isZero([1, 0, 0, 0])).toBe(false);
      expect(color.isZero([0, 1, 0, 0])).toBe(false);
      expect(color.isZero([0, 0, 1, 0])).toBe(false);
    });

    test("returns false for non-zero alpha", () => {
      expect(color.isZero([0, 0, 0, 26])).toBe(false);
      expect(color.isZero([0, 0, 0, 255])).toBe(false);
    });

    test("returns false for undefined input", () => {
      expect(color.isZero(undefined)).toBe(false);
    });

    test("returns false for null input", () => {
      expect(color.isZero(null as any)).toBe(false);
    });

    test("returns false for hex color", () => {
      expect(color.isZero("#000000")).toBe(false);
    });

    test("returns false for constructed color", () => {
      expect(color.isZero(color.construct("#000000"))).toBe(false);
    });
  });
});

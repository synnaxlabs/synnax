// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";
import { z } from "zod";

import { binary } from "@/binary";
import { color } from "@/color";
import { zod } from "@/zod";

describe("color.Color", () => {
  describe("constructor", () => {
    test("from hex", () => {
      expect(color.construct("#7a2c26")).toEqual([122, 44, 38, 255]);
    });
    test("from hex with alpha", () => {
      expect(color.construct("#7a2c26", 128)).toEqual([122, 44, 38, 128]);
    });

    describe("from eight digit hex", () => {
      test("case 1", () => {
        expect(color.construct("#7a2c26ff")).toEqual([122, 44, 38, 255]);
      });
      test("case 2", () => {
        expect(color.construct("#7a2c2605")).toEqual([122, 44, 38, 5]);
      });
      test("ignores the alpha argument", () => {
        expect(color.construct("#7a2c2605", 128)).toEqual([122, 44, 38, 5]);
      });
    });

    test("from rgb", () => {
      expect(color.construct([122, 44, 38])).toEqual([122, 44, 38, 255]);
    });
    test("from rgb with alpha", () => {
      expect(color.construct([122, 44, 38], 7)).toEqual([122, 44, 38, 7]);
    });
    test("from rgba", () => {
      expect(color.construct([122, 44, 38, 128])).toEqual([122, 44, 38, 128]);
    });
    test("keeps a small byte alpha", () => {
      expect(color.construct([122, 44, 38, 1])).toEqual([122, 44, 38, 1]);
      expect(color.construct([122, 44, 38, 2])).toEqual([122, 44, 38, 2]);
    });
    test("from c", () => {
      const c = color.construct(color.construct("#7a2c26"));
      expect(c).toEqual([122, 44, 38, 255]);
    });
    test("throws a ParseError for an invalid crude color", () => {
      expect(() => color.construct([300, 0, 0] as any)).toThrow(zod.ParseError);
      expect(() => color.construct([300, 0, 0] as any)).toThrow(
        /Failed to parse color/,
      );
    });
    test("rejects a fractional tuple alpha", () => {
      expect(() => color.construct([255, 0, 0, 0.5])).toThrow(zod.ParseError);
    });

    test.each<[string, unknown]>([
      ["the empty string", ""],
      ["the empty object", {}],
      ["a struct", { r: 122, g: 44, b: 38, a: 0.5 }],
      ["an rgba255 object", { rgba255: [122, 44, 38, 128] }],
    ])("rejects %s", (_, crude) => {
      expect(color.colorZ.safeParse(crude).success).toBe(false);
    });
  });

  describe("to hex", () => {
    test("without alpha", () => {
      expect(color.hex(color.construct("#7a2c26"))).toEqual("#7a2c26");
    });
    test("with alpha", () => {
      expect(color.hex(color.construct("#7a2c26", 128))).toEqual("#7a2c2680");
    });
    test("with a zero alpha", () => {
      expect(color.hex(color.ZERO)).toEqual("#00000000");
    });
  });

  describe("to RGBA1", () => {
    test("with alpha", () => {
      const c = color.construct("#7a2c26", 51);
      const expected = [122 / 255, 44 / 255, 38 / 255, 0.2];
      expected.forEach((v, i) => {
        expect(color.rgba1(c)[i]).toBeCloseTo(v);
      });
    });
    test("without alpha", () => {
      expect(color.rgba1("#ffffff")).toEqual([1, 1, 1, 1]);
    });
  });

  describe("opacity", () => {
    test.each<[color.Crude, number]>([
      ["#ff000000", 0],
      ["#ff000080", 128 / 255],
      ["#ff0000", 1],
    ])("of %j", (c, expected) => {
      expect(color.opacity(c)).toBeCloseTo(expected);
    });
  });

  describe("aValue", () => {
    test("returns the alpha byte", () => {
      expect(color.aValue("#ff000080")).toEqual(128);
    });
  });

  describe("colorZ", () => {
    test("encodes a color as an 8 digit hex string through the JSON codec", () => {
      const schema = z.object({
        fillColor: color.colorZ,
        stroke: color.colorZ.optional(),
      });
      const encoded = binary.JSON_CODEC.encodeString(
        { fillColor: [255, 0, 0, 1], stroke: "#00ff00" },
        schema,
      );
      expect(JSON.parse(encoded)).toEqual({
        fill_color: "#ff000001",
        stroke: "#00ff00ff",
      });
    });
    test("encodes a list of colors through the JSON codec", () => {
      const schema = z.object({ swatches: z.array(color.colorZ) });
      const encoded = binary.JSON_CODEC.encodeString({ swatches: ["#000000"] }, schema);
      expect(JSON.parse(encoded)).toEqual({ swatches: ["#000000ff"] });
    });
    test("decodes the hex string the JSON codec writes", () => {
      const schema = z.object({ fillColor: color.colorZ });
      const decoded = binary.JSON_CODEC.decodeString(
        '{"fill_color":"#ff000002"}',
        schema,
      );
      expect(decoded.fillColor).toEqual([255, 0, 0, 2]);
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
    const tests: Array<[string, color.HSLA, color.RGBA]> = [
      ["Red", [0, 100, 50, 1], [255, 0, 0, 255]],
      ["Green", [120, 100, 50, 1], [0, 255, 0, 255]],
      ["Blue", [240, 100, 50, 1], [0, 0, 255, 255]],
      ["Yellow", [60, 100, 50, 1], [255, 255, 0, 255]],
      ["Cyan", [180, 100, 50, 1], [0, 255, 255, 255]],
      ["Magenta", [300, 100, 50, 1], [255, 0, 255, 255]],
      ["Black", [0, 0, 0, 1], [0, 0, 0, 255]],
      ["White", [0, 0, 100, 1], [255, 255, 255, 255]],
      ["Mid Gray", [0, 0, 50, 1], [128, 128, 128, 255]],
      ["Semi-transparent Red", [0, 100, 50, 0.5], [255, 0, 0, 128]],
      ["Transparent Red", [0, 100, 50, 0], [255, 0, 0, 0]],
    ];

    tests.forEach(([name, hsla, expected]) => {
      test(name, () => {
        expect(color.fromHSLA(hsla)).toEqual(expected);
      });
    });

    test("handles hue wrapping properly", () => {
      expect(color.fromHSLA([0, 100, 50, 1])).toEqual(
        color.fromHSLA([360, 100, 50, 1]),
      );
    });
  });

  describe("hsla", () => {
    const tests: Array<[string, color.RGBA, color.HSLA]> = [
      ["Red", [255, 0, 0, 255], [0, 100, 50, 1]],
      ["Green", [0, 255, 0, 255], [120, 100, 50, 1]],
      ["Blue", [0, 0, 255, 255], [240, 100, 50, 1]],
      ["Yellow", [255, 255, 0, 255], [60, 100, 50, 1]],
      ["Cyan", [0, 255, 255, 255], [180, 100, 50, 1]],
      ["Magenta", [255, 0, 255, 255], [300, 100, 50, 1]],
      ["Black", [0, 0, 0, 255], [0, 0, 0, 1]],
      ["White", [255, 255, 255, 255], [0, 0, 100, 1]],
      ["Mid Gray", [128, 128, 128, 255], [0, 0, 50, 1]],
      ["Transparent Red", [255, 0, 0, 0], [0, 100, 50, 0]],
      ["Semi-transparent Blue", [0, 0, 255, 51], [240, 100, 50, 0.2]],
      ["Dark Red", [128, 0, 0, 255], [0, 100, 25, 1]],
      ["Light Blue", [128, 128, 255, 255], [240, 100, 75, 1]],
      ["Desaturated Red", [191, 64, 64, 255], [0, 50, 50, 1]],
      ["Slightly Saturated Green", [96, 159, 96, 255], [120, 25, 50, 1]],
    ];

    tests.forEach(([name, rgba, expected]) => {
      test(name, () => {
        const result = color.hsla(rgba);
        for (let i = 0; i < 3; i++) expect(result[i]).toBeCloseTo(expected[i], 0);
        expect(result[3]).toBeCloseTo(expected[3]);
      });
    });

    test("handles RGB array input", () => {
      expect(color.hsla([0, 255, 0])[3]).toEqual(1);
    });

    test("preserves original color after round-trip conversion", () => {
      const originalColors: color.RGBA[] = [
        [255, 0, 0, 255],
        [0, 255, 0, 255],
        [128, 128, 128, 255],
        [255, 255, 255, 128],
        [255, 255, 255, 1],
      ];
      for (const original of originalColors) {
        const converted = color.fromHSLA(color.hsla(original));
        for (let i = 0; i < 3; i++) expect(converted[i]).toBeCloseTo(original[i], 0);
        expect(converted[3]).toEqual(original[3]);
      }
    });
  });

  describe("setOpacity", () => {
    test("sets the opacity of an RGB color", () => {
      expect(color.setOpacity([255, 0, 0], 0.5)).toEqual([255, 0, 0, 128]);
    });
    test("overrides the alpha of an RGBA color", () => {
      expect(color.setOpacity([128, 128, 128, 51], 0.8)).toEqual([128, 128, 128, 204]);
    });
    test("sets the opacity of a hex color", () => {
      expect(color.setOpacity("#0000ff", 0.2)).toEqual([0, 0, 255, 51]);
    });
    test("handles the bounds", () => {
      expect(color.setOpacity("#ff0000", 0)).toEqual([255, 0, 0, 0]);
      expect(color.setOpacity("#ff000080", 1)).toEqual([255, 0, 0, 255]);
    });
    test("rejects an opacity outside 0 to 1", () => {
      expect(() => color.setOpacity("#ff0000", 50)).toThrow(zod.ParseError);
      expect(() => color.setOpacity("#ff0000", -0.1)).toThrow(zod.ParseError);
    });
  });

  describe("rgbaCSS", () => {
    test("converts RGB array to CSS rgba string", () => {
      expect(color.rgbaCSS([255, 0, 0])).toEqual("rgba(255, 0, 0, 1)");
    });
    test("converts RGBA array to CSS rgba string", () => {
      expect(color.rgbaCSS([0, 255, 0, 51])).toEqual("rgba(0, 255, 0, 0.2)");
    });
    test("converts hex to CSS rgba string", () => {
      expect(color.rgbaCSS("#0000ff")).toEqual("rgba(0, 0, 255, 1)");
    });
    test("converts a transparent color to CSS rgba string", () => {
      expect(color.rgbaCSS(color.ZERO)).toEqual("rgba(0, 0, 0, 0)");
    });
    test("converts hex with alpha to CSS rgba string", () => {
      expect(color.rgbaCSS("#ff000080")).toMatch(/rgba\(255, 0, 0, 0\.5\d*\)/);
    });
  });

  describe("rgbCSS", () => {
    test("converts RGB array to CSS rgb string", () => {
      expect(color.rgbCSS([255, 0, 0])).toEqual("rgb(255, 0, 0)");
    });
    test("ignores alpha", () => {
      expect(color.rgbCSS([0, 255, 0, 128])).toEqual("rgb(0, 255, 0)");
    });
    test("converts hex to CSS rgb string", () => {
      expect(color.rgbCSS("#0000ff")).toEqual("rgb(0, 0, 255)");
    });
  });

  describe("rgbString", () => {
    test("converts RGB array to comma-separated string", () => {
      expect(color.rgbString([255, 0, 0])).toEqual("255, 0, 0");
    });
    test("ignores alpha", () => {
      expect(color.rgbString([0, 255, 0, 128])).toEqual("0, 255, 0");
    });
    test("converts hex to comma-separated string", () => {
      expect(color.rgbString("#0000ff")).toEqual("0, 0, 255");
    });
  });

  describe("rgbaString", () => {
    test("carries the rgb channels and an opaque alpha", () => {
      expect(color.rgbaString("#ff0000")).toEqual("255, 0, 0, 1");
    });
    test("carries a translucent alpha", () => {
      expect(color.rgbaString([255, 0, 0, 51])).toEqual("255, 0, 0, 0.2");
    });
    test("carries a fully transparent color", () => {
      expect(color.rgbaString(color.ZERO)).toEqual("0, 0, 0, 0");
    });
    test("is undefined for no color", () => {
      expect(color.rgbaString(undefined)).toBeUndefined();
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
      expect(color.cssString(rgb)).toEqual("rgba(255, 0, 0, 1)");
    });

    test("converts RGBA array to rgba CSS string", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.cssString(rgba)).toEqual("rgba(0, 255, 0, 0.5019607843137255)");
    });

    test("converts hex to rgba CSS string", () => {
      const hex = "#0000ff";
      expect(color.cssString(hex)).toEqual("rgba(0, 0, 255, 1)");
    });

    test("returns CSS variables as-is", () => {
      const cssVar = "var(--primary-color)";
      expect(color.cssString(cssVar)).toEqual(cssVar);
    });

    test("handles eight-digit hex with alpha", () => {
      const hex = "#00ff0080";
      const result = color.cssString(hex);
      // Alpha 0x80 = 128/255 ≈ 0.502
      expect(result).toMatch(/rgba\(0, 255, 0, 0\.5\d*\)/);
    });

    test("handles Color object", () => {
      const c = color.construct([128, 128, 128, 179]);
      expect(color.cssString(c)).toEqual("rgba(128, 128, 128, 0.7019607843137254)");
    });

    test("throws a ParseError for invalid color format", () => {
      expect(() => color.cssString([1, 2] as any)).toThrow(zod.ParseError);
      expect(() => color.cssString([300, 0, 0] as any)).toThrow(zod.ParseError);
      expect(() => color.cssString([300, 0, 0] as any)).toThrow(
        /Failed to parse color/,
      );
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
      expect(color.isCrude([255, 0, 0, 0.5, 1])).toBe(false);
      expect(color.isCrude([-1, 0, 0])).toBe(false);
      expect(color.isCrude([0, 256, 0])).toBe(false);
    });

    test("rejects invalid RGBA arrays", () => {
      expect(color.isCrude([255, 0, 0, 256])).toBe(false);
      expect(color.isCrude([255, 0, 0, -0.1])).toBe(false);
    });

    test("RGBA struct is not a crude color", () => {
      expect(color.isCrude({ r: 255, g: 0, b: 0, a: 255 })).toBe(false);
    });

    test("rejects non-color values", () => {
      expect(color.isCrude(null)).toBe(false);
      expect(color.isCrude(undefined)).toBe(false);
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
      expect(color.isColor([255, 0, -1, 1])).toBe(false); // Negative value
      expect(color.isColor([255, 0, 256, 255])).toBe(false); // Value > 255
      expect(color.isColor([255, 0, 0, 1.5])).toBe(false); // Alpha > 1
      expect(color.isColor([255, 0, 0, -0.1])).toBe(false); // Alpha < 0
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
      expect(color.isColor([0, -10, 0, 1])).toBe(false);
    });

    test("validates alpha is within 0-1 range", () => {
      expect(color.isColor([0, 0, 0, 0])).toBe(true);
      expect(color.isColor([0, 0, 0, 255])).toBe(true);
      expect(color.isColor([0, 0, 0, 128])).toBe(true);
      expect(color.isColor([0, 0, 0, -0.1])).toBe(false);
      expect(color.isColor([0, 0, 0, 1.1])).toBe(false);
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

    test("parses four and eight digit hex with alpha", () => {
      expect(color.fromCSS("#f008")).toEqual([255, 0, 0, 136]);
      expect(color.fromCSS("#ff000080")).toEqual([255, 0, 0, 128]);
    });

    test("parses space syntax with a slash alpha", () => {
      expect(color.fromCSS("rgb(255 0 0 / 50%)")).toEqual([255, 0, 0, 128]);
      expect(color.fromCSS("rgb(100% 0% 0%)")).toEqual([255, 0, 0, 255]);
    });

    test("parses hsl colors", () => {
      expect(color.fromCSS("hsl(0, 100%, 50%)")).toEqual([255, 0, 0, 255]);
      expect(color.fromCSS("hsla(240 100% 50% / 0.25)")).toEqual([0, 0, 255, 64]);
    });

    test("converts hsl hue units to degrees", () => {
      const cyan = [0, 255, 255, 255];
      expect(color.fromCSS("hsl(180deg 100% 50%)")).toEqual(cyan);
      expect(color.fromCSS("hsl(0.5turn 100% 50%)")).toEqual(cyan);
      expect(color.fromCSS(`hsl(${Math.PI}rad 100% 50%)`)).toEqual(cyan);
      expect(color.fromCSS("hsl(200grad 100% 50%)")).toEqual(cyan);
      expect(color.fromCSS("hsl(1.8e2deg 100% 50%)")).toEqual(cyan);
    });

    test("rejects an hsl hue with an unknown unit", () => {
      expect(color.fromCSS("hsl(180px 100% 50%)")).toBeUndefined();
    });

    test("rejects an hsl hue that overflows", () => {
      expect(color.fromCSS("hsl(1e999deg 100% 50%)")).toBeUndefined();
    });

    test("clamps out of range channels", () => {
      expect(color.fromCSS("rgb(300, -5, 0, 2)")).toEqual([255, 0, 0, 255]);
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
      expect(color.fromCSS("#12345")).toBeUndefined();
      expect(color.fromCSS("rgb(1, 2)")).toBeUndefined();
      expect(color.fromCSS("rgb(a, b, c)")).toBeUndefined();
    });
  });

  describe("hsva", () => {
    test("converts primary colors", () => {
      expect(color.hsva("#ff0000")).toEqual([0, 100, 100, 1]);
      expect(color.hsva("#00ff00")).toEqual([120, 100, 100, 1]);
      expect(color.hsva("#0000ff")).toEqual([240, 100, 100, 1]);
    });

    test("converts achromatic colors to zero hue and saturation", () => {
      expect(color.hsva("#000000")).toEqual([0, 0, 0, 1]);
      expect(color.hsva("#ffffff")).toEqual([0, 0, 100, 1]);
    });

    test("keeps the alpha", () => {
      expect(color.hsva([255, 0, 0, 51])[3]).toBeCloseTo(0.2);
    });
  });

  describe("fromHSVA", () => {
    test("converts primary colors", () => {
      expect(color.fromHSVA([0, 100, 100, 1])).toEqual([255, 0, 0, 255]);
      expect(color.fromHSVA([120, 100, 100, 1])).toEqual([0, 255, 0, 255]);
      expect(color.fromHSVA([240, 100, 100, 0.5])).toEqual([0, 0, 255, 128]);
    });

    test("treats a hue of 360 as 0", () => {
      expect(color.fromHSVA([360, 100, 100, 1])).toEqual([255, 0, 0, 255]);
    });

    test("round trips every hex color it produces", () => {
      ["#3e8bff", "#dc136c", "#7ac74f", "#808080", "#010203"].forEach((hex) =>
        expect(color.hex(color.fromHSVA(color.hsva(hex)))).toEqual(hex),
      );
    });

    test("throws on an out of range value", () => {
      expect(() => color.fromHSVA([0, 101, 0, 1])).toThrow();
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
      expect(color.isZero([0, 0, 0, 1])).toBe(false);
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

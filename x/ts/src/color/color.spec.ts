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

    describe("from eight digit hex", () => {
      test("case 1", () => {
        expect(color.construct("#7a2c26ff")).toEqual([122, 44, 38, 255]);
      });
      test("case 2", () => {
        expect(color.construct("#7a2c2605")).toEqual([122, 44, 38, 5]);
      });
    });

    test("from rgb", () => {
      expect(color.construct([122, 44, 38])).toEqual([122, 44, 38, 255]);
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
      ["an unknown name", "reddish"],
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
      expect(color.hex(color.construct("#7a2c2680"))).toEqual("#7a2c2680");
    });
    test("with a zero alpha", () => {
      expect(color.hex(color.TRANSPARENT)).toEqual("#00000000");
    });
  });

  describe("to RGBA1", () => {
    test("with alpha", () => {
      const c = color.construct("#7a2c2633");
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

  describe("colorZ", () => {
    test("encodes a color as its hex string through the JSON codec", () => {
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
        stroke: "#00ff00",
      });
    });
    test("encodes a list of colors through the JSON codec", () => {
      const schema = z.object({ swatches: z.array(color.colorZ) });
      const encoded = binary.JSON_CODEC.encodeString({ swatches: ["#000000"] }, schema);
      expect(JSON.parse(encoded)).toEqual({ swatches: ["#000000"] });
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

  describe("rgbString", () => {
    test("writes the channels and drops the alpha", () => {
      expect(color.rgbString([0, 255, 0, 128])).toEqual("0, 255, 0");
    });
  });

  describe("rgbaString", () => {
    test("writes the channels and the 0-1 alpha", () => {
      expect(color.rgbaString("#ff0000")).toEqual("255, 0, 0, 1");
      expect(color.rgbaString([255, 0, 0, 51])).toEqual("255, 0, 0, 0.2");
    });
    test("is undefined for no color", () => {
      expect(color.rgbaString(undefined)).toBeUndefined();
    });
  });

  describe("equals", () => {
    test("same RGB values are equal", () => {
      const c1: [number, number, number] = [255, 0, 0];
      const c2: [number, number, number] = [255, 0, 0];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("different RGB values are not equal", () => {
      const c1: [number, number, number] = [255, 0, 0];
      const c2: [number, number, number] = [0, 255, 0];
      expect(color.equals(c1, c2)).toBe(false);
    });

    test("RGBA and RGB with default alpha are equal", () => {
      const c1: [number, number, number] = [255, 0, 0];
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
      const c2: [number, number, number] = [255, 0, 0];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("eight-digit hex and RGBA with same values are equal", () => {
      const c1 = "#ff000080";
      const c2: color.RGBA = [255, 0, 0, 128];
      expect(color.equals(c1, c2)).toBe(true);
    });

    test("comparing to undefined", () => {
      const c1: [number, number, number] = [255, 0, 0];
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

    test("writes a color as its hex string", () => {
      expect(color.cssString([255, 0, 0])).toEqual("#ff0000");
      expect(color.cssString([0, 255, 0, 128])).toEqual("#00ff0080");
      expect(color.cssString("rgb(0, 0, 255)")).toEqual("#0000ff");
    });

    test("returns CSS variables as-is", () => {
      const cssVar = "var(--primary-color)";
      expect(color.cssString(cssVar)).toEqual(cssVar);
    });

    test("throws a ParseError for invalid color format", () => {
      expect(() => color.cssString([1, 2] as any)).toThrow(zod.ParseError);
      expect(() => color.cssString([300, 0, 0] as any)).toThrow(zod.ParseError);
      expect(() => color.cssString([300, 0, 0] as any)).toThrow(
        /Failed to parse color/,
      );
    });
  });

  describe("isColor", () => {
    test("valid RGBA array is a Color", () => {
      const rgba: color.RGBA = [0, 255, 0, 128];
      expect(color.isColor(rgba)).toBe(true);
    });

    test("RGB array with 3 elements is not a Color", () => {
      const rgb: [number, number, number] = [255, 0, 0];
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

  describe("construct from a string", () => {
    test.each<[string, color.Color]>([
      ["red", [255, 0, 0, 255]],
      [" RebeccaPurple ", [102, 51, 153, 255]],
      ["transparent", [0, 0, 0, 0]],
      ["#ff0000", [255, 0, 0, 255]],
      ["00ff00", [0, 255, 0, 255]],
      ["#FF000080", [255, 0, 0, 128]],
      ["#f00", [255, 0, 0, 255]],
      ["#f008", [255, 0, 0, 136]],
      ["rgb(255, 0, 0)", [255, 0, 0, 255]],
      ["rgb(255, 0, 0, 0.5)", [255, 0, 0, 128]],
      ["rgba(0, 0, 255, 0.5)", [0, 0, 255, 128]],
      ["rgba(1, 2, 3)", [1, 2, 3, 255]],
      ["rgb(255 0 0 / 50%)", [255, 0, 0, 128]],
      ["rgb(100% 0% 0%)", [255, 0, 0, 255]],
      ["  RGB(1,2,3)  ", [1, 2, 3, 255]],
      ["hsl(0, 100%, 50%)", [255, 0, 0, 255]],
      ["hsl(180deg 100% 50%)", [0, 255, 255, 255]],
      ["hsl(0, 0, 100)", [255, 255, 255, 255]],
      ["hsl(480, 100%, 50%)", [0, 255, 0, 255]],
      ["hsl(-120, 100%, 50%)", [0, 0, 255, 255]],
      ["hsla(240 100% 50% / 0.25)", [0, 0, 255, 64]],
      ["hsla(0, 55%, 43%, 0)", [170, 49, 49, 0]],
    ])("%s", (input, expected) => {
      expect(color.construct(input)).toEqual(expected);
    });

    test.each([
      "",
      "reddish",
      "constructor",
      "#gggggg",
      "#12345",
      "rgb(1, 2)",
      "rgb(1, 2, 3, 0.5, 1)",
      "rgb(a, b, c)",
      "rgb(300, 0, 0)",
      "rgb(-1, 0, 0)",
      "rgba(1, 2, 3, 1.5)",
      "hsl(180turn 100% 50%)",
      "rgb(NaN, 0, 0)",
      "rgb(1e2, 0, 0)",
      "rgb(1 2 3 0.5)",
      "rgb(1 2 / 3 / 4)",
      "rgb(1, 2 3)",
      "rgb(1, 2, 3 / 0.5)",
      "rgb(1 2 3 /)",
      "hsl(120%, 100%, 50%)",
      "hsl(0, 101%, 50%)",
      "hsl(0, 0%, 101%)",
    ])("rejects %j", (input) => {
      expect(() => color.construct(input)).toThrow(zod.ParseError);
    });
  });

  describe("oklch", () => {
    test("should give white a lightness of 1 and no chroma", () => {
      const [l, c, , a] = color.oklch("#ffffff");
      expect(l).toBeCloseTo(1);
      expect(c).toBeCloseTo(0);
      expect(a).toEqual(1);
    });
    test.each(["#ff0000", "#3774d080", "#123456", "#00000000"])(
      "should round trip %s through fromOKLCH",
      (hex) => {
        expect(color.fromOKLCH(color.oklch(hex))).toEqual(color.construct(hex));
      },
    );
    test("should clamp an out-of-gamut color into sRGB", () => {
      const c = color.fromOKLCH([0.9, 0.4, 30, 1]);
      expect(color.isColor(c)).toBe(true);
      expect(c[0]).toEqual(255);
    });
  });

  describe("over", () => {
    test("should paint the foreground over the background at an opacity", () => {
      expect(color.over("#ff0000", "#0000ff", 0.25)).toEqual([64, 0, 191, 255]);
    });
    test("should return the background at opacity 0", () => {
      expect(color.over("#ff0000", "#00ff0080", 0)).toEqual([0, 255, 0, 255]);
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
});

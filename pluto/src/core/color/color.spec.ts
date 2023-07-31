// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { Color } from "@/core/color";

describe("Color", () => {
  describe("constructor", () => {
    test("from hex", () => {
      const color = new Color("#7a2c26");
      expect(color.r).toEqual(122);
      expect(color.g).toEqual(44);
      expect(color.b).toEqual(38);
    });
    test("from hex with alpha", () => {
      const color = new Color("#7a2c26", 0.5);
      expect(color.r).toEqual(122);
      expect(color.g).toEqual(44);
      expect(color.b).toEqual(38);
      expect(color.a).toEqual(0.5);
    });
    describe("from eight digit hex", () => {
      test("case 1", () => {
        const color = new Color("#7a2c26ff");
        expect(color.r).toEqual(122);
        expect(color.g).toEqual(44);
        expect(color.b).toEqual(38);
        expect(color.a).toEqual(1);
      });
      test("case 2", () => {
        const color = new Color("#7a2c2605");
        expect(color.r).toEqual(122);
        expect(color.g).toEqual(44);
        expect(color.b).toEqual(38);
        expect(color.a).toEqual(5 / 255);
      });
    });
    test("from rgb", () => {
      const color = new Color([122, 44, 38]);
      expect(color.r).toEqual(122);
      expect(color.g).toEqual(44);
      expect(color.b).toEqual(38);
    });
    test("from rgba", () => {
      const color = new Color([122, 44, 38, 0.5]);
      expect(color.r).toEqual(122);
      expect(color.g).toEqual(44);
      expect(color.b).toEqual(38);
      expect(color.a).toEqual(0.5);
    });
    test("from color", () => {
      const color = new Color(new Color("#7a2c26"));
      expect(color.r).toEqual(122);
      expect(color.g).toEqual(44);
      expect(color.b).toEqual(38);
    });
  });
  describe("to hex", () => {
    test("without alpha", () => {
      const color = new Color("#7a2c26");
      expect(color.hex).toEqual("#7a2c26");
    });
    test("with alpha", () => {
      const color = new Color("#7a2c26", 0.5);
      expect(color.hex).toEqual("#7a2c267f");
    });
  });
  describe("to RGBA255", () => {
    test("with alpha", () => {
      const color = new Color("#7a2c26", 0.5);
      const expected = [122, 44, 38, 0.5];
      expect(color.rgba255).toEqual(expected);
    });
    test("without alpha", () => {
      const color = new Color("#7a2c26");
      const expected = [122, 44, 38, 1];
      expect(color.rgba255).toEqual(expected);
    });
  });
  describe("to RGBA1", () => {
    test("with alpha", () => {
      const color = new Color("#7a2c26", 0.5);
      const expected = [122 / 255, 44 / 255, 38 / 255, 0.5];
      expected.forEach((v, i) => {
        expect(color.rgba1[i]).toBeCloseTo(v);
      });
    });
    test("without alpha", () => {
      const color = new Color("#7a2c26");
      const expected = [122 / 255, 44 / 255, 38 / 255, 1];
      expected.forEach((v, i) => {
        expect(color.rgba1[i]).toBeCloseTo(v);
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
        const color = new Color(hex);
        expect(color.luminance).toBeCloseTo(expected);
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
        const color1 = new Color(hex1);
        const color2 = new Color(hex2);
        expect(color1.contrast(color2)).toBeCloseTo(expected);
      });
    });
    test("pick color with highest contrast", () => {
      const color = new Color("#000000");
      const color1 = new Color("#ffffff");
      const color2 = new Color("#0000ff");
      expect(color.pickByContrast(color1, color2)).toEqual(color1);
    });
  });
});

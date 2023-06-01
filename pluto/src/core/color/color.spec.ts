// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { easeQuad } from "d3";
import { describe, expect, it } from "vitest";

import { Color, RGBA } from "./color";

describe("convert", () => {
  describe("hexToRGBA", () => {
    const TESTS: Array<[string, RGBA]> = [
      ["#000000", [0, 0, 0, 1]],
      ["#00000000", [0, 0, 0, 0]],
      ["#000000ff", [0, 0, 0, 1]],
      ["#7f11e01f", [127, 17, 224, 0.12]],
    ];
    for (const [hex, expected] of TESTS) {
      it(`should convert ${hex} to ${expected}`, () => {
        expect(new Color(hex).equals(new Color(expected))).toBe(true);
      });
    }
  });
  describe("normalizeRGBA", () => {
    const TESTS: Array<[number, RGBA, RGBA]> = [
      [255, [0, 0, 0, 1], [0, 0, 0, 1]],
      [255, [0, 0, 0, 0], [0, 0, 0, 0]],
      [255, [0, 0, 0, 0.5], [0, 0, 0, 0.5]],
      [255, [127, 17, 224, 0.12], [127 / 255, 17 / 255, 224 / 255, 0.12]],
      [1, [127, 17, 224, 0.12], [127, 17, 224, 0.12]],
    ];
    for (const [divisor, input, expected] of TESTS) {
      it(`should normalize ${input} to ${expected}`, () => {
        const actual = normalizeRGBA(input, divisor);
        expected.forEach((v, i) => {
          expect(actual[i]).toBeCloseTo(v);
        });
      });
    }
  });
  describe("validateHex", () => {
    const TESTS: Array<[string, boolean]> = [
      ["#000000", true],
      ["#00000000", true],
      ["#000000ff", true],
      ["#7f11e01f", true],
      ["#0000000", false],
      ["#000000000", false],
      ["#000000fff", false],
      ["#7f11e01ff", false],
      ["#000000g", false],
      ["#000000gg", false],
    ];
    for (const [hex, expected] of TESTS) {
      it(`should validate ${hex} to ${expected}`, () => {
        expect(validateHex(hex)).toBe(expected);
      });
    }
  });
  describe("addOpacityToHex", () => {
    const TESTS: Array<[string, Opacity, string]> = [
      ["#000000", 99, "#000000FC"],
      ["#000000", 0, "#00000000"],
      ["#000000", 1, "#00000003"],
      ["#000000", 12, "#0000001F"],
      ["#000000FF", 50, "#00000080"],
    ];
    for (const [hex, opacity, expected] of TESTS) {
      it(`should add opacity ${opacity} to ${hex} to ${expected}`, () => {
        expect(addOpacityToHex(hex, opacity)).toBe(expected);
      });
    }
  });
});

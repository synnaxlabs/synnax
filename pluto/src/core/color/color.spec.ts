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
    test("with alpha", () => {
      const color = new Color("#7a2c26", 0.5);
      expect(color.hex).toEqual("#7a2c267f");
    });
  });
});

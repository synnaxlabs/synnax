// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { box } from "@/spatial/box";
import { Scale, XY } from "@/spatial/scale/scale";

type ScaleSpec = [name: string, scale: Scale<number>, i: number, o: number];

describe("Scale", () => {
  const simpleScale = Scale.scale<number>(0, 10).scale(0, 1);
  const translateScale = Scale.scale<number>(0, 10).translate(5).scale(0, 1);
  const translateMagnifyScale = Scale.scale<number>(0, 10)
    .translate(5)
    .magnify(2)
    .scale(0, 1);
  describe("position", () => {
    const positionSpecs: ScaleSpec[] = [
      ["basic", simpleScale, 0, 0],
      ["basic II", simpleScale, 5, 0.5],
      ["reverse basic", simpleScale.reverse(), 0, 0],
      ["reverse basic II", simpleScale.reverse(), 0.5, 5],
      ["translate", translateScale, 0, 0.5],
      ["translate II", translateScale, 5, 1],
      ["reverse translate", translateScale.reverse(), 0.5, 0],
      ["reverse translate II", translateScale.reverse(), 0, -5],
      ["translate magnify", translateMagnifyScale, 0, 1],
      ["translate magnify II", translateMagnifyScale, 5, 2],
      ["reverse translate magnify", translateMagnifyScale.reverse(), 1, 0],
      ["reverse translate magnify II", translateMagnifyScale.reverse(), 0, -5],
    ];
    positionSpecs.forEach(([name, scale, i, o]) => {
      it(`should return ${o} for ${i} on ${name}`, () => {
        expect(scale.pos(i)).toBe(o);
      });
    });
  });
  describe("dimension", () => {
    const dimensionSpecs: ScaleSpec[] = [
      ["basic", simpleScale, 0, 0],
      ["basic II", simpleScale, 5, 0.5],
      ["reverse basic", simpleScale.reverse(), 0, 0],
      ["reverse basic II", simpleScale.reverse(), 0.5, 5],
      ["translate", translateScale, 0, 0],
      ["translate II", translateScale, 5, 0.5],
      ["reverse translate", translateScale.reverse(), 0.5, 5],
      ["translate magnify", translateMagnifyScale, 0, 0],
      ["translate magnify II", translateMagnifyScale, 5, 1],
    ];
    dimensionSpecs.forEach(([name, scale, i, o]) => {
      it(`should return ${o} for ${i} on ${name}`, () => expect(scale.dim(i)).toBe(o));
    });
  });
  describe("XYScale", () => {
    test("converting a DOM rect to decimal coordinates", () => {
      const s = XY.scale(box.construct(100, 100, 1000, 1000)).scale(box.DECIMAL);
      const b1 = s.box(box.construct(100, 100, 1000, 1000));
      expect(box.bottomLeft(b1)).toEqual({ x: 0, y: 0 });
      const b2 = s.box(box.construct(200, 200, 200, 200));
      expect(box.bottomLeft(b2).x).toBeCloseTo(0.1);
      expect(box.bottomLeft(b2).y).toBeCloseTo(0.7);
    });
  });
  describe("transform", () => {
    it("should return the correct transform", () => {
      const s = Scale.scale<number>(0, 10).translate(5).magnify(2).scale(0, 1);
      const t = s.transform;
      expect(t.scale).toBe(0.2);
      expect(t.offset).toBe(1);
    });
    it("should return the correct transform for an XY scale", () => {
      const s = XY.translate({ x: 5, y: 5 }).magnify({ x: 2, y: 2 });
      const t = s.transform;
      expect(t.scale).toEqual({ x: 2, y: 2 });
      expect(t.offset).toEqual({ x: 10, y: 10 });
    });
  });
});

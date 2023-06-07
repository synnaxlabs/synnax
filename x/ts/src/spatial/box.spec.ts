// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, test, expect, it } from "vitest";

import { Box, positionInCenter } from "@/spatial";

describe("Box", () => {
  describe("construction", () => {
    test("from dom rect", () => {
      const b = new Box({ left: 0, top: 0, right: 10, bottom: 10 });
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from two points", () => {
      const b = new Box({ x: 0, y: 0 }, { x: 10, y: 10 });
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from point and dimensions", () => {
      const b = new Box({ x: 0, y: 0 }, { width: 10, height: 10 });
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from point and width and height", () => {
      const b = new Box({ x: 0, y: 0 }, 10, 10);
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from raw params", () => {
      const b = new Box(0, 0, 10, 10);
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
  });
  describe("positionInCenterOf", () => {
    it("should position the box in the center of the other box", () => {
      let b = new Box(0, 0, 10, 10);
      const b2 = new Box(0, 0, 20, 20);
      b = positionInCenter(b, b2);
      expect(b.topLeft).toEqual({ x: 5, y: 5 });
      expect(b.topRight).toEqual({ x: 15, y: 5 });
      expect(b.bottomLeft).toEqual({ x: 5, y: 15 });
      expect(b.bottomRight).toEqual({ x: 15, y: 15 });
    });
  });
});

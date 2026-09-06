// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test } from "vitest";

import { location } from "@/spatial/location";

describe("Location", () => {
  describe("construction", () => {
    const CASES: Array<[string, location.Crude]> = [
      ["from string", "left"],
      ["from direction", "x"],
    ];
    CASES.forEach(([name, arg]) =>
      test(name, () => expect(location.construct(arg)).toEqual("left")),
    );
  });

  describe("rotate", () => {
    test("should rotate clockwise", () => {
      expect(location.rotate("top", "clockwise")).toEqual("left");
      expect(location.rotate("left", "clockwise")).toEqual("bottom");
      expect(location.rotate("bottom", "clockwise")).toEqual("right");
      expect(location.rotate("right", "clockwise")).toEqual("top");
    });

    test("should rotate counterclockwise", () => {
      expect(location.rotate("top", "counterclockwise")).toEqual("right");
      expect(location.rotate("left", "counterclockwise")).toEqual("top");
      expect(location.rotate("bottom", "counterclockwise")).toEqual("left");
      expect(location.rotate("right", "counterclockwise")).toEqual("bottom");
    });
  });

  describe("swapAxis", () => {
    const OUTER = ["top", "bottom", "left", "right"] as const;

    test("should pair each edge with the one facing the same way", () => {
      expect(location.swapAxis("top")).toEqual("left");
      expect(location.swapAxis("left")).toEqual("top");
      expect(location.swapAxis("right")).toEqual("bottom");
      expect(location.swapAxis("bottom")).toEqual("right");
    });

    test("should move the edge onto the other axis", () => {
      OUTER.forEach((l) =>
        expect(location.direction(location.swapAxis(l))).not.toEqual(
          location.direction(l),
        ),
      );
    });

    test("should be its own inverse", () => {
      OUTER.forEach((l) => expect(location.swapAxis(location.swapAxis(l))).toEqual(l));
    });
  });
});

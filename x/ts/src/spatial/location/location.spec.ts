// Copyright 2025 Synnax Labs, Inc.
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
    [
      ["from valueOf", String("left")],
      ["from string", "left"],
      ["from direction", "x"],
    ].forEach(([name, arg]) =>
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
});

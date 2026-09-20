// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Scale } from "@/schematic/node/general/scale";

describe("Scale", () => {
  describe("axis", () => {
    it("should read the orientation rotation writes for a vertical bar", () => {
      expect(Scale.axis("top")).toEqual("y");
    });

    it("should read the orientation rotation writes for a horizontal bar", () => {
      expect(Scale.axis("right")).toEqual("x");
    });

    it("should read a pre-rotation orientation as vertical", () => {
      expect(Scale.axis("left")).toEqual("y");
      expect(Scale.axis("bottom")).toEqual("y");
    });

    it("should read a missing orientation as vertical", () => {
      expect(Scale.axis()).toEqual("y");
    });

    it("should start a new scale vertical", () => {
      expect(Scale.axis(Scale.defaultConfig().orientation)).toEqual("y");
    });
  });
});

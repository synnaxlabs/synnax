// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Scale } from "@/schematic/node/common/scale";
import { telem } from "@/telem/aether";

describe("Scale", () => {
  describe("defaultConfig", () => {
    it("should populate every display field", () => {
      const config = Scale.defaultConfig();
      expect(config.bounds).toEqual({ lower: 0, upper: 100 });
      expect(config.showFill).toBe(true);
      expect(config.showCaret).toBe(true);
      expect(config.showScale).toBe(true);
      expect(config.side).toEqual("right");
      expect(config.units).toEqual("");
      expect(config.level).toEqual("small");
      expect(color.isZero(config.color)).toBe(true);
      expect(color.isZero(config.axisColor)).toBe(true);
      expect(color.isZero(config.textColor)).toBe(true);
    });

    it("should keep overrides over the defaults", () => {
      const config = Scale.defaultConfig({ showCaret: false, side: "left" });
      expect(config.showCaret).toBe(false);
      expect(config.side).toEqual("left");
      expect(config.showFill).toBe(true);
    });
  });

  describe("telem", () => {
    it("should read back the properties a spec was built from", () => {
      const props = {
        channel: 12,
        precision: 4,
        notation: "scientific",
        windowSize: 3,
      } as const;
      expect(Scale.parseTelem(Scale.createTelem(props))).toEqual(props);
    });

    it("should use the defaults for an unset spec", () => {
      expect(Scale.parseTelem()).toEqual({
        channel: 0,
        precision: 2,
        notation: "standard",
        windowSize: 1,
      });
    });

    it("should use the defaults for a spec built by a different pipeline", () => {
      const spec = telem.sourcePipeline("string", {
        connections: [],
        segments: { other: telem.streamChannelValue({ channel: 9 }) },
        outlet: "other",
      });
      expect(Scale.parseTelem(spec).channel).toEqual(0);
    });
  });
});

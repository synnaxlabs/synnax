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
import { Staleness } from "@/vis/staleness";

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
      expect(config.notation).toEqual("standard");
      expect(config.precision).toEqual(2);
      expect(config.stalenessTimeout).toEqual(Staleness.ZERO_CONFIG.stalenessTimeout);
      expect(color.isZero(config.stalenessColor)).toBe(true);
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

  describe("source", () => {
    it("should build the smoothed read pipeline for the configured channel", () => {
      const spec = Scale.source(
        Scale.defaultConfig({ channel: 12, rollingAverage: 3 }),
      );
      const { segments } = telem.sourcePipelinePropsZ.parse(spec.props);
      expect(
        telem.streamChannelValuePropsZ.parse(segments.valueStream.props).channel,
      ).toEqual(12);
      expect(
        telem.rollingAverageProps.parse(segments.rollingAverage.props).windowSize,
      ).toEqual(3);
    });

    it("should read an unset channel as zero and no smoothing", () => {
      const { segments } = telem.sourcePipelinePropsZ.parse(
        Scale.source(Scale.defaultConfig()).props,
      );
      expect(
        telem.streamChannelValuePropsZ.parse(segments.valueStream.props).channel,
      ).toEqual(0);
      expect(
        telem.rollingAverageProps.parse(segments.rollingAverage.props).windowSize,
      ).toEqual(1);
    });
  });
});

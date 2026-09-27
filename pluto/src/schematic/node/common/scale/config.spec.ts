// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Scale } from "@/schematic/node/common/scale";
import { telem } from "@/telem/aether";

describe("Scale", () => {
  describe("source", () => {
    it("should build the smoothed read pipeline for the configured channel", () => {
      const spec = Scale.source(
        schematic.scaleIndicatorConfigZ.parse({ channel: 12, rollingAverage: 3 }),
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
        Scale.source(schematic.scaleIndicatorConfigZ.parse({})).props,
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

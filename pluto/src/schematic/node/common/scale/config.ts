// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { bounds, color, type dimensions } from "@synnaxlabs/x";

import * as CommonTelem from "@/schematic/node/common/telem";
import { type telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";

/** Stored shape of a live scale indicator, shared by every symbol that renders one. */
export type Config = schematic.ScaleIndicatorConfig;

export const DEFAULT_DIMENSIONS: dimensions.Dimensions = { width: 60, height: 160 };

export const defaultConfig = (overrides: Partial<Config> = {}): Config => ({
  ...Staleness.ZERO_CONFIG,
  bounds: bounds.construct(0, 100),
  color: color.ZERO,
  axisColor: color.ZERO,
  textColor: color.ZERO,
  units: "",
  notation: "standard",
  precision: 2,
  showFill: true,
  showCaret: true,
  showScale: true,
  side: "right",
  level: "small",
  ...overrides,
});

/** source builds the smoothed read pipeline the indicator's value is drawn from. */
export const source = ({ channel, rollingAverage }: Config): telem.NumberSourceSpec =>
  CommonTelem.smoothedNumberSource({ channel, rollingAverage });

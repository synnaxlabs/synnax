// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type dimensions } from "@synnaxlabs/x";

import * as CommonTelem from "@/schematic/node/common/telem";
import { type telem } from "@/telem/aether";

/** Stored shape of a live scale indicator, shared by every symbol that renders one. */
export type Config = schematic.ScaleIndicatorConfig;

export const DEFAULT_DIMENSIONS: dimensions.Dimensions = { width: 60, height: 160 };

/** source builds the smoothed read pipeline the indicator's value is drawn from. */
export const source = ({ channel, rollingAverage }: Config): telem.NumberSourceSpec =>
  CommonTelem.smoothedNumberSource({ channel, rollingAverage });

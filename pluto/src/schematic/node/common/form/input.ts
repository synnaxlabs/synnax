// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Input } from "@/input";

export const DIMENSIONS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { y: 2, x: 0.25 },
  bounds: { lower: 0, upper: 2000 },
  endContent: "px",
};

export const PERCENT_BORDER_RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { y: 0.25, x: 0.05 },
  bounds: { lower: 0, upper: 51 },
  endContent: "%",
};

export const STROKE_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: { y: 0.1, x: 0.02 },
  bounds: { lower: 0, upper: 21 },
  endContent: "px",
};

export const VALUE_WIDTH_INPUT_PROPS = {
  dragScale: { x: 1, y: 0.25 },
  bounds: { lower: 40, upper: 500 },
  endContent: "px",
} as const;

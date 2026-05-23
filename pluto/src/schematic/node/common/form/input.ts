// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, type xy } from "@synnaxlabs/x";

import { type Input } from "@/input";
import { type Tabs } from "@/tabs";

export const COMMON_TOGGLE_FORM_TABS: Tabs.Tab[] = [
  { tabKey: "style", name: "Style" },
  { tabKey: "control", name: "Control" },
];

export const DIMENSIONS_DRAG_SCALE: xy.Crude = { y: 2, x: 0.25 };
export const PERCENT_DRAG_SCALE: xy.Crude = { y: 0.25, x: 0.05 };
export const STROKE_WIDTH_DRAG_SCALE: xy.Crude = { y: 0.1, x: 0.02 };
export const DIMENSIONS_BOUNDS: bounds.Bounds = { lower: 0, upper: 2000 };
export const BORDER_RADIUS_BOUNDS: bounds.Bounds = { lower: 0, upper: 51 };
export const STROKE_WIDTH_BOUNDS: bounds.Bounds = { lower: 0, upper: 21 };

export const DIMENSIONS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: DIMENSIONS_DRAG_SCALE,
  bounds: DIMENSIONS_BOUNDS,
  endContent: "px",
};

export const PERCENT_BORDER_RADIUS_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: PERCENT_DRAG_SCALE,
  bounds: BORDER_RADIUS_BOUNDS,
  endContent: "%",
};

export const STROKE_WIDTH_INPUT_PROPS: Partial<Input.NumericProps> = {
  dragScale: STROKE_WIDTH_DRAG_SCALE,
  bounds: STROKE_WIDTH_BOUNDS,
  endContent: "px",
};

export const VALUE_WIDTH_INPUT_PROPS = {
  dragScale: { x: 1, y: 0.25 },
  bounds: { lower: 40, upper: 500 },
  endContent: "px",
} as const;

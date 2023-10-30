// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location, type direction } from "@synnaxlabs/x";

export const Y_AXIS_KEYS = ["y1", "y2", "y3", "y4"] as const;
export type YAxisKey = (typeof Y_AXIS_KEYS)[number];
export type YAxisRecord<T> = Record<YAxisKey, T>;
export type MultiYAxisRecord<T extends any[] | readonly any[]> = Record<YAxisKey, T>;

export const X_AXIS_KEYS = ["x1", "x2"] as const;
export type XAxisKey = (typeof X_AXIS_KEYS)[number];
export type XAxisRecord<T> = Record<XAxisKey, T>;
export type MultiXAxisRecord<T> = Record<XAxisKey, T[]>;

export const AXIS_KEYS = [...Y_AXIS_KEYS, ...X_AXIS_KEYS] as const;
export type AxisKey = (typeof AXIS_KEYS)[number];

export const axisLabel = (key: AxisKey): string => key.toUpperCase();

export const axisDirection = (key: AxisKey): direction.Direction =>
  key[0] as direction.Direction;

export const axisLocation = (key: AxisKey): location.Location => AXIS_LOCATIONS[key];

export const AXIS_LOCATIONS: Record<AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

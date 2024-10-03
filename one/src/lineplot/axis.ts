// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction, type location } from "@synnaxlabs/x";
import { z } from "zod";

export const Y_AXIS_KEYS = ["y1", "y2", "y3", "y4"] as const;
export type YAxisKey = (typeof Y_AXIS_KEYS)[number];
export const yAxisKeyZ = z.enum(Y_AXIS_KEYS);
export type YAxisRecord<T> = Record<YAxisKey, T>;
export const yAxisRecordZ = <T extends any[] | readonly any[]>(valueZ: z.ZodType<T>) =>
  z.record(yAxisKeyZ, valueZ);
export type MultiYAxisRecord<T extends any[] | readonly any[]> = Record<YAxisKey, T>;

export const X_AXIS_KEYS = ["x1", "x2"] as const;
export const xAxisKeyZ = z.enum(X_AXIS_KEYS);
export type XAxisKey = (typeof X_AXIS_KEYS)[number];
export const xAxisRecordZ = <T>(valueZ: z.ZodType<T>) => z.record(xAxisKeyZ, valueZ);
export type XAxisRecord<T> = Record<XAxisKey, T>;
export type MultiXAxisRecord<T> = Record<XAxisKey, T[]>;

export const AXIS_KEYS = [...Y_AXIS_KEYS, ...X_AXIS_KEYS] as const;
export const axisKeyZ = z.enum(AXIS_KEYS);
export type AxisKey = (typeof AXIS_KEYS)[number];

export const axisLabel = (key: AxisKey): string => key.toUpperCase();

export const axisDirection = (key: AxisKey): direction.Direction =>
  key[0] as direction.Direction;

export const axisLocation = (key: AxisKey): location.Outer => AXIS_LOCATIONS[key];

export const AXIS_LOCATIONS: Record<AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

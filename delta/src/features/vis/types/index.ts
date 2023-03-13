// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Direction } from "@synnaxlabs/x";
import { DeepPartial } from "react-hook-form";

export interface Vis {
  variant: string;
  key: string;
}

export const Y_AXIS_KEYS = ["y1", "y2", "y3", "y4"] as const;
export type YAxisKey = typeof Y_AXIS_KEYS[number];
export type MultiYAxisRecord = Record<YAxisKey, readonly string[]>;
export type YAxisRecord = Record<YAxisKey, string>;

export const X_AXIS_KEYS = ["x1", "x2"] as const;
export type XAxisKey = typeof X_AXIS_KEYS[number];
export type MultiXAxisRecord = Record<XAxisKey, readonly string[]>;
export type XAxisRecord = Record<XAxisKey, string>;

export const AXIS_KEYS = [...Y_AXIS_KEYS, ...X_AXIS_KEYS] as const;
export type AxisKey = typeof AXIS_KEYS[number];

export const axisLabel = (key: AxisKey): string => key.toUpperCase();

export const axisDirection = (key: AxisKey): Direction => key[0] as Direction;

export interface ControlledVisProps<V extends Vis, SV extends Vis = V> {
  vis: SV;
  setVis: (vis: DeepPartial<V>) => void;
}

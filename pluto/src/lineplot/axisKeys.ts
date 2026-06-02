// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { type location } from "@synnaxlabs/x";

export const axisLabel = (key: lineplot.AxisKey): string => key.toUpperCase();

export const AXIS_LOCATIONS: Record<lineplot.AxisKey, location.Outer> = {
  y1: "left",
  y2: "right",
  y3: "left",
  y4: "right",
  x1: "bottom",
  x2: "top",
};

export const axisLocation = (key: lineplot.AxisKey): location.Outer =>
  AXIS_LOCATIONS[key];

export const isAxisKey = (s: string): s is lineplot.AxisKey =>
  lineplot.axisKeyZ.safeParse(s).success;

export const isXAxisKey = (s: string): s is lineplot.XAxisKey =>
  lineplot.xAxisKeyZ.safeParse(s).success;

export const isYAxisKey = (s: string): s is lineplot.YAxisKey =>
  lineplot.yAxisKeyZ.safeParse(s).success;

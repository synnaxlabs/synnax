// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction, type location, xy } from "@synnaxlabs/x";

export type DetailedBorderRadius = Record<location.CornerString, xy.XY>;

export type BorderRadius =
  | number
  | Record<direction.Direction, number>
  | Record<location.CornerString, number>
  | DetailedBorderRadius;

export const parseBorderRadius = (radius: BorderRadius): DetailedBorderRadius => {
  if (typeof radius === "number")
    return {
      topLeft: xy.construct(radius),
      topRight: xy.construct(radius),
      bottomLeft: xy.construct(radius),
      bottomRight: xy.construct(radius),
    };
  if ("x" in radius)
    return {
      topLeft: radius,
      topRight: radius,
      bottomLeft: radius,
      bottomRight: radius,
    };
  if (typeof radius.bottomLeft === "number")
    return {
      topLeft: xy.construct(radius.topLeft),
      topRight: xy.construct(radius.topRight),
      bottomLeft: xy.construct(radius.bottomLeft),
      bottomRight: xy.construct(radius.bottomRight),
    };
  return radius as DetailedBorderRadius;
};

export const cssBorderRadius = (radius: DetailedBorderRadius): string => {
  const { topLeft, topRight, bottomLeft, bottomRight } = radius;
  return `${topLeft.x}% ${topRight.x}% ${bottomRight.x}% ${bottomLeft.x}% / ${topLeft.y}% ${topRight.y}% ${bottomRight.y}% ${bottomLeft.y}%`;
};

export const pixelToPercent = (pixel: number, total: number): number =>
  (pixel / total) * 100;

export const DEFAULT_DIMENSIONS = { width: 40, height: 80 };
export const DEFAULT_BORDER_RADIUS = { x: 50, y: 10 };

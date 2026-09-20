// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type border, type dimensions, type xy } from "@synnaxlabs/x";

// Symbols store radii as percentages so corners stay proportional as the user resizes.
export const cssRadius = (radius: border.Radius): string => {
  const { topLeft, topRight, bottomLeft, bottomRight } = radius;
  return `${topLeft.x}% ${topRight.x}% ${bottomRight.x}% ${bottomLeft.x}% / ${topLeft.y}% ${topRight.y}% ${bottomRight.y}% ${bottomLeft.y}%`;
};

export const pixelToPercent = (pixel: number, total: number): number =>
  (pixel / total) * 100;

export const DEFAULT_DIMENSIONS: dimensions.Dimensions = { width: 40, height: 80 };
export const DEFAULT_RADIUS: xy.XY = { x: 50, y: 10 };

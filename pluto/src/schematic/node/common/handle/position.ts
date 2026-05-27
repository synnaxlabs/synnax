// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { Position as RFPosition } from "@xyflow/react";

export const ORIENTATION_RF_POSITIONS: Record<
  location.Outer,
  Record<location.Outer, RFPosition>
> = {
  left: {
    left: RFPosition.Left,
    right: RFPosition.Right,
    top: RFPosition.Top,
    bottom: RFPosition.Bottom,
  },
  right: {
    left: RFPosition.Right,
    right: RFPosition.Left,
    top: RFPosition.Bottom,
    bottom: RFPosition.Top,
  },
  top: {
    left: RFPosition.Bottom,
    right: RFPosition.Top,
    top: RFPosition.Left,
    bottom: RFPosition.Right,
  },
  bottom: {
    left: RFPosition.Top,
    right: RFPosition.Bottom,
    top: RFPosition.Right,
    bottom: RFPosition.Left,
  },
};

export const smart = (
  position: location.Outer,
  orientation: location.Outer,
): RFPosition => ORIENTATION_RF_POSITIONS[orientation][position];

export const swap = (position: RFPosition, bypass: boolean = false): RFPosition => {
  if (bypass) return position;
  switch (position) {
    case RFPosition.Left:
      return RFPosition.Right;
    case RFPosition.Right:
      return RFPosition.Left;
    case RFPosition.Top:
      return RFPosition.Bottom;
    case RFPosition.Bottom:
      return RFPosition.Top;
    default:
      return RFPosition.Top;
  }
};

export const adjust = (
  top: number,
  left: number,
  orientation: location.Outer,
  prevent: boolean = false,
): { left: number; top: number } => {
  if (prevent) return { left, top };
  if (orientation === "left") return { top, left };
  if (orientation === "right") return { top: 100 - top, left: 100 - left };
  if (orientation === "top") return { top: 100 - left, left: top };
  return { top: left, left: 100 - top };
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import {
  Handle,
  type HandleProps,
  Position,
  useUpdateNodeInternals,
} from "@xyflow/react";

import { CSS } from "@/css";

export interface BaseProps extends Omit<HandleProps, "position"> {
  location: location.Outer;
}

const RF_POSITIONS: Record<location.Outer, Position> = {
  top: Position.Top,
  right: Position.Right,
  bottom: Position.Bottom,
  left: Position.Left,
};

export const locationToRFPosition = (location: location.Outer): Position =>
  RF_POSITIONS[location];

export const Base = ({ location, className, ...props }: BaseProps) => {
  try {
    useUpdateNodeInternals();
  } catch {
    return null;
  }
  const position = locationToRFPosition(location);
  return (
    <Handle
      className={CSS.cls(CSS.BE("arc", "handle"), className)}
      position={position}
      {...props}
    />
  );
};

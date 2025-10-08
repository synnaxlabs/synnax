// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, type location } from "@synnaxlabs/x";
import {
  Handle,
  type HandleProps,
  Position,
  useUpdateNodeInternals,
} from "@xyflow/react";

import { CSS } from "@/css";

export interface CoreProps extends Omit<HandleProps, "position"> {
  location: location.Outer;
}

export const locationToRFPosition = (location: location.Outer) =>
  Position[caseconv.capitalize(location) as keyof typeof Position];

export const Core = ({ location, className, ...props }: CoreProps) => {
  try {
    useUpdateNodeInternals();
  } catch {
    return null;
  }
  const position = locationToRFPosition(location);
  return (
    <Handle
      className={CSS(CSS.BE("arc", "handle"), className)}
      position={position}
      {...props}
    />
  );
};

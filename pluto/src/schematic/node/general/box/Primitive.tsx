// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Tank as TankPrimitive } from "@/schematic/node/vessels/tank/Primitive";

interface RenderProps extends Omit<schematic.NodeConfigBox, "variant"> {
  className?: string;
}

export const Box = (props: RenderProps): ReactElement => (
  <TankPrimitive
    {...props}
    dimensions={props.dimensions ?? { width: 25, height: 50 }}
    boxBorderRadius={props.borderRadius ?? 0}
  />
);

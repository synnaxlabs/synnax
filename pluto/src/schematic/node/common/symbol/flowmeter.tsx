// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface FlowmeterLabelProps {
  position?: xy.XY;
  color?: color.Crude;
}

export const FlowmeterLabel = ({
  position,
  color: colorVal,
}: FlowmeterLabelProps): ReactElement => (
  <text
    x={position?.x ?? 57}
    y={position?.y ?? 27}
    style={{ fill: color.cssString(colorVal), fontWeight: 450 }}
    stroke="none"
  >
    F
  </text>
);

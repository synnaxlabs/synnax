// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { usePIDContext } from "@/pid/PIDContext";
import { ValveBodyProps, valveBodySpec } from "@/pid/Valve/valveBodyCore";
import { SVG } from "@/vis/svg";

export const ValveBody = (props: ValveBodyProps): ReactElement | null => {
  return usePIDContext().render("valveBody", props);
};

const valveBodySvg = ({
  position,
  dimensions: { width, height },
  fill,
  stroke,
}: ValveBodyProps): ReactElement => (
  <svg
    width={width}
    height={height}
    x={position.x}
    y={position.y}
    viewBox={SVG.viewBox(valveBodySpec.dimensions)}
  >
    <path d={valveBodySpec.path} stroke={stroke} fill={fill} />
  </svg>
);

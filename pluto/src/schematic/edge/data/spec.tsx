// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type direction, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@synnaxlabs/lyra/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { type Spec } from "@/schematic/edge/spec";

const SYMBOL_INTERVAL = 40;

const SYMBOL_RADIUS_RATIO = 1.5;
const SYMBOL_INTERVAL_PER_WIDTH = 4;

interface SymbolProps {
  position: xy.XY;
  direction: direction.Direction;
  strokeWidth: number;
}

const DataSymbol = ({ position, strokeWidth }: SymbolProps): ReactElement => (
  <circle
    cx={position.x}
    cy={position.y}
    r={strokeWidth * SYMBOL_RADIUS_RATIO}
    fill="var(--pluto-gray-l0)"
    strokeWidth={strokeWidth}
  />
);

export const spec: Spec = Segmented.createSpec(
  "data",
  "Data",
  ({ points, crossings, color: colorVal, strokeWidth }) => (
    <g
      className={CSS.B("symbol-colored")}
      style={{ [CSS.variable("symbol-color")]: color.rgbaString(colorVal) }}
    >
      <Base.Base
        path={Path.rounded(points, crossings)}
        color={colorVal}
        strokeWidth={strokeWidth}
      />
      {Path.computeSymbolPositions(
        points,
        SYMBOL_INTERVAL +
          SYMBOL_INTERVAL_PER_WIDTH * (strokeWidth - Base.DEFAULT_STROKE_WIDTH),
      ).map(({ position, direction }, i) => (
        <DataSymbol
          key={i}
          position={position}
          direction={direction}
          strokeWidth={strokeWidth}
        />
      ))}
    </g>
  ),
);

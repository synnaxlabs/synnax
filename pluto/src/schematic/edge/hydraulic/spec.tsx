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

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { type Spec } from "@/schematic/edge/spec";
import { Primitive } from "@/schematic/node/common/primitive";

const SYMBOL_INTERVAL = 40;
const SYMBOL_INTERVAL_PER_WIDTH = 3;
const SYMBOL_SIZE = 10;

interface SymbolProps {
  position: xy.XY;
  direction: direction.Direction;
  strokeWidth: number;
}

const HydraulicSymbol = ({
  position,
  direction,
  strokeWidth,
}: SymbolProps): ReactElement => {
  const size = SYMBOL_SIZE + strokeWidth - Base.DEFAULT_STROKE_WIDTH;
  const pos = { ...position };
  if (direction === "x") pos.y += size / 2;
  else pos.x += size / 2;
  return (
    <path
      d={`M0,0 L0,-${size} L${size},-${size}`}
      fill="none"
      strokeWidth={strokeWidth}
      transform={`translate(${pos.x},${pos.y}) rotate(270)`}
      strokeLinecap="round"
    />
  );
};

const Line = ({
  points,
  crossings,
  color: colorVal,
  strokeWidth,
}: Segmented.PathProps): ReactElement => {
  const symbolStyle = { [Primitive.SYMBOL_COLOR]: color.rgbaString(colorVal) };
  return (
    <g className={CSS.symbolColored} style={symbolStyle}>
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
        <HydraulicSymbol
          key={i}
          position={position}
          direction={direction}
          strokeWidth={strokeWidth}
        />
      ))}
    </g>
  );
};

export const spec: Spec = Segmented.createSpec("hydraulic", "Hydraulic", Line);

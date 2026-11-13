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
const SYMBOL_SIZE = 10;
const CORNER_CLEARANCE = 5;
const CORNER_CLEARANCE_PER_WIDTH = 4;

interface SymbolProps {
  position: xy.XY;
  direction: direction.Direction;
  strokeWidth: number;
  scale: number;
}

const HydraulicSymbol = ({
  position: { x, y },
  direction,
  strokeWidth,
  scale,
}: SymbolProps): ReactElement => {
  const size = SYMBOL_SIZE * scale;
  const symbolStroke = Math.min(strokeWidth, Base.DEFAULT_STROKE_WIDTH * scale);
  // Reaches as far past the edge of a wide line as the default symbol does.
  const half = size / 2 + (strokeWidth - symbolStroke) / 2;
  const d =
    direction === "x"
      ? `M${x},${y + half} L${x - size},${y + half} L${x - size},${y - half}`
      : `M${x + half},${y} L${x - half},${y} L${x - half},${y - size}`;
  return <path d={d} fill="none" strokeWidth={symbolStroke} strokeLinecap="round" />;
};

const Line = ({
  points,
  crossings,
  color: colorVal,
  strokeWidth,
}: Segmented.PathProps): ReactElement => {
  const symbolStyle = { [Primitive.SYMBOL_COLOR]: color.rgbaString(colorVal) };
  // Grows slower than the line, so a wide line still fits symbols on each run.
  const scale = Math.sqrt(strokeWidth / Base.DEFAULT_STROKE_WIDTH);
  return (
    <g className={CSS.symbolColored} style={symbolStyle}>
      <Base.Base
        path={Path.rounded(points, crossings)}
        color={colorVal}
        strokeWidth={strokeWidth}
      />
      {Path.computeSymbolPositions(
        points,
        SYMBOL_INTERVAL * scale,
        CORNER_CLEARANCE +
          CORNER_CLEARANCE_PER_WIDTH * (strokeWidth - Base.DEFAULT_STROKE_WIDTH),
      ).map(({ position, direction }, i) => (
        <HydraulicSymbol
          key={i}
          position={position}
          direction={direction}
          strokeWidth={strokeWidth}
          scale={scale}
        />
      ))}
    </g>
  );
};

export const spec: Spec = Segmented.createSpec("hydraulic", "Hydraulic", Line);

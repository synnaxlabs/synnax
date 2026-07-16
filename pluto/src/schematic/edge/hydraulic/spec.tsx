// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { type direction, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { type Spec } from "@/schematic/edge/spec";
import { symbolColorVar } from "@/schematic/symbolColor";

const SYMBOL_INTERVAL = 40;
const SYMBOL_SIZE = 10;

interface SymbolProps {
  position: xy.XY;
  direction: direction.Direction;
}

const HydraulicSymbol = ({ position, direction }: SymbolProps): ReactElement => {
  const pos = { ...position };
  if (direction === "x") pos.y += SYMBOL_SIZE / 2;
  else pos.x += SYMBOL_SIZE / 2;
  return (
    <path
      d={`M0,0 L0,-${SYMBOL_SIZE} L${SYMBOL_SIZE},-${SYMBOL_SIZE}`}
      fill="none"
      strokeWidth={2}
      transform={`translate(${pos.x},${pos.y}) rotate(270)`}
      strokeLinecap="round"
    />
  );
};

export const spec: Spec<"hydraulic", schematic.EdgeConfigHydraulic> =
  Segmented.createSpec(
    "hydraulic",
    "Hydraulic",
    ({ points, crossings, color: colorVal }) => (
      <g
        className={CSS.B("symbol-colored")}
        style={{ [CSS.var("symbol-color")]: symbolColorVar(colorVal) }}
      >
        <Base.Base path={Path.rounded(points, crossings)} color={colorVal} />
        {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
          ({ position, direction }, i) => (
            <HydraulicSymbol key={i} position={position} direction={direction} />
          ),
        )}
      </g>
    ),
  );

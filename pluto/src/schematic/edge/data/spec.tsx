// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { symbolColorVar } from "@/schematic/symbolColor";

const SYMBOL_INTERVAL = 40;

interface SymbolProps {
  position: xy.XY;
  direction: direction.Direction;
}

const DataSymbol = ({ position }: SymbolProps): ReactElement => (
  <circle
    cx={position.x}
    cy={position.y}
    r={3}
    fill="var(--pluto-gray-l0)"
    strokeWidth={2}
  />
);

export const spec = Segmented.createSpec(
  "data",
  "Data",
  ({ points, color: colorVal }) => (
    <g
      className={CSS.B("symbol-colored")}
      style={{ [CSS.var("symbol-color")]: symbolColorVar(colorVal) }}
    >
      <Base.Base path={Path.rounded(points)} color={colorVal} />
      {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
        ({ position, direction }, i) => (
          <DataSymbol key={i} position={position} direction={direction} />
        ),
      )}
    </g>
  ),
);

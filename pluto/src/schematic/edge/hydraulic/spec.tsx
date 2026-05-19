// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import { type direction } from "@synnaxlabs/x/direction";
import { type xy } from "@synnaxlabs/x/xy";
import { type ReactElement } from "react";

import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { NAME, VARIANT } from "@/schematic/edge/hydraulic/config";

const SYMBOL_INTERVAL = 40;
const SYMBOL_SIZE = 10;

interface SymbolProps {
  color: color.Crude;
  position: xy.XY;
  direction: direction.Direction;
}

const HydraulicSymbol = ({
  color: colorVal,
  position,
  direction,
}: SymbolProps): ReactElement => {
  const pos = { ...position };
  if (direction === "x") pos.y += SYMBOL_SIZE / 2;
  else pos.x += SYMBOL_SIZE / 2;
  return (
    <path
      d={`M0,0 L0,-${SYMBOL_SIZE} L${SYMBOL_SIZE},-${SYMBOL_SIZE}`}
      stroke={color.cssString(colorVal)}
      fill="none"
      strokeWidth={2}
      transform={`translate(${pos.x},${pos.y}) rotate(270)`}
      strokeLinecap="round"
    />
  );
};

export const spec = Segmented.createSpec(VARIANT, NAME, ({ points, color }) => (
  <>
    <Base.Base path={Path.rounded(points)} color={color} />
    {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
      ({ position, direction }, i) => (
        <HydraulicSymbol
          key={i}
          position={position}
          direction={direction}
          color={color}
        />
      ),
    )}
  </>
));

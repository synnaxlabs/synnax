import { color as colorX, type direction, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Base } from "@/schematic/edge/base";
import { Path } from "@/schematic/edge/path";
import { Segmented } from "@/schematic/edge/segmented";

const SYMBOL_INTERVAL = 40;

interface SymbolProps {
  color: colorX.Color;
  position: xy.XY;
  direction: direction.Direction;
}

const DataSymbol = ({ color, position }: SymbolProps): ReactElement => (
  <circle
    cx={position.x}
    cy={position.y}
    r={3}
    fill="var(--pluto-gray-l0)"
    stroke={colorX.cssString(color)}
    strokeWidth={2}
  />
);

export const dataSpec = Segmented.createSpec("data", "Data", ({ points, color }) => (
  <>
    <Base.Base path={Path.rounded(points)} color={color} />
    {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
      ({ position, direction }, i) => (
        <DataSymbol key={i} position={position} direction={direction} color={color} />
      ),
    )}
  </>
));

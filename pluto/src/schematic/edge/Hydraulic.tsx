import { color as colorX, type direction, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Base } from "@/schematic/edge/base";
import { Path } from "@/schematic/edge/path";
import { Segmented } from "@/schematic/edge/segmented";

const SYMBOL_INTERVAL = 40;
const SYMBOL_SIZE = 10;

interface SymbolProps {
  color: colorX.Color;
  position: xy.XY;
  direction: direction.Direction;
}

const HydraulicSymbol = ({ color, position, direction }: SymbolProps): ReactElement => {
  const pos = { ...position };
  if (direction === "x") pos.y += SYMBOL_SIZE / 2;
  else pos.x += SYMBOL_SIZE / 2;
  return (
    <path
      d={`M0,0 L0,-${SYMBOL_SIZE} L${SYMBOL_SIZE},-${SYMBOL_SIZE}`}
      stroke={colorX.cssString(color)}
      fill="none"
      strokeWidth={2}
      transform={`translate(${pos.x},${pos.y}) rotate(270)`}
      strokeLinecap="round"
    />
  );
};

export const hydraulicSpec = Segmented.createSpec(
  "hydraulic",
  "Hydraulic",
  ({ points, color }) => (
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
  ),
);

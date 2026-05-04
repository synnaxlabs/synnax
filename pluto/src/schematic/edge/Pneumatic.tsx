import { color as colorX, type direction, xy } from "@synnaxlabs/x";
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

const PneumaticSymbol = ({ color, position, direction }: SymbolProps): ReactElement => {
  const pos = { ...position };
  if (direction === "x") pos.y += SYMBOL_SIZE / 3;
  else pos.x -= SYMBOL_SIZE / 3;
  let pointTwo = xy.translateX(pos, -10);
  let rotate = 45;
  if (direction === "y") {
    pointTwo = xy.translateY(pos, -10);
    rotate += 90;
  }
  const stroke = colorX.cssString(color);
  return (
    <>
      <path
        d={`M0,0 L0,-${SYMBOL_SIZE}`}
        stroke={stroke}
        fill="none"
        strokeWidth={2}
        transform={`translate(${pos.x},${pos.y}) rotate(${rotate})`}
        strokeLinecap="round"
      />
      <path
        d={`M0,0 L0,-${SYMBOL_SIZE}`}
        stroke={stroke}
        fill="none"
        strokeWidth={2}
        transform={`translate(${pointTwo.x},${pointTwo.y}) rotate(${rotate})`}
        strokeLinecap="round"
      />
    </>
  );
};

export const pneumaticSpec = Segmented.createSpec(
  "pneumatic",
  "Pneumatic",
  ({ points, color }) => (
    <>
      <Base.Base path={Path.rounded(points)} color={color} />
      {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
        ({ position, direction }, i) => (
          <PneumaticSymbol
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

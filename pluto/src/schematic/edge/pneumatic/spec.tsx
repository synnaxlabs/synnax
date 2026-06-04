// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type direction, xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { NAME, VARIANT } from "@/schematic/edge/pneumatic/config";

const SYMBOL_INTERVAL = 40;
const SYMBOL_SIZE = 10;

interface SymbolProps {
  position: xy.XY;
  direction: direction.Direction;
}

const PneumaticSymbol = ({ position, direction }: SymbolProps): ReactElement => {
  const pos = { ...position };
  if (direction === "x") pos.y += SYMBOL_SIZE / 3;
  else pos.x -= SYMBOL_SIZE / 3;
  let pointTwo = xy.translateX(pos, -10);
  let rotate = 45;
  if (direction === "y") {
    pointTwo = xy.translateY(pos, -10);
    rotate += 90;
  }
  return (
    <>
      <path
        d={`M0,0 L0,-${SYMBOL_SIZE}`}
        stroke="var(--pluto-symbol-display)"
        fill="none"
        strokeWidth={2}
        transform={`translate(${pos.x},${pos.y}) rotate(${rotate})`}
        strokeLinecap="round"
      />
      <path
        d={`M0,0 L0,-${SYMBOL_SIZE}`}
        stroke="var(--pluto-symbol-display)"
        fill="none"
        strokeWidth={2}
        transform={`translate(${pointTwo.x},${pointTwo.y}) rotate(${rotate})`}
        strokeLinecap="round"
      />
    </>
  );
};

export const spec = Segmented.createSpec(
  VARIANT,
  NAME,
  ({ points, color: colorVal }) => (
    <g
      className={CSS.B("symbol-colored")}
      style={{
        [CSS.var("symbol-color")]: color.isZero(colorVal)
          ? undefined
          : color.rgbString(colorVal),
      }}
    >
      <Base.Base path={Path.rounded(points)} color={colorVal} />
      {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
        ({ position, direction }, i) => (
          <PneumaticSymbol key={i} position={position} direction={direction} />
        ),
      )}
    </g>
  ),
);

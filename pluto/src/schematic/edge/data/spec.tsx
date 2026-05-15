// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x/color";
import { direction } from "@synnaxlabs/x/direction";
import { xy } from "@synnaxlabs/x/xy";
import { type ReactElement } from "react";

import { Base } from "@/schematic/edge/common/base";
import { Path } from "@/schematic/edge/common/path";
import { Segmented } from "@/schematic/edge/common/segmented";
import { NAME, VARIANT } from "@/schematic/edge/data/config";

const SYMBOL_INTERVAL = 40;

interface SymbolProps {
  color: color.Crude;
  position: xy.XY;
  direction: direction.Direction;
}

const DataSymbol = ({ color: colorVal, position }: SymbolProps): ReactElement => (
  <circle
    cx={position.x}
    cy={position.y}
    r={3}
    fill="var(--pluto-gray-l0)"
    stroke={color.cssString(colorVal)}
    strokeWidth={2}
  />
);

export const spec = Segmented.createSpec(VARIANT, NAME, ({ points, color }) => (
  <>
    <Base.Base path={Path.rounded(points)} color={color} />
    {Path.computeSymbolPositions(points, SYMBOL_INTERVAL).map(
      ({ position, direction }, i) => (
        <DataSymbol key={i} position={position} direction={direction} color={color} />
      ),
    )}
  </>
));

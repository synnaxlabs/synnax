// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type color, type location, type scale } from "@synnaxlabs/x";
import { type ReactElement, memo, useMemo } from "react";

import { CSS } from "@/css";
import {
  newTickFactory,
  type Tick,
  type TickFactoryProps,
  type TickFactory,
} from "@/vis/axis/ticks";

const TICK_LINE_SIZE = 5;
const TICK_PADDING = 6;

export interface SVGProps extends TickFactoryProps {
  position: location.Outer;
  decimalToDataScale: scale.Scale;
  size: number;
  showGrid?: boolean;
  gridSize?: number;
  color?: color.Crude;
  gridColor?: color.Crude;
}

interface TickMarkProps {
  tick: Tick;
  position: location.Outer;
  color: string;
}

const TickMark = memo(({ tick, position, color }: TickMarkProps): ReactElement => {
  const isHorizontal = position === "top" || position === "bottom";
  const tickDir = position === "bottom" || position === "right" ? 1 : -1;

  const lineProps = isHorizontal
    ? { x1: 0, y1: 0, x2: 0, y2: TICK_LINE_SIZE * tickDir }
    : { x1: 0, y1: 0, x2: TICK_LINE_SIZE * tickDir, y2: 0 };

  const textProps = isHorizontal
    ? {
        x: 0,
        y: (TICK_LINE_SIZE + TICK_PADDING) * tickDir,
        textAnchor: "middle" as const,
        dominantBaseline: tickDir === 1 ? ("hanging" as const) : ("auto" as const),
      }
    : {
        x: (TICK_LINE_SIZE + TICK_PADDING) * tickDir,
        y: 0,
        textAnchor: tickDir === 1 ? ("start" as const) : ("end" as const),
        dominantBaseline: "middle" as const,
      };

  const transform = isHorizontal
    ? `translate(${tick.position}, 0)`
    : `translate(0, ${tick.position})`;

  return (
    <g className={CSS.BE("axis", "tick")} transform={transform}>
      <line className={CSS.BE("axis", "tick-mark")} stroke={color} {...lineProps} />
      <text className={CSS.BE("axis", "label")} fill={color} {...textProps}>
        {tick.label}
      </text>
    </g>
  );
});
TickMark.displayName = "TickMark";

interface GridLineProps {
  tick: Tick;
  position: location.Outer;
  gridSize: number;
  color: string;
}

const GridLine = memo(
  ({ tick, position, gridSize, color }: GridLineProps): ReactElement => {
    const isHorizontal = position === "top" || position === "bottom";
    const gridDir = position === "bottom" || position === "right" ? -1 : 1;

    const lineProps = isHorizontal
      ? { x1: tick.position, y1: 0, x2: tick.position, y2: gridSize * gridDir }
      : { x1: 0, y1: tick.position, x2: gridSize * gridDir, y2: tick.position };

    return (
      <line
        className={CSS.BE("axis", "grid-line")}
        stroke={color}
        strokeOpacity={0.5}
        {...lineProps}
      />
    );
  },
);
GridLine.displayName = "GridLine";

export const SVG = memo(
  ({
    position,
    decimalToDataScale,
    size,
    showGrid = false,
    gridSize = 0,
    color = "var(--pluto-gray-l6)",
    gridColor = "var(--pluto-gray-l3)",
    tickSpacing = 75,
    type = "linear",
  }: SVGProps): ReactElement => {
    const tickFactory: TickFactory = useMemo(
      () => newTickFactory({ tickSpacing, type }),
      [tickSpacing, type],
    );

    const ticks: Tick[] = useMemo(
      () => tickFactory.create({ decimalToDataScale, size }),
      [tickFactory, decimalToDataScale, size],
    );

    const isHorizontal = position === "top" || position === "bottom";

    const axisLineProps = isHorizontal
      ? { x1: 0, y1: 0, x2: size, y2: 0 }
      : { x1: 0, y1: 0, x2: 0, y2: size };

    const colorStr = CSS.colorVar(color) ?? "var(--pluto-gray-l6)";
    const gridColorStr = CSS.colorVar(gridColor) ?? "var(--pluto-gray-l3)";

    return (
      <svg
        className={CSS(CSS.B("axis"), CSS.BM("axis", "svg"), CSS.loc(position))}
        style={{ overflow: "visible" }}
      >
        {showGrid && (
          <g className={CSS.BE("axis", "grid")}>
            {ticks.map((tick) => (
              <GridLine
                key={tick.position}
                tick={tick}
                position={position}
                gridSize={gridSize}
                color={gridColorStr}
              />
            ))}
          </g>
        )}

        <line
          className={CSS.BE("axis", "line")}
          stroke={colorStr}
          {...axisLineProps}
        />

        <g className={CSS.BE("axis", "ticks")}>
          {ticks.map((tick) => (
            <TickMark
              key={tick.position}
              tick={tick}
              position={position}
              color={colorStr}
            />
          ))}
        </g>
      </svg>
    );
  },
);
SVG.displayName = "SVGAxis";

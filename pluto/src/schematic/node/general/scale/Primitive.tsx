// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type schematic } from "@synnaxlabs/client";
import { color } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { symbolColorVar } from "@/schematic/symbolColor";

interface RenderProps extends Pick<schematic.ScaleNodeConfig, "indicator"> {
  className?: string;
}

// Authored at the size the picker shows: neighbors get 0.75 from Primitive.Div, a
// chassis this one cannot take because it forces fill: none over the bar and caret.
const WIDTH = 30;
const HEIGHT = 48;
const CONTAINER_STYLE: CSSProperties = {
  width: WIDTH,
  height: HEIGHT,
  position: "relative",
};

const BAR_LEFT = 4;
const BAR_RIGHT = 16;
const BAR_TOP = 3;
const BAR_BOTTOM = 45;
const TICK_LENGTH = 3;
const CARET_SIZE = 3;
const TICK_RATIOS = [0, 0.5, 1];
// Kept off every tick ratio so the caret can never sit under a tick.
const VALUE_RATIO = 0.75;

const alongBar = (ratio: number): number => BAR_BOTTOM - (BAR_BOTTOM - BAR_TOP) * ratio;

const VALUE_Y = alongBar(VALUE_RATIO);
const TICK_YS = TICK_RATIOS.map(alongBar);

const AXIS_FALLBACK = "var(--pluto-gray-l8)";

export const Scale = ({
  indicator: { color: c, axisColor, showFill, showCaret } = {},
  className,
}: RenderProps): ReactElement => {
  const containerStyle = useMemo<CSSProperties>(
    () => ({ ...CONTAINER_STYLE, [CSS.variable("symbol-color")]: symbolColorVar(c) }),
    [c],
  );
  const axis = color.isZero(axisColor) ? AXIS_FALLBACK : color.hex(axisColor);
  return (
    <div className={CSS.cls(CSS.B("symbol-colored"), className)} style={containerStyle}>
      <svg width={WIDTH} height={HEIGHT} style={{ position: "absolute" }}>
        {showFill && (
          <>
            <rect
              x={BAR_LEFT}
              y={BAR_TOP}
              width={BAR_RIGHT - BAR_LEFT}
              height={BAR_BOTTOM - BAR_TOP}
              rx={2}
              fill="none"
              stroke={axis}
              strokeWidth={1}
            />
            <rect
              x={BAR_LEFT}
              y={VALUE_Y}
              width={BAR_RIGHT - BAR_LEFT}
              height={BAR_BOTTOM - VALUE_Y}
              rx={2}
              fill="var(--pluto-symbol-display)"
            />
          </>
        )}
        {!showFill && (
          <line
            x1={BAR_RIGHT}
            y1={BAR_TOP}
            x2={BAR_RIGHT}
            y2={BAR_BOTTOM}
            stroke={axis}
            strokeWidth={1}
          />
        )}
        {TICK_YS.map((y) => (
          <line
            key={y}
            x1={BAR_RIGHT}
            y1={y}
            x2={BAR_RIGHT + TICK_LENGTH}
            y2={y}
            stroke={axis}
            strokeWidth={1}
          />
        ))}
        {showCaret && (
          <path
            d={`M ${BAR_RIGHT} ${VALUE_Y} L ${BAR_RIGHT + CARET_SIZE} ${VALUE_Y - CARET_SIZE} L ${BAR_RIGHT + CARET_SIZE} ${VALUE_Y + CARET_SIZE} Z`}
            fill="var(--pluto-symbol-display)"
          />
        )}
      </svg>
    </div>
  );
};

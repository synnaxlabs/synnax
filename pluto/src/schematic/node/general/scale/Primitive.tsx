// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { type Config } from "@/schematic/node/general/scale/config";
import { symbolColorVar } from "@/schematic/symbolColor";

interface RenderProps extends Omit<Config, "variant"> {
  className?: string;
}

const CONTAINER_STYLE: CSSProperties = { width: 40, height: 64, position: "relative" };

const PREVIEW_RATIO = 0.6;
const BAR_LEFT = 5;
const BAR_RIGHT = 19;
const BAR_TOP = 4;
const BAR_BOTTOM = 60;
const VALUE_Y = BAR_BOTTOM - (BAR_BOTTOM - BAR_TOP) * PREVIEW_RATIO;
const TICK_YS = [BAR_TOP, VALUE_Y, BAR_BOTTOM];

export const Scale = ({
  color: c,
  style = "fill",
  className,
}: RenderProps): ReactElement => {
  const containerStyle = useMemo<CSSProperties>(
    () => ({ ...CONTAINER_STYLE, [CSS.var("symbol-color")]: symbolColorVar(c) }),
    [c],
  );
  return (
    <div className={CSS(CSS.B("symbol-colored"), className)} style={containerStyle}>
      <svg width="40" height="64" style={{ position: "absolute" }}>
        <rect
          x={BAR_LEFT}
          y={BAR_TOP}
          width={BAR_RIGHT - BAR_LEFT}
          height={BAR_BOTTOM - BAR_TOP}
          rx={2}
          fill="none"
          stroke="var(--pluto-gray-l5)"
          strokeWidth={1}
        />
        {style === "fill" ? (
          <rect
            x={BAR_LEFT}
            y={VALUE_Y}
            width={BAR_RIGHT - BAR_LEFT}
            height={BAR_BOTTOM - VALUE_Y}
            rx={2}
            fill="var(--pluto-symbol-display)"
          />
        ) : (
          <path
            d={`M ${BAR_RIGHT} ${VALUE_Y} L ${BAR_RIGHT + 5} ${VALUE_Y - 5} L ${BAR_RIGHT + 5} ${VALUE_Y + 5} Z`}
            fill="var(--pluto-symbol-display)"
          />
        )}
        {TICK_YS.map((y) => (
          <line
            key={y}
            x1={BAR_RIGHT}
            y1={y}
            x2={BAR_RIGHT + 4}
            y2={y}
            stroke="var(--pluto-gray-l5)"
            strokeWidth={1}
          />
        ))}
      </svg>
    </div>
  );
};

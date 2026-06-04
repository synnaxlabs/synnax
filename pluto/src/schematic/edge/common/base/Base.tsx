// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color } from "@synnaxlabs/x";
import { BaseEdge, type BaseEdgeProps } from "@xyflow/react";
import { type CSSProperties, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";

export interface BaseProps extends Omit<BaseEdgeProps, "color"> {
  color: color.Crude;
}

const INTERACTION_WIDTH = 30;

export const Base = ({
  style: baseStyle,
  color: stroke,
  className,
  ...props
}: BaseProps): ReactElement => {
  const style = useMemo<CSSProperties>(() => {
    // A non-color string (e.g. the connection-line preview's CSS variable) is stroked
    // directly and skips the theme transform.
    if (typeof stroke === "string" && !color.colorZ.safeParse(stroke).success)
      return { ...baseStyle, stroke };
    return {
      ...baseStyle,
      [CSS.var("symbol-color")]: color.isZero(stroke)
        ? undefined
        : color.rgbString(stroke),
      stroke: "var(--pluto-symbol-display)",
    };
  }, [stroke, baseStyle]);
  return (
    <BaseEdge
      {...props}
      className={CSS(CSS.B("symbol-colored"), className)}
      interactionWidth={INTERACTION_WIDTH}
      style={style}
    />
  );
};

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
import { type CSSProperties, type ReactElement } from "react";
import { z } from "zod";

import { CSS } from "@/css";
import { Primitive } from "@/schematic/node/common/primitive";

export interface BaseProps extends Omit<BaseEdgeProps, "color"> {
  color?: color.Crude;
}

const INTERACTION_WIDTH = 30;

export const Base = ({
  style: baseStyle,
  color: stroke,
  className,
  ...props
}: BaseProps): ReactElement => {
  // A non-color string (e.g. the connection-line preview's CSS variable) is stroked
  // directly and skips the theme transform.
  const rawStroke =
    typeof stroke === "string" && !z.validate(color.colorZ, stroke)
      ? stroke
      : undefined;
  const symbolColor = rawStroke == null ? color.rgbaString(stroke) : undefined;
  const style: CSSProperties =
    rawStroke != null
      ? { ...baseStyle, stroke: rawStroke }
      : {
          ...baseStyle,
          [Primitive.SYMBOL_COLOR]: symbolColor,
          stroke: "var(--pluto-symbol-display)",
        };
  return (
    <BaseEdge
      {...props}
      className={CSS.cls(CSS.symbolColored, className)}
      interactionWidth={INTERACTION_WIDTH}
      style={style}
    />
  );
};

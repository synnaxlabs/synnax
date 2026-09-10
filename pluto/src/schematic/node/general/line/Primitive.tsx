// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/schematic/node/general/line/line.css";

import { type color, type xy } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { symbolColorVar } from "@/schematic/symbolColor";

export interface LineProps {
  className?: string;
  color?: color.Crude;
  start?: xy.XY;
  end?: xy.XY;
  strokeWidth?: number;
}

export const Line = ({
  className,
  color: colorVal,
  start = { x: 0, y: 20 },
  end = { x: 40, y: 0 },
  strokeWidth = 2,
}: LineProps): ReactElement => {
  // A zero-area box would match every selection rectangle, so the box is at least 1px.
  const width = Math.max(1, start.x, end.x);
  const height = Math.max(1, start.y, end.y);
  const ends = { x1: start.x, y1: start.y, x2: end.x, y2: end.y };
  return (
    <svg
      className={CSS.cls(className, CSS.B("line"), CSS.B("symbol-colored"))}
      width={width}
      height={height}
      style={{ [CSS.variable("symbol-color")]: symbolColorVar(colorVal) }}
    >
      <line className={CSS.BE("line", "hit")} {...ends} />
      <line strokeWidth={strokeWidth} {...ends} />
    </svg>
  );
};

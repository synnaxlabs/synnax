// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import { box } from "@/spatial/box";
import { location } from "@/spatial/location";
import { xy as base } from "@/spatial/xy";

export const completeXY = base.xy.extend({
  root: location.corner,
  units: z.object({
    x: z.enum(["px", "decimal"]),
    y: z.enum(["px", "decimal"]),
  }),
});

export type CompleteXY = z.infer<typeof completeXY>;

export const xy = completeXY.partial({ root: true, units: true });

export interface XY extends z.infer<typeof xy> {}

interface ToCSSReturn extends Partial<Record<location.Outer, string>> {}

export const toCSS = (pos: XY): ToCSSReturn => {
  const ret: ToCSSReturn = {};
  ret[pos.root?.x ?? "left"] =
    pos?.units?.x === "px" ? `${pos.x}px` : `${pos.x * 100}%`;
  ret[pos.root?.y ?? "top"] = pos?.units?.y === "px" ? `${pos.y}px` : `${pos.y * 100}%`;
  return ret;
};

export const toDecimal = (pos: XY, element: box.Box, container: box.Box): base.XY => {
  const ret = { x: pos.x, y: pos.y };
  if (pos.units?.x === "decimal") {
    if (pos.root?.x === "right") ret.x = 1 - pos.x;
  } else if (pos.root?.x === "right")
    ret.x = 1 - (pos.x + box.width(element)) / box.width(container);
  else ret.x /= box.width(container);
  if (pos.units?.y === "decimal") {
    if (pos.root?.y === "bottom") ret.y = 1 - pos.y;
  } else if (pos.root?.y === "bottom")
    ret.y = 1 - (pos.y + box.height(element)) / box.height(container);
  else ret.y /= box.height(container);
  return ret;
};

export interface CalculateOptions {
  lowerThreshold?: number;
  upperThreshold?: number;
}

export const calculate = (
  pos: XY,
  element: box.Box,
  container: box.Box,
  options?: CalculateOptions,
): XY => {
  const lowerThreshold = options?.lowerThreshold ?? 0.2;
  const upperThreshold = options?.upperThreshold ?? 0.8;
  const ret: Required<XY> = {
    x: pos.x,
    y: pos.y,
    root: { ...location.TOP_LEFT },
    units: { x: "px", y: "px" },
  };
  if (pos.x > upperThreshold) {
    ret.x = (1 - pos.x) * box.width(container) - box.width(element);
    ret.root.x = "right";
  } else if (pos.x < lowerThreshold) ret.x = pos.x * box.width(container);
  else ret.units.x = "decimal";
  if (pos.y > upperThreshold) {
    ret.y = (1 - pos.y) * box.height(container) - box.height(element);
    ret.root.y = "bottom";
  } else if (pos.y < lowerThreshold) ret.y = pos.y * box.height(container);
  else ret.units.y = "decimal";
  ret.x = Math.round(ret.x * 100) / 100;
  return { ...ret, ...base.truncate(ret, 3) };
};

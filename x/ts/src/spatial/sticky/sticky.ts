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

export interface ToDecimalProps {
  position: XY;
  element: box.Box;
  container: box.Box;
}

export const toDecimal = ({
  position,
  element,
  container,
}: ToDecimalProps): base.XY => {
  const ret = { x: position.x, y: position.y };
  if (position.units?.x === "decimal") {
    if (position.root?.x === "right") ret.x = 1 - position.x;
  } else if (position.root?.x === "right")
    ret.x = 1 - (position.x + box.width(element)) / box.width(container);
  else ret.x /= box.width(container);
  if (position.units?.y === "decimal") {
    if (position.root?.y === "bottom") ret.y = 1 - position.y;
  } else if (position.root?.y === "bottom")
    ret.y = 1 - (position.y + box.height(element)) / box.height(container);
  else ret.y /= box.height(container);
  return ret;
};

export interface CalculateProps {
  position: XY;
  element: box.Box;
  container: box.Box;
  lowerThreshold?: number;
  upperThreshold?: number;
}

export const calculate = ({
  position,
  element,
  container,
  lowerThreshold = 0.2,
  upperThreshold = 0.8,
}: CalculateProps): XY => {
  const ret: Required<XY> = {
    x: position.x,
    y: position.y,
    root: { ...location.TOP_LEFT },
    units: { x: "px", y: "px" },
  };
  if (position.x > upperThreshold) {
    ret.x = (1 - position.x) * box.width(container) - box.width(element);
    ret.root.x = "right";
  } else if (position.x < lowerThreshold) ret.x = position.x * box.width(container);
  else ret.units.x = "decimal";
  if (position.y > upperThreshold) {
    ret.y = (1 - position.y) * box.height(container) - box.height(element);
    ret.root.y = "bottom";
  } else if (position.y < lowerThreshold) ret.y = position.y * box.height(container);
  else ret.units.y = "decimal";
  ret.x = Math.round(ret.x * 100) / 100;
  return { ...ret, ...base.truncate(ret, 3) };
};

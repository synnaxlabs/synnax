// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const POSITIONS = ["start", "center", "end"];
export type Position = typeof POSITIONS[number];

export const ORDERS = ["first", "last"] as const;
export type Order = typeof ORDERS[number];

export const Y_LOCATIONS = ["top", "bottom"] as const;
export type YLocation = typeof Y_LOCATIONS[number];
export const X_LOCATIONS = ["left", "right"] as const;
export type XLocation = typeof X_LOCATIONS[number];
export type CenterLocation = "center";

export const CORNERS = ["topLeft", "topRight", "bottomLeft", "bottomRight"] as const;
export type Corner = typeof CORNERS[number];

export const OUTER_LOCATIONS = [...Y_LOCATIONS, ...X_LOCATIONS] as const;
export type OuterLocation = typeof OUTER_LOCATIONS[number];

export const LOCATIONS = [...OUTER_LOCATIONS, "center"] as const;
export type Location = typeof LOCATIONS[number];

export const DIRECTIONS = ["x", "y"] as const;
export type Direction = typeof DIRECTIONS[number];
export const isDirection = (v: string): v is Direction =>
  DIRECTIONS.includes(v as Direction);

export const locToDir = (loc: Location | Direction): Direction => {
  if (isDirection(loc)) return loc;
  return Y_LOCATIONS.includes(loc as YLocation) ? "y" : "x";
};

export const swapDir = (direction: Direction): Direction =>
  direction === "x" ? "y" : "x";

export const locFromDir = (direction: Direction): Location =>
  direction === "x" ? "left" : "top";

const SWAPPED_LOCS = {
  top: "bottom",
  bottom: "top",
  left: "right",
  right: "left",
  center: "center",
} as const;

export const swapLoc = (location: Location): Location => SWAPPED_LOCS[location];

/** A generic 2D point, scale, or offset. */
export interface XY extends Record<Direction, number> {}

export const ZERO_XY: XY = { x: 0, y: 0 };
export const ZERO_DIMS: Dimensions = { width: 0, height: 0 };
export const ONE_XY: XY = { x: 1, y: 1 };
export const INFINITE_XY: XY = { x: Infinity, y: Infinity };

export interface SignedDimensions {
  signedWidth: number;
  signedHeight: number;
}

export interface Dimensions {
  width: number;
  height: number;
}

export interface Transform {
  offset: XY;
  scale: XY;
}

export interface ClientXY {
  clientX: number;
  clientY: number;
}

export type UnparsedXY = number | XY | ClientXY | Dimensions | SignedDimensions;

export const toXY = (pt: UnparsedXY): XY => {
  if (typeof pt === "number") return { x: pt, y: pt };
  if ("clientX" in pt) return { x: pt.clientX, y: pt.clientY };
  if ("width" in pt) return { x: pt.width, y: pt.height };
  if ("signedWidth" in pt) return { x: pt.signedWidth, y: pt.signedHeight };
  return { x: pt.x, y: pt.y };
};

export const toXYEqual = (one?: UnparsedXY, two?: UnparsedXY): boolean => {
  if (one == null || two == null) return one == null && two == null;
  const oneXY = toXY(one);
  const twoXY = toXY(two);
  return oneXY.x === twoXY.x && oneXY.y === twoXY.y;
};

export const locDim = (
  location: Location | Direction,
  point: XY | Dimensions
): number => toXY(point)[locToDir(location)];

export type ClientXYF = (e: ClientXY) => void;

export const dirToDim = (direction: Direction): "width" | "height" =>
  direction === "x" ? "width" : "height";

export interface Bound {
  lower: number;
  upper: number;
}

export interface XYBound {
  x: Bound;
  y: Bound;
}

export const ZERO_BOUND = { lower: 0, upper: 0 };
export const INFINITE_BOUND = { lower: -Infinity, upper: Infinity };
export const DECIMAL_BOUND = { lower: 0, upper: 1 };
export const CLIP_BOUND = { lower: -1, upper: 1 };
export const ZERO_XY_BOUND = { x: ZERO_BOUND, y: ZERO_BOUND };
export const INFINITE_XY_BOUND = { x: INFINITE_BOUND, y: INFINITE_BOUND };
export const DECIMAL_XY_BOUND = { x: DECIMAL_BOUND, y: DECIMAL_BOUND };
export const CLIP_XY_BOUND = { x: CLIP_BOUND, y: CLIP_BOUND };

export const isBound = (v: any): v is Bound =>
  typeof v === "object" && "lower" in v && "upper" in v;

export const makeValidBound = (bound: Bound): Bound =>
  bound.lower > bound.upper ? { lower: bound.upper, upper: bound.lower } : bound;

export const bound = (v1: number | Bound, v2?: number): Bound => {
  if (isBound(v1)) return makeValidBound(v1);
  if (typeof v1 === "number") {
    if (v2 != null) return { lower: v1, upper: v2 };
    return { lower: 0, upper: v1 };
  }
  throw new Error("Invalid bound");
};

export const inBounds = (v: number, bound: Bound): boolean =>
  v >= bound.lower && v <= bound.upper;

export const dimInBounds = (dim: number, bound: Bound): boolean =>
  bound.upper - bound.lower >= dim;

export const isZeroBound = (bound: Bound): boolean =>
  bound.lower === 0 && bound.upper === 0;

export const DECIMAL_COORD_ROOT: Corner = "bottomLeft";

export const cornerLocations = (corner: Corner): [XLocation, YLocation] =>
  CORNER_LOCATIONS[corner];

const CORNER_LOCATIONS: Record<Corner, [XLocation, YLocation]> = {
  topLeft: ["left", "top"],
  topRight: ["right", "top"],
  bottomLeft: ["left", "bottom"],
  bottomRight: ["right", "bottom"],
};

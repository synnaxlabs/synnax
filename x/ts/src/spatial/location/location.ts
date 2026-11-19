// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Alignment,
  type AngularDirection,
  type CornerLocation,
  cornerLocationZ,
  type Direction,
  type Location,
  type LocationXY,
  locationXYZ,
  type Order,
  OUTER_LOCATIONS,
  type OuterLocation,
  outerLocationZ,
  X_LOCATIONS,
  type XLocation,
  xLocationZ,
  Y_LOCATIONS,
  type YLocation,
} from "@/spatial/base";

export {
  type Alignment,
  type Location,
  type Order,
  OUTER_LOCATIONS,
  X_LOCATIONS,
  Y_LOCATIONS,
};

/** Zod schema for {@link X}. */
export const xZ = xLocationZ;
/** Zod schema for {@link Outer}. */
export const outerZ = outerLocationZ;
/** Zod schema for {@link Corner}. */
export const cornerZ = cornerLocationZ;

/** A location on the horizontal axis: "left" or "right". */
export type X = XLocation;
/** A location on the vertical axis: "top" or "bottom". */
export type Y = YLocation;
/** Any edge: "top", "right", "bottom", or "left". */
export type Outer = OuterLocation;
/** One corner, as its two edges. */
export type Corner = CornerLocation;

const OUTER_SET = new Set<string>(OUTER_LOCATIONS);

const SWAPPED: Record<Location, Location> = {
  top: "bottom",
  right: "left",
  bottom: "top",
  left: "right",
  center: "center",
};

const ROTATIONS: Record<Outer, Record<AngularDirection, Outer>> = {
  top: { clockwise: "left", counterclockwise: "right" },
  right: { clockwise: "top", counterclockwise: "bottom" },
  bottom: { clockwise: "right", counterclockwise: "left" },
  left: { clockwise: "bottom", counterclockwise: "top" },
};
/** A location, or a direction standing for its first location. */
export type Crude = Direction | Location;

/** Resolves a {@link Crude} to a location. "x" becomes "left" and "y" becomes "top". */
export function construct(cl: Direction | Outer): Outer;
export function construct(cl: Crude): Location;
export function construct(cl: Crude): Location {
  if (cl === "x") return "left";
  if (cl === "y") return "top";
  return cl;
}

/** @returns the location opposite the given one. "center" is its own opposite. */
export function swap(cl: Direction | Outer): Outer;
export function swap(cl: Crude): Location;
export function swap(cl: Crude): Location {
  return SWAPPED[construct(cl)];
}

/** @returns the edge a quarter turn from the given one, in the given direction. */
export const rotate = (loc: Outer, dir: AngularDirection): Outer => ROTATIONS[loc][dir];

const AXIS_SWAPPED: Record<Outer, Outer> = {
  top: "left",
  left: "top",
  right: "bottom",
  bottom: "right",
};

/**
 * @returns the edge on the other axis facing the same way: right pairs with bottom, and
 * left with top. Rotating a shape carries its edges across with this.
 */
export const swapAxis = (loc: Outer): Outer => AXIS_SWAPPED[loc];

/** @returns the axis a location sits on. */
export const direction = (cl: Crude): Direction => {
  const l = construct(cl);
  if (l === "top" || l === "bottom") return "y";
  return "x";
};

/** Zod schema for {@link XY}. */
export const xy = locationXYZ;
/** A location on both axes, naming a corner, an edge midpoint, or the center. */
export type XY = LocationXY;

/** The nine {@link XY} locations, each frozen. */
export const TOP_LEFT: Corner = Object.freeze({ x: "left", y: "top" });
export const TOP_RIGHT: Corner = Object.freeze({ x: "right", y: "top" });
export const BOTTOM_LEFT: Corner = Object.freeze({ x: "left", y: "bottom" });
export const BOTTOM_RIGHT: Corner = Object.freeze({ x: "right", y: "bottom" });
export const CENTER: XY = Object.freeze({ x: "center", y: "center" });
export const TOP_CENTER: XY = Object.freeze({ x: "center", y: "top" });
export const BOTTOM_CENTER: XY = Object.freeze({ x: "center", y: "bottom" });
export const CENTER_RIGHT: XY = Object.freeze({ x: "right", y: "center" });
export const CENTER_LEFT: XY = Object.freeze({ x: "left", y: "center" });
/** Every {@link XY} location, edges before corners before the center. */
export const XY_LOCATIONS: readonly XY[] = Object.freeze([
  CENTER_LEFT,
  CENTER_RIGHT,
  TOP_CENTER,
  BOTTOM_CENTER,
  TOP_LEFT,
  TOP_RIGHT,
  BOTTOM_LEFT,
  BOTTOM_RIGHT,
  CENTER,
]);

/** @returns whether the two XY locations are the same. */
export const xyEquals = (a: XY, b: XY): boolean => a.x === b.x && a.y === b.y;

/**
 * @returns whether the XY location fits the pattern: every named axis of a partial XY,
 * or either axis of a bare location.
 */
export const xyMatches = (a: XY, l: Partial<XY> | Location): boolean => {
  if (typeof l === "object") {
    let ok = true;
    if ("x" in l) {
      const ok_ = a.x === l.x;
      if (!ok_) ok = false;
    }
    if ("y" in l) {
      const ok_ = a.y === l.y;
      if (!ok_) ok = false;
    }
    return ok;
  }
  return a.x === l || a.y === l;
};

/** @returns whether the value names an edge. */
export const isOuter = (v: string): v is Outer => OUTER_SET.has(v);

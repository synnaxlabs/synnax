// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { caseconv } from "@/caseconv";
import {
  type AngularDirection,
  CENTER_LOCATIONS,
  type CenterLocation,
  centerLocationZ,
  type CornerLocation,
  cornerLocationZ,
  type Direction,
  directionZ,
  type Location,
  LOCATIONS,
  locationZ,
  OUTER_LOCATIONS,
  type OuterLocation,
  outerLocationZ,
  X_LOCATIONS,
  type XLocation,
  xLocationZ,
  Y_LOCATIONS,
  type YLocation,
  yLocationZ,
} from "@/spatial/base";

export {
  CENTER_LOCATIONS,
  type Location,
  LOCATIONS,
  locationZ,
  OUTER_LOCATIONS,
  X_LOCATIONS,
  Y_LOCATIONS,
};

/** Zod schema for {@link X}. */
export const xZ = xLocationZ;
/** Zod schema for {@link Y}. */
export const yZ = yLocationZ;
/** Zod schema for {@link Center}. */
export const centerZ = centerLocationZ;
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
/** The "center" location. */
export type Center = CenterLocation;
/** One corner, as its two edges. */
export type Corner = CornerLocation;
/** One corner, as a single camel-case word. */
export type CornerString = "topLeft" | "topRight" | "bottomLeft" | "bottomRight";

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
/** Zod schema for {@link Crude}. */
export const crudeZ = z.union([
  directionZ,
  z.enum([...OUTER_LOCATIONS, ...CENTER_LOCATIONS]),
]);
/** A location, or a direction standing for its first location. */
export type Crude = z.infer<typeof crudeZ>;

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

/** @returns the axis a location sits on. */
export const direction = (cl: Crude): Direction => {
  const l = construct(cl);
  if (l === "top" || l === "bottom") return "y";
  return "x";
};

/** Zod schema for {@link XY}. */
export const xy = z.object({
  x: xLocationZ.or(centerLocationZ),
  y: yLocationZ.or(centerLocationZ),
});
/** A location on both axes, naming a corner, an edge midpoint, or the center. */
export type XY = z.infer<typeof xy>;

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

/** @returns the XY location's two axes as a tuple. */
export const xyCouple = (a: XY): [Location, Location] => [a.x, a.y];

/** @returns whether the value names an edge. */
export const isOuter = (v: string): v is Outer => OUTER_SET.has(v);

/** @returns whether the location sits on the horizontal axis. */
export const isX = (a: Crude): a is XLocation | CenterLocation =>
  direction(construct(a)) === "x";

/** @returns whether the location sits on the vertical axis. */
export const isY = (a: Crude): a is YLocation => direction(construct(a)) === "y";

/** Renders an XY location as a {@link CornerString}, e.g. "topLeft". */
export const xyToString = (a: XY): string => `${a.x}${caseconv.capitalize(a.y)}`;

/** Builds an {@link XY} from an XY, two locations, or one location used for both. */
export const constructXY = (x: Crude | XY, y?: Crude): XY => {
  let parsedX: Location;
  let parsedY: Location;
  if (typeof x === "object" && "x" in x) {
    parsedX = x.x;
    parsedY = x.y;
  } else {
    parsedX = construct(x);
    parsedY = construct(y ?? x);
  }
  if (
    direction(parsedX) === direction(parsedY) &&
    parsedX !== "center" &&
    parsedY !== "center"
  )
    throw new Error(
      `[XYLocation] - encountered two locations with the same direction: ${parsedX.toString()} - ${parsedY.toString()}`,
    );
  const xy = { ...CENTER };
  if (parsedX !== "center")
    if (isX(parsedX)) xy.x = parsedX;
    else xy.y = parsedX;

  if (parsedY !== "center")
    if (isX(parsedY)) xy.x = parsedY;
    else xy.y = parsedY;

  return xy;
};

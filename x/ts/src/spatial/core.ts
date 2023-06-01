// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const CORNER_LOCATIONS: Record<CornerT, [XLocationT, YLocationT]> = {
  topLeft: ["left", "top"],
  topRight: ["right", "top"],
  bottomLeft: ["left", "bottom"],
  bottomRight: ["right", "bottom"],
};

// Options

const DIRECTIONS = ["x", "y"] as const;
export const Y_LOCATIONS = ["top", "bottom"] as const;
const X_LOCATIONS = ["left", "right"] as const;
const CENTER_LOCATION = "center";
export const POSITIONS = ["start", "center", "end"] as const;
export const ORDERS = ["first", "last"] as const;
export const CORNERS = ["topLeft", "topRight", "bottomLeft", "bottomRight"] as const;
const OUTER_LOCATIONS = [...Y_LOCATIONS, ...X_LOCATIONS] as const;
export const LOCATIONS = [...OUTER_LOCATIONS, "center"] as const;

// Strict definitions

const couple = z.tuple([z.number(), z.number()]);
const direction = z.enum(["x", "y"]);
const yDirection = z.literal("y");
const xDirection = z.literal("x");
const yLocation = z.enum(Y_LOCATIONS);
const xLocation = z.enum(X_LOCATIONS);
const centerLocation = z.literal(CENTER_LOCATION);
const outerLocation = z.enum(OUTER_LOCATIONS);
const location = z.enum(LOCATIONS);
const xy = z.object({ x: z.number(), y: z.number() });
const clientXY = z.object({ clientX: z.number(), clientY: z.number() });
const dimensions = z.object({ width: z.number(), height: z.number() });
const signedDimensions = z.object({
  signedWidth: z.number(),
  signedHeight: z.number(),
});
const position = z.enum(POSITIONS);
const order = z.enum(ORDERS);
const corner = z.enum(CORNERS);
const transform = z.object({ offset: z.number(), scale: z.number() });
export const xyTransform = z.object({ offset: xy, scale: xy });

// Loose definitions

const looseDirection = z.union([direction, location]);
const looseYLocation = z.union([yLocation, yDirection]);
const looseXLocation = z.union([xLocation, xDirection]);
const looseOuterLocation = z.union([outerLocation, direction]);
const looseXY = z.union([xy, dimensions, signedDimensions, couple]);
const looseLocation = looseDirection;
const strictBoundZ = z.object({ lower: z.number(), upper: z.number() });
const looseBoundZ = z.union([strictBoundZ, couple]);
const looseXYTransform = z.object({ offset: looseXY, scale: looseXY });
const looseDimensions = z.union([dimensions, signedDimensions, xy, couple]);

// Type exports

export type Couple = z.infer<typeof couple>;
export type YLocationT = z.infer<typeof yLocation>;
export type XLocationT = z.infer<typeof xLocation>;
export type OuterLocationT = z.infer<typeof outerLocation>;
export type CenterLocationT = typeof CENTER_LOCATION;
export type LocationT = z.infer<typeof location>;
export type DirectionT = z.infer<typeof direction>;
export type XYT = z.infer<typeof xy>;
export type DimensionsT = z.infer<typeof dimensions>;
export type SignedDimensionsT = z.infer<typeof signedDimensions>;
export type BoundT = z.input<typeof looseBoundZ>;
export type PositionT = z.infer<typeof position>;
export type OrderT = z.infer<typeof order>;
export type CornerT = z.infer<typeof corner>;
export type XYTransformT = z.infer<typeof xyTransform>;
export type ClientXYT = z.infer<typeof clientXY>;
export type TransformT = z.infer<typeof transform>;

export type LooseXYT = z.input<typeof looseXY> | XY;
export type LooseYLocationT = z.infer<typeof looseYLocation>;
export type LooseXLocationT = z.infer<typeof looseXLocation>;
export type LooseXYTransformT = z.infer<typeof looseXYTransform>;
export type LooseDirectionT = z.infer<typeof looseDirection> | Direction;
export type LooseLocationT = z.infer<typeof looseLocation> | Location;
export type LooseOuterLocation = z.infer<typeof looseOuterLocation>;
export type LooseDimensionsT = z.infer<typeof looseDimensions>;

export const DECIMAL_COORD_ROOT: CornerT = "bottomLeft";

export const cornerLocations = (corner: CornerT): [XLocationT, YLocationT] =>
  CORNER_LOCATIONS[corner];

export class Direction extends String {
  constructor(direction: LooseDirectionT) {
    if (DIRECTIONS.includes(direction as DirectionT)) super(direction);
    else if (Y_LOCATIONS.includes(direction as YLocationT)) super("y");
    else super("x");
  }

  equals(other: LooseDirectionT): boolean {
    const o = new Direction(other);
    return this.valueOf() === o.valueOf();
  }

  /** @returns "x" if the direction is "y" and "y" if the direction is "x" */
  get inverse(): Direction {
    return new Direction(this.valueOf() === "x" ? "y" : "x");
  }

  /** @returns "top" if the direction is "y" and "left" if the direction is "x" */
  get location(): Location {
    return new Location(this.valueOf() as DirectionT);
  }

  static readonly DIRECTIONS = DIRECTIONS;

  static get x(): Direction {
    return new Direction("x");
  }

  static get y(): Direction {
    return new Direction("y");
  }
}

export class Location extends String {
  constructor(location: LooseLocationT) {
    if (!Direction.DIRECTIONS.includes(location as DirectionT)) super(location);
    else if (location === "x") super("left");
    else super("top");
  }

  equals(other: LooseLocationT): boolean {
    const o = new Location(other);
    return this.valueOf() === o.valueOf();
  }

  get v(): LocationT {
    return this.valueOf() as LocationT;
  }

  get inverse(): Location {
    return new Location(Location.SWAPPED[this.valueOf() as LocationT]);
  }

  get direction(): Direction {
    return new Direction(this.valueOf() as DirectionT);
  }

  get dimension(): "width" | "height" {
    return this.valueOf() === "x" ? "width" : "height";
  }

  static readonly X = X_LOCATIONS;
  static readonly xz = xLocation;

  static readonly Y = Y_LOCATIONS;
  static readonly yz = yLocation;

  static readonly OUTER = OUTER_LOCATIONS;
  static readonly outerZ = outerLocation;

  static readonly CENTER = CENTER_LOCATION;
  static readonly centerZ = centerLocation;

  static readonly cornerZ = corner;

  private static readonly SWAPPED: Record<LocationT, LocationT> = {
    top: "bottom",
    bottom: "top",
    left: "right",
    right: "left",
    center: "center",
  };
}

/**
 * A point in 2D space.
 */
export class XY {
  readonly x: number;
  readonly y: number;

  /**
   * @constructor
   * @param x - The x coordinate OR an object or array that can be parsed into an XY.
   * @param y - An optional y coordinate that is only used if a numeric x coordinate
   * is provided. If x is numeric and y is not provided, y will be set to x.
   */
  constructor(x: number | LooseXYT, y?: number) {
    if (typeof x === "number") {
      this.x = x;
      this.y = y ?? x;
    } else if (Array.isArray(x)) {
      this.x = x[0];
      this.y = x[1];
    } else if ("signedWidth" in x) {
      this.x = x.signedWidth;
      this.y = x.signedHeight;
    } else if ("width" in x) {
      this.x = x.width;
      this.y = x.height;
    } else {
      this.x = x.x;
      this.y = x.y;
    }
  }

  /** @returns an x and y coordinate of zero */
  static get zero(): XY {
    return new XY(0, 0);
  }

  /** @returns an x and y coordinate of one */
  static get one(): XY {
    return new XY(1, 1);
  }

  /** @returns an x and y coordinate of infinity */
  static get infinite(): XY {
    return new XY(Infinity, Infinity);
  }

  /** @returns an XY coordinate translated by the given x value */
  translateX(x: number): XY {
    return new XY(this.x + x, this.y);
  }

  /** @returns an XY coordinate translated by the given y value */
  translateY(y: number): XY {
    return new XY(this.x, this.y + y);
  }

  /** @returns an XY coordinate translated by the given x and y values */
  translate(xy: XY | LooseXYT): XY {
    const t = new XY(xy);
    return new XY(this.x + t.x, this.y + t.y);
  }

  equals(other: XY | LooseXYT): boolean {
    const o = new XY(other);
    return this.x === o.x && this.y === o.y;
  }

  get couple(): Couple {
    return [this.x, this.y];
  }

  /**
   * z is a zod schema for parsing an XY. This schema is loose in that it will
   * accept and convert a variety of inputs into an XY. If you only want to accept
   * strict XYs, use z.
   */
  static readonly looseZ = looseXY.transform((v) => new XY(v));

  /**
   * z is a zod schema for parsing an XY. This schema is strict in that it will
   * only accept an XY as an input.
   */
  static readonly z = xy.transform((v) => new XY(v));
}

export class Dimensions {
  readonly width: number;
  readonly height: number;

  constructor(width: number | LooseDimensionsT, height?: number) {
    if (typeof width === "number") {
      this.width = width;
      this.height = height ?? width;
    } else if (Array.isArray(width)) {
      [this.width, this.height] = width;
    } else if ("x" in width) {
      this.width = width.x;
      this.height = width.y;
    } else if ("signedWidth" in width) {
      this.width = width.signedWidth;
      this.height = width.signedHeight;
    } else {
      this.width = width.width;
      this.height = width.height;
    }
  }

  static get zero(): Dimensions {
    return new Dimensions(0, 0);
  }

  static get one(): Dimensions {
    return new Dimensions(1, 1);
  }

  static get infinite(): Dimensions {
    return new Dimensions(Infinity, Infinity);
  }
}

/**
 * A lower and upper bound of values, where the lower bound is inclusive
 * and the upper bound is exclusive.
 */
export class Bound {
  readonly lower: number;
  readonly upper: number;

  /**
   * @constructor
   * @param lower - The lower bound OR an object or array that can be parsed into a bound.
   * @param upper - An optional upper bound that is only used if a numeric lower bound is provided.
   * If lower is numeric and upper is not provided, upper will be set to lower.
   *
   * The constructor does NOT validate that the lower bound is less than the upper bound, so its
   * possible to create an inverted bound.
   */
  constructor(lower: number | BoundT, upper?: number) {
    if (typeof lower === "number") {
      this.lower = lower;
      this.upper = upper ?? lower;
    } else if (Array.isArray(lower)) {
      [this.lower, this.upper] = lower;
    } else {
      this.lower = lower.lower;
      this.upper = lower.upper;
    }
  }

  /** @returns an upper and lower bound of zero */
  static get zero(): Bound {
    return new Bound(0);
  }

  /**
   * @returns an upper and lower bound of negative and positive infinity.
   * An infinite bound contains all values except for positive infinity.
   */
  static get infinite(): Bound {
    return new Bound(-Infinity, Infinity);
  }

  /** @returns a lower bound of zero and an upper bound of one */
  static get decimal(): Bound {
    return new Bound(0, 1);
  }

  /** @returns clip space with a lower bound of -1 and an upper bound of 1 */
  static get clip(): Bound {
    return new Bound(-1, 1);
  }

  /**
   * @returns true if the bound contains the value. Note that this bound
   * is lower inclusive and upper exclusive.
   * */
  contains(v: number): boolean {
    return v >= this.lower && v < this.upper;
  }

  /** @returns a number representing the distance between the upper and lower bounds */
  span(): number {
    return this.upper - this.lower;
  }

  /** @returns true if both the upper and lower bounds are zero */
  get isZero(): boolean {
    return this.lower === 0 && this.upper === 0;
  }

  /** @returns true if the span of the bound is zero */
  get spanIsZero(): boolean {
    return this.span() === 0;
  }

  /**
   * Finds the combination of upper and lower bounds from the given set that result
   * in the bound with the maximum possible span.
   * */
  static max(bounds: Bound[]): Bound {
    return new Bound({
      lower: Math.max(...bounds.map((b) => b.upper)),
      upper: Math.max(...bounds.map((b) => b.lower)),
    });
  }

  /**
   * Finds the combination of upper and lower bounds from the given set that result
   * in the bound with the minimum possible span.
   * */
  static min(bounds: Bound[]): Bound {
    return new Bound({
      lower: Math.min(...bounds.map((b) => b.lower)),
      upper: Math.min(...bounds.map((b) => b.upper)),
    });
  }

  /**
   * z is a zod schema for parsing a bound. This schema is loose in that it will
   * accept and convert a variety of inputs into a bound. If you only want to accept
   * strict bounds, use strictZ.
   */
  static readonly z = looseBoundZ.transform((v) => new Bound(v));

  /**
   * strictZ is a zod schema for parsing a bound. This schema is strict in that it will
   * only accept a bound as an input.
   * */
  static readonly strictZ = strictBoundZ.transform((v) => new Bound(v));
}

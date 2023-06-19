// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const CORNER_LOCATIONS: Record<Corner, [CrudeXLocation, CrudeYLocation]> = {
  topLeft: ["left", "top"],
  topRight: ["right", "top"],
  bottomLeft: ["left", "bottom"],
  bottomRight: ["right", "bottom"],
};

export const POSITIONS = ["start", "center", "end"] as const;
export const ORDERS = ["first", "last"] as const;
export const CORNERS = ["topLeft", "topRight", "bottomLeft", "bottomRight"] as const;
const DIRECTIONS = ["x", "y"] as const;
const Y_LOCATIONS = ["top", "bottom"] as const;
const X_LOCATIONS = ["left", "right"] as const;
const CENTER_LOCATION = "center";
const OUTER_LOCATIONS = [...Y_LOCATIONS, ...X_LOCATIONS] as const;
const LOCATIONS = [...OUTER_LOCATIONS, "center"] as const;

const stringValueOf = z.instanceof(String);
const numberCouple = z.tuple([z.number(), z.number()]);
const direction = z.enum(["x", "y"]);
const yDirection = z.literal("y");
const xDirection = z.literal("x");
const yLocation = z.enum(Y_LOCATIONS);
const xLocation = z.enum(X_LOCATIONS);
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
const boundsZ = z.object({ lower: z.number(), upper: z.number() });
const dimension = z.enum(["width", "height"]);

const looseDirection = z.union([direction, location, stringValueOf]);
const looseYLocation = z.union([yLocation, yDirection]);
const looseXLocation = z.union([xLocation, xDirection]);
const looseOuterLocation = z.union([outerLocation, direction]);
const looseXY = z.union([xy, clientXY, dimensions, signedDimensions, numberCouple]);
const looseLocation = looseDirection;
const looseBoundZ = z.union([boundsZ, numberCouple]);
const looseXYTransform = z.object({ offset: looseXY, scale: looseXY });
const looseDimensions = z.union([dimensions, signedDimensions, xy, numberCouple]);

export type NumberCouple = z.infer<typeof numberCouple>;
export type CrudeYLocation = z.infer<typeof yLocation>;
export type CrudeXLocation = z.infer<typeof xLocation>;
export type CrudeOuterLocation = z.infer<typeof outerLocation>;
export type CrudeCenterLocation = typeof CENTER_LOCATION;
export type CrudeLocation = z.infer<typeof location>;
export type CrudeDirection = z.infer<typeof direction>;
export type CrudeXY = z.infer<typeof xy>;
export type CrudeDimensions = z.infer<typeof dimensions>;
export type SignedDimensions = z.infer<typeof signedDimensions>;
export type CrudeBounds = z.input<typeof boundsZ>;
export type CrudePosition = z.infer<typeof position>;
export type CrudeOrder = z.infer<typeof order>;
export type Corner = z.infer<typeof corner>;
export type Dimension = z.infer<typeof dimension>;
export type XYTransformT = z.infer<typeof xyTransform>;
export type ClientXYT = z.infer<typeof clientXY>;
export type TransformT = z.infer<typeof transform>;

export type LooseXYT = z.input<typeof looseXY>;
export type LooseCrudeYLocation = z.infer<typeof looseYLocation>;
export type LooseCrudeXLocation = z.infer<typeof looseXLocation>;
export type LooseXYTransformT = z.infer<typeof looseXYTransform>;
export type LooseOuterLocation = z.infer<typeof looseOuterLocation>;
export type LooseDimensionsT = z.infer<typeof looseDimensions>;
export type LooseBoundT = z.infer<typeof looseBoundZ>;
/**
 * Location and Direction classes don't satisfy their primitive type interface,
 * so we need to include them in the loose type.
 */
export type LooseDirectionT = z.infer<typeof looseDirection> | Direction;
export type LooseLocationT = z.infer<typeof looseLocation> | Location;

export const cornerLocations = (corner: Corner): [CrudeXLocation, CrudeYLocation] =>
  CORNER_LOCATIONS[corner];

/**
 * A direction on the screen: "x" or "y".
 */
export class Direction extends String {
  constructor(direction: LooseDirectionT) {
    if (direction instanceof Direction) super(direction.valueOf());
    else if (DIRECTIONS.includes(direction as CrudeDirection)) super(direction);
    else if (Y_LOCATIONS.includes(direction as CrudeYLocation)) super("y");
    else super("x");
  }

  /** @returns the direction in its primitive form i.e. the string "x" or "y". */
  get crude(): CrudeDirection {
    return this.valueOf() as CrudeDirection;
  }

  /**
   * @returns true if the direction and provided direction are semantically
   * equal, converting the provided type to a direction if necessary.
   */
  equals(other: LooseDirectionT): boolean {
    const o = new Direction(other);
    return this.valueOf() === o.valueOf();
  }

  /** @returns "x" if the direction is "y" and "y" if the direction is "x". */
  get inverse(): Direction {
    return new Direction(this.equals("x") ? "y" : "x");
  }

  /** @returns "top" if the direction is "y" and "left" if the direction is "x". */
  get location(): Location {
    return new Location(this.valueOf() as CrudeDirection);
  }

  get dimension(): Dimension {
    return this.equals("x") ? "width" : "height";
  }

  /** The "x" direction. */
  static readonly x = new Direction("x");

  /** The "y" direction. */
  static readonly y = new Direction("y");

  /** A list of all the direction options. */
  static readonly DIRECTIONS = DIRECTIONS;

  /**
   * A Zod schema to parse a loose direction i.e any type that can be
   * converted to a direction.
   */
  static readonly looseZ = looseDirection.transform((v) => new Direction(v));

  /** A Zod schema to parse a strict direction i.e. either "x" or "y". */
  static readonly strictZ = direction.transform((v) => new Direction(v));

  /** Returns true if the provided value can be parsed as a direction. */
  static isValid(other: any): boolean {
    return Direction.looseZ.safeParse(other).success;
  }
}

/**
 * A general location of an element on the screen or regio * of the screen:
 * "top", "left", "right", "bottom", "center". For a type that represents a
 * specific position, see XY.
 */
export class Location extends String {
  constructor(location: LooseLocationT) {
    if (!Direction.DIRECTIONS.includes(location as CrudeDirection))
      super(location.valueOf());
    else if (location === "x") super("left");
    else super("top");
  }

  /**
   * @returns true if the location and provided location are semantically
   * equal, converting the provided type to a direction if necessary.
   */
  equals(other: LooseLocationT): boolean {
    const o = new Location(other);
    return this.valueOf() === o.valueOf();
  }

  /** @returns the value of a location as a primitive javascript scring. */
  get crude(): CrudeLocation {
    return this.valueOf() as CrudeLocation;
  }

  /**
   * @returns the semantic inverse of the location i.e. the inverse of "left"
   * is "right".
   */
  get inverse(): Location {
    return new Location(Location.SWAPPED[this.valueOf() as CrudeLocation]);
  }

  /**
   * @returns the direction best representing the location, where "top" and "bottom"
   * are "y" and "left" and "right" are "x". To get the inverse of this behavio, simply
   * call the "inverse" getter on the returned direction.
   */
  get direction(): Direction {
    return new Direction(this.crude as CrudeDirection);
  }

  /**
   * @returns true if the location is an outer location i.e. not "center".
   */
  get isOuter(): boolean {
    return OUTER_LOCATIONS.includes(this.crude as CrudeOuterLocation);
  }

  /**
   * @returns true if the location is an x location i.e. "left" or "right".
   */
  get isX(): boolean {
    return X_LOCATIONS.includes(this.crude as CrudeXLocation);
  }

  /**
   * @returns true if the location is a y location i.e. "top" or "bottom".
   */
  get isY(): boolean {
    return Y_LOCATIONS.includes(this.crude as CrudeYLocation);
  }

  /**
   * @returns true if the location is a center location i.e. "center".
   */
  get isCenter(): boolean {
    return this.crude === "center";
  }

  /** The "top" location. */
  static readonly top = new Location("top");

  /** The "bottom" location. */
  static readonly bottom = new Location("bottom");

  /** The "left" location. */
  static readonly left = new Location("left");

  /** The "right" location. */
  static readonly right = new Location("right");

  /** The "center" location. */
  static readonly center = new Location("center");

  /**
   * A list of all locations represented by the "x" direction i.e. "left" and "right".
   */
  static readonly X_LOCATIONS = X_LOCATIONS;

  /**
   * A list of all locations represented by the "y" direction i.e. "top" and "bottom".
   */
  static readonly Y_LOCATIONS = Y_LOCATIONS;

  /**
   * A list of all locations represented by the "x" and "y" directions i.e. "top",
   */
  static readonly OUTER = OUTER_LOCATIONS;

  /**
   * A zod schema to parse a strict location i.e. one of "top", "bottom", "left",
   * "right",
   */
  static readonly strictZ = location.transform((v) => new Location(v));

  /**
   * A zod schema to parse a loose location i.e. any type that can be converted
   * to a location.
   */
  static readonly looseZ = looseLocation.transform((v) => new Location(v));

  /**
   * A zod schema to parse a corner i.e. one of "topLeft", "topRight", "bottomLeft",
   */
  static readonly strictCornerZ = corner;

  private static readonly locationOrValue = z.union([
    z.instanceof(Location),
    stringValueOf,
  ]);

  /**
   * A zod schema to parse an X location i.e. one of "left" or "right".
   */
  static readonly strictXZ = xLocation
    .or(Location.locationOrValue)
    .transform((v) => new Location(v))
    .refine((l) => l.isX);

  /**
   * A zod schema to parse a Y location i.e. one of "top" or "bottom".
   */
  static readonly strictYZ = yLocation
    .or(Location.locationOrValue)
    .transform((v) => new Location(v))
    .refine((l) => l.isY);

  /**
   * A zod schema to parse an outer location i.e. one of "top", "bottom", "left",
   * "right".
   */
  static readonly strictOuterZ = outerLocation
    .or(Location.locationOrValue)
    .transform((v) => new Location(v))
    .refine((l) => l.isOuter);

  private static readonly SWAPPED: Record<CrudeLocation, CrudeLocation> = {
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
  /** The x coordinate. */
  readonly x: number;
  /** The y coordinate. */
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
    } else if ("clientX" in x) {
      this.x = x.clientX;
      this.y = x.clientY;
    } else if ("width" in x) {
      this.x = x.width;
      this.y = x.height;
    } else {
      this.x = x.x;
      this.y = x.y;
    }
  }

  /** @returns the XY in its crude form i.e {x: number, y: number} */
  get isZero(): boolean {
    return this.equals(XY.ZERO);
  }

  /** @returns the XY in its crude form i.e {x: number, y: number} */
  get crude(): CrudeXY {
    return { x: this.x, y: this.y };
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
  translate(xy: LooseXYT): XY {
    const t = new XY(xy);
    return new XY(this.x + t.x, this.y + t.y);
  }

  /** @returns true if the XY is semantically equal to the provided XY. */
  equals(other?: LooseXYT | number, y?: number): boolean {
    if (other == null) return false;
    const o = new XY(other, y);
    return this.x === o.x && this.y === o.y;
  }

  /**
   * @returns the XY represented as a couple, where the first item is the x coordinate,
   * and the second item is the y coordinate.
   */
  get couple(): NumberCouple {
    return [this.x, this.y];
  }

  /** An x and y coordinate of zero */
  static readonly ZERO = new XY(0, 0);

  /** An x and y coordinate of one */
  static readonly ONE = new XY(1, 1);

  /** An x and y coordinate of infinity */
  static readonly INFINITE = new XY(Infinity, Infinity);

  /**
   * A zod schema for parsing an XY. This schema is loose in that it will
   * accept and convert a variety of inputs into an XY. If you only want to accept
   * strict XYs, use z.
   */
  static readonly looseZ = looseXY.transform((v) => new XY(v));

  /**
   * A zod schema for parsing an XY. This schema is strict in that it will
   * only accept an XY as an input.
   */
  static readonly z = xy.transform((v) => new XY(v));
}

/**
 * A width and height in 2D space.
 */
export class Dimensions {
  /** The width. */
  readonly width: number;
  /** The height. */
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

  /** @returns the dimensions in its primitive form i.e {width: number, height: number} */
  get crude(): CrudeDimensions {
    return { width: this.width, height: this.height };
  }

  /** Dimensions with zero width and height. */
  static readonly ZERO = new Dimensions(0, 0);
  /** Dimensions with a width and height of 1. */
  static readonly DECIMAL = new Dimensions(1, 1);
  /** Dimensions with a width and height of infinity. */
  static readonly INFINITE = new Dimensions(Infinity, Infinity);

  /**
   * @returns true the dimensions and provided dimensions are semantically equal,
   * converting the provided type to dimensions if necessary.
   */
  equals(other?: LooseDimensionsT): boolean {
    if (other == null) return false;
    const o = new Dimensions(other);
    return this.width === o.width && this.height === o.height;
  }

  /**
   * @returns the dimensions as a couple, where the first item is the width and the second
   * is the height.
   */
  get couple(): NumberCouple {
    return [this.width, this.height];
  }
}

/**
 * A lower and upper bound of values, where the lower bound is inclusive
 * and the upper bound is exclusive.
 */
export class Bounds {
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
  constructor(lower: number | LooseBoundT, upper?: number) {
    if (typeof lower === "number") {
      if (upper != null) {
        this.lower = lower;
        this.upper = upper;
      } else {
        this.lower = 0;
        this.upper = lower;
      }
    } else if (Array.isArray(lower)) {
      [this.lower, this.upper] = lower;
    } else {
      this.lower = lower.lower;
      this.upper = lower.upper;
    }
    [this.lower, this.upper] = this.makeValid();
  }

  /** @returns the bound in its primitive form i.e {lower: number, upper: number} */
  get crude(): CrudeBounds {
    return { lower: this.lower, upper: this.upper };
  }

  private makeValid(): [number, number] {
    if (this.lower > this.upper) return [this.upper, this.lower];
    return [this.lower, this.upper];
  }

  /**
   * @returns true if the bound contains the value. Note that this bound
   * is lower inclusive and upper exclusive.
   * */
  contains(v: number): boolean {
    return v >= this.lower && v < this.upper;
  }

  /** @returns a number representing the distance between the upper and lower bounds */
  get span(): number {
    return this.upper - this.lower;
  }

  /** @returns true if both the upper and lower bounds are zero */
  get isZero(): boolean {
    return this.lower === 0 && this.upper === 0;
  }

  /** @returns true if the span of the bound is zero */
  get spanIsZero(): boolean {
    return this.span === 0;
  }

  /** @returns true if both the lower and upper bounds are finite */
  get isFinite(): boolean {
    return isFinite(this.lower) && isFinite(this.upper);
  }

  /**
   * Finds the combination of upper and lower bounds from the given set that result
   * in the bound with the maximum possible span.
   * */
  static max(bounds: LooseBoundT[]): Bounds {
    const parsed = bounds.map((b) => new Bounds(b));
    return new Bounds({
      lower: Math.min(...parsed.map((b) => b.lower)),
      upper: Math.max(...parsed.map((b) => b.upper)),
    });
  }

  /**
   * Finds the combination of upper and lower bounds from the given set that result
   * in the bound with the minimum possible span.
   * */
  static min(bounds: LooseBoundT[]): Bounds {
    const parsed = bounds.map((b) => new Bounds(b));
    return new Bounds({
      lower: Math.max(...parsed.map((b) => b.lower)),
      upper: Math.min(...parsed.map((b) => b.upper)),
    });
  }

  /** An upper and lower bound of zero */
  static readonly ZERO = new Bounds(0, 0);

  /**
   * An upper and lower bound of negative and positive infinity.
   * An infinite bound contains all values except for positive infinity.
   */
  static readonly INFINITE = new Bounds(-Infinity, Infinity);

  /** A lower bound of zero and an upper bound of one */
  static readonly DECIMAL = new Bounds(1);

  /** @returns clip space with a lower bound of -1 and an upper bound of 1 */
  static readonly CLIP = new Bounds(-1, 1);

  /**
   * z is a zod schema for parsing a bound. This schema is loose in that it will
   * accept and convert a variety of inputs into a bound. If you only want to accept
   * strict bounds, use strictZ.
   */
  static readonly looseZ = looseBoundZ.transform((v) => new Bounds(v));

  /**
   * strictZ is a zod schema for parsing a bound. This schema is strict in that it will
   * only accept a bound as an input.
   * */
  static readonly strictZ = boundsZ.transform((v) => new Bounds(v));
}

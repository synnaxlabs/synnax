// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const POSITIONS = ["start", "center", "end"] as const;
export const ORDERS = ["first", "last"] as const;

const DIRECTIONS = ["x", "y"] as const;
const Y_LOCATIONS = ["top", "bottom"] as const;
const X_LOCATIONS = ["left", "right"] as const;
const CENTER_LOCATION = "center";
const OUTER_LOCATIONS = [...Y_LOCATIONS, ...X_LOCATIONS] as const;
const LOCATIONS = [...OUTER_LOCATIONS, "center"] as const;

const stringValueOf = z.instanceof(String);
const numberCouple = z.tuple([z.number(), z.number()]);
const crudeDirection = z.enum(["x", "y"]);
const crudeYDirection = z.literal("y");
const crudeXDirection = z.literal("x");
const crudeYLocation = z.enum(Y_LOCATIONS);
const crudeXLocation = z.enum(X_LOCATIONS);
const crudeCenterLocation = z.literal(CENTER_LOCATION);
const crudeOuterLocation = z.enum(OUTER_LOCATIONS);
const crudeLocation = z.enum(LOCATIONS);
const crudeXY = z.object({ x: z.number(), y: z.number() });
const crudeClientXY = z.object({ clientX: z.number(), clientY: z.number() });
const crudeDimensions = z.object({ width: z.number(), height: z.number() });
const crudeSignedDimensions = z.object({
  signedWidth: z.number(),
  signedHeight: z.number(),
});
const crudePosition = z.enum(POSITIONS);
export const crudeOrder = z.enum(ORDERS);
const crudeTransform = z.object({ offset: z.number(), scale: z.number() });
export const crudeXYTransform = z.object({ offset: crudeXY, scale: crudeXY });
const crudeBounds = z.object({ lower: z.number(), upper: z.number() });
const crudeDimension = z.enum(["width", "height"]);
const crudeSignedDimension = z.enum(["signedWidth", "signedHeight"]);
const crudeXYCornerLocation = z.object({
  x: crudeXLocation,
  y: crudeYLocation,
});
const crudeXYLocation = z.object({
  x: z.union([crudeXLocation, crudeCenterLocation]),
  y: z.union([crudeYLocation, crudeCenterLocation]),
});

const looseDirection = z.union([crudeDirection, crudeLocation, stringValueOf]);
const looseYLocation = z.union([crudeYLocation, crudeYDirection, stringValueOf]);
const looseXLocation = z.union([crudeXLocation, crudeXDirection, stringValueOf]);
const looseOuterLocation = z.union([looseXLocation, looseYLocation, crudeDirection]);
const looseXY = z.union([
  crudeXY,
  crudeClientXY,
  crudeDimensions,
  crudeSignedDimensions,
  numberCouple,
]);
const looseLocation = z.union([crudeLocation, looseOuterLocation, looseDirection]);
const looseBounds = z.union([crudeBounds, numberCouple]);
const looseXYTransform = z.object({ offset: looseXY, scale: looseXY });
const looseDimensions = z.union([
  crudeDimensions,
  crudeSignedDimensions,
  crudeXY,
  numberCouple,
]);
const looseXYCornerLocation = z.object({
  x: looseXLocation,
  y: looseYLocation,
});
const looseXYLocation = z.object({
  x: z.union([looseXLocation, crudeCenterLocation]),
  y: z.union([looseYLocation, crudeCenterLocation]),
});

export type NumberCouple = z.infer<typeof numberCouple>;
export type CrudeYLocation = z.infer<typeof crudeYLocation>;
export type CrudeXLocation = z.infer<typeof crudeXLocation>;
export type CrudeOuterLocation = z.infer<typeof crudeOuterLocation>;
export type CrudeCenterLocation = typeof CENTER_LOCATION;
export type CrudeLocation = z.infer<typeof crudeLocation>;
export type CrudeDirection = z.infer<typeof crudeDirection>;
export type CrudeXY = z.infer<typeof crudeXY>;
export type CrudeDimensions = z.infer<typeof crudeDimensions>;
export type SignedDimensions = z.infer<typeof crudeSignedDimensions>;
export type CrudeBounds = z.input<typeof crudeBounds>;
export type CrudePosition = z.infer<typeof crudePosition>;
export type CrudeOrder = z.infer<typeof crudeOrder>;
export type Dimension = z.infer<typeof crudeDimension>;
export type SignedDimension = z.infer<typeof crudeSignedDimension>;
export type XYTransformT = z.infer<typeof crudeXYTransform>;
export type ClientXYT = z.infer<typeof crudeClientXY>;
export type TransformT = z.infer<typeof crudeTransform>;

export type LooseXYT = z.input<typeof looseXY>;
export type LooseCrudeYLocation = z.infer<typeof looseYLocation>;
export type LooseCrudeXLocation = z.infer<typeof looseXLocation>;
export type LooseXYTransformT = z.infer<typeof looseXYTransform>;
export type LooseOuterLocation = z.infer<typeof looseOuterLocation>;
export type LooseDimensionsT = z.infer<typeof looseDimensions>;
export type LooseBoundT = z.infer<typeof looseBounds>;
export type CrudeXYLocation = z.infer<typeof crudeXYLocation>;
export type CrudeCornerXYLocation = z.infer<typeof crudeXYCornerLocation>;

/**
 * Location and Direction classes don't satisfy their primitive type interface,
 * so we need to include them in the loose type.
 */
export type LooseDirectionT = z.infer<typeof looseDirection> | Direction;
export type LooseLocationT = z.infer<typeof looseLocation> | Location;
export type LooseXYLocation = z.infer<typeof looseXYLocation> | XYLocation;
export type LooseXYCornerLocation = z.infer<typeof looseXYCornerLocation> | XYLocation;

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
    return this.isX ? "width" : "height";
  }

  get signedDimension(): SignedDimension {
    return this.isX ? "signedWidth" : "signedHeight";
  }

  get isX(): boolean {
    return this.equals("x");
  }

  get isY(): boolean {
    return this.equals("y");
  }

  /** The "x" direction. */
  static readonly X = new Direction("x");

  /** The "y" direction. */
  static readonly Y = new Direction("y");

  /** A list of all the direction options. */
  static readonly DIRECTIONS = DIRECTIONS;

  /**
   * A Zod schema to parse a loose direction i.e any type that can be
   * converted to a direction.
   */
  static readonly looseZ = looseDirection.transform((v) => new Direction(v));

  /** A Zod schema to parse a strict direction i.e. either "x" or "y". */
  static readonly strictZ = crudeDirection.transform((v) => new Direction(v));

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

  static isValid(other: any): boolean {
    return Location.looseZ.safeParse(other).success;
  }

  /** The "top" location. */
  static readonly TOP = new Location("top");

  get isTop(): boolean {
    return this.equals("top");
  }

  /** The "bottom" location. */
  static readonly BOTTOM = new Location("bottom");

  get isBottom(): boolean {
    return this.equals("bottom");
  }

  /** The "left" location. */
  static readonly LEFT = new Location("left");

  get isLeft(): boolean {
    return this.equals("left");
  }

  /** The "right" location. */
  static readonly RIGHT = new Location("right");

  get isRight(): boolean {
    return this.equals("right");
  }

  /** The "center" location. */
  static readonly CENTER = new Location("center");

  get isCenter(): boolean {
    return this.equals("center");
  }

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
  static readonly strictZ = crudeLocation.transform((v) => new Location(v));

  /**
   * A zod schema to parse a loose location i.e. any type that can be converted
   * to a location.
   */
  static readonly looseZ = looseLocation.transform((v) => new Location(v));

  private static readonly locationOrValue = stringValueOf;
  //   // z.instanceof(Location),
  //   stringValueOf,
  // );

  /**
   * A zod schema to parse an X location i.e. one of "left" or "right".
   */
  static readonly strictXZ = crudeXLocation
    .or(Location.locationOrValue)
    .transform((v) => new Location(v))
    .refine((l) => l.isX);

  /**
   * A zod schema to parse a Y location i.e. one of "top" or "bottom".
   */
  static readonly strictYZ = crudeYLocation
    .or(Location.locationOrValue)
    .transform((v) => new Location(v))
    .refine((l) => l.isY);

  /**
   * A zod schema to parse an outer location i.e. one of "top", "bottom", "left",
   * "right".
   */
  static readonly strictOuterZ = crudeOuterLocation
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

export class XYLocation {
  x: Location;
  y: Location;

  constructor(
    x: LooseLocationT | LooseXYLocation | Location | XYLocation,
    y?: LooseLocationT,
  ) {
    let one: Location;
    let two: Location;
    if ((typeof x === "object" && "x" in x) || x instanceof XYLocation) {
      one = new Location(x.x);
      two = new Location(x.y);
    } else {
      one = new Location(x);
      two = new Location(y ?? x);
    }
    if (one.direction.equals(two.direction) && !one.isCenter && !two.isCenter)
      throw new Error(
        `[XYLocation] - encountered two locations with the same direction: ${one.toString()} - ${two.toString()}`,
      );
    if (one.isCenter) {
      if (two.isX) [this.x, this.y] = [two, one];
      else [this.x, this.y] = [one, two];
    } else if (two.isCenter) {
      if (one.isX) [this.x, this.y] = [one, two];
      else [this.x, this.y] = [two, one];
    } else if (one.isX) [this.x, this.y] = [one, two];
    else [this.x, this.y] = [two, one];
  }

  equals(other: LooseXYLocation): boolean {
    const o = new XYLocation(other);
    return this.x.equals(o.x) && this.y.equals(o.y);
  }

  toString(): string {
    return `${this.x.valueOf()}-${this.y.valueOf()}`;
  }

  get crude(): CrudeXYLocation {
    return { x: this.x.crude as CrudeXLocation, y: this.y.crude as CrudeYLocation };
  }

  static readonly TOP_LEFT = new XYLocation("left", "top");
  static readonly TOP_RIGHT = new XYLocation("right", "top");
  static readonly BOTTOM_LEFT = new XYLocation("left", "bottom");
  static readonly BOTTOM_RIGHT = new XYLocation("right", "bottom");
  static readonly CENTER = new XYLocation("center", "center");
  static readonly TOP_CENTER = new XYLocation("center", "top");
  static readonly BOTTOM_CENTER = new XYLocation("center", "bottom");
  static readonly LEFT_CENTER = new XYLocation("left", "center");
  static readonly RIGHT_CENTER = new XYLocation("right", "center");

  static readonly z = crudeXYLocation.transform((v) => new XYLocation(v));
  static readonly cornerZ = crudeXYCornerLocation.transform((v) => new XYLocation(v));
  static readonly looseCornerZ = looseXYCornerLocation
    .transform((v) => new XYLocation(v))
    .or(z.instanceof(XYLocation));
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

  /** scale */
  scale(scale: number): XY {
    return new XY(this.x * scale, this.y * scale);
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
  translate(x: LooseXYT | number | Direction, y?: number): XY {
    if (x instanceof Direction) {
      if (x.isX) return this.translateX(y ?? 0);
      return this.translateY(y ?? 0);
    }
    const t = new XY(x, y);
    return new XY(this.x + t.x, this.y + t.y);
  }

  set(direction: Direction, value: number): XY {
    if (direction.isX) return new XY(value, this.y);
    return new XY(this.x, value);
  }

  /** @returns true if the XY is semantically equal to the provided XY. */
  equals(other?: LooseXYT | number, y?: number): boolean {
    if (other == null) return false;
    const o = new XY(other, y);
    return this.x === o.x && this.y === o.y;
  }

  /** @returns the distance between the point and given point */
  distanceTo(other: LooseXYT): number {
    const o = new XY(other);
    return Math.sqrt(Math.pow(this.x - o.x, 2) + Math.pow(this.y - o.y, 2));
  }

  /** @returns the distance between the x coordinates of the point and given point */
  xDistanceTo(other: LooseXYT): number {
    const o = new XY(other);
    return Math.abs(this.x - o.x);
  }

  /** @returns the distance beween the y coordinates of the point and given point */
  yDistanceTo(other: LooseXYT): number {
    const o = new XY(other);
    return Math.abs(this.y - o.y);
  }

  /** @returns the translation necessary to get from the point to the given point */
  translation(other: LooseXYT): XY {
    const o = new XY(other);
    return new XY(o.x - this.x, o.y - this.y);
  }

  /**
   * @returns the XY represented as a couple, where the first item is the x coordinate,
   * and the second item is the y coordinate.
   */
  get couple(): NumberCouple {
    return [this.x, this.y];
  }

  /**
   * @returns the XY in css coordinate form i.e {left: number, top: number}.
   */
  get css(): { left: number; top: number } {
    return { left: this.x, top: this.y };
  }

  /**
   * @returns the XY in css percentage coordinate form, assuming the XY is a percentage
   * expressed as a decimal.
   */
  get percentCSS(): { left: string; top: string } {
    return { left: `${this.x * 100}%`, top: `${this.y * 100}%` };
  }

  /** @returns true if either the x or y coordinate is NaN */
  get isNan(): boolean {
    return isNaN(this.x) || isNaN(this.y);
  }

  /** An x and y coordinate of zero */
  static readonly ZERO = new XY(0, 0);

  /** An x and y coordinate of one */
  static readonly ONE = new XY(1, 1);

  /** An x and y coordinate of infinity */
  static readonly INFINITE = new XY(Infinity, Infinity);

  /** An x and y coordinate of NaN */
  static readonly NAN = new XY(NaN, NaN);

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
  static readonly z = crudeXY.transform((v) => new XY(v));
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

  pickGreatest(other: Dimensions): Dimensions {
    return new Dimensions({
      width: other.width > this.width ? other.width : this.width,
      height: other.height > this.height ? other.height : this.height,
    });
  }

  /**
   * @returns the dimensions as a couple, where the first item is the width and the second
   * is the height.
   */
  get couple(): NumberCouple {
    return [this.width, this.height];
  }

  /**
   * @returns the swapped dimensions i.e. the width and height are swapped.
   */
  swap(): Dimensions {
    return new Dimensions({ width: this.height, height: this.width });
  }

  svgViewBox(): string {
    return `0 0 ${this.width} ${this.height}`;
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

  clamp(v: number): number {
    if (v < this.lower) return this.lower;
    if (v >= this.upper) return this.upper - 1;
    return v;
  }

  /**
   * @returns true if the bound contains the value. Note that this bound
   * is lower inclusive and upper exclusive.
   * */
  contains(v: number): boolean {
    return v >= this.lower && v < this.upper;
  }

  overlapsWith(other: Bounds): boolean {
    return this.contains(other.lower) || this.contains(other.upper - 1);
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
  static readonly looseZ = looseBounds.transform((v) => new Bounds(v));

  /**
   * strictZ is a zod schema for parsing a bound. This schema is strict in that it will
   * only accept a bound as an input.
   * */
  static readonly strictZ = crudeBounds.transform((v) => new Bounds(v));
}

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import type * as bounds from "@/spatial/bounds/bounds";
import type * as dimensions from "@/spatial/dimensions/dimensions";
import * as direction from "@/spatial/direction/direction";
import * as location from "@/spatial/location/location";
import * as xy from "@/spatial/xy/xy";

const cssPos = z.union([z.number(), z.string()]);

export const cssBox = z.object({
  top: cssPos,
  left: cssPos,
  width: cssPos,
  height: cssPos,
});
export const domRect = z.object({
  left: z.number(),
  top: z.number(),
  right: z.number(),
  bottom: z.number(),
});
export const box = z.object({
  one: xy.xy,
  two: xy.xy,
  root: location.corner,
});

export type Box = z.infer<typeof box>;
export type CSS = z.infer<typeof cssBox>;
export type DOMRect = z.infer<typeof domRect>;

export type Crude = DOMRect | Box | { getBoundingClientRect: () => DOMRect };

/** A box centered at (0,0) with a width and height of 0. */
export const ZERO = { one: xy.ZERO, two: xy.ZERO, root: location.TOP_LEFT };

/**
 * A box centered at (0,0) with a width and height of 1, and rooted in the
 * bottom left. Note that pixel space is typically rooted in the top left.
 */
export const DECIMAL = { one: xy.ZERO, two: xy.ONE, root: location.BOTTOM_LEFT };

export const copy = (b: Box, root?: location.CornerXY): Box => ({
  one: b.one,
  two: b.two,
  root: root ?? b.root,
});

/**
 * Box represents a general box in 2D space. It typically represents a bounding box
 * for a DOM element, but can also represent a box in clip space or decimal space.
 *
 * It's important to note that the behavior of a Box varies depending on its coordinate
 * system.Make sure you're aware of which coordinate system you're using.
 *
 * Many of the properties and methods on a Box access the same semantic value. The
 * different accessors are there for ease of use and semantics.
 */
export const construct = (
  first: number | DOMRect | xy.XY | Box | { getBoundingClientRect: () => DOMRect },
  second?: number | xy.XY | dimensions.Dimensions | dimensions.Signed,
  width: number = 0,
  height: number = 0,
  coordinateRoot?: location.CornerXY,
): Box => {
  const b: Box = {
    one: { ...xy.ZERO },
    two: { ...xy.ZERO },
    root: coordinateRoot ?? location.TOP_LEFT,
  };

  if (typeof first === "number") {
    if (typeof second !== "number")
      throw new Error("Box constructor called with invalid arguments");
    b.one = { x: first, y: second };
    b.two = { x: b.one.x + width, y: b.one.y + height };
    return b;
  }

  if ("one" in first && "two" in first && "root" in first)
    return { ...first, root: coordinateRoot ?? first.root };

  if ("getBoundingClientRect" in first) first = first.getBoundingClientRect();
  if ("left" in first) {
    b.one = { x: first.left, y: first.top };
    b.two = { x: first.right, y: first.bottom };
    return b;
  }

  b.one = first;
  if (second == null) b.two = { x: b.one.x + width, y: b.one.y + height };
  else if (typeof second === "number")
    b.two = { x: b.one.x + second, y: b.one.y + width };
  else if ("width" in second)
    b.two = {
      x: b.one.x + second.width,
      y: b.one.y + second.height,
    };
  else if ("signedWidth" in second)
    b.two = {
      x: b.one.x + second.signedWidth,
      y: b.one.y + second.signedHeight,
    };
  else b.two = second;
  return b;
};

export interface Resize {
  /**
   * Sets the dimensions of the box to the given dimensions.
   * @example resize(b, { width: 10, height: 10 }) // Sets the box to a 10x10 box.
   */
  (b: Crude, dims: dimensions.Dimensions | dimensions.Signed): Box;
  /**
   * Sets the dimension along the given direction to the given amount.
   * @example resize(b, "x", 10) // Sets the width of the box to 10.
   * @example resize(b, "y", 10) // Sets the height of the box to 10.
   */
  (b: Crude, direction: direction.Direction, amount: number): Box;
}

export const resize: Resize = (
  b: Crude,
  dims: dimensions.Dimensions | dimensions.Signed | direction.Direction,
  amount?: number,
): Box => {
  const b_ = construct(b);
  if (typeof dims === "string") {
    if (amount == null) throw new Error("Invalid arguments for resize");
    const dir = direction.construct(dims);
    return construct(
      b_.one,
      undefined,
      dir === "x" ? amount : width(b_),
      dir === "y" ? amount : height(b_),
      b_.root,
    );
  }
  return construct(b_.one, dims, undefined, undefined, b_.root);
};

/**
 * Checks if a box contains a point or another box.
 *
 * @param container - The container box to check against.
 * @param value - The point or box to check if it is contained in the container.
 * @param inclusive - Whether the edges of the box are inclusive or exclusive.
 * @returns true if the box inclusively contains the point or box and false otherwise.
 */
export const contains = (
  container: Crude,
  value: Box | xy.XY,
  inclusive: boolean = true,
): boolean => {
  const b_ = construct(container);
  let comp = (a: number, b: number) => a < b;
  if (inclusive) comp = (a: number, b: number) => a <= b;
  if ("one" in value)
    return (
      comp(left(b_), left(value)) &&
      comp(right(value), right(b_)) &&
      comp(top(b_), top(value)) &&
      comp(bottom(value), bottom(b_))
    );
  return (
    comp(left(b_), value.x) &&
    comp(value.x, right(b_)) &&
    comp(top(b_), value.y) &&
    comp(value.y, bottom(b_))
  );
};

/**
 * @returns true if the given box is semantically equal to this box and false otherwise.
 */
export const equals = (a: Box, b: Box): boolean =>
  xy.equals(a.one, b.one) &&
  xy.equals(a.two, b.two) &&
  location.xyEquals(a.root, b.root);

/**
 * @returns the dimensions of the box. Note that these dimensions are guaranteed to
 * be positive. To get the signed dimensions, use the `signedDims` property.
 */
export const dims = (b: Box): dimensions.Dimensions => ({
  width: width(b),
  height: height(b),
});

/**
 * @returns the dimensions of the box. Note that these dimensions may be negative.
 * To get the unsigned dimensions, use the `dims` property.
 */
export const signedDims = (b: Box): dimensions.Signed => ({
  signedWidth: signedWidth(b),
  signedHeight: signedHeight(b),
});

/**
 * @returns the css representation of the box.
 */
export const css = (b: Box): CSS => ({
  top: top(b),
  left: left(b),
  width: width(b),
  height: height(b),
});

export const dim = (
  b: Crude,
  dir: direction.Crude,
  signed: boolean = false,
): number => {
  const dim: number =
    direction.construct(dir) === "y" ? signedHeight(b) : signedWidth(b);
  return signed ? dim : Math.abs(dim);
};

/** @returns the pont corresponding to the given corner of the box. */
export const xyLoc = (b: Crude, l: location.XY): xy.XY => {
  const b_ = construct(b);
  return {
    x: l.x === "center" ? center(b_).x : loc(b_, l.x),
    y: l.y === "center" ? center(b_).y : loc(b_, l.y),
  };
};

/**
 * @returns a one dimensional coordinate corresponding to the location of the given
 * side of the box i.e. the x coordinate of the left side, the y coordinate of the
 * top side, etc.
 */
export const loc = (b: Crude, loc: location.Location): number => {
  const b_ = construct(b);
  const f = location.xyCouple(b_.root).includes(loc) ? Math.min : Math.max;
  return location.X_LOCATIONS.includes(loc as location.X)
    ? f(b_.one.x, b_.two.x)
    : f(b_.one.y, b_.two.y);
};

/** @returns true if the area of the box is 0 and false otherwise. */
export const areaIsZero = (b: Box): boolean => area(b) === 0;

/** @returns the width of the box. */
export const width = (b: Crude): number => dim(b, "x");

/** @returns the height of the box. */
export const height = (b: Crude): number => dim(b, "y");

/**
 * @returns the signed width of the box, which will be negative if the x value of the
 * first coordinate is less than the x value of the second coordinate.
 */
export const signedWidth = (b: Crude): number => {
  const b_ = construct(b);
  return b_.two.x - b_.one.x;
};

/**
 * @returns the signed height of the box, which will be negative if the y value of the
 * first coordinate is less than the y value of the second coordinate.
 */
export const signedHeight = (b: Crude): number => {
  const b_ = construct(b);
  return b_.two.y - b_.one.y;
};

export const topLeft = (b: Crude): xy.XY => xyLoc(b, location.TOP_LEFT);

export const topCenter = (b: Crude): xy.XY => xyLoc(b, location.TOP_CENTER);

export const topRight = (b: Crude): xy.XY => xyLoc(b, location.TOP_RIGHT);

export const bottomLeft = (b: Crude): xy.XY => xyLoc(b, location.BOTTOM_LEFT);

export const bottomCenter = (b: Crude): xy.XY => xyLoc(b, location.BOTTOM_CENTER);

export const bottomRight = (b: Crude): xy.XY => xyLoc(b, location.BOTTOM_RIGHT);

export const centerLeft = (b: Crude): xy.XY => xyLoc(b, location.CENTER_LEFT);

export const centerRight = (b: Crude): xy.XY => xyLoc(b, location.CENTER_RIGHT);

export const right = (b: Crude): number => loc(b, "right");

export const bottom = (b: Crude): number => loc(b, "bottom");

export const left = (b: Crude): number => loc(b, "left");

export const top = (b: Crude): number => loc(b, "top");

export const center = (b: Crude): xy.XY =>
  xy.translate(topLeft(b), {
    x: signedWidth(b) / 2,
    y: signedHeight(b) / 2,
  });

export const x = (b: Crude): number => {
  const b_ = construct(b);
  return b_.root.x === "left" ? left(b_) : right(b_);
};

export const y = (b: Crude): number => {
  const b_ = construct(b);
  return b_.root.y === "top" ? top(b_) : bottom(b_);
};

export const root = (b: Crude): xy.XY => ({ x: x(b), y: y(b) });

/**
 * @returns the bounds of the box along the x axis i.e. if the box root is top left,
 * the lower bound is the x coordinate of the left side of the box and the upper bound
 * is the x coordinate of the right side of the box.
 * @param b - The box to get the bounds of.
 */
export const xBounds = (b: Crude): bounds.Bounds => {
  const b_ = construct(b);
  return { lower: b_.one.x, upper: b_.two.x };
};

/**
 * @returns the bounds of the box along the y axis i.e. if the box root is top left,
 * the lower bound is the y coordinate of the top side of the box and the upper bound
 * is the y coordinate of the bottom side of the box.
 * @param b - The box to get the bounds of.
 */
export const yBounds = (b: Crude): bounds.Bounds => {
  const b_ = construct(b);
  return { lower: b_.one.y, upper: b_.two.y };
};

export const reRoot = (b: Box, corner: location.CornerXY): Box => copy(b, corner);

export const edgePoints = (b: Crude, loc: location.Location): [xy.XY, xy.XY] => {
  const b_ = construct(b);
  const x = location.X_LOCATIONS.includes(loc as location.X)
    ? "x"
    : location.Y_LOCATIONS.includes(loc as location.Y)
      ? "y"
      : null;
  if (x === null) throw new Error(`Invalid location: ${loc}`);
  const f = loc === "top" || loc === "left" ? Math.min : Math.max;
  const one = { ...b_.one };
  const two = { ...b_.two };
  one[x] = f(b_.one[x], b_.two[x]);
  two[x] = f(b_.one[x], b_.two[x]);
  return [one, two];
};

/**
 * Reposition a box so that it is centered within a given bound.
 *
 * @param target The box to reposition - Only works if the root is topLeft
 * @param bound The box to reposition within - Only works if the root is topLeft
 * @returns the repositioned box
 */
export const positionInCenter = (target_: Crude, bound_: Crude): Box => {
  const target = construct(target_);
  const bound = construct(bound_);
  const x_ = x(bound) + (width(bound) - width(target)) / 2;
  const y_ = y(bound) + (height(bound) - height(target)) / 2;
  return construct({ x: x_, y: y_ }, dims(target));
};

/** */
export const isBox = (value: unknown): value is Box => {
  if (typeof value !== "object" || value == null) return false;
  return "one" in value && "two" in value && "root" in value;
};

export const aspect = (b: Box): number => width(b) / height(b);

interface Translate {
  /** @returns the box translated by the given coordinates. */
  (b: Crude, t: xy.Crude): Box;
  /** @returns the box translated in the given direction by the given amount. */
  (b: Crude, direction: direction.Direction, amount: number): Box;
}

export const translate: Translate = (
  b: Crude,
  t: xy.Crude | direction.Direction,
  amount?: number,
): Box => {
  if (typeof t === "string") {
    if (amount == null) throw new Error(`Undefined amount passed into box.translate`);
    const dir = direction.construct(t);
    t = xy.construct(dir, amount);
  }
  const b_ = construct(b);
  return construct(
    xy.translate(b_.one, t),
    xy.translate(b_.two, t),
    undefined,
    undefined,
    b_.root,
  );
};

/** @returns a box representing the intersection of the two given boxes. */
export const intersection = (a: Box, b: Box): Box => {
  const x = Math.max(left(a), left(b));
  const y = Math.max(top(a), top(b));
  const x2 = Math.min(right(a), right(b));
  const y2 = Math.min(bottom(a), bottom(b));
  if (x > x2 || y > y2) return ZERO;
  return construct({ x, y }, { x: x2, y: y2 }, undefined, undefined, a.root);
};

/** @returns the area of the box. */
export const area = (b: Box): number => width(b) * height(b);

/**
 * Truncates the coordinates of the box to the given precision.
 * @param b - The box to truncate.
 * @param precision - The number of decimal places to truncate to.
 * @returns the truncated box.
 */
export const truncate = (b: Crude, precision: number): Box => {
  const b_ = construct(b);
  return construct(
    xy.truncate(b_.one, precision),
    xy.truncate(b_.two, precision),
    undefined,
    undefined,
    b_.root,
  );
};

/**
 * Constructs a box from a particular corner, and then reinterprets the box in order
 * to define it from a different corner.
 *
 * @example
 * const b = box.construct(0, 0, 10, 10, location.BOTTOM_LEFT, location.TOP_LEFT);
 * // b is now a box rooted in the top left corner with it's first coordinate at
 * // (0, 10) and it's second coordinate at (10, 0).
 *
 * @param x - The x coordinate of the first point.
 * @param y - The y coordinate of the first point.
 * @param width - The width of the box.
 * @param height - The height of the box.
 * @param currRoot - The current root of the box i.e. the corner that x and y represent.
 * @param newRoot - The new root of the box i.e. the corner that the box should be
 * reinterpreted from.
 * @returns the box reinterpreted from the new root.
 */
export const constructWithAlternateRoot = (
  x: number,
  y: number,
  width: number,
  height: number,
  currRoot: location.XY,
  newRoot: location.CornerXY,
): Box => {
  const first = { x, y };
  const second = { x: x + width, y: y + height };
  if (currRoot.x !== newRoot.x)
    if (currRoot.x === "center") {
      first.x -= width / 2;
      second.x -= width / 2;
    } else {
      first.x -= width;
      second.x -= width;
    }
  if (currRoot.y !== newRoot.y)
    if (currRoot.y === "center") {
      first.y -= height / 2;
      second.y -= height / 2;
    } else {
      first.y -= height;
      second.y -= height;
    }
  return construct(first, second, undefined, undefined, newRoot);
};

export const round = (b: Crude): Box => {
  const b_ = construct(b);
  return construct(xy.round(b_.one), xy.round(b_.two), undefined, undefined, b_.root);
};

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { Stringer } from "@/primitive";
import {
  Dimensions,
  XY,
  SignedDimensions,
  Bounds,
  Location,
  CrudeXLocation,
  CrudeXY,
  LooseDirectionT,
  Direction,
  CrudeDimensions,
  CrudeLocation,
  CrudeCornerXYLocation,
  XYLocation,
  LooseXYLocation,
} from "@/spatial/core";

const cssPos = z.union([z.number(), z.string()]);

const cssBox = z.object({
  top: cssPos,
  left: cssPos,
  width: cssPos,
  height: cssPos,
});
const domRect = z.object({
  left: z.number(),
  top: z.number(),
  right: z.number(),
  bottom: z.number(),
});
const box = z.object({
  one: XY.z,
  two: XY.z,
  root: XYLocation.looseCornerZ,
});
const looseBox = z.union([
  cssBox,
  domRect,
  z.object({
    one: XY.looseZ,
    two: XY.looseZ,
    root: XYLocation.looseCornerZ,
  }),
]);

export type BoxT = z.infer<typeof box>;
export type LooseBoxT = z.infer<typeof looseBox>;
export type CSSBox = z.infer<typeof cssBox>;
export type DOMRect = z.infer<typeof domRect>;

/**
 * Box represents a general box in 2D space. It typically represents a bounding box
 * for a DOM element, but can also represent a box in clip space or decimal space.
 *
 * It'simportant to note that the behavior of a Box varies depending on its coordinate
 * system.Make sure you're aware of which coordinate system you're using.
 *
 * Many of the properties and methods on a Box access the same semantic value. The
 * different accessors are there for ease of use and semantics.
 */
export class Box implements Stringer {
  readonly one: XY;
  readonly two: XY;
  readonly root: XYLocation;

  readonly isBox: true = true;

  constructor(
    first:
      | number
      | DOMRect
      | CrudeXY
      | Box
      | { getBoundingClientRect: () => DOMRect }
      | BoxT,
    second?: number | CrudeXY | CrudeDimensions | SignedDimensions,
    width: number = 0,
    height: number = 0,
    coordinateRoot?: CrudeCornerXYLocation | XYLocation
  ) {
    if (first instanceof Box) {
      this.one = first.one;
      this.two = first.two;
      this.root = new XYLocation(coordinateRoot ?? first.root);
      return;
    }

    this.root = new XYLocation(coordinateRoot ?? XYLocation.TOP_LEFT);

    if (typeof first === "number") {
      if (typeof second !== "number")
        throw new Error("Box constructor called with invalid arguments");
      this.one = new XY({ x: first, y: second });
      this.two = new XY({ x: this.one.x + width, y: this.one.y + height });
      return;
    }

    if ("getBoundingClientRect" in first) first = first.getBoundingClientRect();
    if ("left" in first) {
      this.one = new XY({ x: first.left, y: first.top });
      this.two = new XY({ x: first.right, y: first.bottom });
      return;
    }

    if ("one" in first) {
      this.one = first.one;
      this.two = first.two;
      this.root = first.root;
      return;
    }

    this.one = new XY(first);
    if (second == null) {
      this.two = new XY({ x: this.one.x + width, y: this.one.y + height });
    } else if (typeof second === "number")
      this.two = new XY({
        x: this.one.x + second,
        y: this.one.y + width,
      });
    else if ("width" in second)
      this.two = new XY({
        x: this.one.x + second.width,
        y: this.one.y + second.height,
      });
    else if ("signedWidth" in second)
      this.two = new XY({
        x: this.one.x + second.signedWidth,
        y: this.one.y + second.signedHeight,
      });
    else this.two = new XY(second);
  }

  /**
   * Checks if a box contains a point or another box.
   *
   * @param value - The point or box to check.
   * @returns true if the box inclusively contains the point or box and false otherwise.
   */
  contains(value: Box | XY): boolean {
    if ("signedWidth" in value)
      return (
        value.left >= this.left &&
        value.right <= this.right &&
        value.top >= this.top &&
        value.bottom <= this.bottom
      );
    return (
      value.x >= this.left &&
      value.x <= this.right &&
      value.y >= this.top &&
      value.y <= this.bottom
    );
  }

  /**
   * @returns true if the given box is semantically equal to this box and false otherwise.
   */
  equals(box: Box): boolean {
    return (
      this.one.equals(box.one) && this.two.equals(box.two) && this.root.equals(box.root)
    );
  }

  /**
   * @returns the dimensions of the box. Note that these dimensions are guaranteed to
   * be positive. To get the signed dimensions, use the `signedDims` property.
   */
  get dims(): Dimensions {
    return new Dimensions({ width: this.width, height: this.height });
  }

  /**
   * @returns the dimensions of the box. Note that these dimensions may be negative.
   * To get the unsigned dimensions, use the `dims` property.
   */
  get signedDims(): SignedDimensions {
    return { signedWidth: this.signedWidth, signedHeight: this.signedHeight };
  }

  /**
   * @returns the css representation of the box.
   */
  get css(): CSSBox {
    return {
      top: this.top,
      left: this.left,
      width: this.width,
      height: this.height,
    };
  }

  dim(dir: LooseDirectionT, signed: boolean = false): number {
    const dir_ = new Direction(dir);
    const dim: number = dir_.valueOf() === "y" ? this.signedHeight : this.signedWidth;
    return signed ? dim : Math.abs(dim);
  }

  /** @returns the pont corresponding to the given corner of the box. */
  xyLoc(point: LooseXYLocation): XY {
    const loc = new XYLocation(point);
    return new XY({
      x: loc.x.isCenter ? this.center.x : this.loc(loc.x),
      y: loc.y.isCenter ? this.center.y : this.loc(loc.y),
    });
  }

  /**
   * @returns a one dimensional coordinate corresponding to the location of the given
   * side of the box i.e. the x coordinate of the left side, the y coordinate of the
   * top side, etc.
   */
  loc(loc: CrudeLocation | Location): number {
    const loc_ = new Location(loc);
    const f = this.root.toString().includes(loc_.crude) ? Math.min : Math.max;
    return Location.X_LOCATIONS.includes(loc_.crude as CrudeXLocation)
      ? f(this.one.x, this.two.x)
      : f(this.one.y, this.two.y);
  }

  locPoint(loc: CrudeLocation | Location): XY {
    const l = this.loc(loc);
    if (Location.X_LOCATIONS.includes(loc as CrudeXLocation))
      return new XY({ x: l, y: this.center.y });
    return new XY({ x: this.center.x, y: l });
  }

  get isZero(): boolean {
    return this.one.x === this.two.x && this.one.y === this.two.y;
  }

  get width(): number {
    return this.dim("x");
  }

  get height(): number {
    return this.dim("y");
  }

  get signedWidth(): number {
    return this.two.x - this.one.x;
  }

  get signedHeight(): number {
    return this.two.y - this.one.y;
  }

  get topLeft(): XY {
    return this.xyLoc(XYLocation.TOP_LEFT);
  }

  get topRight(): XY {
    return this.xyLoc(XYLocation.TOP_RIGHT);
  }

  get bottomLeft(): XY {
    return this.xyLoc(XYLocation.BOTTOM_LEFT);
  }

  get bottomRight(): XY {
    return this.xyLoc(XYLocation.BOTTOM_RIGHT);
  }

  get right(): number {
    return this.loc("right");
  }

  get bottom(): number {
    return this.loc("bottom");
  }

  get left(): number {
    return this.loc("left");
  }

  get top(): number {
    return this.loc("top");
  }

  get center(): XY {
    return this.topLeft.translate({
      x: this.signedWidth / 2,
      y: this.signedHeight / 2,
    });
  }

  get x(): number {
    return this.root.x.equals("left") ? this.left : this.right;
  }

  get y(): number {
    return this.root.y.equals("top") ? this.top : this.bottom;
  }

  get xBounds(): Bounds {
    return new Bounds(this.one.x, this.two.x);
  }

  get yBounds(): Bounds {
    return new Bounds(this.one.y, this.two.y);
  }

  copy(root?: CrudeCornerXYLocation | XYLocation): Box {
    return new Box(this.one, this.two, 0, 0, root ?? this.root);
  }

  toString(): string {
    return `Top Left: ${this.topLeft.x}, ${this.topLeft.y} Bottom Right: ${this.bottomRight.x}, ${this.bottomRight.y}`;
  }

  reRoot(corner: CrudeCornerXYLocation | XYLocation): Box {
    return this.copy(corner);
  }

  /** A box centered at (0,0) with a width and height of 0. */
  static readonly ZERO = new Box(0, 0, 0, 0);

  /**
   * A box centered at (0,0) with a width and height of 1, and rooted in the
   * bottom left. Note that pixel space is typically rooted in the top left.
   */
  static readonly DECIMAL = new Box(0, 0, 1, 1, XYLocation.BOTTOM_LEFT);

  static readonly z = box.transform((b) => new Box(b));
}

/**
 * Reposition a box so that it is visible within a given bound.
 *
 * @param target The box to reposition - Only works if the root is topLeft
 * @param bound The box to reposition within - Only works if the root is topLeft
 *
 * @returns the repsoitioned box and a boolean indicating if the box was repositioned
 * or not.
 */
export const positionSoVisible = (
  target: HTMLElement | Box,
  bound: HTMLElement | Box
): [Box, boolean] => {
  if (target instanceof HTMLElement) target = new Box(target);
  if (bound instanceof HTMLElement) bound = new Box(bound);
  if (bound.contains(target)) return [target, false];
  let nextPos: XY;
  if (target.right > bound.width)
    nextPos = new XY({
      x: target.x - target.width,
      y: target.y,
    });
  else nextPos = new XY({ x: target.x, y: target.y - target.height });
  return [new Box(nextPos, target.dims), true];
};

/**
 * Reposition a box so that it is centered within a given bound.
 *
 * @param target The box to reposition - Only works if the root is topLeft
 * @param bound The box to reposition within - Only works if the root is topLeft
 * @returns the repsoitioned box
 */
export const positionInCenter = (
  target: HTMLElement | Box,
  bound: HTMLElement | Box
): Box => {
  if (target instanceof HTMLElement) target = new Box(target);
  if (bound instanceof HTMLElement) bound = new Box(bound);
  const x = bound.x + (bound.width - target.width) / 2;
  const y = bound.y + (bound.height - target.height) / 2;
  return new Box({ x, y }, target.dims);
};

export const isBox = (value: unknown): value is Box => {
  if (typeof value !== "object" || value == null) return false;
  return "isBox" in value && (value as Box).isBox;
};

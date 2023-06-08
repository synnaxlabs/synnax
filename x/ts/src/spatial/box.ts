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
  CornerT,
  XY,
  SignedDimensionsT,
  Bounds,
  OuterLocationT,
  Location,
  XLocationT,
  XYT,
  LooseDirectionT,
  Direction,
  DimensionsT,
} from "@/spatial/core";

const cssBox = z.object({
  top: z.number(),
  left: z.number(),
  width: z.number(),
  height: z.number(),
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
  root: Location.strictCornerZ,
});
const looseBox = z.union([
  cssBox,
  domRect,
  z.object({
    one: XY.looseZ,
    two: XY.looseZ,
    root: Location.strictCornerZ,
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
 * It'simportant to note that the behavior of a Box varies depending on its coordinate system.
 * Make sure you're aware of which coordinate system you're using.
 *
 * Many of the properties and methods on a Box access the same semantic value. The different
 * accessors are there for ease of use and semantics.
 */
export class Box implements Stringer {
  readonly one: XY;
  readonly two: XY;
  readonly root: CornerT;

  readonly isBox: true = true;

  constructor(
    first:
      | number
      | DOMRect
      | XYT
      | Box
      | { getBoundingClientRect: () => DOMRect }
      | BoxT,
    second?: number | XYT | DimensionsT | SignedDimensionsT,
    width: number = 0,
    height: number = 0,
    coordinateRoot?: CornerT
  ) {
    if (first instanceof Box) {
      this.one = first.one;
      this.two = first.two;
      this.root = coordinateRoot ?? first.root;
      return;
    }

    this.root = coordinateRoot ?? "topLeft";

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

  contains(box: Box | XY): boolean {
    if ("signedWidth" in box)
      return (
        box.left >= this.left &&
        box.right <= this.right &&
        box.top >= this.top &&
        box.bottom <= this.bottom
      );
    return (
      box.x >= this.left &&
      box.x <= this.right &&
      box.y >= this.top &&
      box.y <= this.bottom
    );
  }

  get dims(): Dimensions {
    return new Dimensions({ width: this.width, height: this.height });
  }

  get signedDims(): SignedDimensionsT {
    return { signedWidth: this.signedWidth, signedHeight: this.signedHeight };
  }

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

  corner(corner: CornerT): XY {
    switch (corner) {
      case "topLeft":
        return new XY({ x: this.left, y: this.top });
      case "bottomRight":
        return new XY({ x: this.right, y: this.bottom });
      case "topRight":
        return new XY({ x: this.right, y: this.top });
      case "bottomLeft":
        return new XY({ x: this.left, y: this.bottom });
    }
  }

  loc(loc: OuterLocationT): number {
    const f = this.root.toLowerCase().includes(loc) ? Math.min : Math.max;
    return Location.X_LOCATIONS.includes(loc as XLocationT)
      ? f(this.one.x, this.two.x)
      : f(this.one.y, this.two.y);
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
    return this.corner("topLeft");
  }

  get topRight(): XY {
    return this.corner("topRight");
  }

  get bottomLeft(): XY {
    return this.corner("bottomLeft");
  }

  get bottomRight(): XY {
    return this.corner("bottomRight");
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

  get x(): number {
    return this.root.toLowerCase().includes("left") ? this.left : this.right;
  }

  get y(): number {
    return this.root.toLowerCase().includes("top") ? this.top : this.bottom;
  }

  get xBound(): Bounds {
    return new Bounds(this.one.x, this.two.x);
  }

  get yBound(): Bounds {
    return new Bounds(this.one.y, this.two.y);
  }

  copy(root?: CornerT): Box {
    return new Box(this.one, this.two, 0, 0, root ?? this.root);
  }

  toString(): string {
    return `Top Left: ${this.topLeft.x}, ${this.topLeft.y} Bottom Right: ${this.bottomRight.x}, ${this.bottomRight.y}`;
  }

  reRoot(corner: CornerT): Box {
    return this.copy(corner);
  }

  static readonly ZERO = new Box(0, 0, 0, 0);

  static readonly DECIMAL = new Box(0, 0, 1, 1, "bottomLeft");

  static readonly z = box;
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

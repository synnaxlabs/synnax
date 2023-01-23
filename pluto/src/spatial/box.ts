/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import { Stringer } from "@synnaxlabs/x";

import {
  Dimensions,
  OuterLocation,
  Corner,
  XY,
  ZERO_XY,
  Direction,
  SignedDimensions,
  X_LOCATIONS,
  XLocation,
  Bound,
} from "./core";

/** represents a partial JS DOMRect */
export interface DOMRect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

export interface BoxCopyProps {
  leftRectOrPoint?: number | DOMRect | XY;
  topPointWidthOrDims?: number | XY | Dimensions;
  widthOrHeight?: number;
  height?: number;
  preserveSign?: boolean;
}

/** BoxProps represents the properties of a Box. */
export interface BoxProps {
  /** y coordinate of the numerically lowest y value in the box. */
  readonly y: number;
  /** x coordinate of the numerically lowest x value in the box. */
  readonly x: number;
  /** y coordinate of the top of the box. */
  readonly top: number;
  /** x coordinate of the left side of the box. */
  readonly left: number;
  /** y coordinate of the bottom of the box. */
  readonly bottom: number;
  /** x coordinate of the right side of the box. */
  readonly right: number;
  /** point representing the top left corner of the box. */
  readonly topLeft: XY;
  /** point representing the top right corner of the box. */
  readonly bottomRight: XY;
  /** point representing the bottom left corner of the box. */
  readonly bottomLeft: XY;
  /** point representing the bottom right corner of the box. */
  readonly topRight: XY;
  /** the absolute width of the box. */
  readonly width: number;
  /** the absolute height of the box. */
  readonly height: number;
  /** the signed width of the box. */
  readonly signedWidth: number;
  /** the signed height of the box. */
  readonly signedHeight: number;
  /** the absolute dimensions of the box. */
  readonly dims: Dimensions;
  /** the signed dimensions of the box. */
  readonly signedDims: SignedDimensions;
}

export interface CSSPosition {
  top: number;
  left: number;
  width: number;
  height: number;
}

export type BoxConstructorProps = [
  leftRectOrPoint: number | DOMRect | XY,
  topPointWidthOrDims?: number | XY | Dimensions,
  widthOrHeight?: number,
  height?: number
];

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
  readonly root: Corner;

  readonly isBox: true = true;

  constructor(
    first: number | DOMRect | XY | Box | { getBoundingClientRect: () => DOMRect },
    second?: number | XY | Dimensions | SignedDimensions,
    width: number = 0,
    height: number = 0,
    coordinateRoot?: Corner
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
      this.one = { x: first, y: second };
      this.two = { x: this.one.x + width, y: this.one.y + height };
      return;
    }

    if ("getBoundingClientRect" in first) first = first.getBoundingClientRect();
    if ("left" in first) {
      this.one = { x: first.left, y: first.top };
      this.two = { x: first.right, y: first.bottom };
      return;
    }

    this.one = first;
    if (second == null) {
      this.two = { x: this.one.x + width, y: this.one.y + height };
    } else if (typeof second === "number")
      this.two = {
        x: this.one.x + second,
        y: this.one.y + width,
      };
    else if ("width" in second)
      this.two = {
        x: this.one.x + second.width,
        y: this.one.y + second.height,
      };
    else if ("signedWidth" in second)
      this.two = {
        x: this.one.x + second.signedWidth,
        y: this.one.y + second.signedHeight,
      };
    else this.two = second;
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
    return { width: this.width, height: this.height };
  }

  get signedDims(): SignedDimensions {
    return { signedWidth: this.signedWidth, signedHeight: this.signedHeight };
  }

  get css(): CSSPosition {
    return {
      top: this.top,
      left: this.left,
      width: this.width,
      height: this.height,
    };
  }

  dim(dir: Direction, signed: boolean = false): number {
    const dim: number = dir === "y" ? this.signedHeight : this.signedWidth;
    return signed ? dim : Math.abs(dim);
  }

  corner(corner: Corner): XY {
    switch (corner) {
      case "topLeft":
        return { x: this.left, y: this.top };
      case "bottomRight":
        return { x: this.right, y: this.bottom };
      case "topRight":
        return { x: this.right, y: this.top };
      case "bottomLeft":
        return { x: this.left, y: this.bottom };
    }
  }

  loc(loc: OuterLocation): number {
    const f = this.root.toLowerCase().includes(loc) ? Math.min : Math.max;
    return X_LOCATIONS.includes(loc as XLocation)
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

  get xBound(): Bound {
    const [lower, upper] = [this.one.x, this.two.x].sort((a, b) => a - b);
    return { lower, upper };
  }

  get yBound(): Bound {
    const [lower, upper] = [this.one.y, this.two.y].sort((a, b) => a - b);
    return { lower, upper };
  }

  copy(root?: Corner): Box {
    return new Box(this.one, this.two, 0, 0, root ?? this.root);
  }

  toString(): string {
    return `Top Left: ${this.topLeft.x}, ${this.topLeft.y} Bottom Right: ${this.bottomRight.x}, ${this.bottomRight.y}`;
  }

  reRoot(corner: Corner): Box {
    return this.copy(corner);
  }
}

export type BoxF = (box: Box) => void;

export const ZERO_BOX: Box = new Box(ZERO_XY, ZERO_XY);
export const DECIMAL_BOX = new Box(0, 0, 1, 1, "bottomLeft");

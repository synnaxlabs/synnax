/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import {
  Dimensions,
  OuterLocation,
  Corner,
  XY,
  ZERO_XY,
  Direction,
  toXY,
  SignedDimensions,
} from "./core";

export interface DOMRect {
  left: number;
  top: number;
  right: number;
  bottom: number;
}

/**
 * A rectangle in 2D space. It most often defines a bounding box for a DOM element in
 * pixel space, but can also have dimensions in clip space or decimal space.
 */
export class Box {
  private readonly one: XY;
  private readonly two: XY;

  constructor(
    leftRectOrPoint: number | DOMRect | XY,
    topPointWidthOrDims?: number | XY | Dimensions,
    widthOrHeight: number = 0,
    height: number = 0
  ) {
    if (typeof leftRectOrPoint === "object" && "left" in leftRectOrPoint) {
      this.one = { x: leftRectOrPoint.left, y: leftRectOrPoint.top };
      this.two = { x: leftRectOrPoint.right, y: leftRectOrPoint.bottom };
    } else if (typeof leftRectOrPoint === "object") {
      this.one = leftRectOrPoint;
      if (topPointWidthOrDims != null) {
        if (typeof topPointWidthOrDims !== "object")
          this.two = {
            x: this.one.x + topPointWidthOrDims,
            y: this.one.y + widthOrHeight,
          };
        else this.two = toXY(topPointWidthOrDims);
      } else this.two = { x: this.one.x + widthOrHeight, y: this.one.y + height };
    } else {
      this.one = { x: leftRectOrPoint, y: topPointWidthOrDims as number };
      this.two = { x: this.one.x + widthOrHeight, y: this.one.y + height };
    }
  }

  clampBy(parent: Box): Box {
    const topLeft = {
      x: Math.max(parent.left, Math.min(parent.right, this.left)),
      y: Math.max(parent.top, Math.min(parent.bottom, this.top)),
    };
    const bottomRight = {
      x: Math.max(parent.left, Math.min(parent.right, this.right)),
      y: Math.max(parent.top, Math.min(parent.bottom, this.bottom)),
    };
    const negativeWidth = this.signedWidth < 0;
    const negativeHeight = this.signedHeight < 0;
    return new Box(
      {
        x: negativeWidth ? bottomRight.x : topLeft.x,
        y: negativeHeight ? bottomRight.y : topLeft.y,
      },
      {
        x: negativeWidth ? topLeft.x : bottomRight.x,
        y: negativeHeight ? topLeft.y : bottomRight.y,
      }
    );
  }

  contains(box: Box | XY): boolean {
    if ("signedWidth" in box) {
      return (
        box.left >= this.left &&
        box.right <= this.right &&
        box.top >= this.top &&
        box.bottom <= this.bottom
      );
    }
    return (
      box.x >= this.left &&
      box.x <= this.right &&
      box.y >= this.top &&
      box.y <= this.bottom
    );
  }

  toDecimal(parent: Box): Box {
    const topLeft = {
      x: (this.left - parent.left) / parent.width,
      y: (this.top - parent.top) / parent.height,
    };
    const bottomRight = {
      x: (this.right - parent.left) / parent.width,
      y: (this.bottom - parent.top) / parent.height,
    };
    const negativeWidth = this.signedWidth < 0;
    const negativeHeight = this.signedHeight < 0;
    return new Box(
      {
        x: negativeWidth ? bottomRight.x : topLeft.x,
        y: negativeHeight ? bottomRight.y : topLeft.y,
      },
      {
        x: negativeWidth ? topLeft.x : bottomRight.x,
        y: negativeHeight ? topLeft.y : bottomRight.y,
      }
    );
  }

  get dims(): Dimensions {
    return { width: this.width, height: this.height };
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
    switch (loc) {
      case "top":
        return Math.min(this.one.y, this.two.y);
      case "bottom":
        return Math.max(this.one.y, this.two.y);
      case "left":
        return Math.min(this.one.x, this.two.x);
      case "right":
        return Math.max(this.one.x, this.two.x);
    }
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
    return this.left;
  }

  get y(): number {
    return this.top;
  }

  translate(v: XY): Box {
    return new Box(
      { x: this.one.x + v.x, y: this.one.y + v.y },
      { x: this.two.x + v.x, y: this.two.y + v.y }
    );
  }

  translateBySignedDims(dims: SignedDimensions): Box {
    return this.translate({ x: dims.signedWidth, y: dims.signedHeight });
  }

  resize(v: XY): Box {
    const bottomRight = this.bottomRight;
    return new Box(this.topLeft, {
      x: bottomRight.x - v.x,
      y: bottomRight.y - v.y,
    });
  }

  scaleDims(v: XY): Box {
    const topLeft = this.topLeft;
    return new Box(this.topLeft, {
      x: topLeft.x + this.width * v.x,
      y: topLeft.y + this.height * v.y,
    });
  }

  scale(v: XY): Box {
    return new Box(
      { x: this.one.x * v.x, y: this.one.y * v.y },
      { x: this.two.x * v.x, y: this.two.y * v.y }
    );
  }

  scaleByDims(v: Dimensions): Box {
    return new Box(
      { x: this.one.x * v.width, y: this.one.y * v.height },
      { x: this.two.x * v.width, y: this.two.y * v.height }
    );
  }

  copy(): Box {
    return new Box(this.one, this.two);
  }
}

export const calculateBottomOffset = (parent: Box, child: Box): number =>
  parent.height - (child.top - parent.top) - child.height;

export const ZERO_BOX: Box = new Box(ZERO_XY, ZERO_XY);

export type BoxF = (box: Box) => void;

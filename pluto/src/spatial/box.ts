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

import { Dimensions, XY, ZERO_XY } from "./core";

/** A sized rectangle positioned in space. Usually represents a DOM element. */
export interface Box extends Dimensions {
  /** The x coordinate of the left edge. */
  left: number;
  /** The y coordinate of the top edge. */
  top: number;
  /** Coordinate of the top left corner. */
  topLeft: XY;
  /** Coordinate of the bottom right corner. */
  bottomRight: XY;
  /**
   * Returns a new box translated by the given coordinates. Positive x moves to the
   * right. Positive y moves downward.
   */
  translate: (v: XY) => Box;
  /***
   * Resizes the box by the given coordinates. Positive x increases the width.
   * Positive y increases the height.
   */
  resize: (v: XY) => Box;
  /**
   * Scales the box by the given coordinates i.e. multiplies the width and height by
   * the given values.
   */
  scale: (v: XY) => Box;
}

export class CSSBox implements Box {
  private readonly _width: number;
  private readonly _height: number;
  private readonly _left: number;
  private readonly _top: number;

  constructor(
    width: number | DOMRect,
    height: number = 0,
    left: number = 0,
    top: number = 0
  ) {
    if (width instanceof DOMRect) {
      this._width = width.width;
      this._height = width.height;
      this._left = width.left;
      this._top = width.top;
      return;
    }
    this._width = width;
    this._height = height;
    this._left = left;
    this._top = top;
  }

  get width(): number {
    return this._width;
  }

  get height(): number {
    return this._height;
  }

  get left(): number {
    return this._left;
  }

  get top(): number {
    return this._top;
  }

  get topLeft(): XY {
    return {
      x: this.left,
      y: this.top,
    };
  }

  get bottomRight(): XY {
    return {
      x: this.left + this.width,
      y: this.top + this.height,
    };
  }

  translate(v: XY): Box {
    return new CSSBox(this.width, this.height, this.left + v.x, this.top + v.y);
  }

  resize(v: XY): Box {
    return new CSSBox(this.width - v.x, this.height - v.y, this.left, this.top);
  }

  scale(v: XY): Box {
    return new CSSBox(this.width * v.x, this.height * v.y, this.left, this.top);
  }
}

export class PointBox implements Box {
  private readonly _one: XY;
  private readonly _two: XY;

  constructor(topLeft: XY, bottomRight: XY) {
    this._one = topLeft;
    this._two = bottomRight;
  }

  get width(): number {
    return this.bottomRight.x - this.topLeft.x;
  }

  get height(): number {
    return this.bottomRight.y - this.topLeft.y;
  }

  get left(): number {
    return this.topLeft.x;
  }

  get top(): number {
    return this.topLeft.y;
  }

  get topLeft(): XY {
    return this.value(Math.min);
  }

  get bottomRight(): XY {
    return this.value(Math.max);
  }

  value(compare: (...v: number[]) => number): XY {
    return {
      x: compare(this._one.x, this._two.x),
      y: compare(this._one.y, this._two.y),
    };
  }

  translate(v: XY): Box {
    return new PointBox(
      { x: this._one.x + v.x, y: this._one.y + v.y },
      { x: this._two.x + v.x, y: this._two.y + v.y }
    );
  }

  resize(v: XY): Box {
    return new PointBox(
      { x: this._one.x + v.x, y: this._one.y + v.y },
      { x: this._two.x - v.x, y: this._two.y - v.y }
    );
  }

  scale(v: XY): Box {
    return new PointBox(
      { x: this._one.x * v.x, y: this._one.y * v.y },
      { x: this._two.x * v.x, y: this._two.y * v.y }
    );
  }
}

export const calculateBottomOffset = (parent: Box, child: Box): number =>
  parent.height - (child.top - parent.top) - child.height;

export const ZERO_BOX: Box = new PointBox(ZERO_XY, ZERO_XY);

export type BoxHandle = (box: Box) => void;

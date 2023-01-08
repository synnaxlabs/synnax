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

export interface Box extends Dimensions {
  left: number;
  top: number;
  topLeft: XY;
  bottomRight: XY;
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
}

export class PointBox {
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
}

export const calculateBottomOffset = (parent: Box, child: Box): number =>
  parent.height - (child.top - parent.top) - child.height;

export const ZERO_BOX: Box = {
  width: 0,
  height: 0,
  left: 0,
  top: 0,
  topLeft: ZERO_XY,
  bottomRight: ZERO_XY,
};

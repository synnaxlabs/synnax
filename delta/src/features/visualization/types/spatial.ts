// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export interface Box {
  width: number;
  height: number;
  left: number;
  top: number;
  topLeft: XY;
  bottomRight: XY;
}

/** A generic 2D point, scale, or offset. */
export interface XY {
  x: number;
  y: number;
}

export const ZERO_XY: XY = { x: 0, y: 0 };

/**
 * Represents a color in RGBA format. RGBA tuples can have any value range (0-255, 0-1, etc.)
 * and should be normalized by any {@link Renderer} that uses them.
 */
export type RGBATuple = [number, number, number, number];

export class CSSBox implements Box {
  private readonly _width: number;
  private readonly _height: number;
  private readonly _left: number;
  private readonly _top: number;

  constructor(width: number, height: number, left: number, top: number) {
    this._width = width;
    this._height = height;
    this._left = left;
    this._top = top;
  }

  static fromDomRect(rect: DOMRect): CSSBox {
    return new CSSBox(rect.width, rect.height, rect.left, rect.top);
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

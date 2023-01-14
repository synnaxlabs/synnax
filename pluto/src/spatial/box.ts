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

import { Dimensions, OuterLocation, Corner, XY, ZERO_XY } from "./core";

/** A sized rectangle positioned in space. Usually represents a DOM element. */
export interface Box extends Dimensions {
  readonly left: number;
  readonly top: number;
  readonly right: number;
  readonly bottom: number;
  loc: (loc: OuterLocation) => number;
  corner: (corner: Corner) => XY;
  readonly topLeft: XY;
  readonly topRight: XY;
  readonly bottomLeft: XY;
  readonly bottomRight: XY;
  readonly isZero: boolean;
  /*
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
  readonly height: number;
  readonly width: number;
  readonly left: number;
  readonly top: number;

  constructor(
    widthOrRect: number | DOMRect,
    height: number = 0,
    left: number = 0,
    top: number = 0
  ) {
    if (widthOrRect instanceof DOMRect) {
      this.height = widthOrRect.height;
      this.width = widthOrRect.width;
      this.left = widthOrRect.left;
      this.top = widthOrRect.top;
      return;
    }
    this.height = height;
    this.width = widthOrRect;
    this.left = left;
    this.top = top;
  }

  corner(corner: "topLeft" | "topRight" | "bottomLeft" | "bottomRight"): XY {
    switch (corner) {
      case "topLeft":
        return { x: this.loc("left"), y: this.loc("top") };
      case "topRight":
        return { x: this.loc("right"), y: this.loc("top") };
      case "bottomLeft":
        return { x: this.loc("left"), y: this.loc("bottom") };
      case "bottomRight":
        return { x: this.loc("right"), y: this.loc("bottom") };
    }
  }

  loc(loc: OuterLocation): number {
    switch (loc) {
      case "top":
        return this.top;
      case "bottom":
        return this.top + this.height;
      case "left":
        return this.left;
      case "right":
        return this.left + this.width;
    }
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

  translate(v: XY): Box {
    return new CSSBox(this.width, this.height, this.left + v.x, this.top + v.y);
  }

  resize(v: XY): Box {
    return new CSSBox(this.width - v.x, this.height - v.y, this.left, this.top);
  }

  scale(v: XY): Box {
    return new CSSBox(this.width * v.x, this.height * v.y, this.left, this.top);
  }

  get isZero(): boolean {
    return this.width === 0 && this.height === 0;
  }
}

export class PointBox implements Box {
  private readonly _one: XY;
  private readonly _two: XY;

  constructor(one: XY, two: XY) {
    this._one = { x: Math.min(one.x, two.x), y: Math.min(one.y, two.y) };
    this._two = { x: Math.max(one.x, two.x), y: Math.max(one.y, two.y) };
  }

  get isZero(): boolean {
    return this._one.x === this._two.x && this._one.y === this._two.y;
  }

  corner(corner: Corner): XY {
    switch (corner) {
      case "topLeft":
        return this._one;
      case "topRight":
        return { x: this._two.x, y: this._one.y };
      case "bottomLeft":
        return { x: this._one.x, y: this._two.y };
      case "bottomRight":
        return this._two;
    }
  }

  loc(loc: OuterLocation): number {
    switch (loc) {
      case "top":
        return this._one.y;
      case "bottom":
        return this._two.y;
      case "left":
        return this._one.x;
      case "right":
        return this._two.x;
    }
  }

  get width(): number {
    return this.loc("right") - this.loc("left");
  }

  get height(): number {
    return this.loc("bottom") - this.loc("top");
  }

  value(compare: (...v: number[]) => number): XY {
    return {
      x: compare(this._one.x, this._two.x),
      y: compare(this._one.y, this._two.y),
    };
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

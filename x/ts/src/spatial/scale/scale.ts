// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { clamp } from "@/clamp/clamp";
import { numeric } from "@/numeric";
import * as bounds from "@/spatial/bounds/bounds";
import { type Box, isBox } from "@/spatial/box/box";
import * as box from "@/spatial/box/box";
import type * as dims from "@/spatial/dimensions/dimensions";
import * as location from "@/spatial/location/location";
import * as xy from "@/spatial/xy/xy";

export const crudeXYTransform = z.object({ offset: xy.crudeZ, scale: xy.crudeZ });

export type XYTransformT = z.infer<typeof crudeXYTransform>;

export type BoundVariant = "domain" | "range";

type ValueType = "position" | "dimension";

type Operation<T extends numeric.Value = number> = (
  currScale: bounds.Bounds<T> | null,
  type: ValueType,
  value: T,
  reverse: boolean,
) => OperationReturn<T>;

type OperationReturn<T extends numeric.Value = number> = [bounds.Bounds<T> | null, T];

interface TypedOperation<T extends numeric.Value = number> extends Operation<T> {
  type: "translate" | "magnify" | "scale" | "invert" | "clamp" | "re-bound";
}

const curriedTranslate =
  <T extends numeric.Value>(translate: T): Operation<T> =>
  (currScale, type, v, reverse) => {
    if (type === "dimension") return [currScale, v];
    return [currScale, (reverse ? v - translate : v + translate) as T];
  };

const curriedMagnify =
  <T extends numeric.Value>(magnify: T): Operation<T> =>
  (currScale, _type, v, reverse) => [
    currScale,
    (reverse ? v / magnify : v * magnify) as T,
  ];

const curriedScale =
  <T extends numeric.Value>(bound: bounds.Bounds<T>): Operation<T> =>
  (currScale, type, v) => {
    if (currScale === null) return [bound, v];
    const { lower: prevLower, upper: prevUpper } = currScale;
    const { lower: nextLower, upper: nextUpper } = bound;
    const prevRange = prevUpper - prevLower;
    const nextRange = nextUpper - nextLower;
    if (type === "dimension") return [bound, (v * (nextRange / prevRange)) as T];
    const nextV = (v - prevLower) * (nextRange / prevRange) + nextLower;
    return [bound, nextV as T];
  };

const curriedReBound =
  <T extends numeric.Value>(bound: bounds.Bounds<T>): Operation<T> =>
  (_, __, v) => [bound, v];

const curriedInvert =
  <T extends numeric.Value>(): Operation<T> =>
  (currScale, type, v) => {
    if (currScale === null) throw new Error("cannot invert without bounds");
    if (type === "dimension") return [currScale, v];
    const { lower, upper } = currScale;
    return [currScale, (upper - (v - lower)) as T];
  };

const curriedClamp =
  <T extends numeric.Value>(bound: bounds.Bounds<T>): Operation<T> =>
  (currScale, _, v) => {
    const { lower, upper } = bound;
    v = clamp<T>(v, lower, upper);
    return [currScale, v];
  };

/**
 * Scale implements a chain of operations that can be used to transform a numeric value.
 */
export class Scale<T extends numeric.Numeric = number> {
  ops: TypedOperation<T>[] = [];
  currBounds: bounds.Bounds<T> | null = null;
  currType: ValueType | null = null;
  private reversed = false;

  constructor() {
    this.ops = [];
  }

  static translate<T extends numeric.Numeric = number>(value: T): Scale<T> {
    return new Scale<T>().translate(value);
  }

  static magnify<T extends numeric.Numeric = number>(value: T): Scale<T> {
    return new Scale<T>().magnify(value);
  }

  static scale<T extends numeric.Numeric>(
    lowerOrBound: T | bounds.Bounds<T>,
    upper?: T,
  ): Scale<T> {
    return new Scale<T>().scale(lowerOrBound, upper);
  }

  translate(value: T): Scale<T> {
    const next = this.new();
    const f = curriedTranslate(value) as TypedOperation<T>;
    f.type = "translate";
    next.ops.push(f);
    return next;
  }

  magnify(value: T): Scale<T> {
    const next = this.new();
    const f = curriedMagnify(value) as TypedOperation<T>;
    f.type = "magnify";
    next.ops.push(f);
    return next;
  }

  scale(lowerOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct<T>(lowerOrBound, upper);
    const next = this.new();
    const f = curriedScale<T>(b) as TypedOperation<T>;
    f.type = "scale";
    next.ops.push(f);
    return next;
  }

  clamp(lowerOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = curriedClamp(b) as TypedOperation<T>;
    f.type = "clamp";
    next.ops.push(f);
    return next;
  }

  reBound(lowerOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = curriedReBound(b) as TypedOperation<T>;
    f.type = "re-bound";
    next.ops.push(f);
    return next;
  }

  invert(): Scale<T> {
    const f = curriedInvert() as TypedOperation<T>;
    f.type = "invert";
    const next = this.new();
    next.ops.push(f);
    return next;
  }

  pos(v: T): T {
    return this.exec("position", v);
  }

  dim(v: T): T {
    return this.exec("dimension", v);
  }

  private new(): Scale<T> {
    const scale = new Scale<T>();
    scale.ops = this.ops.slice();
    scale.reversed = this.reversed;
    return scale;
  }

  private exec(vt: ValueType, v: T): T {
    this.currBounds = null;
    return this.ops.reduce<OperationReturn<T>>(
      ([b, v]: OperationReturn<T>, op: Operation<T>): OperationReturn<T> =>
        op(b, vt, v, this.reversed),
      [null, v],
    )[1];
  }

  reverse(): Scale<T> {
    const scale = this.new();
    scale.ops.reverse();
    // Switch the order of the operations to place scale operation 'BEFORE' the subsequent
    // non - scale operations for example, if we have a reversed [scale A, scale B,
    // translate A, magnify, translate B, scale C] we want to reverse it to [scale C,
    // scale B, translate B, magnify, translate A, scale A]
    const swaps: Array<[number, number]> = [];
    scale.ops.forEach((op, i) => {
      if (op.type === "scale" || swaps.some(([low, high]) => i >= low && i <= high))
        return;
      const nextScale = scale.ops.findIndex((op, j) => op.type === "scale" && j > i);
      if (nextScale === -1) return;
      swaps.push([i, nextScale]);
    });
    swaps.forEach(([low, high]) => {
      const s = scale.ops.slice(low, high);
      s.unshift(scale.ops[high]);
      scale.ops.splice(low, high - low + 1, ...s);
    });
    scale.reversed = !scale.reversed;
    return scale;
  }

  static readonly IDENTITY = new Scale();
}

export const xyScaleToTransform = (scale: XY): XYTransformT => ({
  scale: {
    x: scale.x.dim(1),
    y: scale.y.dim(1),
  },
  offset: {
    x: scale.x.pos(0),
    y: scale.y.pos(0),
  },
});

export class XY {
  x: Scale;
  y: Scale;
  currRoot: location.CornerXY | null;

  constructor(
    x: Scale = new Scale(),
    y: Scale = new Scale(),
    root: location.CornerXY | null = null,
  ) {
    this.x = x;
    this.y = y;
    this.currRoot = root;
  }

  static translate(xy: number | xy.XY, y?: number): XY {
    return new XY().translate(xy, y);
  }

  static translateX(x: number): XY {
    return new XY().translateX(x);
  }

  static translateY(y: number): XY {
    return new XY().translateY(y);
  }

  static clamp(box: Box): XY {
    return new XY().clamp(box);
  }

  static magnify(xy: xy.XY): XY {
    return new XY().magnify(xy);
  }

  static scale(box: dims.Dimensions | Box): XY {
    return new XY().scale(box);
  }

  static reBound(box: Box): XY {
    return new XY().reBound(box);
  }

  translate(cp: number | xy.Crude, y?: number): XY {
    const _xy = xy.construct(cp, y);
    const next = this.copy();
    next.x = this.x.translate(_xy.x);
    next.y = this.y.translate(_xy.y);
    return next;
  }

  translateX(x: number): XY {
    const next = this.copy();
    next.x = this.x.translate(x);
    return next;
  }

  translateY(y: number): XY {
    const next = this.copy();
    next.y = this.y.translate(y);
    return next;
  }

  magnify(xy: xy.XY): XY {
    const next = this.copy();
    next.x = this.x.magnify(xy.x);
    next.y = this.y.magnify(xy.y);
    return next;
  }

  scale(b: Box | dims.Dimensions): XY {
    const next = this.copy();
    if (isBox(b)) {
      const prevRoot = this.currRoot;
      next.currRoot = b.root;
      if (prevRoot != null && !location.xyEquals(prevRoot, b.root)) {
        if (prevRoot.x !== b.root.x) next.x = next.x.invert();
        if (prevRoot.y !== b.root.y) next.y = next.y.invert();
      }
      next.x = next.x.scale(box.xBounds(b));
      next.y = next.y.scale(box.yBounds(b));
      return next;
    }
    next.x = next.x.scale(b.width);
    next.y = next.y.scale(b.height);
    return next;
  }

  reBound(b: Box): XY {
    const next = this.copy();
    next.x = this.x.reBound(box.xBounds(b));
    next.y = this.y.reBound(box.yBounds(b));
    return next;
  }

  clamp(b: Box): XY {
    const next = this.copy();
    next.x = this.x.clamp(box.xBounds(b));
    next.y = this.y.clamp(box.yBounds(b));
    return next;
  }

  copy(): XY {
    const n = new XY();
    n.currRoot = this.currRoot;
    n.x = this.x;
    n.y = this.y;
    return n;
  }

  reverse(): XY {
    const next = this.copy();
    next.x = this.x.reverse();
    next.y = this.y.reverse();
    return next;
  }

  pos(xy: xy.XY): xy.XY {
    return { x: this.x.pos(xy.x), y: this.y.pos(xy.y) };
  }

  box(b: Box): Box {
    return box.construct(
      this.pos(b.one),
      this.pos(b.two),
      0,
      0,
      this.currRoot ?? b.root,
    );
  }

  static readonly IDENTITY = new XY();
}

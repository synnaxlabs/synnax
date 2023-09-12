// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { clamp } from "@/clamp";
import * as bounds from "@/spatial/bounds";
import { type Box, isBox } from "@/spatial/box";
import * as box from "@/spatial/box";
import type * as dims from "@/spatial/dimensions";
import * as location from "@/spatial/location";
import * as xy from "@/spatial/xy";

export const crudeXYTransform = z.object({ offset: xy.crudeZ, scale: xy.crudeZ });

export type XYTransformT = z.infer<typeof crudeXYTransform>;

export type BoundVariant = "domain" | "range";

type ValueType = "position" | "dimension";

type Operation = (
  currScale: bounds.Bounds | null,
  type: ValueType,
  number: number,
  reverse: boolean,
) => OperationReturn;

type OperationReturn = [bounds.Bounds | null, number];

interface TypedOperation extends Operation {
  type: "translate" | "magnify" | "scale" | "invert" | "clamp" | "re-bound";
}

const curriedTranslate =
  (translate: number): Operation =>
  (currScale, type, v, reverse) => {
    if (type === "dimension") return [currScale, v];
    return [currScale, reverse ? v - translate : v + translate];
  };

const curriedMagnify =
  (magnify: number): Operation =>
  (currScale, _type, v, reverse) => [currScale, reverse ? v / magnify : v * magnify];

const curriedScale =
  (bound: bounds.Bounds): Operation =>
  (currScale, type, v) => {
    if (currScale === null) return [bound, v];
    const { lower: prevLower, upper: prevUpper } = currScale;
    const { lower: nextLower, upper: nextUpper } = bound;
    const prevRange = prevUpper - prevLower;
    const nextRange = nextUpper - nextLower;
    if (type === "dimension") return [bound, v * (nextRange / prevRange)];
    const nextV = (v - prevLower) * (nextRange / prevRange) + nextLower;
    return [bound, nextV];
  };

const curriedReBound =
  (bound: bounds.Bounds): Operation =>
  (currScale, type, v) => [bound, v];

const curriedInvert = (): Operation => (currScale, type, v) => {
  if (currScale === null) throw new Error("cannot invert without bounds");
  if (type === "dimension") return [currScale, v];
  const { lower, upper } = currScale;
  return [currScale, upper - (v - lower)];
};

const curriedClamp =
  (bound: bounds.Bounds): Operation =>
  (currScale, _, v) => {
    const { lower, upper } = bound;
    v = clamp(v, lower, upper);
    return [currScale, v];
  };

export class Scale {
  ops: TypedOperation[] = [];
  currBounds: bounds.Bounds | null = null;
  currType: ValueType | null = null;
  private reversed = false;

  constructor() {
    this.ops = [];
  }

  static translate(value: number): Scale {
    return new Scale().translate(value);
  }

  static magnify(value: number): Scale {
    return new Scale().magnify(value);
  }

  static scale(lowerOrBound: number | bounds.Bounds, upper?: number): Scale {
    return new Scale().scale(lowerOrBound, upper);
  }

  translate(value: number): Scale {
    const next = this.new();
    const f = curriedTranslate(value) as TypedOperation;
    f.type = "translate";
    next.ops.push(f);
    return next;
  }

  magnify(value: number): Scale {
    const next = this.new();
    const f = curriedMagnify(value) as TypedOperation;
    f.type = "magnify";
    next.ops.push(f);
    return next;
  }

  scale(lowerOrBound: number | bounds.Bounds, upper?: number): Scale {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = curriedScale(b) as TypedOperation;
    f.type = "scale";
    next.ops.push(f);
    return next;
  }

  clamp(lowerOrBound: number | bounds.Bounds, upper?: number): Scale {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = curriedClamp(b) as TypedOperation;
    f.type = "clamp";
    next.ops.push(f);
    return next;
  }

  reBound(lowerOrBound: number | bounds.Bounds, upper?: number): Scale {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = curriedReBound(b) as TypedOperation;
    f.type = "re-bound";
    next.ops.push(f);
    return next;
  }

  invert(): Scale {
    const f = curriedInvert() as TypedOperation;
    f.type = "invert";
    const next = this.new();
    next.ops.push(f);
    return next;
  }

  pos(v: number): number {
    return this.exec("position", v);
  }

  dim(v: number): number {
    return this.exec("dimension", v);
  }

  private new(): Scale {
    const scale = new Scale();
    scale.ops = this.ops.slice();
    scale.reversed = this.reversed;
    return scale;
  }

  private exec(vt: ValueType, v: number): number {
    this.currBounds = null;
    return this.ops.reduce<OperationReturn>(
      ([b, v]: OperationReturn, op: Operation): OperationReturn =>
        op(b, vt, v, this.reversed),
      [null, v],
    )[1];
  }

  reverse(): Scale {
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

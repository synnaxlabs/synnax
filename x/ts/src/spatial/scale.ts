// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { clamp } from "@/clamp";
import { Box } from "@/spatial/box";
import {
  Bounds,
  XY,
  XYTransformT,
  LooseXYT,
  LooseBoundT,
  XYLocation,
} from "@/spatial/core";

export type ScaleBound = "domain" | "range";

type ValueType = "position" | "dimension";

type Operation = (
  currScale: Bounds | null,
  type: ValueType,
  number: number,
  reverse: boolean
) => OperationReturn;

type OperationReturn = [Bounds | null, number];

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
  (currScale, _type, v, reverse) =>
    [currScale, reverse ? v / magnify : v * magnify];

const curriedScale =
  (bound: Bounds): Operation =>
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
  (bound: Bounds): Operation =>
  (currScale, type, v) =>
    [bound, v];

const curriedInvert = (): Operation => (currScale, type, v) => {
  if (currScale === null) throw new Error("cannot invert without bounds");
  if (type === "dimension") return [currScale, v];
  const { lower, upper } = currScale;
  return [currScale, upper - (v - lower)];
};

const curriedClamp =
  (bound: Bounds): Operation =>
  (currScale, type, v) => {
    if (currScale === null) return [currScale, v];
    const { lower, upper } = currScale;
    v = clamp(v, lower, upper);
    return [currScale, v];
  };

export class Scale {
  ops: TypedOperation[] = [];
  currBounds: Bounds | null = null;
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

  static scale(lowerOrBound: number | LooseBoundT, upper?: number): Scale {
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

  scale(lowerOrBound: number | LooseBoundT, upper?: number): Scale {
    const b = new Bounds(lowerOrBound, upper);
    const next = this.new();
    const f = curriedScale(b) as TypedOperation;
    f.type = "scale";
    next.ops.push(f);
    return next;
  }

  clamp(lowerOrBound: number | LooseBoundT, upper?: number): Scale {
    const b = new Bounds(lowerOrBound, upper);
    const next = this.new();
    const f = curriedClamp(b) as TypedOperation;
    f.type = "clamp";
    next.ops.push(f);
    return next;
  }

  reBound(lowerOrBound: number | LooseBoundT, upper?: number): Scale {
    const b = new Bounds(lowerOrBound, upper);
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
      [null, v]
    )[1];
  }

  reverse(): Scale {
    const scale = this.new();
    scale.ops.reverse();
    // switch the order of the operations to place scale operation 'BEFORE' the subsequent
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
}

export const xyScaleToTransform = (scale: XYScale): XYTransformT => ({
  scale: {
    x: scale.x.dim(1),
    y: scale.y.dim(1),
  },
  offset: {
    x: scale.x.pos(0),
    y: scale.y.pos(0),
  },
});

export class XYScale {
  x: Scale;
  y: Scale;
  currRoot: XYLocation | null;

  constructor(
    x: Scale = new Scale(),
    y: Scale = new Scale(),
    root: XYLocation | null = null
  ) {
    this.x = x;
    this.y = y;
    this.currRoot = root;
  }

  static translate(xy: number | LooseXYT, y?: number): XYScale {
    return new XYScale().translate(xy, y);
  }

  static translateX(x: number): XYScale {
    return new XYScale().translateX(x);
  }

  static translateY(y: number): XYScale {
    return new XYScale().translateY(y);
  }

  static clamp(box: Box): XYScale {
    return new XYScale().clamp(box);
  }

  static magnify(xy: XY): XYScale {
    return new XYScale().magnify(xy);
  }

  static scale(box: Box): XYScale {
    return new XYScale().scale(box);
  }

  static reBound(box: Box): XYScale {
    return new XYScale().reBound(box);
  }

  translate(xy: number | LooseXYT, y?: number): XYScale {
    const _xy = new XY(xy, y);
    const next = this.copy();
    next.x = this.x.translate(_xy.x);
    next.y = this.y.translate(_xy.y);
    return next;
  }

  translateX(x: number): XYScale {
    const next = this.copy();
    next.x = this.x.translate(x);
    return next;
  }

  translateY(y: number): XYScale {
    const next = this.copy();
    next.y = this.y.translate(y);
    return next;
  }

  magnify(xy: XY): XYScale {
    const next = this.copy();
    next.x = this.x.magnify(xy.x);
    next.y = this.y.magnify(xy.y);
    return next;
  }

  scale(box: Box): XYScale {
    const next = this.copy();
    const prevRoot = this.currRoot;
    next.currRoot = box.root;
    if (prevRoot != null && prevRoot !== box.root) {
      if (!prevRoot.x.equals(box.root.x)) next.x = next.x.invert();
      if (!prevRoot.y.equals(box.root.y)) next.y = next.y.invert();
    }
    next.x = next.x.scale(box.xBounds);
    next.y = next.y.scale(box.yBounds);
    return next;
  }

  reBound(box: Box): XYScale {
    const next = this.copy();
    next.x = this.x.reBound(box.xBounds);
    next.y = this.y.reBound(box.yBounds);
    return next;
  }

  clamp(box: Box): XYScale {
    const next = this.copy();
    next.x = this.x.clamp(box.xBounds);
    next.y = this.y.clamp(box.yBounds);
    return next;
  }

  copy(): XYScale {
    const n = new XYScale();
    n.currRoot = this.currRoot;
    n.x = this.x;
    n.y = this.y;
    return n;
  }

  pos(xy: LooseXYT): XY {
    const xy_ = new XY(xy);
    return new XY({ x: this.x.pos(xy_.x), y: this.y.pos(xy_.y) });
  }

  box(box: Box): Box {
    return new Box(
      this.pos(box.one),
      this.pos(box.two),
      0,
      0,
      this.currRoot ?? box.root
    );
  }
}

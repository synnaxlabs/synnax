// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box } from "./box";
import {
  toBound,
  Bound,
  Corner,
  cornerLocations as cornerLocs,
  Dimensions,
  SignedDimensions,
  toXY,
  XY,
  Direction,
  XYTransform,
} from "./core";

import { clamp } from "@/clamp";

export type ScaleBound = "domain" | "range";

type ValueType = "position" | "dimension";

type Operation = (
  currScale: Bound | null,
  type: ValueType,
  number: number,
  reverse: boolean
) => OperationReturn;

type OperationReturn = [Bound | null, number];

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
  (bound: Bound): Operation =>
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
  (bound: Bound): Operation =>
  (currScale, type, v) =>
    [bound, v];

const curriedInvert = (): Operation => (currScale, type, v) => {
  if (currScale === null) throw new Error("cannot invert without bounds");
  if (type === "dimension") return [currScale, v];
  const { lower, upper } = currScale;
  return [currScale, upper - (v - lower)];
};

const curriedClamp =
  (bound: Bound): Operation =>
  (currScale, type, v) => {
    if (currScale === null) return [currScale, v];
    const { lower, upper } = currScale;
    v = clamp(v, lower, upper);
    return [currScale, v];
  };

export class Scale {
  ops: TypedOperation[] = [];
  currBounds: Bound | null = null;
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

  static scale(lowerOrBound: number | Bound, upper?: number): Scale {
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

  scale(lowerOrBound: number | Bound, upper?: number): Scale {
    const b = toBound(lowerOrBound, upper);
    const next = this.new();
    const f = curriedScale(b) as TypedOperation;
    f.type = "scale";
    next.ops.push(f);
    return next;
  }

  clamp(lowerOrBound: number | Bound, upper?: number): Scale {
    const b = toBound(lowerOrBound, upper);
    const next = this.new();
    const f = curriedClamp(b) as TypedOperation;
    f.type = "clamp";
    next.ops.push(f);
    return next;
  }

  reBound(lowerOrBound: number | Bound, upper?: number): Scale {
    const b = toBound(lowerOrBound, upper);
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

export type XYScale = Record<Direction, Scale>;

export const xyScaleToTransform = (scale: XYScale): XYTransform => ({
  scale: {
    x: scale.x.pos(1),
    y: scale.y.pos(1),
  },
  offset: {
    x: scale.x.pos(0),
    y: scale.y.pos(0),
  },
});

export class BoxScale {
  x: Scale;
  y: Scale;
  currRoot: Corner | null;

  constructor() {
    this.x = new Scale();
    this.y = new Scale();
    this.currRoot = null;
  }

  static translate(xy: XY | Dimensions | SignedDimensions): BoxScale {
    return new BoxScale().translate(xy);
  }

  static clamp(box: Box): BoxScale {
    return new BoxScale().clamp(box);
  }

  static magnify(xy: XY): BoxScale {
    return new BoxScale().magnify(xy);
  }

  static scale(box: Box): BoxScale {
    return new BoxScale().scale(box);
  }

  translate(xy: XY | Dimensions | SignedDimensions): BoxScale {
    const _xy = toXY(xy);
    const next = this.new();
    next.x = this.x.translate(_xy.x);
    next.y = this.y.translate(_xy.y);
    return next;
  }

  magnify(xy: XY): BoxScale {
    const next = this.new();
    next.x = this.x.magnify(xy.x);
    next.y = this.y.magnify(xy.y);
    return next;
  }

  scale(box: Box): BoxScale {
    const next = this.new();
    const prevRoot = this.currRoot;
    next.currRoot = box.root;
    if (prevRoot != null && prevRoot !== box.root) {
      const [prevX, prevY] = cornerLocs(prevRoot);
      const [currX, currY] = cornerLocs(box.root);
      if (prevX !== currX) next.x = next.x.invert();
      if (prevY !== currY) next.y = next.y.invert();
    }
    next.x = next.x.scale(box.xBound);
    next.y = next.y.scale(box.yBound);
    return next;
  }

  clamp(box: Box): BoxScale {
    const next = this.new();
    next.x = this.x.clamp(box.xBound);
    next.y = this.y.clamp(box.yBound);
    return next;
  }

  private new(): BoxScale {
    const n = new BoxScale();
    n.currRoot = this.currRoot;
    n.x = this.x;
    n.y = this.y;
    return n;
  }

  pos(xy: XY): XY {
    return { x: this.x.pos(xy.x), y: this.y.pos(xy.y) };
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@/math";
import { type numeric } from "@/numeric";
import { bounds } from "@/spatial/bounds";
import { box } from "@/spatial/box";
import { type dimensions } from "@/spatial/dimensions";
import { location } from "@/spatial/location";
import { xy } from "@/spatial/xy";

export interface XYTransformT {
  offset: xy.Crude;
  scale: xy.Crude;
}

interface TransformT<T extends numeric.Value> {
  offset: T;
  scale: T;
}

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

const typed = <T extends numeric.Value>(
  type: TypedOperation<T>["type"],
  op: Operation<T>,
): TypedOperation<T> => Object.assign(op, { type });

const curriedTranslate =
  <T extends numeric.Value>(translate: T): Operation<T> =>
  (currScale, type, v, reverse) => {
    if (type === "dimension") return [currScale, v];
    return [currScale, reverse ? math.sub(v, translate) : math.add(v, translate)];
  };

const curriedMagnify =
  <T extends numeric.Value>(magnify: T): Operation<T> =>
  (currScale, _type, v, reverse) => [
    currScale,
    reverse ? math.div(v, magnify) : math.mult(v, magnify),
  ];

const curriedScale =
  <T extends numeric.Value>(bound: bounds.Bounds<T>): Operation<T> =>
  (currScale, type, v) => {
    if (currScale === null) return [bound, v];
    const { lower: prevLower, upper: prevUpper } = currScale;
    const { lower: nextLower, upper: nextUpper } = bound;
    const prevRange = math.sub(prevUpper, prevLower);
    const nextRange = math.sub(nextUpper, nextLower);
    const ratio = math.div(nextRange, prevRange);
    if (type === "dimension") return [bound, math.mult(v, ratio)];
    return [bound, math.add(math.mult(math.sub(v, prevLower), ratio), nextLower)];
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
    return [currScale, math.sub(upper, math.sub(v, lower))];
  };

const curriedClamp =
  <T extends numeric.Value>(bound: bounds.Bounds<T>): Operation<T> =>
  (currScale, _, v) => {
    v = bounds.clamp<T>(bound, v);
    return [currScale, v];
  };

/**
 * Scale implements a chain of operations that can be used to transform numeric values.
 * A scale can either operate on standard JS numbers or on BigInts, but not both.
 */
export class Scale<T extends numeric.Value = number> {
  private readonly ops: TypedOperation<T>[];
  private readonly reversed: boolean;

  private constructor(ops: TypedOperation<T>[] = [], reversed: boolean = false) {
    this.ops = ops;
    this.reversed = reversed;
  }

  /** @returns a new scale whose first operation translates by value. */
  static translate<T extends numeric.Value = number>(value: T): Scale<T> {
    return new Scale<T>().translate(value);
  }

  /** @returns a new scale whose first operation multiplies by value. */
  static magnify<T extends numeric.Value = number>(value: T): Scale<T> {
    return new Scale<T>().magnify(value);
  }

  /**
   * @returns a new scale whose initial band is the given bounds. Call {@link scale}
   * again to map that band onto another. A lone upper implies a lower of 0.
   *
   * @example Scale.scale<number>(1).scale(100).pos(1) // => 100
   */
  static scale<T extends numeric.Value>(lowerOrUpper: T, upper?: T): Scale<T>;

  static scale<T extends numeric.Value>(bound: bounds.Bounds<T>): Scale<T>;

  static scale<T extends numeric.Value = number>(upperOrBound: T, upper?: T): Scale<T> {
    return new Scale<T>().scale(upperOrBound, upper);
  }

  /** @returns a copy whose next operation translates by value. */
  translate(value: T): Scale<T> {
    const next = this.new();
    const f = typed<T>("translate", curriedTranslate(value));
    next.ops.push(f);
    return next;
  }

  /** @returns a copy whose next operation multiplies by value. */
  magnify(value: T): Scale<T> {
    const next = this.new();
    const f = typed<T>("magnify", curriedMagnify(value));
    next.ops.push(f);
    return next;
  }
  /**
   * @returns a copy whose next operation re-scales onto the given bounds. A lone upper
   * implies a lower of 0.
   *
   * @example Scale.scale<number>(1).scale(100).pos(1) // => 100
   */
  scale(lowerOrUpper: T, upper?: T): Scale<T>;

  scale(bounds: bounds.Bounds<T>): Scale<T>;

  scale(upperOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct<T>(upperOrBound, upper);
    const next = this.new();
    const f = typed<T>("scale", curriedScale<T>(b));
    next.ops.push(f);
    return next;
  }

  /**
   * @returns a copy that clamps every value to the given bounds. A lone upper implies a
   * lower of 0.
   *
   * @example Scale.scale(0, 1).clamp(0, 0.5).pos(1) // => 0.5
   */
  clamp(lowerOrUpper: T, upper?: T): Scale<T>;

  clamp(bounds: bounds.Bounds<T>): Scale<T>;

  clamp(lowerOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = typed<T>("clamp", curriedClamp(b));
    next.ops.push(f);
    return next;
  }

  /**
   * @returns a copy whose bounds are replaced, with no scaling applied to the values
   * passing through it.
   */
  reBound(lower: T, upper?: T): Scale<T>;

  /**
   * @returns a copy of he scale with a re-bounding operation applied. This operation
   */
  reBound(bound: bounds.Bounds<T>): Scale<T>;

  reBound(lowerOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = typed<T>("re-bound", curriedReBound(b));
    next.ops.push(f);
    return next;
  }

  invert(): Scale<T> {
    const f = typed<T>("invert", curriedInvert());
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
    return new Scale<T>(this.ops.slice(), this.reversed);
  }

  private exec(vt: ValueType, v: T): T {
    return this.ops.reduce<OperationReturn<T>>(
      ([b, v]: OperationReturn<T>, op: Operation<T>): OperationReturn<T> =>
        op(b, vt, v, this.reversed),
      [null, v],
    )[1];
  }

  reverse(): Scale<T> {
    const ops = this.ops.slice().reverse();
    // Switch the order of the operations to place scale operation 'BEFORE' the
    // subsequent non - scale operations for example, if we have a reversed [scale A,
    // scale B, translate A, magnify, translate B, scale C] we want to reverse it to
    // [scale C, scale B, translate B, magnify, translate A, scale A]
    const swaps: Array<[number, number]> = [];
    ops.forEach((op, i) => {
      if (op.type === "scale" || swaps.some(([low, high]) => i >= low && i <= high))
        return;
      const nextScale = ops.findIndex((op, j) => op.type === "scale" && j > i);
      if (nextScale === -1) return;
      swaps.push([i, nextScale]);
    });
    swaps.forEach(([low, high]) => {
      const s = ops.slice(low, high);
      s.unshift(ops[high]);
      ops.splice(low, high - low + 1, ...s);
    });
    return new Scale<T>(ops, !this.reversed);
  }

  get transform(): TransformT<T> {
    return { scale: this.dim(1 as T), offset: this.pos(0 as T) };
  }

  static readonly IDENTITY = new Scale();
}

export class XY {
  readonly x: Scale<number>;
  readonly y: Scale<number>;
  private readonly currRoot: location.Corner | null;

  constructor(
    x: Scale<number> = Scale.IDENTITY,
    y: Scale<number> = Scale.IDENTITY,
    root: location.Corner | null = null,
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

  static clamp(b: box.Box): XY {
    return new XY().clamp(b);
  }

  static magnify(xy: xy.XY): XY {
    return new XY().magnify(xy);
  }

  static scale(b: dimensions.Dimensions | box.Box): XY {
    return new XY().scale(b);
  }

  static reBound(b: box.Box): XY {
    return new XY().reBound(b);
  }

  translate(cp: number | xy.Crude, y?: number): XY {
    const _xy = xy.construct(cp, y);
    return new XY(this.x.translate(_xy.x), this.y.translate(_xy.y), this.currRoot);
  }

  translateX(x: number): XY {
    return new XY(this.x.translate(x), this.y, this.currRoot);
  }

  translateY(y: number): XY {
    return new XY(this.x, this.y.translate(y), this.currRoot);
  }

  magnify(xy: xy.XY): XY {
    return new XY(this.x.magnify(xy.x), this.y.magnify(xy.y), this.currRoot);
  }

  scale(b: box.Box | dimensions.Dimensions): XY {
    if (!("root" in b))
      return new XY(this.x.scale(b.width), this.y.scale(b.height), this.currRoot);
    let { x, y } = this;
    const prevRoot = this.currRoot;
    if (prevRoot != null && !location.xyEquals(prevRoot, b.root)) {
      if (prevRoot.x !== b.root.x) x = x.invert();
      if (prevRoot.y !== b.root.y) y = y.invert();
    }
    return new XY(x.scale(box.xBounds(b)), y.scale(box.yBounds(b)), b.root);
  }

  reBound(b: box.Box): XY {
    return new XY(
      this.x.reBound(box.xBounds(b)),
      this.y.reBound(box.yBounds(b)),
      this.currRoot,
    );
  }

  clamp(b: box.Box): XY {
    return new XY(
      this.x.clamp(box.xBounds(b)),
      this.y.clamp(box.yBounds(b)),
      this.currRoot,
    );
  }

  copy(): XY {
    return new XY(this.x, this.y, this.currRoot);
  }

  reverse(): XY {
    return new XY(this.x.reverse(), this.y.reverse(), this.currRoot);
  }

  pos(xy: xy.XY): xy.XY {
    return { x: this.x.pos(xy.x), y: this.y.pos(xy.y) };
  }

  dim(xy: xy.XY): xy.XY {
    return { x: this.x.dim(xy.x), y: this.y.dim(xy.y) };
  }

  box(b: box.Box): box.Box {
    return box.construct(
      this.pos(b.one),
      this.pos(b.two),
      0,
      0,
      this.currRoot ?? b.root,
    );
  }

  get transform(): XYTransformT {
    return {
      scale: this.dim({ x: 1, y: 1 }),
      offset: this.pos({ x: 0, y: 0 }),
    };
  }

  static readonly IDENTITY = new XY();
}

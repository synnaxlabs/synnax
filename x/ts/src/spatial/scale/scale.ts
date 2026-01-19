// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { clamp } from "@/clamp/clamp";
import { type numeric } from "@/numeric";
import { bounds } from "@/spatial/bounds";
import { box } from "@/spatial/box";
import { type Box, isBox } from "@/spatial/box/box";
import { type dimensions } from "@/spatial/dimensions";
import { location } from "@/spatial/location";
import { xy } from "@/spatial/xy";

export const crudeXYTransform = z.object({ offset: xy.crudeZ, scale: xy.crudeZ });
export type XYTransformT = z.infer<typeof crudeXYTransform>;

export const transform = z.object({ offset: z.number(), scale: z.number() });
export type TransformT<T extends numeric.Value = number> = {
  offset: T;
  scale: T;
};

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
    return [
      currScale,
      (reverse ? v - translate : (v as number) + (translate as number)) as T,
    ];
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
    const prevRange = (prevUpper - prevLower) as T;
    const nextRange = (nextUpper - nextLower) as T;
    if (type === "dimension") return [bound, (v * (nextRange / prevRange)) as T];
    // @ts-expect-error - typescript can't do the math correctly
    const nextV = ((v - prevLower) * (nextRange / prevRange) + nextLower) as T;
    return [bound, nextV];
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
 * Scale implements a chain of operations that can be used to transform numeric values.
 * A scale can either operate on standard JS numbers or on BigInts, but not both.
 */
export class Scale<T extends numeric.Value = number> {
  ops: TypedOperation<T>[] = [];
  currBounds: bounds.Bounds<T> | null = null;
  currType: ValueType | null = null;
  private reversed = false;

  constructor() {
    this.ops = [];
  }

  /**
   * @returns a new scale with a translation as its first operation. Any number provided
   * to the {@link pos} operation on the scale will be translated by the specified value.
   * @param value - The amount to translate by.
   */
  static translate<T extends numeric.Value = number>(value: T): Scale<T> {
    return new Scale<T>().translate(value);
  }

  /**
   * @returns a new scale with a magnification as its first operation. Any number provided
   * to the {@link pos} or {@link dim} operation will be multiplied by the specified value.
   * @param value - The amount to translate by.
   */
  static magnify<T extends numeric.Value = number>(value: T): Scale<T> {
    return new Scale<T>().magnify(value);
  }

  /**
   * @returns a new scale that uses the given lower and upper bounds as it's initial
   * scale. This scale will be used to define the initial 'band' of values defined within
   * the scale. Once the scale is created, call {@link scale} again to scale the initial
   * band to the new scaled band.
   *
   * @example
   *    const s = Scale.scale<number>(1).scale(100)
   *    console.log(s.pos(0))
   *    // 0
   *    console.log(s.pos(1))
   *    // 100
   *
   * @param upper - The upper bound of the scale. This overload assumes that the lower
   * bound is 0.
   */
  static scale<T extends numeric.Value>(upper: T): Scale<T>;

  /**
   * @returns a new scale that uses the given lower and upper bounds as it's initial
   * scale. This scale will be used to define the initial 'band' of values defined within
   * the scale. Once the scale is created, call {@link scale} again to scale the initial
   * band to the new scaled band.
   *
   * @example
   *    const s = Scale.scale<number>(0, 1).scale(0, 100)
   *    console.log(s.pos(0))
   *    // 0
   *    console.log(s.pos(1))
   *    // 100
   *
   * @param lower - The lower bound of the scale.
   * @param upper - The upper bound of the scale.
   */
  static scale<T extends numeric.Value>(lower: T, upper: T): Scale<T>;

  /**
   * @returns a new scale that uses the given bounds as it's initial
   * scale. This scale will be used to define the initial 'band' of values defined within
   * the scale. Once the scale is created, call {@link scale} again to scale the initial
   * band to the new scaled band.
   *
   * @example
   *    const s = Scale.scale<number>({ lower: 0, upper: 1 }).scale({lower: 0, upper: 100 })
   *    console.log(s.pos(0))
   *    // 0
   *    console.log(s.pos(1))
   *    // 100
   *
   * @param bound - The bound to scale by. See {@link bounds.Bounds} for more info.
   */
  static scale<T extends numeric.Value>(bound: bounds.Bounds<T>): Scale<T>;

  static scale<T extends numeric.Value = number>(upperOrBound: T, upper?: T): Scale<T> {
    return new Scale<T>().scale(upperOrBound, upper);
  }

  /**
   * @returns a copy of the scale with a translation as its next operation. Any
   * number provided to the {@link pos} method on the scale will be translated by the
   * specified value.
   * @param value - The amount to translate by.
   */
  translate(value: T): Scale<T> {
    const next = this.new();
    const f = curriedTranslate(value) as TypedOperation<T>;
    f.type = "translate";
    next.ops.push(f);
    return next;
  }

  /**
   * @returns a copy of the scale with a translation as its next operation. Any number
   * provided to the {@link pos} or {@link dim} method on the scale will be multiplied
   * by the specified value.
   * @param value - The amount to magnify by.
   */
  magnify(value: T): Scale<T> {
    const next = this.new();
    const f = curriedMagnify(value) as TypedOperation<T>;
    f.type = "magnify";
    next.ops.push(f);
    return next;
  }
  /**
   * @returns a copy of the scale with a 're-scaling' as its next operation. This will
   * translate numbers provided to {@link pos} and {@link dim} to the new scale.
   *
   * @example
   *    const s = Scale.scale<number>(1).scale(100)
   *    console.log(s.pos(0))
   *    // 0
   *    console.log(s.pos(1))
   *    // 100
   *
   * @param upper - The upper bound of the scale. This overload assumes that the lower
   * bound is 0.
   */
  scale(upper: T): Scale<T>;

  /**
   * @returns a copy of the scale with a 're-scaling' as its next operation. This will
   * translate numbers provided to {@link pos} and {@link dim} to the new scale.
   *
   * @example
   *    const s = Scale.scale<number>(0, 1).scale(0, 100)
   *    console.log(s.pos(0))
   *    // 0
   *    console.log(s.pos(1))
   *    // 100
   *
   * @param lower - The lower bound of the new scale.
   * @param upper - The upper bound of the new scale.
   */
  scale(lower: T, upper: T): Scale<T>;

  /**
   * @returns a copy of the scale with a 're-scaling' as its next operation. This will
   * translate numbers provided to {@link pos} and {@link dim} to the new scale.
   *
   * @example
   *    const s = Scale.scale<number>({ lower: 0, upper: 1 }).scale({lower: 0, upper: 100 })
   *    console.log(s.pos(0))
   *    // 0
   *    console.log(s.pos(1))
   *    // 100
   *
   * @param bound - The bound to scale by. See {@link bounds.Bounds} for more info.
   */
  scale(bounds: bounds.Bounds<T>): Scale<T>;

  /** This overload is for internal use only */
  scale(upperOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T>;

  scale(upperOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct<T>(upperOrBound, upper);
    const next = this.new();
    const f = curriedScale<T>(b) as TypedOperation<T>;
    f.type = "scale";
    next.ops.push(f);
    return next;
  }

  /**
   * @returns a copy of the scale with a clamping operation applied. Any number passed
   * to the scale wil be clamped to the specified bounds.
   *
   * @example
   *  const s = Scale.scale(0, 1).clamp(0, 0.5)
   *  console.log(s.pos(1))
   *  // 0.5
   *
   * @param upper - The upper bound to clamp by. WARNING: This operation assumes
   * that the lower bound of the clamp is 0.
   */
  clamp(upper: T): Scale<T>;

  /**
   * @returns a copy of the scale with a clamping operation applied. Any number passed
   * to the scale wil be clamped to the specified bounds.
   *
   * @example
   *  const s = Scale.scale(0, 1).clamp(0, 0.5)
   *  console.log(s.pos(1))
   *  // 0.5
   *
   * @param lower - The lower bound of the scale.
   * @param upper - The upper bound of the scale.
   */
  clamp(lower: T, upper: T): Scale<T>;

  /**
   * @returns a copy of the scale with a clamping operation applied. Any number passed
   * to the scale will be clamped to the specified bounds.
   *
   * @example
   *  const s = Scale.scale(0, 1).clamp({ lower: 0, upper: 0.5 })
   *  console.log(s.pos(1))
   *  // 0.5
   *
   * @param bounds - The bounds to clamp by.
   */
  clamp(bounds: bounds.Bounds<T>): Scale<T>;

  clamp(lowerOrBound: T | bounds.Bounds<T>, upper?: T): Scale<T> {
    const b = bounds.construct(lowerOrBound, upper);
    const next = this.new();
    const f = curriedClamp(b) as TypedOperation<T>;
    f.type = "clamp";
    next.ops.push(f);
    return next;
  }

  /**
   * @returns a copy of the scale with a re-bounding operation applied. This operation
   * adjusts the bounds of the scale WITHOUT applying a scaling operation to values
   * passed through the scale.
   *
   * @example
   *  const s = Scale.scale(0, 1).reBound(0, 100)
   *  console.log(s.bound)
   *
   * @param lower - The new lower bound.
   * @param upper - The new upper bound.
   */
  reBound(lower: T, upper?: T): Scale<T>;

  /**
   * @returns a copy of he scale with a re-bounding operation applied. This operation
   *
   * @param bound
   */
  reBound(bound: bounds.Bounds<T>): Scale<T>;

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

  get transform(): TransformT<T> {
    return { scale: this.dim(1 as T), offset: this.pos(0 as T) };
  }

  static readonly IDENTITY = new Scale();
}

export class XY {
  x: Scale<number>;
  y: Scale<number>;
  currRoot: location.CornerXY | null;

  constructor(
    x: Scale<number> = new Scale<number>(),
    y: Scale<number> = new Scale<number>(),
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

  static scale(box: dimensions.Dimensions | Box): XY {
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

  scale(b: Box | dimensions.Dimensions): XY {
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

  dim(xy: xy.XY): xy.XY {
    return { x: this.x.dim(xy.x), y: this.y.dim(xy.y) };
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

  get transform(): XYTransformT {
    return {
      scale: this.dim({ x: 1, y: 1 }),
      offset: this.pos({ x: 0, y: 0 }),
    };
  }

  static readonly IDENTITY = new XY();
}

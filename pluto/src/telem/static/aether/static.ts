// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Series,
  type GLBufferController,
  nativeTypedArray,
  Rate,
  DataType,
  TimeRange,
  bounds,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type telem } from "@/telem/core";

export class Factory implements telem.Factory {
  type = "static";

  create(key: string, spec: telem.Spec): telem.Telem | null {
    switch (spec.type) {
      case XY.TYPE:
        return new XY(key, spec.props);
      case IterativeXY.TYPE:
        return new IterativeXY(key, spec.props);
      case Numeric.TYPE:
        return new Numeric(key, spec.props);
      default:
        return null;
    }
  }
}

export const xyPropsZ = z.object({
  x: z.array(nativeTypedArray),
  y: z.array(nativeTypedArray),
  xOffsets: z.array(z.number()).optional().default([]),
  yOffsets: z.array(z.number()).optional().default([]),
});

export type XYProps = z.input<typeof xyPropsZ>;

class XYCore {
  key: string;
  _x: Series[];
  _y: Series[];
  onChangeHandler?: () => void;

  constructor(key: string, x: Series[], y: Series[]) {
    this.key = key;
    this._x = x;
    this._y = y;
  }

  async x(gl?: GLBufferController): Promise<Series[]> {
    if (gl != null) this._x.map((x) => x.updateGLBuffer(gl));
    return this._x;
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    if (gl != null) this._y.map((y) => y.updateGLBuffer(gl));
    return this._y;
  }

  async yBounds(): Promise<bounds.Bounds> {
    const y = await this.y();
    return bounds.max(y.map((x) => x.bounds));
  }

  async xBounds(): Promise<bounds.Bounds> {
    const x = await this.x();
    return bounds.max(x.map((x) => x.bounds));
  }

  release(gl: GLBufferController): void {}

  onChange(f: () => void): void {
    this.onChangeHandler = f;
  }

  invalidate(): void {
    if (this.onChangeHandler != null) this.onChangeHandler();
  }

  cleanup(): void {}
}

export class XY extends XYCore implements telem.XYSource {
  static readonly TYPE = "static-xy";

  variant = "xy";

  constructor(key: string, props_: any) {
    const props = xyPropsZ.parse(props_);
    super(
      key,
      props.x.map((x, i) => {
        const arr = new Series(
          x,
          DataType.FLOAT32,
          TimeRange.ZERO,
          props.xOffsets[i] ?? 0,
        );
        arr.acquire();
        return arr;
      }),
      props.y.map((y, i) => {
        const arr = new Series(
          y,
          DataType.FLOAT32,
          TimeRange.ZERO,
          props.yOffsets[i] ?? 0,
        );
        arr.acquire();
        return arr;
      }),
    );
  }

  setProps(props_: any): void {
    const props = xyPropsZ.parse(props_);
    this._x = props.x.map(
      (x, i) => new Series(x, DataType.FLOAT32, TimeRange.ZERO, props.xOffsets[i] ?? 0),
    );
    this._y = props.y.map(
      (y, i) => new Series(y, DataType.FLOAT32, TimeRange.ZERO, props.yOffsets[i] ?? 0),
    );
    this.onChangeHandler?.();
  }
}

export const iterativeXYPropsZ = xyPropsZ.extend({
  rate: Rate.z,
  yOffset: z.number().optional().default(0),
});

export type IterativeXYProps = z.input<typeof iterativeXYPropsZ>;

export class IterativeXY extends XYCore implements telem.XYSource {
  position: number;
  interval?: number;

  static readonly TYPE = "iterative-xy";

  variant = "xy";

  constructor(key: string, props_: any) {
    const { x, y, rate, yOffset } = iterativeXYPropsZ.parse(props_);
    super(
      key,
      x.map((x) => new Series(x)),
      y.map((y) => new Series(y, DataType.FLOAT32, TimeRange.ZERO, yOffset)),
    );
    this.position = 0;
    this.start(rate);
  }

  setProps(props_: any): void {
    const props = iterativeXYPropsZ.parse(props_);
    this._x = props.x.map((x) => new Series(x));
    this._y = props.y.map(
      (y) => new Series(y, DataType.FLOAT32, TimeRange.ZERO, props.yOffset),
    );
  }

  async x(gl?: GLBufferController): Promise<Series[]> {
    const x = this._x.map((x) => x.slice(0, this.position));
    if (gl != null) x.map((x) => x.updateGLBuffer(gl));
    return x;
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    const y = this._y.map((y) => y.slice(0, this.position));
    if (gl != null) y.map((y) => y.updateGLBuffer(gl));
    return y;
  }

  start(rate: Rate): void {
    if (this.interval != null) clearInterval(this.interval);
    this.interval = setInterval(() => {
      this.onChangeHandler?.();
      this.incrementPosition();
    }, rate.period.milliseconds) as unknown as number;
  }

  private incrementPosition(): void {
    this.position++;
  }

  cleanup(): void {
    clearInterval(this.interval);
    this.interval = undefined;
  }
}

export const numericPropsZ = z.number();

export type NumericProps = z.infer<typeof numericPropsZ>;

export class Numeric implements telem.NumericSource {
  static readonly TYPE = "static-point";

  variant = "numeric";

  key: string;
  _value: number;

  constructor(key: string, props_: any) {
    this.key = key;
    this._value = numericPropsZ.parse(props_);
  }

  setProps(props_: any): void {
    this._value = numericPropsZ.parse(props_);
  }

  async value(): Promise<number> {
    return this._value;
  }

  onChange(): void {}

  release(): void {}

  cleanup(): void {}

  invalidate(): void {}
}

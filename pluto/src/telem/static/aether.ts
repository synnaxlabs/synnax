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
  GLBufferController,
  Bounds,
  nativeTypedArray,
  Rate,
  DataType,
  TimeRange,
} from "@synnaxlabs/x";
import { z } from "zod";

import { XYTelemSource } from "@/core/vis/telem";
import { NumericTelemSource } from "@/core/vis/telem/TelemSource";
import { TelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";

export class StaticTelemFactory implements TelemFactory {
  type = "static";

  create(key: string, type: string, props: any): ModifiableTelemSourceMeta | null {
    switch (type) {
      case StaticXYTelem.TYPE:
        return new StaticXYTelem(key, props);
      case IterativeXYTelem.TYPE:
        return new IterativeXYTelem(key, props);
      case StaticPointTelem.TYPE:
        return new StaticPointTelem(key, props);
      default:
        return null;
    }
  }
}

export const staticXYTelemProps = z.object({
  x: z.array(nativeTypedArray),
  y: z.array(nativeTypedArray),
  xOffsets: z.array(z.number()).optional().default([]),
  yOffsets: z.array(z.number()).optional().default([]),
});

export type StaticXYTelemProps = z.input<typeof staticXYTelemProps>;

class StaticXYTelemCore {
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

  async yBounds(): Promise<Bounds> {
    const y = await this.y();
    return Bounds.max(y.map((x) => x.bounds));
  }

  async xBounds(): Promise<Bounds> {
    const x = await this.x();
    return Bounds.max(x.map((x) => x.bounds));
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

export class StaticXYTelem extends StaticXYTelemCore implements XYTelemSource {
  static readonly TYPE = "static-xy";

  variant = "xy";

  constructor(key: string, props_: any) {
    const props = staticXYTelemProps.parse(props_);
    super(
      key,
      props.x.map((x, i) => {
        const arr = new Series(
          x,
          DataType.FLOAT32,
          TimeRange.ZERO,
          props.xOffsets[i] ?? 0
        );
        arr.acquire();
        return arr;
      }),
      props.y.map((y, i) => {
        const arr = new Series(
          y,
          DataType.FLOAT32,
          TimeRange.ZERO,
          props.yOffsets[i] ?? 0
        );
        arr.acquire();
        return arr;
      })
    );
  }

  setProps(props_: any): void {
    const props = staticXYTelemProps.parse(props_);
    this._x = props.x.map(
      (x, i) => new Series(x, DataType.FLOAT32, TimeRange.ZERO, props.xOffsets[i] ?? 0)
    );
    this._y = props.y.map(
      (y, i) => new Series(y, DataType.FLOAT32, TimeRange.ZERO, props.yOffsets[i] ?? 0)
    );
    this.onChangeHandler?.();
  }
}

export const iterativeXYTelemProps = staticXYTelemProps.extend({
  rate: Rate.z,
  yOffset: z.number().optional().default(0),
});

export type IterativeXYTelemProps = z.input<typeof iterativeXYTelemProps>;

export class IterativeXYTelem extends StaticXYTelemCore implements XYTelemSource {
  position: number;
  interval?: number;

  static readonly TYPE = "iterative-xy";

  variant = "xy";

  constructor(key: string, props_: any) {
    const { x, y, rate, yOffset } = iterativeXYTelemProps.parse(props_);
    super(
      key,
      x.map((x) => new Series(x)),
      y.map((y) => new Series(y, DataType.FLOAT32, TimeRange.ZERO, yOffset))
    );
    this.position = 0;
    this.start(rate);
  }

  setProps(props_: any): void {
    const props = iterativeXYTelemProps.parse(props_);
    this._x = props.x.map((x) => new Series(x));
    this._y = props.y.map(
      (y) => new Series(y, DataType.FLOAT32, TimeRange.ZERO, props.yOffset)
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

export const staticPointTelemProps = z.number();

export type StaticPointTelemProps = z.infer<typeof staticPointTelemProps>;

export class StaticPointTelem implements NumericTelemSource {
  static readonly TYPE = "static-point";

  variant = "numeric";

  key: string;
  _value: number;

  constructor(key: string, props_: any) {
    this.key = key;
    this._value = staticPointTelemProps.parse(props_);
  }

  setProps(props_: any): void {
    this._value = staticPointTelemProps.parse(props_);
  }

  async value(): Promise<number> {
    return this._value;
  }

  onChange(): void {}

  release(): void {}

  cleanup(): void {}

  invalidate(): void {}
}

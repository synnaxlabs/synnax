// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  LazyArray,
  GLBufferController,
  Bounds,
  nativeTypedArray,
  Rate,
  DataType,
  TimeRange,
} from "@synnaxlabs/x";
import { z } from "zod";

import { XYTelemSource } from "@/core/vis/telem";
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
  _x: LazyArray[];
  _y: LazyArray[];
  onChangeHandler?: () => void;

  constructor(key: string, x: LazyArray[], y: LazyArray[]) {
    this.key = key;
    this._x = x;
    this._y = y;
  }

  async x(gl?: GLBufferController): Promise<LazyArray[]> {
    if (gl != null) this._x.map((x) => x.updateGLBuffer(gl));
    return this._x;
  }

  async y(gl?: GLBufferController): Promise<LazyArray[]> {
    if (gl != null) this._y.map((y) => y.updateGLBuffer(gl));
    return this._y;
  }

  async xBounds(): Promise<Bounds> {
    return Bounds.max(this._x.map((x) => x.bounds));
  }

  async yBounds(): Promise<Bounds> {
    return Bounds.max(this._y.map((y) => y.bounds));
  }

  release(gl: GLBufferController): void {
    this._x.map((x) => x.release(gl));
    this._y.map((y) => y.release(gl));
  }

  onChange(f: () => void): void {
    this.onChangeHandler = f;
  }

  invalidate(): void {
    if (this.onChangeHandler != null) this.onChangeHandler();
  }

  cleanup(): void {}
}

export class StaticXYTelem extends StaticXYTelemCore implements XYTelemSource {
  static readonly TYPE = "static";

  variant = "xy";

  constructor(key: string, props_: any) {
    const props = staticXYTelemProps.parse(props_);
    super(
      key,
      props.x.map((x, i) => {
        const arr = new LazyArray(
          x,
          DataType.FLOAT32,
          TimeRange.ZERO,
          props.xOffsets[i] ?? 0
        );
        arr.acquire();
        return arr;
      }),
      props.y.map((y, i) => {
        const arr = new LazyArray(
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
      (x, i) =>
        new LazyArray(x, DataType.FLOAT32, TimeRange.ZERO, props.xOffsets[i] ?? 0)
    );
    this._y = props.y.map(
      (y, i) =>
        new LazyArray(y, DataType.FLOAT32, TimeRange.ZERO, props.yOffsets[i] ?? 0)
    );
    this.onChangeHandler?.();
  }
}

export const iterativeXYTelemProps = staticXYTelemProps.extend({
  rate: Rate.z,
  yOffset: z.number().optional().default(0),
});

export type IterativeXYTelemProps = z.infer<typeof iterativeXYTelemProps>;

export class IterativeXYTelem extends StaticXYTelemCore implements XYTelemSource {
  position: number;
  interval?: number;

  static readonly TYPE = "iterative";

  variant = "xy";

  constructor(key: string, props_: any) {
    const { x, y, rate, yOffset } = iterativeXYTelemProps.parse(props_);
    super(
      key,
      x.map((x) => new LazyArray(x)),
      y.map((y) => new LazyArray(y, DataType.FLOAT32, TimeRange.ZERO, yOffset))
    );
    this.position = 0;
    this.start(rate);
  }

  setProps(props_: any): void {
    const props = iterativeXYTelemProps.parse(props_);
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map(
      (y) => new LazyArray(y, DataType.FLOAT32, TimeRange.ZERO, props.yOffset)
    );
  }

  async x(gl?: GLBufferController): Promise<LazyArray[]> {
    const x = this._x.map((x) => x.slice(0, this.position));
    if (gl != null) x.map((x) => x.updateGLBuffer(gl));
    return x;
  }

  async y(gl?: GLBufferController): Promise<LazyArray[]> {
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

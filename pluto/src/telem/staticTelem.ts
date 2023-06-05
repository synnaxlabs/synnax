import { Bound, GLBufferControl, LazyArray, NativeTypedArray } from "@synnaxlabs/x";

import {
  DynamicXYTelemSource,
  TelemSourceMeta,
  XYTelemSource,
} from "@/core/vis/telem/TelemSource";

export interface StaticTelemProps {
  x: NativeTypedArray[];
  y: NativeTypedArray[];
}

export class StaticTelem implements XYTelemSource {
  _x: LazyArray[];
  _y: LazyArray[];
  key: string;

  type = "xy";

  constructor(key: string, props: StaticTelemProps) {
    this.key = key;
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }

  async x(gl: GLBufferControl): Promise<LazyArray[]> {
    this._x.map((x) => x.updateGLBuffer(gl));
    return this._x;
  }

  async y(gl: GLBufferControl): Promise<LazyArray[]> {
    this._y.map((y) => y.updateGLBuffer(gl));
    return this._y;
  }

  async xBound(): Promise<Bound> {
    return Bound.max(this._x.map((x) => x.bound));
  }

  async yBound(): Promise<Bound> {
    return Bound.max(this._y.map((y) => y.bound));
  }

  setProps(props: StaticTelemProps): void {
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }
}

export interface DynamicTelemProps {
  x: NativeTypedArray[];
  y: NativeTypedArray[];
  updateRate: number;
}

export class DynamicTelem implements DynamicXYTelemSource {
  _x: LazyArray[];
  _y: LazyArray[];
  key: string;
  handler: (() => void) | null;
  position: number;

  type = "xy";

  constructor(key: string, props: DynamicTelemProps) {
    this.key = key;
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
    this.handler = null;
    this.position = 0;

    setInterval(() => {
      if (this.handler != null) this.handler();
      this.position++;
    }, props.updateRate);
  }

  onChange(f: () => void): void {
    this.handler = f;
  }

  async x(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (gl != null) this._x.map((x) => x.updateGLBuffer(gl));
    return this._x.map((x) => x.slice(0, this.position));
  }

  async y(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (gl != null) this._y.map((y) => y.updateGLBuffer(gl));
    return this._y.map((y) => y.slice(0, this.position));
  }

  async xBound(): Promise<Bound> {
    return Bound.max(this._x.map((x) => x.bound));
  }

  async yBound(): Promise<Bound> {
    return Bound.max(this._y.map((y) => y.bound));
  }

  setProps(props: DynamicTelemProps): void {
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }
}

export class StaticTelemFactory {
  new(
    key: string,
    type: string,
    props: StaticTelemProps | DynamicTelemProps
  ): TelemSourceMeta {
    if (type === "dynamic") return new DynamicTelem(key, props as DynamicTelemProps);
    return new StaticTelem(key, props);
  }
}

import {
  Bound,
  GLBufferControl,
  LazyArray,
  NativeTypedArray,
  maxBound,
} from "@synnaxlabs/x";

import { XYTelemSource } from "@/core/vis/telem/TelemSource";

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

  async x(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (gl != null) this._x.map((x) => x.updateGLBuffer(gl));
    return this._x;
  }

  async y(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (gl != null) this._y.map((y) => y.updateGLBuffer(gl));
    return this._y;
  }

  async xBound(): Promise<Bound> {
    return maxBound(this._x.map((x) => x.bound));
  }

  async yBound(): Promise<Bound> {
    return maxBound(this._y.map((y) => y.bound));
  }

  setProps(props: StaticTelemProps): void {
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }
}

export class StaticTelemFactory {
  new(key: string, props: StaticTelemProps): StaticTelem {
    return new StaticTelem(key, props);
  }
}

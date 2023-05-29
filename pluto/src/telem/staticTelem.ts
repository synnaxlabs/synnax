import {
  Bound,
  GLBufferControl,
  LazyArray,
  NativeTypedArray,
  maxBound,
} from "@synnaxlabs/x";

import { useTelemSourceControl } from "./Context";

import { XYTelemSource, XYTelemSourceMeta } from "@/core/vis/telem/TelemSource";

export interface UseStaticTelemProps {
  x: NativeTypedArray[];
  y: NativeTypedArray[];
}

export const useStaticTelem = (props: UseStaticTelemProps): XYTelemSourceMeta => {
  const key = useTelemSourceControl("static", props, [
    ...props.x.map((x) => x.buffer),
    ...props.y.map((y) => y.buffer),
  ]);
  return {
    type: "xy",
    key,
  };
};

export class StaticTelem implements XYTelemSource {
  _x: LazyArray[];
  _y: LazyArray[];
  key: string;

  type = "xy";

  constructor(key: string, props: UseStaticTelemProps) {
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

  setProps(props: UseStaticTelemProps): void {
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }
}

export class StaticTelemFactory {
  new(key: string, props: UseStaticTelemProps): StaticTelem {
    return new StaticTelem(key, props);
  }
}

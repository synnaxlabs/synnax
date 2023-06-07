// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LazyArray, GLBufferControl, Bound, nativeTypedArray } from "@synnaxlabs/x";
import { z } from "zod";

import { XYTelemSource } from "@/core/vis/telem";

export const staticXYTelemProps = z.object({
  x: z.array(nativeTypedArray),
  y: z.array(nativeTypedArray),
});

export type StaticXYTelemProps = z.infer<typeof staticXYTelemProps>;

export class StaticXYTelem implements XYTelemSource {
  key: string;
  _x: LazyArray[];
  _y: LazyArray[];

  type = "xy";

  constructor(key: string, props_: any) {
    const props = staticXYTelemProps.parse(props_);
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
    return Bound.max(this._x.map((x) => x.bound));
  }

  async yBound(): Promise<Bound> {
    return Bound.max(this._y.map((y) => y.bound));
  }

  setProps(props_: any): void {
    const props = staticXYTelemProps.parse(props_);
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }
}

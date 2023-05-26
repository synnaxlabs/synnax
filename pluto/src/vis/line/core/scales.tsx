// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Deep, dirToDim, Scale } from "@synnaxlabs/x";

import { Bounds } from "./bounds";

import { axisDirection, AxisKey } from "@/vis/Axis";
import { Viewport } from "@/vis/line/core/viewport";

export interface InternalState {
  normal: Partial<Record<AxisKey, Scale>>;
  offset: Partial<Record<AxisKey, Scale>>;
  decimal: Partial<Record<AxisKey, Scale>>;
}

const ZERO_INTERNAL_STATE: InternalState = {
  normal: {},
  offset: {},
  decimal: {},
};

export class Scales {
  private state: InternalState;

  constructor() {
    this.state = Deep.copy(ZERO_INTERNAL_STATE);
  }

  build(viewport: Viewport, bounds: Bounds): void {
    const scales = Deep.copy(ZERO_INTERNAL_STATE);
    bounds.forEach((key, normal, offset) => {
      const dir = axisDirection(key);
      const dim = dirToDim(dir);
      const loc = dir === "x" ? "left" : "bottom";
      const mag = 1 / viewport.box[dim];
      const trans = -viewport.box[loc];
      const decimal = Scale.scale(normal).scale(1);
      scales.decimal[key] = decimal;
      scales.normal[key] = decimal.translate(trans).magnify(mag);
      scales.offset[key] = Scale.scale(offset).scale(1).translate(trans).magnify(mag);
    });
    this.state = scales;
  }

  forEach(
    fn: (key: AxisKey, normal: Scale, offset: Scale, decimal: Scale) => void
  ): void {
    (Object.keys(this.state.normal) as AxisKey[]).forEach((key) => {
      const normal = this.state.normal[key] as Scale;
      const offset = this.state.offset[key] as Scale;
      const decimal = this.state.decimal[key] as Scale;
      fn(key, normal, offset, decimal);
    });
  }

  hasAxis(key: AxisKey): boolean {
    return key in this.state.normal;
  }

  offset(key: AxisKey): Scale | undefined {
    return this.state.offset[key];
  }

  normal(key: AxisKey): Scale | undefined {
    return this.state.normal[key];
  }

  decimal(key: AxisKey): Scale | undefined {
    return this.state.decimal[key];
  }

  get valid(): boolean {
    return Object.keys(this.state.normal).length > 0;
  }
}

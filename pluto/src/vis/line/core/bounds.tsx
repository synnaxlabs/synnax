// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { addSamples, Bound, Deep, SampleValue, ZERO_BOUND } from "@synnaxlabs/x";

import { VisArray } from "@/telem/visArray";
import { AxisKey, Y_AXIS_KEYS, YAxisKey } from "@/vis/Axis";
import { Telem } from "@/vis/line/core/telem";

export interface InternalState {
  normal: Partial<Record<AxisKey, Bound>>;
  offset: Partial<Record<AxisKey, Bound>>;
}

export interface BoundState {
  driven: boolean;
  bound: Bound;
}

export type BoundsState = Record<AxisKey, BoundState>;

const ZERO_BOUND_STATE: BoundState = {
  driven: false,
  bound: ZERO_BOUND,
};

const ZERO_BOUNDS_STATE: BoundsState = {
  x1: ZERO_BOUND_STATE,
  x2: ZERO_BOUND_STATE,
  y1: ZERO_BOUND_STATE,
  y2: ZERO_BOUND_STATE,
  y3: ZERO_BOUND_STATE,
  y4: ZERO_BOUND_STATE,
};

const buildBound = (
  arrays: VisArray[],
  padding: number,
  includeOffset: boolean
): Bound => {
  const upper = Number(
    arrays.reduce((acc: SampleValue, arr) => {
      let max = arr.arr.max;
      if (!includeOffset) max = addSamples(max, -arr.arr.sampleOffset);
      return max > acc ? max : acc;
    }, -Infinity)
  );
  const lower = Number(
    arrays.reduce((acc: SampleValue, arr) => {
      let min = arr.arr.min;
      if (!includeOffset) min = addSamples(min, -arr.arr.sampleOffset);
      return min < acc ? min : acc;
    }, Infinity)
  );
  const _padding = (upper - lower) * padding;
  if (upper === lower) return { lower: lower - 1, upper: upper + 1 };
  return { lower: lower - _padding, upper: upper + _padding };
};

export class Bounds {
  private state: BoundsState;
  private internal: InternalState;

  constructor() {
    this.internal = {
      normal: {},
      offset: {},
    };
    this.state = Bounds.zeroState();
  }

  static zeroState(): BoundsState {
    return Deep.copy(ZERO_BOUNDS_STATE);
  }

  update(state: BoundsState): void {
    this.state = state;
  }

  build(telem: Telem, padding: number): void {
    this.internal = { normal: {}, offset: {} };
    telem.forEachAxis((key, data) => {
      if (data.length === 0) return;
      const flat = data.flatMap((d) => d.data);
      const addPadding = Y_AXIS_KEYS.includes(key as YAxisKey);
      const coreBound = this.state[key];
      if (!coreBound.driven) {
        this.internal.normal[key] = buildBound(flat, addPadding ? padding : 0, false);
        this.internal.offset[key] = buildBound(flat, addPadding ? padding : 0, true);
      } else {
        this.internal.normal[key] = coreBound.bound;
        this.internal.offset[key] = {
          lower: 0,
          upper: coreBound.bound.upper - coreBound.bound.lower,
        };
      }
    });
    const toUpdate: Partial<Record<AxisKey, BoundState>> = {};
    (Object.entries(this.internal.normal) as Array<[AxisKey, Bound]>).forEach(
      ([key, bound]) => {
        const coreBound = this.state[key];
        if (coreBound.driven) return;
        toUpdate[key] = {
          ...coreBound,
          bound,
        };
      }
    );
  }

  forEach(callback: (key: AxisKey, normal: Bound, offset: Bound) => void): void {
    (Object.keys(this.internal.normal) as AxisKey[]).forEach((key: AxisKey) =>
      callback(
        key,
        this.internal.normal[key] as Bound,
        this.internal.offset[key] as Bound
      )
    );
  }
}

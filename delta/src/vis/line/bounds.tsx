import { useMemo } from "react";

import { addSamples, Bound, SampleValue } from "@synnaxlabs/x";

import { AxisKey, YAxisKey, Y_AXIS_KEYS } from "@/vis/axis";
import { Data } from "@/vis/line/data";
import { TelemetryClientResponse } from "@/vis/telem/client";

export interface BoundsCoreState {
  normal: Partial<Record<AxisKey, Bound>>;
  offset: Partial<Record<AxisKey, Bound>>;
}

const buildBound = (
  data: TelemetryClientResponse[],
  padding: number,
  includeOffset: boolean
): Bound => {
  const arrays = data.flatMap(({ arrays }) => arrays);
  const upper = Number(
    arrays.reduce((acc: SampleValue, arr) => {
      let max = arr.max;
      if (!includeOffset) max = addSamples(max, -arr.offset);
      return max > acc ? max : acc;
    }, -Infinity)
  );
  const lower = Number(
    arrays.reduce((acc: SampleValue, arr) => {
      let min = arr.min;
      if (!includeOffset) min = addSamples(min, -arr.offset);
      return min < acc ? min : acc;
    }, Infinity)
  );
  const _padding = (upper - lower) * padding;
  if (upper === lower) return { lower: lower - 1, upper: upper + 1 };
  return { lower: lower - _padding, upper: upper + _padding };
};

export class Bounds {
  private readonly core: BoundsCoreState;

  constructor(state: BoundsCoreState) {
    this.core = state;
  }

  static use(data: Data, padding: number): Bounds {
    return useMemo(() => {
      const state: BoundsCoreState = { normal: {}, offset: {} };
      data.forEachAxis((key, responses) => {
        if (responses.length === 0) return;
        const addPadding = Y_AXIS_KEYS.includes(key as YAxisKey);
        state.normal[key] = buildBound(responses, addPadding ? padding : 0, false);
        state.offset[key] = buildBound(responses, addPadding ? padding : 0, true);
      });
      return new Bounds(state);
    }, [data, padding]);
  }

  forEach(callback: (key: AxisKey, normal: Bound, offset: Bound) => void): void {
    (Object.keys(this.core.normal) as AxisKey[]).forEach((key: AxisKey) =>
      callback(key, this.core.normal[key] as Bound, this.core.offset[key] as Bound)
    );
  }

  get valid(): boolean {
    return Object.keys(this.core.normal).length > 0;
  }
}

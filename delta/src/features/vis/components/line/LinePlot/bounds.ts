import { addSamples, Bound, SampleValue } from "@synnaxlabs/x";

import { LineVisData } from "./data";

import { TelemetryClientResponse } from "@/features/vis/telem/client";
import { AxisKey } from "@/features/vis/types";

export interface BoundsState {
  normal: Partial<Record<AxisKey, Bound>>;
  offset: Partial<Record<AxisKey, Bound>>;
}

const initial = (): BoundsState => ({
  offset: {},
  normal: {},
});

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

const build = (data: LineVisData, padding: number): BoundsState => {
  const state: BoundsState = initial();
  (Object.keys(data) as AxisKey[]).forEach((key) => {
    if (data[key].length === 0) return;
    state.normal[key] = buildBound(data[key], 0, false);
    state.offset[key] = buildBound(data[key], 0, true);
  });
  return state;
};

export const Bounds = {
  initial,
  build,
};

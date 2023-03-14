import { addSamples, Bound, SampleValue } from "@synnaxlabs/x";

import { Data } from "./data";

import { TelemetryClientResponse } from "@/features/vis/telem/client";
import { AxisKey, YAxisKey, Y_AXIS_KEYS } from "@/features/vis/types";

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

const build = (data: Data, padding: number): BoundsState => {
  const state: BoundsState = initial();
  data.forEachAxis((key, responses) => {
    if (responses.length === 0) return;
    const addPadding = Y_AXIS_KEYS.includes(key as YAxisKey);
    state.normal[key] = buildBound(responses, addPadding ? padding : 0, false);
    state.offset[key] = buildBound(responses, addPadding ? padding : 0, true);
  });
  return state;
};

export const Bounds = {
  initial,
  build,
};

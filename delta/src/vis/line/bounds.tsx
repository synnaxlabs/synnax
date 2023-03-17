import { useMemo } from "react";

import { addSamples, Bound, SampleValue } from "@synnaxlabs/x";

import { selectRequiredVis, VisStoreState } from "../store";

import { BoundsState, BoundState, LineVis } from "./core";

import { useMemoSelect } from "@/hooks";
import { LayoutStoreState } from "@/layout";
import { AxisKey, YAxisKey, Y_AXIS_KEYS } from "@/vis/axis";
import { Data } from "@/vis/line/data";
import { TelemetryClientResponse } from "@/vis/telem/client";

export interface InternalState {
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
  private readonly core: InternalState;

  constructor(state: InternalState) {
    this.core = state;
  }

  static useSelect(key: string): BoundsState {
    return useMemoSelect(
      (state: VisStoreState & LayoutStoreState) =>
        selectRequiredVis<LineVis>(state, key, "line").bounds,
      [key]
    );
  }

  static use(key: string, data: Data, padding: number): Bounds {
    const core = Bounds.useSelect(key);
    return useMemo(() => {
      const state: InternalState = { normal: {}, offset: {} };
      data.forEachAxis((key, responses) => {
        if (responses.length === 0) return;
        const addPadding = Y_AXIS_KEYS.includes(key as YAxisKey);
        const coreBound = core[key];
        if (!coreBound.driven) {
          state.normal[key] = buildBound(responses, addPadding ? padding : 0, false);
          state.offset[key] = buildBound(responses, addPadding ? padding : 0, true);
        } else {
          state.normal[key] = coreBound.bound;
          state.offset[key] = {
            lower: 0,
            upper: coreBound.bound.upper - coreBound.bound.lower,
          };
        }
      });
      const toUpdate: Partial<Record<AxisKey, BoundState>> = {};
      (Object.entries(state.normal) as Array<[AxisKey, Bound]>).forEach(
        ([key, bound]) => {
          const coreBound = core[key];
          if (coreBound.driven) return;
          toUpdate[key] = {
            ...coreBound,
            bound,
          };
        }
      );

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

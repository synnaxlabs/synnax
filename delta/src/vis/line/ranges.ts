// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemo } from "react";

import { TimeStamp } from "@synnaxlabs/x";

import { useMemoSelect } from "@/hooks";
import { LayoutStoreState } from "@/layout";
import { LineVis } from "@/vis/line/core";
import { selectRequiredVis, VisualizationStoreState } from "@/vis/store";
import { XAxisKey, XAxisRecord, X_AXIS_KEYS } from "@/vis/types";
import { Range, selectRanges, WorkspaceStoreState } from "@/workspace";

export type RangesCoreState = XAxisRecord<readonly string[]>;

export const ZERO_RANGES_CORE_STATE: RangesCoreState = {
  x1: [] as string[],
  x2: [] as string[],
};

export class Ranges {
  readonly core: RangesCoreState;
  readonly ranges: Record<string, Range>;

  private constructor(core: RangesCoreState, ranges: Record<string, Range>) {
    this.core = core;
    this.ranges = ranges;
  }

  static use(key: string): Ranges {
    const { core, ranges } = useMemoSelect(
      (state: VisualizationStoreState & LayoutStoreState & WorkspaceStoreState) => {
        const core = selectRequiredVis<LineVis>(state, "line", key).ranges;
        const ranges = Ranges.rangesFromArray(selectRanges(state, Ranges.keys(core)));
        return { core, ranges };
      },
      [key]
    );
    return useMemo(() => new Ranges(core, ranges), [core, ranges]);
  }

  private static keys(core: RangesCoreState): string[] {
    return Object.values(core).flat();
  }

  private static rangesFromArray(ranges: Range[]): Record<string, Range> {
    return Object.fromEntries(ranges.map((r) => [r.key, r]));
  }

  forEach(callback: (range: Range, axes: XAxisKey[]) => void): void {
    Object.entries(this.ranges).forEach(([key, range]) => {
      const axes = X_AXIS_KEYS.filter((axis) => this.core[axis].includes(key));
      callback(range, axes);
    });
  }

  get array(): Range[] {
    return Object.values(this.ranges);
  }

  get isLive(): boolean {
    const now = TimeStamp.now();
    return this.array.some((r) => new TimeStamp(r.end).after(now));
  }
}

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Deep } from "@synnaxlabs/x";

import { Range, RangePayload, rangeFromPayload } from "@/telem/range";
import { XAxisKey, XAxisRecord, X_AXIS_KEYS } from "@/core/vis/Axis";

export type RangesState = XAxisRecord<readonly RangePayload[]>;

const ZERO_RANGES_STATE: RangesState = {
  x1: [] as RangePayload[],
  x2: [] as RangePayload[],
};

export class Ranges {
  private state: RangesState;

  constructor() {
    this.state = Deep.copy(ZERO_RANGES_STATE);
  }

  static zeroState(): RangesState {
    return Deep.copy(ZERO_RANGES_STATE);
  }

  update(state: RangesState): void {
    this.state = state;
  }

  forEach(callback: (range: Range, axes: XAxisKey[]) => void): void {
    this.array.forEach((range) => {
      const axes = X_AXIS_KEYS.filter((axis) =>
        this.state[axis].map((r) => r.key).includes(range.key)
      );
      callback(range, axes);
    });
  }

  axis(key: XAxisKey): readonly Range[] {
    return this.state[key].map((r) => rangeFromPayload(r));
  }

  get array(): Range[] {
    return Object.values(this.state)
      .flat()
      .map((r) => rangeFromPayload(r));
  }
}

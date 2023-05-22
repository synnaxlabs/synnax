// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Deep, TimeStamp } from "@synnaxlabs/x";

import { XAxisKey, X_AXIS_KEYS } from "@/vis/axis";
import {
  ZERO_RANGES_STATE,
  GOOD_STATUS,
  RangesState,
  Status,
  StatusProvider,
  Range,
} from "@/vis/line/core";

export class Ranges implements StatusProvider {
  readonly ranges: RangesState;
  readonly status: Status;

  private constructor(state: RangesState, status: Status = GOOD_STATUS) {
    this.ranges = state;
    this.status = status;
  }

  static use(state: RangesState): Ranges {
    return new Ranges(state);
  }

  static isValid(core: RangesState): boolean {
    return Object.values(core).flat().length > 0;
  }

  private static rangesFromArray(ranges: Range[]): Record<string, Range> {
    return Object.fromEntries(ranges.map((r) => [r.key, r]));
  }

  forEach(callback: (range: Range, axes: XAxisKey[]) => void): void {
    this.array.forEach((range) => {
      const axes = X_AXIS_KEYS.filter((axis) =>
        this.ranges[axis].map((r) => r.key).includes(range.key)
      );
      callback(range, axes);
    });
  }

  axis(key: XAxisKey): readonly Range[] {
    return this.ranges[key];
  }

  get array(): Range[] {
    return Object.values(this.ranges).flat();
  }

  get isLive(): boolean {
    const now = TimeStamp.now();
    return this.array.some((r) => r.range.end.after(now));
  }

  get valid(): boolean {
    return this.array.length > 0;
  }

  static zero(): Ranges {
    return new Ranges(Deep.copy(ZERO_RANGES_STATE));
  }
}

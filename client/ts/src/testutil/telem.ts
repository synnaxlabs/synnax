// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type DataType,
  type TimeSpan,
  TimeStamp,
  type TypedArray,
} from "@synnaxlabs/x";

export const secondsLinspace = (start: number, n: number): TimeStamp[] =>
  Array.from({ length: n }, (_, i) => start + i).map((n) => TimeStamp.seconds(n));

/** A clock that moves only when advanced. */
export class ManualClock {
  private time: TimeStamp;

  constructor(start: TimeStamp = TimeStamp.seconds(1000)) {
    this.time = start;
  }

  /** @returns the time on the clock. */
  now = (): TimeStamp => this.time;

  /** Moves the clock forward by the span. */
  advance(span: TimeSpan): void {
    this.time = this.time.add(span);
  }
}

export const randomSeries = (length: number, dataType: DataType): TypedArray => {
  const bytes = new Uint8Array(length * dataType.density.valueOf());
  for (let i = 0; i < bytes.byteLength; i++) bytes[i] = Math.floor(Math.random() * 256);
  return new dataType.Array(bytes.buffer);
};

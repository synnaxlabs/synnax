// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeStamp, TimeSpan, TimeStamp } from "@synnaxlabs/x";

/** Tick step sizes, from 1 nanosecond to 1 second in 1-2-5 increments. */
export const TIME_SCALE_STEPS: TimeSpan[] = [
  TimeSpan.NANOSECOND,
  TimeSpan.nanoseconds(2),
  TimeSpan.nanoseconds(5),
  TimeSpan.nanoseconds(10),
  TimeSpan.nanoseconds(20),
  TimeSpan.nanoseconds(50),
  TimeSpan.nanoseconds(100),
  TimeSpan.nanoseconds(200),
  TimeSpan.nanoseconds(500),
  TimeSpan.MICROSECOND,
  TimeSpan.microseconds(2),
  TimeSpan.microseconds(5),
  TimeSpan.microseconds(10),
  TimeSpan.microseconds(20),
  TimeSpan.microseconds(50),
  TimeSpan.microseconds(100),
  TimeSpan.microseconds(200),
  TimeSpan.microseconds(500),
  TimeSpan.MILLISECOND,
  TimeSpan.milliseconds(2),
  TimeSpan.milliseconds(5),
  TimeSpan.milliseconds(10),
  TimeSpan.milliseconds(20),
  TimeSpan.milliseconds(50),
  TimeSpan.milliseconds(100),
  TimeSpan.milliseconds(200),
  TimeSpan.milliseconds(500),
  TimeSpan.SECOND,
];

const MICROSECOND_FORMAT_THRESHOLD = TimeSpan.microseconds(50);

export interface PreciseTimeScaleProps {
  /** The domain of the time scale as [start, end] timestamps */
  domain: [CrudeTimeStamp, CrudeTimeStamp];
  /** The range of the scale as [start, end] numbers for visual representation */
  range: [number, number];
}

/**
 * Maps a time domain onto a numeric range, holding nanosecond precision in BigInt
 * throughout.
 *
 * @example preciseTimeScale().domain([start, end]).range([0, 1000]).scale(t)
 */
export class PreciseTimeScale {
  private _domain: [TimeStamp, TimeStamp];
  private _range: [number, number];
  private _span: TimeSpan;

  constructor() {
    this._domain = [new TimeStamp(0n), new TimeStamp(1n)];
    this._range = [0, 1];
    this._span = this._domain[1].span(this._domain[0]);
  }

  /** Reads the time domain, or sets it and returns this for chaining. */
  domain(): [TimeStamp, TimeStamp];
  domain(domain: [CrudeTimeStamp, CrudeTimeStamp]): this;
  domain(domain?: [CrudeTimeStamp, CrudeTimeStamp]): [TimeStamp, TimeStamp] | this {
    if (domain === undefined) return this._domain;
    this._domain = [new TimeStamp(domain[0]), new TimeStamp(domain[1])];
    this._span = this._domain[1].span(this._domain[0]);
    return this;
  }

  /** Reads the numeric range, or sets it and returns this for chaining. */
  range(): [number, number];
  range(range: [number, number]): this;
  range(range?: [number, number]): [number, number] | this {
    if (range === undefined) return this._range;
    this._range = range;
    return this;
  }

  /** @returns the position of the timestamp within the range. */
  scale(value: CrudeTimeStamp): number {
    const ts = new TimeStamp(value);
    const v = ts.valueOf();
    const start = this._domain[0].valueOf();
    const span = this._domain[1].valueOf() - start;
    const rangeSpan = this._range[1] - this._range[0];
    return (Number(v - start) / Number(span)) * rangeSpan + this._range[0];
  }

  /**
   * @returns evenly spaced tick positions across the domain. The count is a target: the
   * step is rounded to a {@link TIME_SCALE_STEPS} interval, so the result may differ.
   */
  ticks(count: number): TimeStamp[] {
    const step = this.calculateOptimalStep(count).valueOf();
    const start = this._domain[0].valueOf();
    const end = this._domain[1].valueOf();
    const alignedStart = ((start + step - 1n) / step) * step;
    const stops = Number((end - alignedStart) / step);
    return Array.from({ length: stops + 1 }, (_, i) => {
      const nanoValue = alignedStart + BigInt(i) * step;
      return new TimeStamp(nanoValue);
    }).filter((ts) => ts.afterEq(this._domain[0]) && ts.beforeEq(this._domain[1]));
  }

  /** @returns the smallest {@link TIME_SCALE_STEPS} entry that fits targetCount ticks. */
  private calculateOptimalStep(targetCount: number): TimeSpan {
    const rawStepNanoseconds = this._span.valueOf() / BigInt(Math.ceil(targetCount));
    let bestStep = TIME_SCALE_STEPS[0];
    for (const step of TIME_SCALE_STEPS) {
      if (step.valueOf() > rawStepNanoseconds) break;
      bestStep = step;
    }
    return bestStep;
  }

  /** Formats a tick label, in microseconds below a 50µs span and milliseconds above. */
  formatTick(value: TimeStamp): string {
    if (this._span.lessThan(MICROSECOND_FORMAT_THRESHOLD)) {
      const remainder = value.remainder(TimeSpan.MILLISECOND);
      return `${remainder.microseconds.toString()}µs`;
    }
    const remainder = value.remainder(TimeSpan.SECOND);
    return `${remainder.milliseconds.toString()}ms`;
  }
}

export const preciseTimeScale = (): PreciseTimeScale => new PreciseTimeScale();

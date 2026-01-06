// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeStamp, TimeSpan, TimeStamp } from "@synnaxlabs/x";

/**
 * Predefined time scale steps used for generating tick marks.
 * Ranges from 1 nanosecond to 1 second in standardized increments.
 */
export const TIME_SCALE_STEPS: TimeSpan[] = [
  TimeSpan.NANOSECOND, // 1ns
  TimeSpan.nanoseconds(2), // 2ns
  TimeSpan.nanoseconds(5), // 5ns
  TimeSpan.nanoseconds(10), // 10ns
  TimeSpan.nanoseconds(20), // 20ns
  TimeSpan.nanoseconds(50), // 50ns
  TimeSpan.nanoseconds(100), // 100ns
  TimeSpan.nanoseconds(200), // 200ns
  TimeSpan.nanoseconds(500), // 500ns
  TimeSpan.MICROSECOND, // 1µs
  TimeSpan.microseconds(2), // 2µs
  TimeSpan.microseconds(5), // 5µs
  TimeSpan.microseconds(10), // 10µs
  TimeSpan.microseconds(20), // 20µs
  TimeSpan.microseconds(50), // 50µs
  TimeSpan.microseconds(100), // 100µs
  TimeSpan.microseconds(200), // 200µs
  TimeSpan.microseconds(500), // 500µs
  TimeSpan.MILLISECOND, // 1ms
  TimeSpan.milliseconds(2), // 2ms
  TimeSpan.milliseconds(5), // 5ms
  TimeSpan.milliseconds(10), // 10ms
  TimeSpan.milliseconds(20), // 20ms
  TimeSpan.milliseconds(50), // 50ms
  TimeSpan.milliseconds(100), // 100ms
  TimeSpan.milliseconds(200), // 200ms
  TimeSpan.milliseconds(500), // 500ms
  TimeSpan.SECOND, // 1s
];

/**
 * Configuration properties for creating a PreciseTimeScale.
 */
export interface PreciseTimeScaleProps {
  /** The domain of the time scale as [start, end] timestamps */
  domain: [CrudeTimeStamp, CrudeTimeStamp];
  /** The range of the scale as [start, end] numbers for visual representation */
  range: [number, number];
}

/**
 * PreciseTimeScale provides a high-precision time scaling utility for visualizing time-based data.
 * It supports nanosecond precision and provides methods for scaling timestamps to pixel coordinates,
 * generating tick marks, and formatting time values.
 *
 * The scale maps a time domain to a numeric range (typically pixels) while maintaining
 * nanosecond precision using BigInt internally.
 *
 * @example
 * ```typescript
 * const scale = preciseTimeScale()
 *   .domain([new Date('2023-01-01'), new Date('2023-01-02')])
 *   .range([0, 1000]);
 *
 * // Convert a timestamp to a pixel position
 * const pixelPos = scale.scale(new Date('2023-01-01T12:00:00'));
 *
 * // Generate tick marks
 * const ticks = scale.ticks(5);
 * ```
 */
export class PreciseTimeScale {
  private _domain: [TimeStamp, TimeStamp];
  private _range: [number, number];
  private _span: TimeSpan;

  /**
   * Creates a new PreciseTimeScale instance with default domain [0, 1] and range [0, 1].
   */
  constructor() {
    // Default initialization
    this._domain = [new TimeStamp(0n), new TimeStamp(1n)];
    this._range = [0, 1];
    this._span = this._domain[1].span(this._domain[0]);
  }

  /**
   * Gets or sets the time domain of the scale.
   *
   * @param domain - Optional domain to set as [start, end] timestamps
   * @returns Current domain if no argument provided, or this instance for chaining
   */
  domain(): [TimeStamp, TimeStamp];
  domain(domain: [CrudeTimeStamp, CrudeTimeStamp]): this;
  domain(domain?: [CrudeTimeStamp, CrudeTimeStamp]): [TimeStamp, TimeStamp] | this {
    if (domain === undefined) return this._domain;
    this._domain = [new TimeStamp(domain[0]), new TimeStamp(domain[1])];
    this._span = this._domain[1].span(this._domain[0]);
    return this;
  }

  /**
   * Gets or sets the numeric range of the scale.
   *
   * @param range - Optional range to set as [start, end] numbers
   * @returns Current range if no argument provided, or this instance for chaining
   */
  range(): [number, number];
  range(range: [number, number]): this;
  range(range?: [number, number]): [number, number] | this {
    if (range === undefined) return this._range;
    this._range = range;
    return this;
  }

  /**
   * Scales a timestamp to its corresponding position in the range.
   *
   * @param value - The timestamp to scale
   * @returns The scaled numeric value within the range
   */
  scale(value: CrudeTimeStamp): number {
    const ts = new TimeStamp(value);
    const v = ts.valueOf();
    const start = this._domain[0].valueOf();
    const span = this._domain[1].valueOf() - start;
    const rangeSpan = this._range[1] - this._range[0];
    return (Number(v - start) / Number(span)) * rangeSpan + this._range[0];
  }

  /**
   * Generates an array of evenly spaced tick marks within the domain.
   * The ticks are automatically adjusted to use human-friendly time intervals.
   *
   * @param count - Desired number of ticks (actual count may differ to maintain nice intervals)
   * @returns Array of timestamps representing tick positions
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

  /**
   * Calculates the optimal step size for tick generation based on the desired tick count.
   * Uses predefined steps from TIME_SCALE_STEPS to ensure human-readable intervals.
   *
   * @param targetCount - Desired number of ticks
   * @returns The optimal TimeSpan step size
   */
  private calculateOptimalStep(targetCount: number): TimeSpan {
    const rawStepNanoseconds = this._span.valueOf() / BigInt(Math.ceil(targetCount));
    let bestStep = TIME_SCALE_STEPS[0];
    for (const step of TIME_SCALE_STEPS) {
      if (step.valueOf() > rawStepNanoseconds) break;
      bestStep = step;
    }
    return bestStep;
  }

  /**
   * Formats a timestamp for display on tick marks.
   * Automatically adjusts the format based on the current time scale:
   * - For spans < 50µs: displays microseconds
   * - For spans >= 50µs: displays milliseconds
   *
   * @param value - The timestamp to format
   * @returns Formatted string representation of the timestamp
   */
  formatTick(value: TimeStamp): string {
    if (this._span.lessThan(TimeSpan.microseconds(50))) {
      const remainder = value.remainder(TimeSpan.MILLISECOND);
      return `${remainder.microseconds.toString()}µs`;
    }
    const remainder = value.remainder(TimeSpan.SECOND);
    return `${remainder.milliseconds.toString()}ms`;
  }
}

/**
 * Creates a new PreciseTimeScale instance.
 *
 * @returns A new PreciseTimeScale instance with default settings
 */
export const preciseTimeScale = (): PreciseTimeScale => new PreciseTimeScale();

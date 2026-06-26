// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DataType,
  math,
  MultiSeries,
  Series,
  type TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import {
  convertSeriesToSupportedGL,
  resolveGLDataType,
} from "@/telem/aether/convertSeries";

/** Response from a write to the @link Dynamic cache. */
export interface DynamicWriteResponse {
  /** A list of series that were flushed from the cache during the write i.e. the new
   * writes were not able to fit in the current buffer, so a new one was allocated
   * and the old one(s) were flushed. */
  flushed: MultiSeries;
  /** A list of series that were allocated during the write. */
  allocated: MultiSeries;
}

/** Props for the @link Dynamic cache. */
export interface DynamicProps {
  /**
   * Sets the maximum size of the buffer that the cache will maintain before flushing
   * data out to the caller.
   */
  dynamicBufferSize: number | TimeSpan;
  /**
   * Sets the data type for the series written to the cache. Used for buffer allocation
   * purposes.
   */
  dataType: DataType;
  /**
   * Function that the cache will use to pull the current time.
   */
  now?: () => TimeStamp;
}

// These are the smallest and largest sizes for a dynamically calculated buffer size.
const MIN_SIZE = 100;
const MAX_SIZE = 1e6;
// The default size returned when there have not been enough writes yet and the
// maximum number of default writes.
const DEF_SIZE = 1e4;
const MAX_DEF_WRITES = 100;

// When we allocate series for variable rate data types, we're allocating the number
// of bytes instead of samples. This multiplier is used as a rough estimation for the number
// of bytes per sample.
const VARIABLE_DT_MULTIPLIER = 40;

/**
 * A cache for channel data that maintains a single, rolling Series as a buffer
 * for channel data.
 */
export class Dynamic {
  private readonly props: DynamicProps;

  private counter = 0;
  /** Current buffer */
  private curr: Series | null;
  private avgRate: number = 0;
  private timeOfLastWrite: TimeStamp;
  private totalWrites: number = 0;
  private now = () => TimeStamp.now();

  /**
   * @constructor
   *
   * @param cap - The capacity of the cache buffer.
   * @param dataType - The data type of the channel.
   */
  constructor(props: DynamicProps) {
    this.props = props;
    this.curr = null;
    if (props.now != null) this.now = props.now;
    this.timeOfLastWrite = this.now();
  }

  /** @returns the number of samples currently held in the cache. */
  get length(): number {
    return this.curr?.length ?? 0;
  }

  /**
   * @returns the current buffer being written to by the cache. Under no circumstances
   * should this be modified by the caller.
   */
  get leadingBuffer(): Series | null {
    return this.curr;
  }

  /**
   * Writes the given arrays to the cache.
   *
   * @returns a list of buffers that were filled by the cache during the write. If
   * the current buffer is able to fit all writes, no buffers will be returned.
   */
  write(series: MultiSeries): DynamicWriteResponse {
    const responses = series.series.flatMap((s) => this._write(s));
    return {
      flushed: new MultiSeries(responses.flatMap((res) => res.flushed.series)),
      allocated: new MultiSeries(responses.flatMap((res) => res.allocated.series)),
    };
  }

  private allocate(
    capacity: number,
    alignment: bigint,
    start: TimeStamp,
    sampleHint?: math.Numeric,
  ): Series {
    this.counter++;
    const dt = this.props.dataType;
    // Bigint channels render into float32 GL buffers, so each value is stored as a
    // small delta off a per-buffer anchor to avoid losing precision. For timestamps,
    // now() is close enough to the values being written. For int64 and uint64 the
    // values can be anything, so the first sample we see is used as the anchor.
    // Non-bigint channels do not have this problem and stay at zero.
    const sampleOffset = dt.equals(DataType.TIMESTAMP)
      ? start.valueOf()
      : dt.usesBigInt && sampleHint != null
        ? BigInt(sampleHint.valueOf())
        : 0;
    return Series.alloc({
      capacity: dt.isVariable ? capacity * VARIABLE_DT_MULTIPLIER : capacity,
      dataType: resolveGLDataType(dt),
      timeRange: start.range(TimeStamp.MAX),
      sampleOffset,
      glBufferUsage: "dynamic",
      alignment,
      key: `dynamic-${this.counter}`,
    });
  }

  private _write(series: Series): DynamicWriteResponse {
    const cap = this.nextBufferSize();
    const res: DynamicWriteResponse = {
      flushed: new MultiSeries([]),
      allocated: new MultiSeries([]),
    };
    // This only happens on the first write to the cache
    if (this.curr == null) {
      this.curr = this.allocate(cap, series.alignment, this.now(), series.data[0]);
      res.allocated.push(this.curr);
    } else {
      // overlap > 0: the incoming series steps back into samples the current buffer
      // already holds. overlap < 0: there is a gap between the buffer and the series.
      const overlap = Number(
        this.curr.alignment + BigInt(this.curr.length) - series.alignment,
      );
      if (overlap > 1) {
        // The stream re-sent samples we already have. Drop the duplicated leading
        // samples and continue the current buffer rather than allocating a new,
        // overlapping fragment. Without this, an overlapping frame on every cycle
        // fragments the cache into dozens of overlapping series that the renderer must
        // draw — the source of the console freeze and the tangled line plot.
        // sub (not slice) returns a zero-copy view; only the non-duplicate tail is
        // written to the buffer below, so this does strictly less work than a
        // contiguous frame.
        if (overlap >= series.length) return res;
        series = series.sub(overlap);
      } else if (overlap < -1) {
        // The incoming series starts after a gap; flush the current buffer and
        // allocate a new one at the incoming alignment.
        const now = this.now();
        this.curr.timeRange.end = now;
        res.flushed.push(this.curr);
        this.curr = this.allocate(cap, series.alignment, now, series.data[0]);
        res.allocated.push(this.curr);
      }
    }
    const converted = convertSeriesToSupportedGL(series, this.curr.sampleOffset);
    const amountWritten = this.curr.write(converted);
    // This means that the current buffer is large enough to fit the entire incoming
    // series. We're done in this caseconv.
    if (amountWritten === series.length) {
      this.updateAvgRate(series);
      return res;
    }
    // Push the current buffer to the flushed list.
    const now = this.now();
    this.curr.timeRange.end = now;
    res.flushed.push(this.curr);
    this.curr = this.allocate(
      cap,
      series.alignment + BigInt(amountWritten),
      now,
      series.data[amountWritten],
    );
    res.allocated.push(this.curr);
    const nextRes = this._write(series.slice(amountWritten));
    res.flushed.push(nextRes.flushed);
    res.allocated.push(nextRes.allocated);
    return res;
  }

  private updateAvgRate(series: Series): void {
    if (typeof this.props.dynamicBufferSize === "number") return;
    // average rate is a weighted average of the rate of the last sample and the average
    // rate currently in the buffer.
    const newRate = series.length / this.now().span(this.timeOfLastWrite).seconds;
    if (this.totalWrites > 0 && isFinite(newRate) && newRate > 0)
      this.avgRate =
        (this.avgRate * (this.totalWrites - 1) + newRate) / this.totalWrites;
    this.totalWrites++;
    this.timeOfLastWrite = this.now();
  }

  private nextBufferSize(): number {
    const { dynamicBufferSize } = this.props;
    if (typeof dynamicBufferSize === "number") return dynamicBufferSize;
    if (this.totalWrites < MAX_DEF_WRITES) return DEF_SIZE;
    const size = math.roundToNearestMagnitude(this.avgRate * dynamicBufferSize.seconds);
    return Math.round(Math.max(Math.min(size, MAX_SIZE), MIN_SIZE));
  }

  /**
   * Closes the cache and releases all resources associated with it. After close()
   * is called, the cache should not be used again.
   */
  close(): void {
    this.curr = null;
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import {
  DataType,
  math,
  MultiSeries,
  Series,
  type TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { IDENTITY_TRANSFORM, type Transform } from "@/framer/cache/transform";

/** Response from a write to the {@link Dynamic} cache. */
export interface WriteResponse {
  /** Series flushed because the write did not fit in the current buffer. */
  flushed: MultiSeries;
  /** A list of series that were allocated during the write. */
  allocated: MultiSeries;
}

/** Props for the {@link Dynamic} cache. */
export interface DynamicProps {
  /**
   * Sets the maximum size of the buffer that the cache will maintain before flushing
   * data out to the caller.
   */
  dynamicBufferSize: number | TimeSpan;
  /** Applied to every series before it enters a buffer. Defaults to identity. */
  transform?: Transform;
  /** Used for logging. */
  instrumentation?: alamos.Instrumentation;
  /** Pulls the current time. */
  now?: () => TimeStamp;
}

// Bounds for dynamically calculated buffer sizes.
const MIN_SIZE = 100;
const MAX_SIZE = 1e6;
// Size used until enough writes have arrived to estimate a rate.
const DEF_SIZE = 1e4;
const MAX_DEF_WRITES = 100;

// Variable-rate types allocate bytes, not samples. Rough bytes-per-sample estimate.
const VARIABLE_DT_MULTIPLIER = 40;

/**
 * A cache for channel data that maintains a single, rolling Series as a buffer
 * for channel data. The buffer's data type is derived from the first written series,
 * so the cache needs no channel metadata.
 */
export class Dynamic {
  private readonly props: Required<Omit<DynamicProps, "now">>;

  private counter = 0;
  private curr: Series | null = null;
  /** End timestamp of the last fully written stamped series in the current buffer.
   * Null when the buffer holds unstamped or partially written data, in which case
   * the flush falls back to the wall clock. */
  private currDataEnd: TimeStamp | null = null;
  private avgRate: number = 0;
  private timeOfLastWrite: TimeStamp;
  private totalWrites: number = 0;
  private readonly now: () => TimeStamp;

  constructor(props: DynamicProps) {
    const {
      dynamicBufferSize,
      transform = IDENTITY_TRANSFORM,
      instrumentation = alamos.NOOP,
      now = () => TimeStamp.now(),
    } = props;
    this.props = { dynamicBufferSize, transform, instrumentation };
    this.now = now;
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
  write(series: MultiSeries): WriteResponse {
    const res: WriteResponse = {
      flushed: new MultiSeries([]),
      allocated: new MultiSeries([]),
    };
    const list = series.series;
    for (let i = 0; i < list.length; i++) this._write(list[i], res);
    return res;
  }

  private allocate(
    capacity: number,
    alignment: bigint,
    start: TimeStamp,
    source: Series,
    sampleIndex: number,
  ): Series {
    this.counter++;
    const dt = this.props.transform.resolveDataType(source.dataType);
    // Bigint series narrowed to a non-bigint buffer store each value as a small delta
    // off a per-buffer anchor to avoid losing precision. For timestamps, now() is
    // close enough to the values being written. For int64 and uint64 the values can
    // be anything, so the first sample we see is used as the anchor.
    const narrowing = source.dataType.usesBigInt && !dt.usesBigInt;
    let sampleOffset: math.Numeric = 0;
    if (narrowing)
      if (source.dataType.equals(DataType.TIMESTAMP)) sampleOffset = start.valueOf();
      else if (sampleIndex < source.length)
        sampleOffset = BigInt(source.data[sampleIndex].valueOf());
    return Series.alloc({
      capacity: dt.isVariable ? capacity * VARIABLE_DT_MULTIPLIER : capacity,
      dataType: dt,
      timeRange: start.range(TimeStamp.MAX),
      sampleOffset,
      glBufferUsage: "dynamic",
      alignment,
      key: `dynamic-${this.counter}`,
    });
  }

  // Unstamped series (virtual channels, writes without an in-frame index) fall back
  // to the wall clock.
  private allocStart(series: Series): TimeStamp {
    return series.timeRange.isZero ? this.now() : series.timeRange.start;
  }

  private allocCurr(
    res: WriteResponse,
    alignment: bigint,
    start: TimeStamp,
    source: Series,
    sampleIndex: number,
  ): Series {
    this.curr = this.allocate(
      this.nextBufferSize(),
      alignment,
      start,
      source,
      sampleIndex,
    );
    res.allocated.push(this.curr);
    return this.curr;
  }

  private flushCurr(res: WriteResponse): void {
    if (this.curr == null) return;
    this.curr.timeRange.end = this.currDataEnd ?? this.now();
    this.currDataEnd = null;
    res.flushed.push(this.curr);
    this.curr = null;
  }

  private _write(series: Series, res: WriteResponse): void {
    const { transform, instrumentation: ins } = this.props;
    if (this.curr != null) {
      const resolved = transform.resolveDataType(series.dataType);
      if (!this.curr.dataType.equals(resolved)) {
        // The channel's data type changed (e.g. a calculated channel was
        // reconfigured).
        ins.L.warn("buffer data type changed, resetting", {
          prev: this.curr.dataType.toString(),
          next: resolved.toString(),
        });
        this.flushCurr(res);
      } else if (series.alignment < this.curr.alignment) {
        // The alignment counter rewound (e.g. the Core restarted and reset its
        // in-memory counter).
        ins.L.warn("alignment regressed, resetting buffer", {
          buffer: this.curr.alignment.toString(),
          incoming: series.alignment.toString(),
        });
        this.flushCurr(res);
      }
    }
    let curr = this.curr;
    if (curr == null)
      curr = this.allocCurr(res, series.alignment, this.allocStart(series), series, 0);
    else {
      // overlap > 0: the incoming series steps back into samples the current buffer
      // already holds. overlap < 0: there is a gap between the buffer and the series.
      const overlap = Number(curr.alignment + BigInt(curr.length) - series.alignment);
      if (overlap > 0) {
        // Drop the re-sent leading samples; a fresh allocation here would fragment
        // the cache with overlapping series. sub() is zero-copy.
        if (overlap >= series.length) return;
        series = series.sub(overlap);
      } else if (overlap < 0) {
        this.flushCurr(res);
        curr = this.allocCurr(
          res,
          series.alignment,
          this.allocStart(series),
          series,
          0,
        );
      }
    }
    while (true) {
      const converted = transform.convert(series, curr.sampleOffset);
      const amountWritten = curr.write(converted);
      if (amountWritten === series.length) {
        this.currDataEnd = series.timeRange.isZero ? null : series.timeRange.end;
        this.updateAvgRate(series);
        return;
      }
      // The timestamp at the split point is unknowable from the data series, so
      // both sides fall back to the wall clock.
      this.currDataEnd = null;
      this.flushCurr(res);
      curr = this.allocCurr(
        res,
        series.alignment + BigInt(amountWritten),
        this.now(),
        series,
        amountWritten,
      );
      series = series.slice(amountWritten);
    }
  }

  private updateAvgRate(series: Series): void {
    if (typeof this.props.dynamicBufferSize === "number") return;
    const now = this.now();
    const newRate = series.length / now.span(this.timeOfLastWrite).seconds;
    if (this.totalWrites > 0 && isFinite(newRate) && newRate > 0)
      this.avgRate =
        (this.avgRate * (this.totalWrites - 1) + newRate) / this.totalWrites;
    this.totalWrites++;
    this.timeOfLastWrite = now;
  }

  private nextBufferSize(): number {
    const { dynamicBufferSize } = this.props;
    if (typeof dynamicBufferSize === "number") return dynamicBufferSize;
    if (this.totalWrites < MAX_DEF_WRITES) return DEF_SIZE;
    const size = math.roundToNearestMagnitude(this.avgRate * dynamicBufferSize.seconds);
    return Math.round(Math.max(Math.min(size, MAX_SIZE), MIN_SIZE));
  }

  /**
   * Flushes the leading buffer, stamping its end with the last stamped write or
   * the wall clock.
   * @returns the flushed buffer, or null when there is none.
   */
  flush(): Series | null {
    const res: WriteResponse = {
      flushed: new MultiSeries([]),
      allocated: new MultiSeries([]),
    };
    this.flushCurr(res);
    return res.flushed.series[0] ?? null;
  }

  /** Closes the cache. It must not be used afterwards. */
  close(): void {
    this.curr = null;
  }
}

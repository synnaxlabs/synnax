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
export interface DynamicWriteResponse {
  /** A list of series that were flushed from the cache during the write i.e. the new
   * writes were not able to fit in the current buffer, so a new one was allocated
   * and the old one(s) were flushed. */
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
// of bytes instead of samples. This multiplier is used as a rough estimation for the
// number of bytes per sample.
const VARIABLE_DT_MULTIPLIER = 40;

/**
 * A cache for channel data that maintains a single, rolling Series as a buffer
 * for channel data. The buffer's data type is derived from the first written series,
 * so the cache needs no channel metadata.
 */
export class Dynamic {
  private readonly props: Required<Omit<DynamicProps, "now">>;

  private counter = 0;
  /** Current buffer */
  private curr: Series | null;
  private avgRate: number = 0;
  private timeOfLastWrite: TimeStamp;
  private totalWrites: number = 0;
  private now = () => TimeStamp.now();

  constructor(props: DynamicProps) {
    const {
      dynamicBufferSize,
      transform = IDENTITY_TRANSFORM,
      instrumentation = alamos.NOOP,
    } = props;
    this.props = { dynamicBufferSize, transform, instrumentation };
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
    const res: DynamicWriteResponse = {
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
    // be anything, so the first sample we see is used as the anchor. Buffers that
    // keep bigint storage do not have this problem and stay at zero.
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

  private flushCurr(res: DynamicWriteResponse): void {
    if (this.curr == null) return;
    this.curr.timeRange.end = this.now();
    res.flushed.push(this.curr);
    this.curr = null;
  }

  private _write(series: Series, res: DynamicWriteResponse): void {
    const { transform, instrumentation: ins } = this.props;
    if (this.curr != null) {
      const resolved = transform.resolveDataType(series.dataType);
      if (!this.curr.dataType.equals(resolved)) {
        // The channel's data type changed (e.g. a calculated channel was
        // reconfigured). Flush the old buffer and restart at the incoming type.
        ins.L.warn("buffer data type changed, resetting", {
          prev: this.curr.dataType.toString(),
          next: resolved.toString(),
        });
        this.flushCurr(res);
      } else if (series.alignment < this.curr.alignment) {
        // The alignment counter rewound (e.g. the Core restarted and reset its
        // in-memory counter). Restart at the incoming alignment instead of treating
        // the write as a duplicate and dropping it.
        ins.L.warn("alignment regressed, resetting buffer", {
          buffer: this.curr.alignment.toString(),
          incoming: series.alignment.toString(),
        });
        this.flushCurr(res);
      }
    }
    if (this.curr == null) {
      this.curr = this.allocate(
        this.nextBufferSize(),
        series.alignment,
        this.now(),
        series,
        0,
      );
      res.allocated.push(this.curr);
    } else {
      // overlap > 0: the incoming series steps back into samples the current buffer
      // already holds. overlap < 0: there is a gap between the buffer and the series.
      const overlap = Number(
        this.curr.alignment + BigInt(this.curr.length) - series.alignment,
      );
      if (overlap > 1) {
        // The stream re-sent samples the buffer already holds. Drop the duplicate
        // leading samples and continue the buffer; allocating a new series here
        // would fragment the cache into overlapping fragments on every cycle.
        // sub() returns a zero-copy view.
        if (overlap >= series.length) return;
        series = series.sub(overlap);
      } else if (overlap < -1) {
        // The incoming series starts after a gap; flush the current buffer and
        // allocate a new one at the incoming alignment.
        this.flushCurr(res);
        this.curr = this.allocate(
          this.nextBufferSize(),
          series.alignment,
          this.now(),
          series,
          0,
        );
        res.allocated.push(this.curr);
      }
    }
    while (true) {
      const converted = transform.convert(series, this.curr.sampleOffset);
      const amountWritten = this.curr.write(converted);
      // This means that the current buffer is large enough to fit the entire incoming
      // series. We're done in this case.
      if (amountWritten === series.length) {
        this.updateAvgRate(series);
        return;
      }
      // Push the current buffer to the flushed list.
      this.flushCurr(res);
      this.curr = this.allocate(
        this.nextBufferSize(),
        series.alignment + BigInt(amountWritten),
        this.now(),
        series,
        amountWritten,
      );
      res.allocated.push(this.curr);
      series = series.slice(amountWritten);
    }
  }

  private updateAvgRate(series: Series): void {
    if (typeof this.props.dynamicBufferSize === "number") return;
    const now = this.now();
    // average rate is a weighted average of the rate of the last sample and the
    // average rate currently in the buffer.
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
   * Closes the cache and releases all resources associated with it. After close()
   * is called, the cache should not be used again.
   */
  close(): void {
    this.curr = null;
  }
}

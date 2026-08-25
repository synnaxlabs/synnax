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
  Size,
  TimeRange,
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
 * @returns true when every byte of the series fits in the buffer's free space. Measured
 * in bytes because a variable-length buffer counts its capacity that way.
 */
const fits = (buffer: Series, series: Series): boolean =>
  series.byteLength.valueOf() <=
  buffer.byteCapacity.valueOf() - buffer.byteLength.valueOf();

// Bounds on compacting a buffer that rotated early. Both have to trip: the fraction
// spares a nearly full buffer, the floor one whose waste is not worth a copy.
const COMPACT_MAX_FILL = 0.25;
const COMPACT_MIN_WASTE = Size.kilobytes(64);

/**
 * @returns true when copying the buffer's samples into a right-sized series is worth
 * the copy. A held buffer never qualifies: the holder keeps the original alive, so a
 * copy would only add a second allocation, and it would break the object identity a
 * consumer that both reads and streams deduplicates on.
 */
const shouldCompact = (buffer: Series): boolean => {
  if (buffer.refCount > 0 || buffer.length === 0) return false;
  const used = buffer.byteLength.valueOf();
  const capacity = buffer.byteCapacity.valueOf();
  return (
    capacity - used > COMPACT_MIN_WASTE.valueOf() && used < capacity * COMPACT_MAX_FILL
  );
};

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
   * @returns the time range the buffer's samples actually cover: its start to the last
   * stamped write, or the wall clock when the buffer holds unstamped data. Unlike the
   * buffer's provisional time range, the end never claims future time. Null when there
   * is no buffer.
   */
  get dataTimeRange(): TimeRange | null {
    if (this.curr == null) return null;
    return new TimeRange(this.curr.timeRange.start, this.currDataEnd ?? this.now());
  }

  /**
   * @returns a list of buffers that were filled by the cache during the write. If the
   * current buffer is able to fit all writes, no buffers will be returned.
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
      else if (source.length > 0) sampleOffset = BigInt(source.data[0].valueOf());
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

  /**
   * @returns the sample capacity for a buffer that must hold the given series whole:
   * the configured size, or the series' own size when that is larger.
   */
  private capacityFor(source: Series): number {
    const dt = this.props.transform.resolveDataType(source.dataType);
    const required = dt.isVariable
      ? Math.ceil(source.byteLength.valueOf() / VARIABLE_DT_MULTIPLIER)
      : source.length;
    return Math.max(this.nextBufferSize(), required);
  }

  // Unstamped series (virtual channels, writes without an in-frame index) fall back
  // to the wall clock.
  private allocStart(series: Series): TimeStamp {
    return series.timeRange.isZero ? this.now() : series.timeRange.start;
  }

  private allocCurr(
    res: WriteResponse,
    source: Series,
    start: TimeStamp | null,
  ): Series {
    this.curr = this.allocate(
      this.capacityFor(source),
      source.alignment,
      start ?? this.allocStart(source),
      source,
    );
    res.allocated.push(this.curr);
    return this.curr;
  }

  /** @returns the timestamp the flushed buffer's end was stamped with, or null when
   * there was no buffer. */
  private flushCurr(res: WriteResponse): TimeStamp | null {
    if (this.curr == null) return null;
    const end = this.currDataEnd ?? this.now();
    this.curr.timeRange.end = end;
    this.currDataEnd = null;
    res.flushed.push(this.compacted(this.curr, res));
    this.curr = null;
    return end;
  }

  /**
   * @returns a right-sized copy of a buffer that rotated with most of its space
   * unused, or the buffer itself. A rotated buffer holds its whole allocation until
   * it is collected, which is wasteful when an oversized series ended it early.
   */
  private compacted(buffer: Series, res: WriteResponse): Series {
    // A buffer allocated during this same write has not reached its subscribers yet,
    // so its reference count cannot yet report who is about to hold it.
    if (!shouldCompact(buffer) || res.allocated.series.includes(buffer)) return buffer;
    return buffer.compact();
  }

  private _write(series: Series, res: WriteResponse): void {
    const { transform, instrumentation: ins } = this.props;
    if (this.curr != null) {
      const resolved = transform.resolveDataType(series.dataType);
      if (!this.curr.dataType.equals(resolved)) {
        // The channel's data type changed (e.g. a calculated channel was reconfigured).
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
    let trimmed = false;
    if (curr != null) {
      // overlap > 0: the incoming series steps back into samples the current buffer
      // already holds. overlap < 0: there is a gap between the buffer and the series.
      const overlap = Number(curr.alignment + BigInt(curr.length) - series.alignment);
      if (overlap > 0) {
        // Drop the re-sent leading samples; a fresh allocation here would fragment
        // the cache with overlapping series. sub() is zero-copy.
        if (overlap >= series.length) return;
        series = series.sub(overlap);
        trimmed = true;
      } else if (overlap < 0) {
        this.flushCurr(res);
        curr = null;
      }
    }
    const convert = (buffer: Series): Series =>
      transform.convert(series, buffer.sampleOffset);
    let rotationStart: TimeStamp | null = null;
    if (curr != null) {
      const converted = convert(curr);
      // A series never splits across two buffers. The timestamp where a split falls is
      // not in the data, so both sides would have to guess the boundary they share, and
      // a guess lands outside the data whenever samples are irregular.
      if (fits(curr, converted)) curr.write(converted);
      else {
        const end = this.flushCurr(res);
        // A trimmed series keeps the whole frame's start, which is earlier than the
        // samples it still holds. Those samples begin where the buffer they were
        // trimmed against ends, so the next buffer starts there.
        if (trimmed) rotationStart = end;
        curr = null;
      }
    }
    if (curr == null) {
      curr = this.allocCurr(res, series, rotationStart);
      curr.write(convert(curr));
    }
    this.currDataEnd = series.timeRange.isZero ? null : series.timeRange.end;
    this.updateAvgRate(series);
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  DataType,
  MultiSeries,
  type Series,
  type TimeRange,
  TimeStamp,
} from "@synnaxlabs/x";

import { UnexpectedError } from "@/errors";
import { Dynamic, type DynamicProps } from "@/framer/cache/dynamic";
import { type Stamps } from "@/framer/cache/stamps";
import {
  type DirtyReadResult,
  type GCMetrics,
  Static,
  type StaticProps,
} from "@/framer/cache/static";
import { IDENTITY_TRANSFORM } from "@/framer/cache/transform";

export interface UnaryProps
  extends StaticProps, Pick<DynamicProps, "dynamicBufferSize"> {}

/**
 * Caches data for a single channel: a dynamic leading buffer for live writes backed
 * by a static cache for flushed and historical data.
 * @throws {UnexpectedError} from every method after close().
 */
export class Unary {
  private closed: boolean = false;
  private readonly props: UnaryProps;
  private readonly static: Static;
  private readonly dynamic: Dynamic;
  // Whether the transform narrows timestamps, so stamps need an exact copy to survive.
  private readonly narrowed: boolean;
  private exact: Unary | null = null;
  /** The entry of this channel's index channel. An index channel points at itself. */
  index: Unary | null = null;
  /** The write stamps of a channel with no index, else null. */
  stamps: Stamps | null = null;

  constructor(props: UnaryProps) {
    const { transform = IDENTITY_TRANSFORM } = props;
    this.props = props;
    this.static = new Static(props);
    this.dynamic = new Dynamic({
      dynamicBufferSize: props.dynamicBufferSize,
      transform,
      instrumentation: props.instrumentation,
    });
    this.narrowed = !transform
      .resolveDataType(DataType.TIMESTAMP)
      .equals(DataType.TIMESTAMP);
  }

  writeDynamic(series: MultiSeries): MultiSeries {
    this.checkOpen("writeDynamic");
    const { stamps } = this;
    if (stamps != null) for (const s of series.series) stamps.record(s);
    this.trackExact(series);
    this.exact?.writeDynamic(series);
    const { flushed, allocated } = this.dynamic.write(series);
    // Flushed buffers stay streamed until a fetch replaces them.
    if (flushed.length > 0) this.static.write(flushed, true);
    const leading = this.dynamic.leadingBuffer;
    if (leading != null) this.static.retainDataType(leading.dataType);
    return allocated;
  }

  get leadingBuffer(): Series | null {
    this.checkOpen("leadingBuffer");
    return this.dynamic.leadingBuffer;
  }

  /** @returns the stamp of the last sample in the leading buffer, else null. */
  get lastWrite(): TimeStamp | null {
    this.checkOpen("lastWrite");
    return this.dynamic.lastWrite;
  }

  /** @returns the stamp of the sample at alignment, from the index or from the write
   * that carried it, else null. */
  timeAt(alignment: bigint): TimeStamp | null {
    this.checkOpen("timeAt");
    if (this.index != null) return this.index.exactAt(alignment);
    return this.stamps?.at(alignment) ?? null;
  }

  /**
   * Flushes the live leading buffer into the static cache with its real end time,
   * so reads treat the span after it as a gap to fetch. Call when streaming for
   * the channel stops.
   */
  flushDynamic(): void {
    this.checkOpen("flushDynamic");
    this.exact?.flushDynamic();
    const flushed = this.dynamic.flush();
    if (flushed != null && flushed.length > 0)
      this.static.write(new MultiSeries([flushed]), true);
  }

  writeStatic(series: MultiSeries): void {
    this.checkOpen("writeStatic");
    const { stamps } = this;
    if (stamps != null) for (const s of series.series) stamps.recordFetched(s);
    this.trackExact(series);
    this.exact?.writeStatic(series);
    this.static.write(series);
    const dt = this.static.dataType;
    if (dt != null) this.dynamic.retainDataType(dt);
  }

  /** Records tr as fetched so reads stop reporting it as a gap even when it holds no
   * samples. Call after a fetch for tr completes. */
  markFetched(tr: TimeRange): void {
    this.checkOpen("markFetched");
    this.exact?.markFetched(tr);
    this.static.markFetched(tr);
  }

  /**
   * Reads cached data overlapping the given time range. The result includes the live
   * leading buffer when the data it actually holds overlaps the range, judged by the
   * last stamped write rather than the buffer's provisional MAX end, so the buffer
   * never leaks into reads of spans it has no samples for. It never claims coverage
   * either: its samples carry provisional leading alignments that cannot pair with
   * fetched data on another channel, so its span is still reported as a gap to fetch. A
   * span may therefore return in both its streamed and fetched representation.
   */
  read(tr: TimeRange): DirtyReadResult {
    this.checkOpen("read");
    const res = this.static.dirtyRead(tr);
    const buf = this.dynamic.leadingBuffer;
    const dataTr = this.dynamic.dataTimeRange;
    if (
      buf == null ||
      dataTr == null ||
      buf.length === 0 ||
      !dataTr.overlapsWith(tr) ||
      // The buffer's leading alignments cannot be ordered against fetched data, so
      // the fetched form of a span wins.
      this.static.overlapsFetched(dataTr)
    )
      return res;
    res.series.push(buf);
    return res;
  }

  /**
   * Garbage collects the entry.
   * @param retain - Alignment spans whose stamps must survive however stale they are.
   */
  gc(retain: bounds.Bounds<bigint>[] = []): GCMetrics {
    this.checkOpen("gc");
    // Stamps live in the exact copy when the transform narrows them, else here.
    const res = this.static.gc(this.narrowed ? [] : retain);
    const exact = this.exact?.gc(retain);
    if (exact == null) return res;
    return {
      purgedSeries: res.purgedSeries + exact.purgedSeries,
      purgedBytes: res.purgedBytes.add(exact.purgedBytes),
    };
  }

  /** @returns the alignment spans of every cached series a consumer holds. */
  heldAlignments(): bounds.Bounds<bigint>[] {
    this.checkOpen("heldAlignments");
    const held = this.static.heldAlignments();
    const leading = this.dynamic.leadingBuffer;
    if (leading != null && leading.refCount > 0) held.push(leading.alignmentBounds);
    return held;
  }

  close(): void {
    this.closed = true;
    this.exact?.close();
    this.dynamic.close();
    this.static.close();
  }

  // Indexes are timestamp channels. Their stamps stay exact beside the narrowed ones.
  private trackExact(series: MultiSeries): void {
    if (!this.narrowed || this.exact != null || series.series.length === 0) return;
    if (!series.dataType.equals(DataType.TIMESTAMP)) return;
    this.exact = new Unary({ ...this.props, transform: IDENTITY_TRANSFORM });
  }

  // The exact buffers are this entry's own when nothing narrows them.
  private exactAt(alignment: bigint): TimeStamp | null {
    const src = this.narrowed ? this.exact : this;
    if (src == null) return null;
    const leading = src.dynamic.leadingBuffer;
    const series =
      leading != null && bounds.contains(leading.alignmentBounds, alignment)
        ? leading
        : src.static.find(alignment);
    if (series == null || !series.dataType.equals(DataType.TIMESTAMP)) return null;
    return new TimeStamp(series.atAlignment(alignment) as bigint);
  }

  private checkOpen(op: string): void {
    if (this.closed)
      throw new UnexpectedError(`${op} called on a closed telemetry cache entry`);
  }
}

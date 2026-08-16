// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MultiSeries, type Series, TimeRange } from "@synnaxlabs/x";

import { UnexpectedError } from "@/errors";
import { Dynamic, type DynamicProps } from "@/framer/cache/dynamic";
import {
  type DirtyReadResult,
  type GCMetrics,
  Static,
  type StaticProps,
} from "@/framer/cache/static";

export interface UnaryProps
  extends StaticProps, Pick<DynamicProps, "dynamicBufferSize"> {}

/**
 * Caches data for a single channel: a dynamic leading buffer for live writes backed
 * by a static cache for flushed and historical data.
 * @throws {UnexpectedError} from every method after close().
 */
export class Unary {
  private closed: boolean = false;
  private readonly static: Static;
  private readonly dynamic: Dynamic;

  constructor(props: UnaryProps) {
    this.static = new Static(props);
    this.dynamic = new Dynamic({
      dynamicBufferSize: props.dynamicBufferSize,
      transform: props.transform,
      instrumentation: props.instrumentation,
    });
  }

  writeDynamic(series: MultiSeries): MultiSeries {
    this.checkOpen("writeDynamic");
    const { flushed, allocated } = this.dynamic.write(series);
    // Flushed buffers stay streamed until a fetch replaces them.
    if (flushed.length > 0) this.static.write(flushed, true);
    return allocated;
  }

  get leadingBuffer(): Series | null {
    this.checkOpen("leadingBuffer");
    return this.dynamic.leadingBuffer;
  }

  /**
   * Flushes the live leading buffer into the static cache with its real end time,
   * so reads treat the span after it as a gap to fetch. Call when streaming for
   * the channel stops.
   */
  flushDynamic(): void {
    this.checkOpen("flushDynamic");
    const flushed = this.dynamic.flush();
    if (flushed != null && flushed.length > 0)
      this.static.write(new MultiSeries([flushed]), true);
  }

  writeStatic(series: MultiSeries): void {
    this.checkOpen("writeStatic");
    this.static.write(series);
  }

  /**
   * Reads cached data overlapping the given time range. The result includes the
   * live leading buffer when it overlaps, and its gaps are clipped at the buffer's
   * start so a fetch never re-reads data the stream already delivered.
   */
  read(tr: TimeRange): DirtyReadResult {
    this.checkOpen("read");
    const res = this.static.dirtyRead(tr);
    const buf = this.dynamic.leadingBuffer;
    if (buf == null || buf.length === 0 || !buf.timeRange.overlapsWith(tr)) return res;
    res.series.push(buf);
    const bufStart = buf.timeRange.start;
    res.gaps = res.gaps.flatMap((gap) => {
      if (gap.start.afterEq(bufStart)) return [];
      if (gap.end.after(bufStart)) return [new TimeRange(gap.start, bufStart)];
      return [gap];
    });
    return res;
  }

  gc(): GCMetrics {
    this.checkOpen("gc");
    return this.static.gc();
  }

  close(): void {
    this.closed = true;
    this.dynamic.close();
    this.static.close();
  }

  private checkOpen(op: string): void {
    if (this.closed)
      throw new UnexpectedError(`${op} called on a closed telemetry cache entry`);
  }
}

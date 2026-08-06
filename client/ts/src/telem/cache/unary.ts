// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MultiSeries, type Series, type TimeRange } from "@synnaxlabs/x";

import { UnexpectedError } from "@/errors";
import { Dynamic, type DynamicProps } from "@/telem/cache/dynamic";
import {
  type DirtyReadResult,
  type GCMetrics,
  Static,
  type StaticProps,
} from "@/telem/cache/static";

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
    // Buffers that have been flushed out of the dynamic cache are written to the
    // static cache.
    if (flushed.length > 0) this.static.write(flushed);
    return allocated;
  }

  get leadingBuffer(): Series | null {
    this.checkOpen("leadingBuffer");
    return this.dynamic.leadingBuffer;
  }

  writeStatic(series: MultiSeries): void {
    this.checkOpen("writeStatic");
    this.static.write(series);
  }

  read(tr: TimeRange): DirtyReadResult {
    this.checkOpen("read");
    return this.static.dirtyRead(tr);
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { type channel, type TimeRange } from "@synnaxlabs/client";
import { MultiSeries, type Series, Size } from "@synnaxlabs/x";

import { Dynamic, type DynamicProps } from "@/telem/client/cache/dynamic";
import {
  type CacheGCMetrics as CacheGCResult,
  type DirtyReadResult,
  Static,
  type StaticProps,
} from "@/telem/client/cache/static";

interface UnaryProps extends StaticProps, Pick<DynamicProps, "dynamicBufferSize"> {
  channel: channel.Payload;
}

export class Unary {
  readonly channel: channel.Payload;
  private closed: boolean = false;
  private readonly ins: alamos.Instrumentation;
  private readonly static: Static;
  private readonly dynamic: Dynamic;

  constructor(props: UnaryProps) {
    this.channel = props.channel;
    this.ins = props.instrumentation ?? alamos.NOOP;
    this.static = new Static(props);
    this.dynamic = new Dynamic({
      dynamicBufferSize: props.dynamicBufferSize,
      dataType: this.channel.dataType,
    });
  }

  writeDynamic(series: MultiSeries): MultiSeries {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted dynamic write to a closed cache for channel ${this.channel.name}`,
      );
      return new MultiSeries([]);
    }
    const { flushed, allocated } = this.dynamic.write(series);
    // Buffers that have been flushed out of the dynamic cache are written to the
    // static cache.
    if (flushed.length > 0) this.static.write(flushed);
    return allocated;
  }

  get leadingBuffer(): Series | null {
    return this.dynamic.leadingBuffer;
  }

  writeStatic(series: MultiSeries): void {
    if (this.closed)
      return this.ins.L.warn(
        `Ignoring attempted static write to a closed cache for channel ${this.channel.name}`,
      );
    this.static.write(series);
  }

  read(tr: TimeRange): DirtyReadResult {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted dirty read from a closed cache for channel ${this.channel.name}`,
      );
      return { series: new MultiSeries([]), gaps: [tr] };
    }
    return this.static.dirtyRead(tr);
  }

  gc(): CacheGCResult {
    if (this.closed) {
      this.ins.L.warn(
        `Ignoring attempted garbage collection on a closed cache for channel ${this.channel.name}`,
      );
      return { purgedSeries: 0, purgedBytes: Size.bytes(0) };
    }
    return this.static.gc();
  }

  close(): void {
    this.closed = true;
    this.dynamic.close();
    this.static.close();
  }
}

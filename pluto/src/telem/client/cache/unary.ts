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

// ===== TEMPORARY sy-4326 wire instrumentation. Remove once root cause found. =====
// Logs each streamed frame that fails to continue the previous one for the same
// channel: the gap, its direction (FWD = missed samples, BACK = re-sent/overlapping
// samples), and any alignment-domain change. This is what the wire actually delivers,
// before the cache touches it. Bounded per-channel and overall so it can't flood.
const sy4326PrevEnd = new Map<string, number>();
const sy4326PrevDom = new Map<string, number>();
const sy4326PerChan = new Map<string, number>();
let sy4326Total = 0;
const sy4326Wire = (name: string, series: MultiSeries): void => {
  for (const s of series.series) {
    const dom = Number(s.alignment >> 32n);
    const sample = Number(s.alignment & 0xffffffffn);
    const prevEnd = sy4326PrevEnd.get(name);
    const prevDom = sy4326PrevDom.get(name);
    if (prevEnd != null) {
      const gap = sample - prevEnd;
      const domChanged = prevDom !== dom;
      const perChan = sy4326PerChan.get(name) ?? 0;
      if ((Math.abs(gap) > 1 || domChanged) && perChan < 6 && sy4326Total < 60) {
        sy4326PerChan.set(name, perChan + 1);
        sy4326Total++;
        console.warn(
          `[sy4326 wire] ${name} ${gap >= 0 ? "FWD" : "BACK"} gap=${gap}` +
            `${domChanged ? ` DOMAIN ${prevDom}->${dom}` : ""} ` +
            `prevEnd=${prevEnd} sample=${sample} dom=${dom} len=${s.length}`,
        );
      }
    }
    sy4326PrevEnd.set(name, sample + s.length);
    sy4326PrevDom.set(name, dom);
  }
};

// Logs the series ranges returned by each historical read (dom:start-end), so we can
// confirm whether the overlap also comes from historical reads or only from streaming.
let sy4326StaticTotal = 0;
const sy4326Static = (name: string, series: MultiSeries): void => {
  if (sy4326StaticTotal >= 40) return;
  sy4326StaticTotal++;
  const ranges = series.series.map((s) => {
    const start = Number(s.alignment & 0xffffffffn);
    return `${Number(s.alignment >> 32n)}:${start}-${start + s.length}`;
  });
  console.warn(
    `[sy4326 static] ${name} read -> ${series.series.length} series: ${ranges.join(" ")}`,
  );
};

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
    sy4326Wire(this.channel.name, series);
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
    sy4326Static(this.channel.name, series);
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

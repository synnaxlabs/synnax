// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type framer, status as cstatus } from "@synnaxlabs/client";
import {
  bounds,
  DataType,
  type destructor,
  MultiSeries,
  type Series,
  Series as XSeries,
  TimeRange,
  TimeSpan,
  TimeStamp,
  unique,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type status } from "@/status/aether";
import { type CreateOptions } from "@/telem/aether/factory";
import { type Client, DISCONNECTED_STATUS } from "@/telem/aether/remote";
import {
  AbstractSource,
  type SeriesSource,
  type SeriesSourceSpec,
  type SeriesValue,
} from "@/telem/aether/telem";

export const triggerEdgeZ = z.enum(["rising", "falling"]);
export type TriggerEdge = z.infer<typeof triggerEdgeZ>;

export const triggeredDataPropsZ = z.object({
  /** The channel whose samples form the frame. */
  channel: z.number(),
  /** The channel scanned for the level crossing. */
  trigger: z.number(),
  level: z.number().default(0),
  edge: triggerEdgeZ.default("rising"),
  /** The span of one frame. */
  span: TimeSpan.z,
  /** The fraction of the span shown before the crossing. */
  pretrigger: z.number().default(0.1),
  /** How long to wait for a crossing before showing the latest span instead. */
  timeout: TimeSpan.z.default(TimeSpan.seconds(1)),
  /** `x` emits seconds since the crossing, `y` the channel's samples. */
  output: z.enum(["x", "y"]),
});

export type TriggeredDataProps = z.input<typeof triggeredDataPropsZ>;

interface Frame {
  x: Series;
  y: Series;
}

const EMPTY: SeriesValue = [bounds.INVALID, new MultiSeries([])];

const lastTimestamp = (index: MultiSeries): number | null => {
  for (let i = index.series.length - 1; i >= 0; i--) {
    const s = index.series[i];
    if (s.length > 0) return Number(s.at(-1, true));
  }
  return null;
};

/**
 * Cuts one frame of a channel around a level crossing on a trigger channel, as an
 * oscilloscope does. The frame spans `span`, with `pretrigger` of it before the
 * crossing, and X is seconds since the crossing so successive frames land on top of
 * each other. When no crossing arrives within `timeout`, the latest span is shown so
 * the plot never blanks. While held, the last frame is served.
 */
export class TriggeredData
  extends AbstractSource<typeof triggeredDataPropsZ>
  implements SeriesSource
{
  static readonly TYPE = "triggered-data";
  schema = triggeredDataPropsZ;

  private readonly client: Client | null;
  private readonly onStatusChange?: status.Adder;
  private readonly now: () => TimeStamp;
  private readonly buffers = new Map<channel.Key, MultiSeries>();
  private keys: {
    data: channel.Key;
    index: channel.Key;
    trigger: channel.Key;
    triggerIndex: channel.Key;
  } | null = null;
  private stopStreaming?: destructor.Destructor;
  private valid = false;
  private generation = 0;
  private held = false;
  private frame: Frame | null = null;
  private pending: number | null = null;
  private scanned: bigint = -1n;
  private previous = NaN;
  private lastTriggerAt: TimeStamp | null = null;

  constructor(
    client: Client | null,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = () => TimeStamp.now(),
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.now = now;
  }

  setHold(held: boolean): void {
    this.held = held;
    if (!held) this.scan();
  }

  value(): SeriesValue {
    if (!this.valid) void this.read();
    const { frame } = this;
    if (frame == null) return EMPTY;
    if (this.props.output === "y") return [frame.y.bounds, new MultiSeries([frame.y])];
    const span = this.props.span.seconds;
    const { pretrigger } = this.props;
    return [
      { lower: -pretrigger * span, upper: (1 - pretrigger) * span },
      new MultiSeries([frame.x]),
    ];
  }

  cleanup(): void {
    this.generation++;
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.buffers.forEach((b) => b.release());
    this.buffers.clear();
    this.valid = false;
    this.frame = null;
  }

  private buffer(key: channel.Key): MultiSeries {
    let b = this.buffers.get(key);
    if (b == null) {
      b = new MultiSeries([]);
      this.buffers.set(key, b);
    }
    return b;
  }

  private pushNew(key: channel.Key, series: Series[]): void {
    const buf = this.buffer(key);
    for (const s of series) {
      if (buf.series.includes(s)) continue;
      s.acquire();
      const at = buf.series.findIndex((held) => held.alignment > s.alignment);
      if (at === -1) buf.push(s);
      else buf.series.splice(at, 0, s);
    }
  }

  /** Never rejects: a failure invalidates the read and reaches onStatusChange. */
  private async read(): Promise<void> {
    const generation = this.generation;
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(DISCONNECTED_STATUS);
      return;
    }
    try {
      const { channel, trigger, span, timeout } = this.props;
      const data = await client.channels.retrieve(channel);
      const trig = trigger === channel ? data : await client.channels.retrieve(trigger);
      if (generation !== this.generation) return;
      if (data.index === 0 || trig.index === 0) {
        this.onStatusChange?.({
          variant: "warning",
          message: "Triggered window needs indexed channels",
        });
        return;
      }
      this.keys = {
        data: data.key,
        index: data.index,
        trigger: trig.key,
        triggerIndex: trig.index,
      };
      const keys = unique.unique(Object.values(this.keys));
      const handler: framer.StreamHandler = (res) => {
        if (generation !== this.generation) return;
        for (const key of keys) {
          const series = res.get(key);
          if (series != null) this.pushNew(key, series.series);
        }
        this.scan();
        this.gc();
      };
      this.stopStreaming?.();
      this.stopStreaming = client.feed.stream(handler, keys).close;
      const keep = this.keepFor(span, timeout);
      for (const key of keys) {
        const res = await client.feed.read(this.now().spanRange(-keep), key);
        if (generation !== this.generation) return;
        this.pushNew(key, res.series);
      }
      this.scan();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(
        cstatus.fromException(e, "Failed to stream triggered data"),
      );
    }
  }

  private keepFor(span: TimeSpan, timeout: TimeSpan): TimeSpan {
    return new TimeSpan(
      Math.max(Number(span.valueOf()) * 3, Number(timeout.valueOf()) * 2),
    );
  }

  private scan(): void {
    if (this.held || this.keys == null) return;
    const { level, edge, span, pretrigger, timeout } = this.props;
    const trig = this.buffer(this.keys.trigger);
    const trigIndex = this.buffer(this.keys.triggerIndex);
    for (const s of trig.series) {
      const start =
        this.scanned < s.alignment ? 0 : Number(this.scanned - s.alignment) + 1;
      for (let i = start; i < s.length; i++) {
        const a = s.alignment + BigInt(i);
        const v = Number(s.at(i, true));
        const crossed =
          edge === "rising"
            ? this.previous < level && v >= level
            : this.previous > level && v <= level;
        if (crossed && this.pending == null) {
          const t = trigIndex.atAlignment(a);
          if (t != null) {
            this.pending = Number(t);
            this.lastTriggerAt = this.now();
          }
        }
        this.previous = v;
        this.scanned = a;
      }
    }
    const index = this.buffer(this.keys.index);
    const latest = lastTimestamp(index);
    if (latest == null) return;
    const spanNs = Number(span.valueOf());
    if (this.pending != null) {
      const end = this.pending + (1 - pretrigger) * spanNs;
      if (latest < end) return;
      this.cut(this.pending);
      this.pending = null;
      return;
    }
    const waited =
      this.lastTriggerAt == null
        ? Infinity
        : Number(this.now().sub(this.lastTriggerAt).valueOf());
    if (waited < Number(timeout.valueOf())) return;
    this.cut(latest - (1 - pretrigger) * spanNs);
  }

  private cut(t: number): void {
    if (this.keys == null) return;
    const { span, pretrigger } = this.props;
    const spanNs = Number(span.valueOf());
    const lo = t - pretrigger * spanNs;
    const hi = t + (1 - pretrigger) * spanNs;
    const index = this.buffer(this.keys.index);
    const data = this.buffer(this.keys.data);
    const xs: number[] = [];
    const ys: number[] = [];
    for (const ix of index.series) {
      if (ix.length === 0) continue;
      const start = ix.binarySearch(lo);
      let end = ix.binarySearch(hi);
      if (end < ix.length && Number(ix.at(end, true)) === hi) end++;
      for (let i = start; i < end; i++) {
        const v = data.atAlignment(ix.alignment + BigInt(i) * ix.alignmentMultiple);
        if (v == null) continue;
        xs.push((Number(ix.at(i, true)) - t) / 1e9);
        ys.push(Number(v));
      }
    }
    if (xs.length < 2) return;
    const timeRange = new TimeRange(lo, hi);
    this.frame = {
      x: new XSeries({
        data: Float32Array.from(xs),
        dataType: DataType.FLOAT32,
        timeRange,
        alignment: 0n,
      }),
      y: new XSeries({
        data: Float32Array.from(ys),
        dataType: DataType.FLOAT32,
        timeRange,
        alignment: 0n,
      }),
    };
    this.notify();
  }

  private gc(): void {
    const { span, timeout } = this.props;
    const threshold = this.now().sub(this.keepFor(span, timeout));
    this.buffers.forEach((buf) => {
      while (buf.series.length > 1 && buf.series[0].timeRange.end.before(threshold))
        buf.series.shift()?.release();
    });
  }
}

export const triggeredData = (props: TriggeredDataProps): SeriesSourceSpec => ({
  type: TriggeredData.TYPE,
  props,
  variant: "source",
  valueType: "series",
});

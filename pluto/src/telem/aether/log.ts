// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import {
  DataType,
  type destructor,
  type Series,
  status as xstatus,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type status } from "@/status/aether";
import { type CreateOptions } from "@/telem/aether/factory";
import {
  AbstractSource,
  type LogEntry,
  type LogSource,
  type LogSourceSpec,
} from "@/telem/aether/telem";
import { type client } from "@/telem/client";

const MAX_ENTRIES = 100_000;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(z.number().or(z.string())),
  timeSpan: TimeSpan.z,
  keepFor: TimeSpan.z.optional(),
});

export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;

interface ChannelMeta {
  key: channel.Key;
  name: string;
  indexKey: channel.Key;
  dataType: DataType;
  virtual: boolean;
  // Reference to the dynamic cache's leading buffer for this channel. writeDynamic()
  // only returns newly *allocated* buffers — subsequent writes go into the existing
  // buffer in-place and return an empty MultiSeries. We store the allocated buffer
  // on first write and advance readCursor on every callback to read only new samples.
  leadingBuffer: Series | null;
  readCursor: number;
  // Whitespace appended after "]" in multi-channel mode to align the value column
  // across channels of different name lengths. Computed once in read().
  padding: string;
}

export class StreamMultiChannelLog
  extends AbstractSource<typeof streamMultiChannelLogPropsZ>
  implements LogSource
{
  static readonly TYPE = "stream-multi-channel-log";
  schema = streamMultiChannelLogPropsZ;

  private readonly client: client.Client;
  private readonly onStatusChange?: status.Adder;
  private readonly now: () => TimeStamp;
  private channelMeta: Map<channel.Key, ChannelMeta> = new Map();
  private entries: LogEntry[] = [];
  private stopStreaming?: destructor.Destructor;
  private valid = false;

  constructor(
    client: client.Client,
    props: unknown,
    options?: CreateOptions,
    now: () => TimeStamp = () => TimeStamp.now(),
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.now = now;
  }

  value(): LogEntry[] {
    if (this.props.channels.length === 0) return this.entries;
    if (!this.valid) void this.read();
    return this.entries;
  }

  private async read(): Promise<void> {
    try {
      this.valid = true;
      this.stopStreaming?.();
      this.channelMeta.clear();

      const channels = await Promise.all(
        this.props.channels.map((ch) => this.client.retrieveChannel(ch)),
      );

      const maxNameLen = Math.max(...channels.map((ch) => ch.name.length));
      for (const ch of channels)
        this.channelMeta.set(ch.key, {
          key: ch.key,
          name: ch.name,
          indexKey: ch.index,
          leadingBuffer: null,
          readCursor: 0,
          dataType: new DataType(ch.dataType),
          virtual: ch.virtual,
          padding: " ".repeat(maxNameLen - ch.name.length),
        });

      const streamKeys = channels.map((ch) => ch.key);
      this.stopStreaming = await this.client.stream((res) => {
        // Intentionally use receipt time rather than the sample's actual timestamp,
        // and intentionally do not sort. The log is an arrival-order display — using
        // receipt time keeps entries strictly append-only and avoids out-of-order
        // jumps caused by natural network latency between channels. Ms-level blur
        // between receipt and sample time is not meaningful for a human reading a log.
        const now = this.now();
        const before = this.entries.length;
        for (const [channelKey, chMeta] of this.channelMeta) {
          const allocated = res.get(channelKey);
          const isJSON = chMeta.dataType.equals(DataType.JSON);
          const pushSamples = (buf: Series, start: number): void => {
            for (let i = start; i < buf.length; i++) {
              const raw = buf.at(i, true);
              this.entries.push({
                channelKey: chMeta.key,
                channelName: chMeta.name,
                channelPadding: chMeta.padding,
                timestamp: now.valueOf(),
                value: isJSON ? JSON.stringify(raw) : String(raw),
              });
            }
          };
          if (allocated != null && allocated.series.length > 0) {
            // Drain the old leading buffer's unread tail before switching.
            if (chMeta.leadingBuffer != null)
              pushSamples(chMeta.leadingBuffer, chMeta.readCursor);
            // Drain intermediate allocations (burst crossed multiple boundaries).
            for (let s = 0; s < allocated.series.length - 1; s++)
              pushSamples(allocated.series[s], 0);
            [chMeta.leadingBuffer, chMeta.readCursor] = [
              allocated.series[allocated.series.length - 1],
              0,
            ];
          }
          // Read newly written samples from the current leading buffer.
          const buf = chMeta.leadingBuffer;
          if (buf == null || buf.length <= chMeta.readCursor) continue;
          pushSamples(buf, chMeta.readCursor);
          chMeta.readCursor = buf.length;
        }
        this.gcEntries();
        if (this.entries.length !== before) this.notify();
      }, streamKeys);
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(xstatus.fromException(e, "failed to stream log channels"));
    }
  }

  private gcEntries(): void {
    const keepFor = this.props.keepFor ?? this.props.timeSpan;
    const threshold = this.now().sub(keepFor).valueOf();
    // Find the index of the first entry that should be kept (O(n)), then remove
    // everything before it in one splice call. Prefer this over a shift() loop, where
    // each individual shift is also O(n) because it moves all remaining elements forward
    // in memory — making the loop O(n²) when many entries are expired.
    const cutoff = this.entries.findIndex((e) => e.timestamp >= threshold);
    if (cutoff > 0) this.entries.splice(0, cutoff);
    if (this.entries.length > MAX_ENTRIES)
      this.entries.splice(0, this.entries.length - MAX_ENTRIES);
  }

  cleanup(): void {
    this.stopStreaming?.();
    this.stopStreaming = undefined;
    this.entries = [];
    this.channelMeta.clear();
    this.valid = false;
  }
}

export const streamMultiChannelLog = (
  props: StreamMultiChannelLogProps,
): LogSourceSpec => ({
  type: StreamMultiChannelLog.TYPE,
  props,
  variant: "source",
  valueType: "log",
});

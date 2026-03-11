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

export type StreamMultiChannelLogProps = z.input<
  typeof streamMultiChannelLogPropsZ
>;

interface ChannelMeta {
  key: channel.Key;
  name: string;
  indexKey: channel.Key;
  dataType: DataType;
  virtual: boolean;
}

export class StreamMultiChannelLog
  extends AbstractSource<typeof streamMultiChannelLogPropsZ>
  implements LogSource
{
  static readonly TYPE = "stream-multi-channel-log";
  schema = streamMultiChannelLogPropsZ;

  private readonly client: client.Client;
  private readonly onStatusChange?: status.Adder;
  private channelMeta: Map<channel.Key, ChannelMeta> = new Map();
  private entries: LogEntry[] = [];
  private stopStreaming?: destructor.Destructor;
  private valid = false;

  constructor(client: client.Client, props: unknown, options?: CreateOptions) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
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

      for (const ch of channels)
        this.channelMeta.set(ch.key, {
          key: ch.key,
          name: ch.name,
          indexKey: ch.index,
          dataType: new DataType(ch.dataType),
          virtual: ch.virtual,
        });

      const streamKeys = channels.map((ch) => ch.key);
      this.stopStreaming = await this.client.stream((res) => {
        for (const [meta] of this.channelMeta) {
          const series = res.get(meta);
          if (series == null) continue;
          const chMeta = this.channelMeta.get(meta)!;
          // Intentionally use receipt time rather than the sample's actual timestamp,
          // and intentionally do not sort. The log is an arrival-order display — using
          // receipt time keeps entries strictly append-only and avoids out-of-order
          // jumps caused by natural network latency between channels. Ms-level blur
          // between receipt and sample time is not meaningful for a human reading a log.
          const now = TimeStamp.now();
          for (const s of series.series) {
            const isJSON = chMeta.dataType.equals(DataType.JSON);
            for (let i = 0; i < s.length; i++) {
              const raw = s.at(i, true);
              this.entries.push({
                channelKey: chMeta.key,
                channelName: chMeta.name,
                timestamp: now.valueOf(),
                value: isJSON ? JSON.stringify(raw) : String(raw),
              });
            }
          }
        }
        this.gcEntries();
        this.notify();
      }, streamKeys);
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(
        xstatus.fromException(e, "failed to stream log channels"),
      );
    }
  }

  private gcEntries(): void {
    const keepFor = this.props.keepFor ?? this.props.timeSpan;
    const threshold = TimeStamp.now().sub(keepFor).valueOf();
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

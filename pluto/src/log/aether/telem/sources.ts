// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel } from "@synnaxlabs/client";
import {
  compare,
  DataType,
  type destructor,
  type Series,
  status as xstatus,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type LogEntry, type LogSource, type LogSourceSpec } from "@/log/aether/telem/types";
import { type status } from "@/status/aether";
import { type CreateOptions } from "@/telem/aether/factory";
import { AbstractSource } from "@/telem/aether/telem";
import { type client } from "@/telem/client";

const MAX_ENTRIES = 100_000;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(channel.keyZ.or(z.string())),
  timeSpan: TimeSpan.z,
  keepFor: TimeSpan.z.optional(),
});

export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;

interface ChannelMeta {
  key: channel.Key;
  dataType: DataType;
  leadingBuffer: Series | null;
  readCursor: number;
  skipSeed: boolean;
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
  private _evictedCount: number = 0;
  private _channels: Array<number | string> = [];
  private readGeneration = 0;

  get evictedCount(): number {
    return this._evictedCount;
  }

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
    this._channels = this.props.channels;
  }

  value(): LogEntry[] {
    if (this._channels.length === 0) return this.entries;
    if (!this.valid) void this.read();
    return this.entries;
  }

  setChannels(channels: Array<number | string>): void {
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    this.valid = false;
    if (channels.length === 0) {
      this.stopStreaming?.();
      this.stopStreaming = undefined;
      this.entries = [];
      this.channelMeta.clear();
      this._evictedCount = 0;
    }
    this.notify();
  }

  private async read(): Promise<void> {
    try {
      this.valid = true;
      // Generation counter prevents stale async completions: if setChannels() triggers
      // a new read() while this one is awaiting, the older read bails out.
      const generation = ++this.readGeneration;
      this.stopStreaming?.();

      const channels = await Promise.all(
        this._channels.map((ch) => this.client.retrieveChannel(ch)),
      );
      // Superseded by a newer read() call while we were awaiting.
      if (generation !== this.readGeneration) return;

      // Scrub entries from channels that were removed.
      const newKeys = new Set(channels.map((ch) => ch.key));
      const removedKeys = new Set(
        [...this.channelMeta.keys()].filter((k) => !newKeys.has(k)),
      );
      if (removedKeys.size > 0)
        this.entries = this.entries.filter((e) => !removedKeys.has(e.channelKey));

      // When channels change mid-session, the new stream seeds with buffered data
      // for ALL channels. Skip the seed to avoid re-displaying existing entries.
      const isRestart = this.channelMeta.size > 0;
      this.channelMeta.clear();
      for (const ch of channels)
        this.channelMeta.set(ch.key, {
          key: ch.key,
          leadingBuffer: null,
          readCursor: 0,
          dataType: new DataType(ch.dataType),
          skipSeed: isRestart,
        });

      const streamKeys = channels.map((ch) => ch.key);
      this.stopStreaming = await this.client.stream((res) => {
        const now = this.now();
        let pushed = 0;
        for (const [key, chMeta] of this.channelMeta) {
          const allocated = res.get(key);
          const isJSON = chMeta.dataType.equals(DataType.JSON);
          const pushSamples = (buf: Series, start: number): void => {
            for (let i = start; i < buf.length; i++) {
              const raw = buf.at(i, true);
              this.entries.push({
                channelKey: chMeta.key,
                timestamp: now.valueOf(),
                value: isJSON ? JSON.stringify(raw) : String(raw),
              });
              pushed++;
            }
          };
          if (allocated != null && allocated.series.length > 0) {
            // First callback after channel change: skip buffered data.
            if (chMeta.skipSeed) {
              const lastSeries = allocated.series[allocated.series.length - 1];
              chMeta.leadingBuffer = lastSeries;
              chMeta.readCursor = lastSeries.length;
              chMeta.skipSeed = false;
              continue;
            }
            // Drain unread tail of the old leading buffer before switching.
            if (chMeta.leadingBuffer != null)
              pushSamples(chMeta.leadingBuffer, chMeta.readCursor);
            // Drain intermediate buffers (burst crossed multiple allocations).
            for (let s = 0; s < allocated.series.length - 1; s++)
              pushSamples(allocated.series[s], 0);
            // Switch to the newest buffer.
            [chMeta.leadingBuffer, chMeta.readCursor] = [
              allocated.series[allocated.series.length - 1],
              0,
            ];
          }
          // Read new in-place writes to the current leading buffer.
          const buf = chMeta.leadingBuffer;
          if (buf == null || buf.length <= chMeta.readCursor) continue;
          if (chMeta.skipSeed) {
            chMeta.readCursor = buf.length;
            chMeta.skipSeed = false;
            continue;
          }
          pushSamples(buf, chMeta.readCursor);
          chMeta.readCursor = buf.length;
        }
        this._evictedCount = this.gcEntries();
        if (pushed > 0) this.notify();
      }, streamKeys);
      this.notify();
    } catch (e) {
      this.valid = false;
      this.onStatusChange?.(xstatus.fromException(e, "failed to stream log channels"));
    }
  }

  // Evicts stale entries from the front of the array. Returns the count so
  // the caller can adjust scroll offset and selection indices.
  private gcEntries(): number {
    const keepFor = this.props.keepFor ?? this.props.timeSpan;
    const threshold = this.now().sub(keepFor).valueOf();
    // Single splice instead of shift() loop to avoid O(n²).
    const cutoff = this.entries.findIndex((e) => e.timestamp >= threshold);
    let evicted = 0;
    if (cutoff > 0) {
      this.entries.splice(0, cutoff);
      evicted += cutoff;
    }
    if (this.entries.length > MAX_ENTRIES) {
      const excess = this.entries.length - MAX_ENTRIES;
      this.entries.splice(0, excess);
      evicted += excess;
    }
    return evicted;
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

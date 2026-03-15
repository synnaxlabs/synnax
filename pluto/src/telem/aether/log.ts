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
  compare,
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
  displayName: string;
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
  // When true, the first stream callback for this channel should skip all data
  // (advance readCursor to end of buffer). This prevents the stream's initial seed
  // from duplicating entries we already have for previously-active channels.
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
  private _aliases: Record<string, string> = {};
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
    // Short-circuit if channels haven't actually changed.
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    this.valid = false;
    if (channels.length === 0) {
      this.stopStreaming?.();
      this.stopStreaming = undefined;
    }
    this.notify();
  }

  setAliases(aliases: Record<string, string>): void {
    this._aliases = aliases;
    let changed = false;
    for (const [key, meta] of this.channelMeta) {
      const displayName = aliases[String(key)] || meta.name;
      if (meta.displayName !== displayName) {
        meta.displayName = displayName;
        changed = true;
      }
    }
    if (!changed) return;
    this.recomputePadding();
    for (const entry of this.entries) {
      const meta = this.channelMeta.get(entry.channelKey);
      if (meta == null) continue;
      entry.channelName = meta.displayName;
      entry.channelPadding = meta.padding;
    }
    this.notify();
  }

  private recomputePadding(): void {
    const maxLen = Math.max(
      0,
      ...[...this.channelMeta.values()].map((m) => m.displayName.length),
    );
    for (const meta of this.channelMeta.values())
      meta.padding = " ".repeat(maxLen - meta.displayName.length);
  }

  private async read(): Promise<void> {
    try {
      this.valid = true;
      this.stopStreaming?.();

      const channels = await Promise.all(
        this._channels.map((ch) => this.client.retrieveChannel(ch)),
      );

      // Compute removed channels BEFORE clearing meta
      const newKeys = new Set(channels.map((ch) => ch.key));
      const removedKeys = new Set(
        [...this.channelMeta.keys()].filter((k) => !newKeys.has(k)),
      );

      // Filter entries from removed channels
      if (removedKeys.size > 0)
        this.entries = this.entries.filter((e) => !removedKeys.has(e.channelKey));

      // When restarting an existing stream, skip the seed for ALL channels to avoid
      // duplicating entries (previously-active) or dumping historical cache (newly-added).
      // On initial start (no previous channels), allow the seed so data appears immediately.
      const isRestart = this.channelMeta.size > 0;
      this.channelMeta.clear();
      for (const ch of channels) {
        const displayName = this._aliases[String(ch.key)] || ch.name;
        this.channelMeta.set(ch.key, {
          key: ch.key,
          name: ch.name,
          displayName,
          indexKey: ch.index,
          leadingBuffer: null,
          readCursor: 0,
          dataType: new DataType(ch.dataType),
          virtual: ch.virtual,
          padding: "",
          skipSeed: isRestart,
        });
      }
      this.recomputePadding();

      // Update names and padding on existing entries (channel may have been
      // renamed or max name length may have changed).
      for (const entry of this.entries) {
        const meta = this.channelMeta.get(entry.channelKey);
        if (meta == null) continue;
        entry.channelName = meta.displayName;
        entry.channelPadding = meta.padding;
      }

      const streamKeys = channels.map((ch) => ch.key);
      this.stopStreaming = await this.client.stream((res) => {
        // Intentionally use receipt time rather than the sample's actual timestamp,
        // and intentionally do not sort. The log is an arrival-order display — using
        // receipt time keeps entries strictly append-only and avoids out-of-order
        // jumps caused by natural network latency between channels. Ms-level blur
        // between receipt and sample time is not meaningful for a human reading a log.
        const now = this.now();
        let pushed = 0;
        for (const [channelKey, chMeta] of this.channelMeta) {
          const allocated = res.get(channelKey);
          const isJSON = chMeta.dataType.equals(DataType.JSON);
          const pushSamples = (buf: Series, start: number): void => {
            for (let i = start; i < buf.length; i++) {
              const raw = buf.at(i, true);
              this.entries.push({
                channelKey: chMeta.key,
                channelName: chMeta.displayName,
                channelPadding: chMeta.padding,
                timestamp: now.valueOf(),
                value: isJSON ? JSON.stringify(raw) : String(raw),
              });
              pushed++;
            }
          };
          if (allocated != null && allocated.series.length > 0) {
            if (chMeta.skipSeed) {
              // First callback after stream restart for a previously-active channel.
              // The stream seeds us with existing dynamic buffer contents that we've
              // already consumed — skip all of it to avoid duplicate entries.
              const lastSeries = allocated.series[allocated.series.length - 1];
              chMeta.leadingBuffer = lastSeries;
              chMeta.readCursor = lastSeries.length;
              chMeta.skipSeed = false;
              continue;
            }
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
          // If skipSeed is still true here (no allocation in seed, just leading buffer
          // data), skip the existing data and clear the flag.
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

  private gcEntries(): number {
    const keepFor = this.props.keepFor ?? this.props.timeSpan;
    const threshold = this.now().sub(keepFor).valueOf();
    // Find the index of the first entry that should be kept (O(n)), then remove
    // everything before it in one splice call. Prefer this over a shift() loop, where
    // each individual shift is also O(n) because it moves all remaining elements forward
    // in memory — making the loop O(n²) when many entries are expired.
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

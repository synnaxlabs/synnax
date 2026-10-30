// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  type framer,
  status as cstatus,
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  compare,
  DataType,
  dynamicWindow,
  type Series,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  type LogEntry,
  type LogSource,
  type LogSourceSpec,
} from "@/log/aether/telem/types";
import { type status } from "@/status/aether";
import { telem } from "@/telem/aether";

const MAX_ENTRIES = 100_000;
// Past maxEntries, a log evicts this share of maxEntries beyond the excess, so a full
// log moves its entries once per share instead of on every pull.
const EVICTION_HEADROOM = 0.1;
// The newest stored samples of each channel a log shows when it opens.
const PAGE_SIZE = 10_000;

const streamMultiChannelLogPropsZ = z.object({
  channels: z.array(channel.keyZ.or(z.string())),
  timeSpan: TimeSpan.z,
  keepFor: TimeSpan.z.optional(),
});

export type StreamMultiChannelLogProps = z.input<typeof streamMultiChannelLogPropsZ>;

// The snapshot and the count of appended samples a channel's entries were last read
// up to.
interface Cursor {
  snapshot: framer.Snapshot;
  appended: number;
}

export class StreamMultiChannelLog
  extends telem.AbstractSource<typeof streamMultiChannelLogPropsZ>
  implements LogSource
{
  static readonly TYPE = "stream-multi-channel-log";
  schema = streamMultiChannelLogPropsZ;

  private readonly client: telem.Client | null;
  private readonly onStatusChange?: status.Adder;
  private readonly now: () => TimeStamp;
  private readonly maxEntries: number;
  private readonly failures: telem.FailureReporter;
  private subscription: framer.Subscription | null = null;
  private readonly cursors = new Map<channel.Key, Cursor>();
  private entries: LogEntry[] = [];
  // status is the worst status of the channels' snapshots at the last pull.
  private status = telem.SUCCESS_STATUS;
  private valid = false;
  private _channels: channel.Key[] = [];

  constructor(
    client: telem.Client | null,
    props: unknown,
    options?: telem.CreateOptions,
    now: () => TimeStamp = () => TimeStamp.now(),
    maxEntries: number = MAX_ENTRIES,
  ) {
    super(props);
    this.client = client;
    this.onStatusChange = options?.onStatusChange;
    this.failures = new telem.FailureReporter(this.onStatusChange);
    this.now = now;
    this.maxEntries = maxEntries;
    this._channels = this.props.channels.filter(
      (ch): ch is number => typeof ch === "number",
    );
  }

  /** @returns the entries, with the worst status of the channels' snapshots. */
  value(): telem.Result<LogEntry[]> {
    if (this._channels.length > 0 && !this.valid) this.subscribe();
    return { data: this.entries, status: this.status };
  }

  /** Channels that stay keep their entries and samples, so none show twice. */
  setChannels(channels: channel.Key[]): void {
    if (compare.primitiveArrays(this._channels, channels) === compare.EQUAL) return;
    this._channels = channels;
    const kept = new Set(channels);
    this.entries = this.entries.filter((e) => kept.has(e.channelKey));
    this.cursors.forEach((_, key) => {
      if (!kept.has(key)) this.cursors.delete(key);
    });
    const { subscription } = this;
    if (channels.length === 0) {
      subscription?.close();
      this.subscription = null;
      this.status = telem.SUCCESS_STATUS;
      this.valid = false;
    } else if (subscription != null) {
      subscription.update(this.request());
      this.pull(subscription);
    }
    this.notify();
  }

  private request(): framer.LatestRequest {
    return {
      keys: [...this._channels],
      window: dynamicWindow(this.props.timeSpan),
      limit: { variant: "latest", count: PAGE_SIZE },
    };
  }

  private subscribe(): void {
    this.valid = true;
    const { client } = this;
    if (client == null) {
      this.onStatusChange?.(telem.DISCONNECTED_STATUS);
      return;
    }
    const subscription = client.feed.subscribe(this.request());
    this.subscription = subscription;
    this.status = telem.LOADING_STATUS;
    subscription.onChange(() => {
      if (this.pull(subscription)) this.notify();
    });
  }

  // Appends the samples each channel received since the last pull, and returns whether
  // the entries or the status changed. A channel's entries wait while it loads, so its
  // history lands in one pull before its live samples and takes the time it loaded.
  private pull(subscription: framer.Subscription): boolean {
    const now = this.now();
    let pushed = 0;
    let worst = telem.SUCCESS_STATUS;
    for (const key of this._channels) {
      const snapshot = subscription.value(key);
      const { variant } = snapshot.status;
      worst = cstatus.worst(worst, snapshot.status)!;
      if (variant !== "loading") pushed += this.append(key, snapshot, now);
    }
    const statusChanged = worst !== this.status;
    this.status = worst;
    this.failures.observe(worst);
    const evicted = this.gcEntries();
    return pushed > 0 || evicted > 0 || statusChanged;
  }

  // Appends the samples the channel appended since the key's cursor, which are the
  // newest samples of the snapshot.
  private append(key: channel.Key, snapshot: framer.Snapshot, now: TimeStamp): number {
    const cursor = this.cursors.get(key);
    if (cursor?.snapshot === snapshot) return 0;
    const { series, appended } = snapshot;
    if (appended == null)
      throw new UnexpectedError(`snapshot of channel ${key} has no appended count`);
    let skip = Math.max(0, series.length - appended + (cursor?.appended ?? 0));
    let pushed = 0;
    for (const s of series.series) {
      for (let j = skip; j < s.length; j++) pushed += this.push(key, s, j, now);
      skip = Math.max(0, skip - s.length);
    }
    this.cursors.set(key, { snapshot, appended });
    return pushed;
  }

  private push(key: channel.Key, series: Series, i: number, now: TimeStamp): number {
    const value = series.asString(i, true);
    if (!series.dataType.equals(DataType.STRING) || !value.includes("\n")) {
      this.entries.push({ channelKey: key, timestamp: now, value });
      return 1;
    }
    const lines = value.split(/\r?\n/);
    lines.forEach((line, j) =>
      this.entries.push({
        channelKey: key,
        timestamp: now,
        value: line,
        continuation: j > 0,
      }),
    );
    return lines.length;
  }

  // Evicts stale entries from the front of the array, and returns how many.
  private gcEntries(): number {
    const keepFor = this.props.keepFor ?? this.props.timeSpan;
    const threshold = this.now().sub(keepFor);
    // Single splice instead of shift() loop to avoid O(n²).
    const cutoff = this.entries.findIndex((e) => e.timestamp >= threshold);
    let evicted = 0;
    if (cutoff > 0) {
      this.entries.splice(0, cutoff);
      evicted += cutoff;
    }
    if (this.entries.length > this.maxEntries) {
      let excess =
        this.entries.length -
        this.maxEntries +
        Math.floor(this.maxEntries * EVICTION_HEADROOM);
      // Skip leading continuations so the new head isn't an orphan.
      while (excess < this.entries.length && this.entries[excess].continuation)
        excess++;
      this.entries.splice(0, excess);
      evicted += excess;
    }
    return evicted;
  }

  cleanup(): void {
    this.subscription?.close();
    this.subscription = null;
    this.cursors.clear();
    this.entries = [];
    this.status = telem.SUCCESS_STATUS;
    this.valid = false;
    this.failures.reset();
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

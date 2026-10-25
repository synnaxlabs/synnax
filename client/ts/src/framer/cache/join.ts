// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Series, TimeRange, type TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { search } from "@/framer/cache/block";

/** A run of a streamed index series' samples that falls in one raw block. */
export interface Part {
  /** The position of the run's first sample in the series. */
  from: number;
  /** The position after the run's last sample. */
  to: number;
  /** The raw block that holds the run. */
  block: number;
  /** The block offset of the run's first sample, or null when the block held it. */
  offset: number | null;
}

/** A streamed index series and the blocks its samples went into. */
export interface Mapped {
  series: Series;
  parts: Part[];
}

// The streamed index series a join keeps, so data series written with them find their
// block offsets.
const RECENT = 1024;

const DOMAIN_MASK = (1n << 32n) - 1n;

/** One channel of a join. */
export class Member {
  /** True once the member's catch-up read has landed. */
  joined = false;
  /** Streamed series held while the member waits for its catch-up read. */
  held: Series[] = [];
  /** The number of callers that joined the member. */
  refs = 0;
  /**
   * True when the channel has no history to read, so its first sample in a block takes
   * the offset its timestamp gives.
   */
  historyless = false;

  constructor(readonly key: channel.Key) {}
}

/**
 * The join of one index channel's stream, and the streams of the data channels on it,
 * to their raw blocks. Streamed index samples get block offsets in time order, and
 * data samples take the offsets of the index samples with their timestamps.
 */
export class Join {
  readonly members = new Map<channel.Key, Member>();
  private recent: Mapped[] = [];
  private last: Series | null = null;

  constructor(readonly index: channel.Key) {}

  /** @returns the index channel's member. */
  get indexMember(): Member {
    return this.members.get(this.index)!;
  }

  /**
   * @returns false when the series does not continue the last one in Core alignment,
   * which means the relay dropped frames between them. A new Core domain starts at
   * sample 0.
   */
  continues(s: Series): boolean {
    const { last } = this;
    if (last == null) return true;
    if (s.alignment >> 32n === last.alignment >> 32n)
      return s.alignment <= last.alignment + BigInt(last.length);
    return (s.alignment & DOMAIN_MASK) === 0n;
  }

  /** Records where a streamed index series' samples went. */
  remember(mapped: Mapped): void {
    this.last = mapped.series;
    this.recent.push(mapped);
    if (this.recent.length > RECENT) this.recent.splice(0, this.recent.length - RECENT);
  }

  /** Forgets the last index series, so the next one starts a new run. */
  restart(): void {
    this.last = null;
  }

  /**
   * @returns the remembered index series that holds the timestamp, and the timestamp's
   * position in it.
   */
  locate(ts: TimeStamp): { mapped: Mapped; pos: number } | null {
    const t = ts.valueOf();
    for (let i = this.recent.length - 1; i >= 0; i--) {
      const mapped = this.recent[i];
      const { series } = mapped;
      if (!series.timeRange.contains(ts)) continue;
      const pos = search(series, t);
      if (pos < series.length && BigInt(series.data[pos]) === t) return { mapped, pos };
    }
    return null;
  }
}

/**
 * @returns the samples [from, to) of the series with block alignment and the time range
 * of the index samples in times at the same positions.
 */
export const piece = (
  s: Series,
  from: number,
  to: number,
  times: Series,
  timesFrom: number,
  block: number,
  offset: number,
): Series => {
  const sub = s.sub(from, to);
  const start = BigInt(times.data[timesFrom]);
  const end = BigInt(times.data[timesFrom + (to - from) - 1]) + 1n;
  return new Series({
    data: sub.data,
    dataType: sub.dataType,
    sampleOffset: sub.sampleOffset,
    timeRange: new TimeRange(start, end),
    alignment: (BigInt(block) << 32n) | BigInt(offset),
  });
};

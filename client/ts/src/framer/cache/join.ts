// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MultiSeries, Series, TimeRange, TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import {
  alignmentOf,
  domainOf,
  offsetOf,
  RAW_LEVEL,
  type RawBlock,
  rawIndex,
  search,
} from "@/framer/cache/block";
import { tileRange } from "@/framer/cache/grid";
import { type Store } from "@/framer/cache/store";

// A run of a streamed index series' samples that falls in one raw block. The offset is
// the block offset of the run's first sample, or null when the block held the run.
interface Run {
  from: number;
  to: number;
  block: number;
  offset: number | null;
}

// A streamed index series and the blocks its samples went into.
interface Mapped {
  series: Series;
  runs: Run[];
}

/** A member of a join that waits for a catch-up read, and the wait it is in. */
export interface Wait {
  key: channel.Key;
  wait: number;
}

// The streamed series a join keeps: index series, so the data series of later frames
// find their block offsets, and the series a member holds while it waits.
const RECENT = 1024;

// Appends the item, keeping at least the newest RECENT items. The oldest leave in bulk,
// so an append costs O(1) amortized.
const keep = <T>(list: T[], item: T): T[] => {
  list.push(item);
  return list.length < 2 * RECENT ? list : list.slice(-RECENT);
};

class Member {
  joined = false;
  // held holds streamed series while the member waits for its catch-up read. The
  // oldest leave past a cap, since the catch-up read fetches their span anyway.
  held: Series[] = [];
  refs = 0;
  // historyless is true when the channel has no history to read, so its first sample
  // in a block takes the offset its timestamp gives.
  historyless = false;
  // from is the first raw block the member's catch-up read must fetch, when known.
  from: number | null = null;
  // fresh is true from a landing until the member places a streamed sample.
  fresh = false;
  // wait numbers the member's waits, so a landing knows the wait its read began in.
  wait = 0;

  constructor(
    readonly key: channel.Key,
    readonly join: Join,
  ) {}

  take(): Series[] {
    const { held } = this;
    this.held = [];
    return held;
  }

  hold(s: Series): void {
    this.held = keep(this.held, s);
  }
}

class Join {
  readonly members = new Map<channel.Key, Member>();
  readonly indexMember: Member;
  private recent: Mapped[] = [];
  private last: Series | null = null;

  constructor(readonly index: channel.Key) {
    this.indexMember = new Member(index, this);
    this.members.set(index, this.indexMember);
  }

  // Returns false when the series does not continue the last one in Core alignment,
  // which means the relay dropped frames between them. A new Core domain starts at
  // sample 0.
  continues(s: Series): boolean {
    const { last } = this;
    if (last == null) return true;
    if (domainOf(s.alignment) === domainOf(last.alignment))
      return s.alignment <= last.alignment + BigInt(last.length);
    return offsetOf(s.alignment) === 0;
  }

  remember(mapped: Mapped): void {
    this.last = mapped.series;
    this.recent = keep(this.recent, mapped);
  }

  // Forgets the last index series, so the next one starts a new run.
  restart(): void {
    this.last = null;
  }

  // Returns the remembered index series that holds the Core alignment, and the
  // alignment's position in it.
  locate(alignment: bigint): { mapped: Mapped; pos: number } | null {
    for (let i = this.recent.length - 1; i >= 0; i--) {
      const mapped = this.recent[i];
      const pos = alignment - mapped.series.alignment;
      if (pos >= 0n && pos < BigInt(mapped.series.length))
        return { mapped, pos: Number(pos) };
    }
    return null;
  }

  // Returns true when the series ends before every remembered index series.
  precedes(s: Series): boolean {
    const first = this.recent[0];
    return first != null && s.alignment + BigInt(s.length) <= first.series.alignment;
  }
}

/** The buffers a write allocated, and the joins it sent back to wait. */
export interface Written {
  allocated: Map<channel.Key, MultiSeries>;
  /** The index channels of the joins that wait for a catch-up read. */
  waiting: Set<channel.Key>;
}

/**
 * Writes the stream's frames into the store, and joins the streams of joined channels
 * to their raw blocks. A join holds the streamed series of its index channel and of
 * the data channels on it while it catches up with a read. Then index samples get
 * block offsets in time order, and data samples take the offsets of the index samples
 * that share their Core alignment, from the same frame or an earlier one. A gap in the
 * stream, a new data type, or a change to the history of an open block sends a join
 * back to wait. Every method that can do so returns the index channels of the joins
 * that wait.
 */
export class Joins {
  private readonly joins = new Map<channel.Key, Join>();
  private readonly members = new Map<channel.Key, Member>();
  private readonly waits = new Set<channel.Key>();

  constructor(private readonly store: Store) {}

  /**
   * Adds the channel and its index to the join of the index. Each new member waits for
   * a catch-up read.
   */
  join(ch: channel.Key, index: channel.Key): void {
    let j = this.joins.get(index);
    if (j == null) {
      j = new Join(index);
      this.joins.set(index, j);
    }
    this.addMember(j, index);
    if (ch !== index) this.addMember(j, ch);
  }

  /** Removes the channel and its index from the join, undoing one call to join. */
  leave(ch: channel.Key, index: channel.Key): void {
    const j = this.joins.get(index);
    if (j == null) return;
    if (ch !== index) this.removeMember(j, ch);
    this.removeMember(j, index);
    if (j.members.size === 0) this.joins.delete(index);
  }

  /** @returns true when the channel is in a join. */
  has(ch: channel.Key): boolean {
    return this.members.has(ch);
  }

  /** @returns the members of the index's join that wait for a catch-up read. */
  waiting(index: channel.Key): Wait[] {
    const j = this.joins.get(index);
    if (j == null) return [];
    return [...j.members.values()]
      .filter((m) => !m.joined)
      .map((m) => ({ key: m.key, wait: m.wait }));
  }

  /** @returns true when the channel's join has caught up with its stream. */
  joined(ch: channel.Key): boolean {
    const m = this.members.get(ch);
    return m != null && m.joined && m.join.indexMember.joined;
  }

  /**
   * @returns true when the channel's last landing placed no streamed sample before the
   * channel waited again, which means the landing failed to join it.
   */
  bounced(ch: channel.Key): boolean {
    const m = this.members.get(ch);
    return m != null && !m.joined && m.fresh;
  }

  /** @returns true when the join of the index holds a calculated channel. */
  calculated(index: channel.Key): boolean {
    const j = this.joins.get(index);
    if (j == null) return false;
    for (const k of j.members.keys()) if (this.store.isCalculated(k)) return true;
    return false;
  }

  /**
   * @returns the first raw block a catch-up read of the channel must fetch: the block
   * of its last joined sample, of its first held sample, or of at, whichever is first.
   */
  catchUpFrom(ch: channel.Key, at: TimeStamp): number {
    let from = rawIndex(at);
    const m = this.members.get(ch);
    if (m == null) return from;
    if (m.from != null) from = Math.min(from, m.from);
    const first = m.held.find((s) => s.length > 0);
    if (first == null) return from;
    if (m === m.join.indexMember)
      return Math.min(from, rawIndex(new TimeStamp(first.data[0])));
    if (first.timeRange.isZero) return from;
    return Math.min(from, rawIndex(first.timeRange.start));
  }

  /**
   * Marks the members of the index's join as caught up, and writes the series they held
   * in the order they arrived. A member that waited again since its wait began stays
   * waiting, since its read can miss samples the stream skipped.
   * @param waits - The waits the catch-up reads began in.
   * @param historyless - True when the members have no history to read.
   * @returns the index channels of the joins that wait.
   */
  landed(index: channel.Key, waits: Wait[], historyless = false): Set<channel.Key> {
    const j = this.joins.get(index);
    if (j == null) return this.drain();
    waits.forEach(({ key, wait }) => {
      const m = j.members.get(key);
      if (m == null || m.wait !== wait) return;
      m.joined = true;
      m.historyless = historyless;
      m.from = null;
      m.fresh = true;
    });
    const im = j.indexMember;
    if (!im.joined) return this.drain();
    this.writeIndex(j, im.take());
    j.members.forEach((m) => {
      if (m !== im && m.joined) this.writeData(m, m.take());
    });
    return this.drain();
  }

  /** Writes a streamed frame, holding the series of joined channels that wait. */
  write(frame: Map<channel.Key, Series[]>): Written {
    const allocated = new Map<channel.Key, MultiSeries>();
    frame.forEach((series, ch) => {
      const j = this.joins.get(ch);
      if (j != null) allocated.set(ch, this.writeIndex(j, series));
    });
    frame.forEach((series, ch) => {
      if (this.joins.has(ch)) return;
      const m = this.members.get(ch);
      allocated.set(
        ch,
        m == null
          ? this.store.write(ch, new MultiSeries(series))
          : this.writeData(m, series),
      );
    });
    return { allocated, waiting: this.drain() };
  }

  /**
   * Drops the cached samples of the channel over the time range, or of every channel
   * when the channel is null. A joined channel whose open block the span touches waits
   * for a new catch-up read.
   * @returns the index channels of the joins that wait.
   */
  invalidate(
    ch: channel.Key | null = null,
    tr: TimeRange = TimeRange.MAX,
  ): Set<channel.Key> {
    if (ch == null) this.joins.forEach((j) => this.waitJoin(j));
    else {
      const m = this.members.get(ch);
      const open = this.store.open(ch);
      if (m != null && open?.joined === true && open.range.overlapsWith(tr))
        this.waitMember(m);
    }
    this.store.invalidate(ch, tr);
    return this.drain();
  }

  /**
   * Closes the channel's open block. Call it when the stream stops delivering the
   * channel. A joined channel waits for a new catch-up read.
   * @returns the index channels of the joins that wait.
   */
  flush(ch: channel.Key): Set<channel.Key> {
    const m = this.members.get(ch);
    if (m != null) this.waitMember(m);
    this.store.flush(ch);
    return this.drain();
  }

  /** Drops every join. */
  close(): void {
    this.joins.clear();
    this.members.clear();
    this.waits.clear();
  }

  private drain(): Set<channel.Key> {
    const waits = new Set(this.waits);
    this.waits.clear();
    return waits;
  }

  private addMember(j: Join, ch: channel.Key): void {
    let m = j.members.get(ch);
    if (m == null) {
      m = new Member(ch, j);
      j.members.set(ch, m);
    }
    this.members.set(ch, m);
    m.refs++;
  }

  private removeMember(j: Join, ch: channel.Key): void {
    const m = j.members.get(ch);
    if (m == null || --m.refs > 0) return;
    this.store.flush(ch);
    j.members.delete(ch);
    this.members.delete(ch);
  }

  // Sends the member back to wait for a catch-up read. The index member takes the
  // whole join with it.
  private waitMember(m: Member): void {
    const j = m.join;
    if (m === j.indexMember) {
      j.restart();
      j.members.forEach((other) => this.unjoin(other));
    } else this.unjoin(m);
    this.waits.add(j.index);
  }

  private waitJoin(j: Join): void {
    this.waitMember(j.indexMember);
  }

  private unjoin(m: Member): void {
    m.joined = false;
    m.wait++;
    const open = this.store.open(m.key);
    if (open?.joined !== true) return;
    m.from = Math.min(m.from ?? open.index, open.index);
    this.store.flush(m.key);
  }

  private writeIndex(j: Join, series: Series[]): MultiSeries {
    const allocated = new MultiSeries([]);
    const im = j.indexMember;
    for (const s of series) {
      if (!im.joined) {
        im.hold(s);
        continue;
      }
      // The Core sums the alignments of a calculation's inputs, so the stream of a
      // calculated channel skips alignments where it skips no samples.
      if (!j.continues(s) && !this.calculated(j.index)) {
        this.waitJoin(j);
        im.hold(s);
        continue;
      }
      const runs = this.runs(j.index, s);
      const fresh = runs.filter((r): r is Run & { offset: number } => r.offset != null);
      if (fresh.length > 0 && this.store.retainDataType(j.index, s.dataType)) {
        this.waitJoin(j);
        im.hold(s);
        continue;
      }
      for (const { from, to, block, offset } of fresh)
        allocated.push(
          this.store.append(
            this.store.raw(j.index, block),
            piece(s, from, to, s, from, block, offset),
          ),
        );
      if (fresh.length > 0) im.fresh = false;
      j.remember({ series: s, runs });
    }
    return allocated;
  }

  // Splits an index series into the runs of each raw block, from the end of each
  // block's coverage.
  private runs(index: channel.Key, s: Series): Run[] {
    const runs: Run[] = [];
    for (let lo = 0; lo < s.length;) {
      const b = rawIndex(new TimeStamp(s.data[lo]));
      const end = tileRange({ level: RAW_LEVEL, index: b }).end.valueOf();
      const hi = search(s, end, lo);
      const block = this.store.raw(index, b);
      const fresh = Math.max(lo, search(s, block.coveredEnd.valueOf(), lo));
      if (fresh > lo) runs.push({ from: lo, to: fresh, block: b, offset: null });
      if (fresh < hi)
        runs.push({ from: fresh, to: hi, block: b, offset: block.coveredCount });
      lo = hi;
    }
    return runs;
  }

  private writeData(m: Member, series: Series[]): MultiSeries {
    const allocated = new MultiSeries([]);
    const j = m.join;
    for (const s of series) {
      if (!m.joined || !j.indexMember.joined) {
        m.hold(s);
        continue;
      }
      const at = j.locate(s.alignment);
      // The stream carried the series before its index. The catch-up read fetched its
      // span, or the gap check of the next series finds the hole.
      if (at == null && j.precedes(s)) continue;
      if (
        at == null ||
        at.pos + s.length > at.mapped.series.length ||
        !this.place(m, s, at.mapped, at.pos, allocated)
      ) {
        this.waitMember(m);
        m.hold(s);
      }
    }
    return allocated;
  }

  // Writes the samples of a data series into the blocks its index samples went into.
  // Returns false on a new data type, and on a gap: the series skips samples its blocks
  // have not taken.
  private place(
    m: Member,
    s: Series,
    { series: times, runs }: Mapped,
    pos: number,
    allocated: MultiSeries,
  ): boolean {
    const end = pos + s.length;
    const pieces: Array<{ block: RawBlock; p: Series }> = [];
    for (const run of runs) {
      if (run.to <= pos || run.from >= end || run.offset == null) continue;
      const block = this.store.raw(m.key, run.block);
      let from = Math.max(run.from, pos);
      const to = Math.min(run.to, end);
      let offset = run.offset + (from - run.from);
      const held = block.coveredCount - offset;
      if (held >= to - from) continue;
      if (held >= 0) {
        from += held;
        offset += held;
      } else if (!m.historyless || block.coveredCount !== 0) {
        m.from = Math.min(m.from ?? run.block, run.block);
        return false;
      }
      pieces.push({
        block,
        p: piece(s, from - pos, to - pos, times, from, run.block, offset),
      });
    }
    if (pieces.length === 0) return true;
    if (this.store.retainDataType(m.key, s.dataType)) return false;
    pieces.forEach(({ block, p }) => allocated.push(this.store.append(block, p)));
    m.fresh = false;
    return true;
  }
}

// Returns the samples [from, to) of the series with block alignment and the time range
// of the index samples in times at the same positions.
const piece = (
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
    alignment: alignmentOf(block, offset),
  });
};

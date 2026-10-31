// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Series, TimeRange, TimeStamp } from "@synnaxlabs/x";

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
// the block offset of the run's first sample. The samples from fresh on are new to the
// block.
interface Run {
  from: number;
  to: number;
  block: number;
  offset: number;
  fresh: number;
}

// A streamed index series and the blocks its samples went into.
interface Mapped {
  series: Series;
  runs: Run[];
}

// The positions [from, to) of an index series that fall in one raw block, and the block
// offset of the first.
interface Span {
  block: number;
  from: number;
  to: number;
  offset: number;
}

// Returns the spans of the positions [pos, end) of an index series, one per run.
const spansOf = (runs: Run[], pos: number, end: number): Span[] =>
  runs
    .filter((r) => r.to > pos && r.from < end)
    .map((r) => {
      const from = Math.max(r.from, pos);
      return {
        block: r.block,
        from,
        to: Math.min(r.to, end),
        offset: r.offset + (from - r.from),
      };
    });

// Returns true when the streamed series starts at the first sample a writer wrote.
const startsRun = (s: Series): boolean => offsetOf(s.alignment) === 0;

// A held series takes its Core alignment past every block alignment, so the x and y of
// a waiting join still pair.
const HELD_ALIGNMENT = 1n << 62n;

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

  constructor(
    readonly key: channel.Key,
    readonly join: Join,
    // wait numbers the member's wait among the waits of every member, so a landing
    // knows the wait its read began in.
    public wait: number,
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

  constructor(
    readonly index: channel.Key,
    wait: number,
  ) {
    this.indexMember = new Member(index, this, wait);
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

/**
 * Joins the streams of joined channels to their raw blocks in the store. A join holds
 * the streamed series of its index channel and of the data channels on it while it
 * catches up with a read. Then index samples get block offsets in time order, and data
 * samples take the offsets of the index samples that share their Core alignment, from
 * the same frame or an earlier one. A gap in the stream, a new data type, or a change
 * to the history of an open block sends a join back to wait. Every method that can do
 * so returns the index channels of the joins that wait.
 */
export class Joins {
  private readonly joins = new Map<channel.Key, Join>();
  private readonly members = new Map<channel.Key, Member>();
  private readonly waits = new Set<channel.Key>();
  private lastWait = 0;
  // Copies of held series at the alignments they draw at, so each read returns the
  // same objects.
  private readonly copies = new WeakMap<Series, Series>();
  private readonly pieces = new WeakMap<Series, Series[]>();

  constructor(private readonly store: Store) {}

  /**
   * Adds the channel and its index to the join of the index. Each new member waits for
   * a catch-up read.
   */
  join(ch: channel.Key, index: channel.Key): void {
    let j = this.joins.get(index);
    if (j == null) {
      j = new Join(index, ++this.lastWait);
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

  /**
   * @returns the streamed series that the channel's join holds for it while it waits
   * for a catch-up read, in alignments where x and y pair. Once the index joins, they
   * take the block offsets of their index samples.
   */
  held(ch: channel.Key): Series[] {
    const m = this.members.get(ch);
    if (m == null) return [];
    const j = m.join;
    if (!j.indexMember.joined) return m.held.map((s) => this.copy(s));
    if (m.joined) return [];
    return m.held.flatMap((s) => this.cut(j, s));
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

  // Returns true when the join of the index holds a calculated channel.
  private calculated(index: channel.Key): boolean {
    const j = this.joins.get(index);
    if (j == null) return false;
    for (const k of j.members.keys()) if (this.store.isCalculated(k)) return true;
    return false;
  }

  /**
   * @returns the first raw block a catch-up read of the channel must fetch: the block
   * of a gap its stream hit, of its first held sample, or of at, whichever is first. A
   * stream drop never moves it back, since windows read the raw blocks they need before
   * it.
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
    // The join numbers the next index sample after the newest one stored, so the store
    // must keep that block until the sample arrives.
    const newest = this.store.newestBlock(j.index);
    if (im.joined && this.store.open(j.index) == null && Number.isFinite(newest))
      this.store.join(this.store.raw(j.index, newest));
    j.members.forEach((m) => {
      if (m !== im && m.joined) this.writeData(m, m.take());
    });
    return this.drain();
  }

  /**
   * Writes the series of joined channels in a streamed frame, holding those of members
   * that wait. Other channels' series stay out of the store.
   */
  write(frame: Map<channel.Key, Series[]>): Set<channel.Key> {
    frame.forEach((series, ch) => {
      const j = this.joins.get(ch);
      if (j != null) this.writeIndex(j, series);
    });
    frame.forEach((series, ch) => {
      const m = this.members.get(ch);
      if (m != null && !this.joins.has(ch)) this.writeData(m, series);
    });
    return this.drain();
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
      m = new Member(ch, j, ++this.lastWait);
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
    m.wait = ++this.lastWait;
    if (this.store.open(m.key)?.joined === true) this.store.flush(m.key);
  }

  private copy(s: Series): Series {
    let copy = this.copies.get(s);
    if (copy != null) return copy;
    const { data, dataType, sampleOffset, timeRange } = s;
    copy = new Series({
      data,
      dataType,
      sampleOffset,
      timeRange,
      alignment: HELD_ALIGNMENT + s.alignment,
    });
    this.copies.set(s, copy);
    return copy;
  }

  // Cuts a held data series into pieces at the block offsets of the index samples that
  // share their Core alignment, or none before the join takes those samples.
  private cut(j: Join, s: Series): Series[] {
    const cached = this.pieces.get(s);
    if (cached != null) return cached;
    const at = j.locate(s.alignment);
    if (at == null || at.pos + s.length > at.mapped.series.length) {
      // The join forgot the index series of a series that precedes the ones it keeps.
      if (j.precedes(s)) this.pieces.set(s, []);
      return [];
    }
    const { series: times, runs } = at.mapped;
    const series = spansOf(runs, at.pos, at.pos + s.length).map(
      ({ block, from, to, offset }) =>
        piece(s, from - at.pos, to - at.pos, times, from, block, offset),
    );
    this.pieces.set(s, series);
    return series;
  }

  private writeIndex(j: Join, series: Series[]): void {
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
      const fresh = runs.filter((r) => r.fresh < r.to);
      if (fresh.length > 0 && this.store.retainDataType(j.index, s.dataType)) {
        this.waitJoin(j);
        im.hold(s);
        continue;
      }
      for (const { from, fresh: at, to, block, offset } of fresh)
        this.store.append(
          this.store.raw(j.index, block),
          piece(s, at, to, s, at, block, offset + at - from),
          at === 0 && startsRun(s),
        );
      if (fresh.length > 0) im.fresh = false;
      j.remember({ series: s, runs });
    }
  }

  // Splits an index series into the runs of each raw block. Samples stamped after the
  // newest one a block holds are fresh, even when a read claimed coverage past them.
  private runs(index: channel.Key, s: Series): Run[] {
    const runs: Run[] = [];
    for (let lo = 0; lo < s.length;) {
      const b = rawIndex(new TimeStamp(s.data[lo]));
      const end = tileRange({ level: RAW_LEVEL, index: b }).end.valueOf();
      const hi = search(s, end, lo);
      const block = this.store.raw(index, b);
      const newest = newestOf(block);
      const fresh = Math.max(lo, search(s, newest.end, lo));
      runs.push({
        from: lo,
        to: hi,
        block: b,
        // A run the block holds in full can sit before the block's newest samples.
        offset:
          fresh === hi
            ? offsetAt(block, BigInt(s.data[lo]))
            : newest.count - fresh + lo,
        fresh,
      });
      lo = hi;
    }
    return runs;
  }

  private writeData(m: Member, series: Series[]): void {
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
        !this.place(m, s, at.mapped, at.pos)
      ) {
        this.waitMember(m);
        m.hold(s);
      }
    }
  }

  // Writes the samples of a data series into the blocks its index samples went into.
  // Returns false on a new data type, and on a gap: the series skips samples its blocks
  // have not taken.
  private place(
    m: Member,
    s: Series,
    { series: times, runs }: Mapped,
    pos: number,
  ): boolean {
    const pieces: Array<{ block: RawBlock; p: Series; newRun: boolean }> = [];
    for (const span of spansOf(runs, pos, pos + s.length)) {
      const block = this.store.raw(m.key, span.block);
      let { from, offset } = span;
      const { to } = span;
      const held = block.coveredCount - offset;
      if (held >= to - from) continue;
      if (held >= 0) {
        from += held;
        offset += held;
      } else if (!m.historyless || block.coveredCount !== 0) {
        m.from = Math.min(m.from ?? span.block, span.block);
        return false;
      }
      pieces.push({
        block,
        p: piece(s, from - pos, to - pos, times, from, span.block, offset),
        newRun: from === pos && startsRun(s),
      });
    }
    if (pieces.length === 0) return true;
    if (this.store.retainDataType(m.key, s.dataType)) return false;
    pieces.forEach(({ block, p, newRun }) => this.store.append(block, p, newRun));
    m.fresh = false;
    return true;
  }
}

// Returns the end of the span of the newest sample of an index channel's block, and the
// block offset after it. A block that holds none gives its start and offset 0.
const newestOf = (block: RawBlock): { end: bigint; count: number } => {
  const live = block.buffer?.leadingBuffer;
  const newest = live != null && live.length > 0 ? live : block.fetched.at(-1);
  if (newest == null || newest.length === 0)
    return { end: block.range.start.valueOf(), count: 0 };
  return {
    end: BigInt(newest.data[newest.length - 1]) + 1n,
    count: offsetOf(newest.alignment) + newest.length,
  };
};

// Returns the block offset of the index sample stamped t.
const offsetAt = (block: RawBlock, t: bigint): number => {
  const live = block.buffer?.leadingBuffer;
  const series = live == null ? block.fetched : [...block.fetched, live];
  return series.reduce(
    (offset, s) =>
      s.length > 0 && BigInt(s.data[0]) <= t
        ? Math.max(offset, offsetOf(s.alignment) + search(s, t))
        : offset,
    0,
  );
};

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

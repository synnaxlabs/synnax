// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import {
  type DataType,
  type destructor,
  MultiSeries,
  type Series,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import {
  type Block,
  RAW_LEVEL,
  RawBlock,
  rawIndex,
  rawKey,
  search,
} from "@/framer/cache/block";
import { Dynamic, type DynamicProps } from "@/framer/cache/dynamic";
import { tileRange, tileSpan } from "@/framer/cache/grid";
import { hold, Join, type Mapped, Member, type Part, piece } from "@/framer/cache/join";

export interface StoreProps extends Partial<
  Pick<DynamicProps, "dynamicBufferSize" | "now">
> {
  /**
   * Byte size the store evicts down to, least recently used first. Each block counts
   * its samples and a small fixed overhead. A held block is never evicted, so the store
   * can exceed the budget.
   * @default Size.megabytes(128)
   */
  budget?: Size;
  /**
   * How long a fetched block answers reads before it is fetched again, so that writes
   * into the past show up. A held block keeps answering.
   * @default TimeSpan.minutes(10)
   */
  staleThreshold?: TimeSpan;
  /** Used for logging. */
  instrumentation?: alamos.Instrumentation;
}

const DEFAULT_BUDGET = Size.megabytes(128);
const DEFAULT_STALE_THRESHOLD = TimeSpan.minutes(10);

// ENTRY_OVERHEAD estimates the memory of a block apart from its samples, so that blocks
// with no samples count against the budget too.
const ENTRY_OVERHEAD = 256;

// The stream's buffer of one channel, and the raw block it writes into.
interface Live {
  buffer: Dynamic;
  block: RawBlock | null;
}

/**
 * Receives a span of a channel's history that changed, or of every channel's when the
 * channel is null.
 */
export type InvalidateHandler = (ch: channel.Key | null, tr: TimeRange) => void;

/** Receives the index channel of a join whose members wait for a catch-up read. */
export type WaitHandler = (index: channel.Key) => void;

/**
 * Caches blocks of channel samples under one byte budget. The least recently used block
 * leaves first. Every block of a channel holds one data type. Streamed samples go into
 * the raw block of their time, which stays open until the stream moves past it. A
 * joined channel's streamed samples get block alignment and extend the block's
 * coverage. Fetched blocks stay valid until a history change names them while the
 * history signals stream, and expire otherwise.
 */
export class Store {
  private readonly budget: number;
  private readonly staleThreshold: TimeSpan;
  private readonly dynamicBufferSize: number | TimeSpan;
  private readonly instrumentation: alamos.Instrumentation;
  private readonly now: () => TimeStamp;
  // Map order is recency order: a get moves its block to the end.
  private readonly blocks = new Map<string, Block>();
  // counted holds the bytes each block counted against the budget at its last set.
  private readonly counted = new Map<string, number>();
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private readonly live = new Map<channel.Key, Live>();
  private bytes = 0;
  // joins holds each join by its index channel, and joinOf each member's join.
  private readonly joins = new Map<channel.Key, Join>();
  private readonly joinOf = new Map<channel.Key, Join>();
  // Calculated channels are computed on read, so no history change names them.
  private readonly calculated = new Set<channel.Key>();
  private signalKey: channel.Key | null = null;
  // signalsSince is when the stream last acked the history signals, or null when it
  // does not carry them.
  private signalsSince: TimeStamp | null = null;
  private readonly invalidateHandlers = new Set<InvalidateHandler>();
  private readonly waitHandlers = new Set<WaitHandler>();

  constructor({
    budget = DEFAULT_BUDGET,
    staleThreshold = DEFAULT_STALE_THRESHOLD,
    dynamicBufferSize = tileSpan(RAW_LEVEL),
    instrumentation = alamos.NOOP,
    now = () => TimeStamp.now(),
  }: StoreProps = {}) {
    this.budget = budget.valueOf();
    this.staleThreshold = staleThreshold;
    this.dynamicBufferSize = dynamicBufferSize;
    this.instrumentation = instrumentation;
    this.now = now;
  }

  /** @returns the block cached under the key, marking it the most recently used. */
  get(key: string): Block | undefined {
    const block = this.blocks.get(key);
    if (block == null) return undefined;
    this.blocks.delete(key);
    this.blocks.set(key, block);
    return block;
  }

  /**
   * Caches the block under the key as the most recently used, replacing any block
   * there, and evicts least recently used blocks past the budget. The block it sets is
   * never evicted. Call it again after the block changes, so its new size counts
   * against the budget.
   */
  set(key: string, block: Block): void {
    const bytes = block.bytes + ENTRY_OVERHEAD;
    this.bytes += bytes - (this.counted.get(key) ?? 0);
    this.counted.set(key, bytes);
    this.blocks.delete(key);
    this.blocks.set(key, block);
    for (const [k, b] of this.blocks) {
      if (this.bytes <= this.budget || k === key) break;
      if (!b.held) this.delete(k);
      else {
        // A held block is in use, so it moves behind the key as the most recently used.
        this.blocks.delete(k);
        this.blocks.set(k, b);
      }
    }
  }

  /** Removes the block cached under the key. */
  delete(key: string): void {
    const bytes = this.counted.get(key);
    if (bytes == null) return;
    this.blocks.delete(key);
    this.counted.delete(key);
    this.bytes -= bytes;
  }

  /**
   * @returns true when the block's fetched samples are too old to answer reads. A held
   * block never expires, and neither does a block that history signals keep valid.
   */
  expired(block: Block): boolean {
    if (block.held || this.signaled(block)) return false;
    return !this.now().span(block.fetchedAt).lessThan(this.staleThreshold);
  }

  // A block fetched while the stream carried the history signals stays valid until a
  // signal names it.
  private signaled(block: Block): boolean {
    return (
      this.signalsSince != null &&
      !this.calculated.has(block.channel) &&
      !block.fetchedAt.before(this.signalsSince)
    );
  }

  /** @returns true when the channel is calculated. */
  isCalculated(ch: channel.Key): boolean {
    return this.calculated.has(ch);
  }

  /** Records whether the channel is calculated, so its blocks keep expiring. */
  setCalculated(ch: channel.Key, calculated: boolean): void {
    if (calculated) this.calculated.add(ch);
    else this.calculated.delete(ch);
  }

  /** Sets the channel that carries the history signals. */
  setSignalKey(key: channel.Key): void {
    this.signalKey = key;
  }

  /** Records the keys the stream has acked. */
  setAcked(keys: Set<channel.Key>): void {
    const flowing = this.signalKey != null && keys.has(this.signalKey);
    if (!flowing) this.signalsSince = null;
    else this.signalsSince ??= this.now();
  }

  /** Registers a handler for changed spans of history. */
  onInvalidate(handler: InvalidateHandler): destructor.Destructor {
    this.invalidateHandlers.add(handler);
    return () => this.invalidateHandlers.delete(handler);
  }

  /** Registers a handler for joins that wait for a catch-up read. */
  onWait(handler: WaitHandler): destructor.Destructor {
    this.waitHandlers.add(handler);
    return () => this.waitHandlers.delete(handler);
  }

  /**
   * Drops the cached samples of the channel over the time range, or of every channel
   * when the channel is null. A joined channel whose open block the span touches waits
   * for a new catch-up read.
   */
  invalidate(ch: channel.Key | null = null, tr: TimeRange = TimeRange.MAX): void {
    if (ch == null) this.joins.forEach((j) => this.waitJoin(j));
    else {
      const j = this.joinOf.get(ch);
      const block = this.live.get(ch)?.block;
      if (j != null && block?.joined === true && block.range.overlapsWith(tr))
        if (ch === j.index) this.waitJoin(j);
        else this.waitMember(j, j.members.get(ch)!);
    }
    const touched = [...this.blocks].filter(
      ([, b]) => (ch == null || b.channel === ch) && b.range.overlapsWith(tr),
    );
    for (const [key, block] of touched)
      if (!(block instanceof RawBlock)) this.delete(key);
      else if (block.expire()) this.set(key, block);
    this.invalidateHandlers.forEach((h) => h(ch, tr));
  }

  /** @returns the data type of the channel's cached samples. */
  dataType(ch: channel.Key): DataType | undefined {
    return this.dataTypes.get(ch);
  }

  /**
   * @returns the channel's raw block at the index, cached empty when absent. An expired
   * block comes back without its fetched samples.
   * @throws {UnexpectedError} if the block under the raw block's key is not a raw block.
   */
  raw(ch: channel.Key, index: number): RawBlock {
    const cached = this.findRaw(ch, index);
    const key = rawKey(ch, index);
    if (cached == null) {
      const block = new RawBlock(ch, index, this.now);
      this.set(key, block);
      return block;
    }
    if (this.expired(cached) && cached.expire()) this.set(key, cached);
    return cached;
  }

  /**
   * @returns the channel's raw block at the index, or undefined when none is cached.
   * @throws {UnexpectedError} if the block under the raw block's key is not a raw block.
   */
  findRaw(ch: channel.Key, index: number): RawBlock | undefined {
    const key = rawKey(ch, index);
    const cached = this.get(key);
    if (cached == null) return undefined;
    if (!(cached instanceof RawBlock))
      throw new UnexpectedError(`block ${key} is not a raw block`);
    return cached;
  }

  /**
   * Writes a streamed frame into the raw blocks of its samples' times. A series with no
   * time range goes into the block of the wall clock. A joined channel's series wait
   * while the join catches up, and take block alignment once it has.
   * @returns the buffers the write allocated for each channel of the frame. A write
   * that fits the open buffer appends in place and allocates none.
   */
  write(frame: Map<channel.Key, Series[]>): Map<channel.Key, MultiSeries> {
    const out = new Map<channel.Key, MultiSeries>();
    frame.forEach((series, ch) => {
      const j = this.joins.get(ch);
      if (j != null) out.set(ch, this.writeIndex(j, series));
    });
    frame.forEach((series, ch) => {
      if (this.joins.has(ch)) return;
      const j = this.joinOf.get(ch);
      out.set(
        ch,
        j == null ? this.writePlain(ch, series) : this.writeData(j, ch, series),
      );
    });
    return out;
  }

  /**
   * Adds the channel and its index to the join of the index. The channel waits for a
   * catch-up read before its streamed samples enter its blocks.
   */
  join(ch: channel.Key, index: channel.Key): void {
    let j = this.joins.get(index);
    if (j == null) {
      j = new Join(index);
      this.joins.set(index, j);
    }
    this.addMember(j, index);
    if (ch !== index) this.addMember(j, ch);
    this.waitHandlers.forEach((h) => h(index));
  }

  /** Removes the channel and its index from the join, undoing one call to join. */
  leave(ch: channel.Key, index: channel.Key): void {
    const j = this.joins.get(index);
    if (j == null) return;
    if (ch !== index) this.removeMember(j, ch);
    this.removeMember(j, index);
    if (j.members.size === 0) this.joins.delete(index);
  }

  /** @returns the members of the index's join that wait for a catch-up read. */
  waiting(index: channel.Key): channel.Key[] {
    return this.members(index).filter(
      (k) => !this.joinOf.get(k)!.members.get(k)!.joined,
    );
  }

  /** @returns the members of the index's join. */
  members(index: channel.Key): channel.Key[] {
    return [...(this.joins.get(index)?.members.keys() ?? [])];
  }

  /** @returns true when the channel's join has caught up with its stream. */
  joined(ch: channel.Key): boolean {
    const j = this.joinOf.get(ch);
    return j != null && j.indexMember.joined && j.members.get(ch)?.joined === true;
  }

  /**
   * @returns the first raw block a catch-up read of the member must fetch: the block of
   * its last joined sample, of its first held series, or of at, whichever is first.
   */
  catchUpFrom(ch: channel.Key, at: TimeStamp): number {
    const m = this.joinOf.get(ch)?.members.get(ch);
    let from = rawIndex(at);
    if (m?.from != null) from = Math.min(from, m.from);
    const first = m?.held[0];
    if (first != null && !first.timeRange.isZero)
      from = Math.min(from, rawIndex(first.timeRange.start));
    return from;
  }

  /**
   * Marks the members of the index's join as caught up, and writes the series they held
   * in the order they arrived.
   * @param historyless - True when the members have no history to read.
   */
  landed(index: channel.Key, keys: channel.Key[], historyless = false): void {
    const j = this.joins.get(index);
    if (j == null) return;
    keys.forEach((k) => {
      const m = j.members.get(k);
      if (m == null) return;
      m.joined = true;
      m.historyless = historyless;
      m.from = null;
    });
    const im = j.indexMember;
    if (!im.joined) return;
    this.writeIndex(j, im.held.splice(0));
    j.members.forEach((m) => {
      if (m !== im && m.joined) this.writeData(j, m.key, m.held.splice(0));
    });
  }

  private addMember(j: Join, ch: channel.Key): void {
    let m = j.members.get(ch);
    if (m == null) {
      m = new Member(ch);
      j.members.set(ch, m);
      this.joinOf.set(ch, j);
    }
    m.refs++;
  }

  private removeMember(j: Join, ch: channel.Key): void {
    const m = j.members.get(ch);
    if (m == null || --m.refs > 0) return;
    const live = this.live.get(ch);
    if (live != null) this.seal(live);
    j.members.delete(ch);
    this.joinOf.delete(ch);
  }

  // Sends every member of the join back to wait for a catch-up read.
  private waitJoin(j: Join): void {
    j.restart();
    j.members.forEach((m) => this.waitMember(j, m));
  }

  private waitMember(j: Join, m: Member): void {
    m.joined = false;
    const live = this.live.get(m.key);
    if (live?.block?.joined === true) {
      m.from = Math.min(m.from ?? live.block.index, live.block.index);
      this.seal(live);
    }
    this.waitHandlers.forEach((h) => h(j.index));
  }

  private writeIndex(j: Join, series: Series[]): MultiSeries {
    const allocated = new MultiSeries([]);
    const im = j.indexMember;
    for (const s of series) {
      if (!im.joined) {
        hold(im, s);
        continue;
      }
      // A new data type sends the join back to wait, as a gap does.
      if (this.retainDataType(j.index, s.dataType) || !j.continues(s)) {
        this.waitJoin(j);
        hold(im, s);
        continue;
      }
      const parts: Part[] = [];
      for (let lo = 0; lo < s.length;) {
        const b = rawIndex(new TimeStamp(s.data[lo]));
        const hi = search(
          s,
          tileRange({ level: RAW_LEVEL, index: b }).end.valueOf(),
          lo,
        );
        const block = this.raw(j.index, b);
        const fresh = Math.max(lo, search(s, block.coveredEnd.valueOf(), lo));
        if (fresh > lo) parts.push({ from: lo, to: fresh, block: b, offset: null });
        if (fresh < hi) {
          const offset = block.coveredCount;
          parts.push({ from: fresh, to: hi, block: b, offset });
          allocated.push(this.append(block, piece(s, fresh, hi, s, fresh, b, offset)));
        }
        lo = hi;
      }
      j.remember({ series: s, parts });
    }
    return allocated;
  }

  private writeData(j: Join, ch: channel.Key, series: Series[]): MultiSeries {
    const allocated = new MultiSeries([]);
    const m = j.members.get(ch)!;
    for (const s of series) {
      if (!m.joined || !j.indexMember.joined) {
        hold(m, s);
        continue;
      }
      if (this.retainDataType(ch, s.dataType)) {
        hold(m, s);
        continue;
      }
      // A catch-up read already holds every sample of a series that ends in covered
      // span, which can be older than the index series the join remembers.
      const last = this.raw(ch, rawIndex(s.timeRange.end.sub(TimeSpan.nanoseconds(1))));
      if (s.timeRange.end.beforeEq(last.coveredEnd)) continue;
      const at = j.locate(s.timeRange.start);
      if (at == null || at.pos + s.length > at.mapped.series.length) {
        this.waitMember(j, m);
        hold(m, s);
        continue;
      }
      if (!this.writeParts(m, s, at.mapped, at.pos, allocated)) {
        this.waitMember(j, m);
        hold(m, s);
      }
    }
    return allocated;
  }

  // Writes the samples of a data series into the blocks its index samples went into.
  // Returns false on a gap: the series skips samples its blocks have not taken.
  private writeParts(
    m: Member,
    s: Series,
    { series: times, parts }: Mapped,
    pos: number,
    allocated: MultiSeries,
  ): boolean {
    const end = pos + s.length;
    for (const part of parts) {
      if (part.to <= pos || part.from >= end || part.offset == null) continue;
      const block = this.raw(m.key, part.block);
      let from = Math.max(part.from, pos);
      const to = Math.min(part.to, end);
      let offset = part.offset + (from - part.from);
      const held = block.coveredCount - offset;
      if (held >= to - from) continue;
      if (held < 0) {
        if (!m.historyless || block.coveredCount !== 0) return false;
      } else {
        from += held;
        offset += held;
      }
      allocated.push(
        this.append(
          block,
          piece(s, from - pos, to - pos, times, from, part.block, offset),
        ),
      );
    }
    return true;
  }

  // Appends a block-aligned piece to the open buffer of its channel, opening the block
  // when the stream moves into it.
  private append(block: RawBlock, p: Series): MultiSeries {
    const live = this.liveOf(block.channel);
    if (live.block !== block || !block.joined) {
      if (live.block != null) this.seal(live, live.block.index < block.index);
      block.buffer = live.buffer;
      block.joined = true;
      live.block = block;
    }
    const res = live.buffer.write(new MultiSeries([p]));
    if (res.flushed.series.length > 0)
      block.fill(res.flushed.series, block.coveredEnd, block.coveredCount);
    block.extend(p.timeRange.end, Number(p.alignment & 0xffffffffn) + p.length);
    this.set(rawKey(block.channel, block.index), block);
    return res.allocated;
  }

  private writePlain(ch: channel.Key, series: Series[]): MultiSeries {
    const allocated = new MultiSeries([]);
    for (const s of series) {
      this.retainDataType(ch, s.dataType);
      const live = this.liveOf(ch);
      const index = rawIndex(s.timeRange.isZero ? this.now() : s.timeRange.start);
      if (live.block != null && live.block.index !== index) this.seal(live);
      let block = live.block;
      const opened = block == null;
      if (block == null) {
        block = this.raw(ch, index);
        block.buffer = live.buffer;
        live.block = block;
      }
      const res = live.buffer.write(new MultiSeries([s]));
      block.stream(res.flushed.series);
      allocated.push(res.allocated);
      if (opened || res.flushed.series.length > 0 || res.allocated.series.length > 0)
        this.set(rawKey(ch, index), block);
    }
    return allocated;
  }

  /**
   * Closes the channel's open block, stamping the end of its buffer. Call it when the
   * stream stops delivering the channel, so reads fetch the span it will not deliver. A
   * joined channel waits for a new catch-up read.
   */
  flush(ch: channel.Key): void {
    const j = this.joinOf.get(ch);
    if (j != null)
      if (ch === j.index) this.waitJoin(j);
      else this.waitMember(j, j.members.get(ch)!);
    const live = this.live.get(ch);
    if (live != null) this.seal(live);
  }

  /** @returns the buffer the stream is writing the channel's samples into. */
  leadingBuffer(ch: channel.Key): Series | null {
    return this.live.get(ch)?.buffer.leadingBuffer ?? null;
  }

  /**
   * Records the channel's data type. A channel's data type can change, and series of
   * two data types cannot be read together, so a new data type drops every block of the
   * channel and its stream's buffer.
   * @returns true if the data type changed.
   */
  retainDataType(ch: channel.Key, dataType: DataType): boolean {
    const held = this.dataTypes.get(ch);
    this.dataTypes.set(ch, dataType);
    if (held == null || held.equals(dataType)) return false;
    this.instrumentation.L.warn("channel data type changed, dropping its blocks", {
      channel: ch,
      prev: held.toString(),
      next: dataType.toString(),
    });
    const j = this.joinOf.get(ch);
    if (j != null)
      if (ch === j.index) this.waitJoin(j);
      else this.waitMember(j, j.members.get(ch)!);
    for (const [key, block] of this.blocks) if (block.channel === ch) this.delete(key);
    const live = this.live.get(ch);
    if (live == null) return true;
    live.buffer.retainDataType(dataType);
    if (live.block != null) {
      live.block.buffer = null;
      live.block.joined = false;
    }
    live.block = null;
    return true;
  }

  /** Drops every block, stream buffer, and join. */
  close(): void {
    this.live.forEach(({ buffer }) => buffer.close());
    this.live.clear();
    this.joins.clear();
    this.joinOf.clear();
    this.invalidateHandlers.clear();
    this.waitHandlers.clear();
    this.blocks.clear();
    this.counted.clear();
    this.dataTypes.clear();
    this.bytes = 0;
  }

  // Closes the open block. A joined block keeps its streamed samples as fetched ones,
  // and is complete when the stream moved past it.
  private seal(live: Live, complete = false): void {
    const { block } = live;
    if (block == null) return;
    const flushed = live.buffer.flush();
    if (flushed != null && flushed.length > 0)
      if (block.joined) block.fill([flushed], block.coveredEnd, block.coveredCount);
      else block.stream([flushed]);
    if (block.joined && complete) block.extend(block.range.end, block.coveredCount);
    block.joined = false;
    block.buffer = null;
    live.block = null;
    this.set(rawKey(block.channel, block.index), block);
  }

  private liveOf(ch: channel.Key): Live {
    let live = this.live.get(ch);
    if (live == null) {
      live = {
        buffer: new Dynamic({
          dynamicBufferSize: this.dynamicBufferSize,
          instrumentation: this.instrumentation.child(`live-${ch}`),
          now: this.now,
        }),
        block: null,
      };
      this.live.set(ch, live);
    }
    return live;
  }
}

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

import { channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import {
  type Block,
  offsetOf,
  RAW_LEVEL,
  RawBlock,
  rawKey,
} from "@/framer/cache/block";
import { Dynamic, type DynamicProps } from "@/framer/cache/dynamic";
import { tileSpan } from "@/framer/cache/grid";

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
  /**
   * The key of the Core node the client connects to, once known. History signals name
   * only the channels stored on it.
   */
  node?: () => number | undefined;
}

const DEFAULT_BUDGET = Size.megabytes(128);
const DEFAULT_STALE_THRESHOLD = TimeSpan.minutes(10);

// ENTRY_OVERHEAD estimates the memory of a block apart from its samples, so that blocks
// with no samples count against the budget too.
const ENTRY_OVERHEAD = 256;

// Moves the samples of the buffer's leading series into the block as fetched ones.
const flushInto = (block: RawBlock, buffer: Dynamic): void => {
  const flushed = buffer.flush();
  if (flushed != null && flushed.length > 0)
    block.fill([flushed], block.coveredEnd, block.coveredCount);
};

// A cached block and the bytes it counted against the budget at its last set.
interface Entry {
  block: Block;
  bytes: number;
}

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

/**
 * @returns true when a calculation writes the channel's samples. It writes a calculated
 * channel and the virtual index of one.
 */
export const calculated = (p: channel.Payload): boolean =>
  channel.isCalculated(p) || (p.virtual && p.isIndex);

/**
 * Caches blocks of channel samples under one byte budget. The least recently used block
 * leaves first. Every block of a channel holds one data type. A join appends streamed
 * samples to the raw block of their time, which stays open until the stream moves past
 * it. A sealed block stays valid until a history change names it while the history
 * signals stream. Other blocks expire.
 */
export class Store {
  private readonly budget: number;
  private readonly staleThreshold: TimeSpan;
  private readonly dynamicBufferSize: number | TimeSpan;
  private readonly instrumentation: alamos.Instrumentation;
  private readonly now: () => TimeStamp;
  private readonly node: () => number | undefined;
  // Map order is recency order: a get moves its entry to the end.
  private readonly entries = new Map<string, Entry>();
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private readonly live = new Map<channel.Key, Live>();
  // newest holds the newest raw block that held samples of each channel. A block that
  // leaves stays its newest, until the channel's data type changes.
  private readonly newest = new Map<channel.Key, number>();
  // epochs counts the changes to the series of each channel's raw blocks.
  private readonly epochs = new Map<channel.Key, number>();
  private bytes = 0;
  private readonly calculated = new Set<channel.Key>();
  // leaseholders holds the node of each stored channel.
  private readonly leaseholders = new Map<channel.Key, number>();
  private signalKey: channel.Key | null = null;
  // signalsSince is when the stream last acked the history signals, or null when it
  // does not carry them.
  private signalsSince: TimeStamp | null = null;
  private readonly invalidateHandlers = new Set<InvalidateHandler>();

  constructor({
    budget = DEFAULT_BUDGET,
    staleThreshold = DEFAULT_STALE_THRESHOLD,
    dynamicBufferSize = tileSpan(RAW_LEVEL),
    instrumentation = alamos.NOOP,
    now = () => TimeStamp.now(),
    node = () => undefined,
  }: StoreProps = {}) {
    this.budget = budget.valueOf();
    this.staleThreshold = staleThreshold;
    this.dynamicBufferSize = dynamicBufferSize;
    this.instrumentation = instrumentation;
    this.now = now;
    this.node = node;
  }

  /** @returns the block cached under the key, marking it the most recently used. */
  get(key: string): Block | undefined {
    const entry = this.entries.get(key);
    if (entry == null) return undefined;
    this.entries.delete(key);
    this.entries.set(key, entry);
    return entry.block;
  }

  /**
   * Caches the block under the key as the most recently used, replacing any block
   * there, and evicts least recently used blocks past the budget. The block it sets is
   * never evicted. Call it again after the block changes, so its new size counts
   * against the budget.
   */
  set(key: string, block: Block): void {
    if (block instanceof RawBlock) {
      this.changed(block.channel);
      if (block.coveredCount > 0)
        this.newest.set(
          block.channel,
          Math.max(block.index, this.newest.get(block.channel) ?? -Infinity),
        );
    }
    const bytes = block.bytes + ENTRY_OVERHEAD;
    this.bytes += bytes - (this.entries.get(key)?.bytes ?? 0);
    this.entries.delete(key);
    this.entries.set(key, { block, bytes });
    this.evict(key);
  }

  /**
   * Moves a subscription's pins from the blocks it served to the blocks it serves now,
   * then evicts least recently used blocks past the budget.
   */
  repin(prev: Set<Block>, next: Set<Block>): void {
    prev.forEach((b) => {
      if (!next.has(b)) b.pins--;
    });
    next.forEach((b) => {
      if (!prev.has(b)) b.pins++;
    });
    this.evict(null);
  }

  /**
   * Holds the block until the fetch settles, then evicts least recently used blocks
   * past the budget.
   * @returns the fetch, which the block keeps as pending until it settles.
   */
  hold(block: RawBlock, fetch: Promise<void>): Promise<void> {
    const pending = fetch.finally(() => {
      if (block.pending !== pending) return;
      block.pending = null;
      this.evict(null);
    });
    block.pending = pending;
    return pending;
  }

  // Evicts least recently used blocks that are not held, other than the kept one, until
  // the store fits its budget.
  private evict(kept: string | null): void {
    // A Map iterator walks past the holes that deletes leave, so the check comes first.
    if (this.bytes <= this.budget) return;
    let visits = this.entries.size;
    for (const [k, entry] of this.entries) {
      if (this.bytes <= this.budget || visits-- === 0) break;
      if (k === kept) continue;
      if (!entry.block.held) this.delete(k);
      else {
        // A held block is in use, so it moves to the back as the most recently used.
        this.entries.delete(k);
        this.entries.set(k, entry);
      }
    }
  }

  /** Removes the block cached under the key. */
  delete(key: string): void {
    const entry = this.entries.get(key);
    if (entry == null) return;
    if (entry.block instanceof RawBlock) this.changed(entry.block.channel);
    this.entries.delete(key);
    this.bytes -= entry.bytes;
  }

  /**
   * @returns true when the block's fetched samples are too old to answer reads. A held
   * block never expires, nor does a sealed block that history signals keep valid.
   */
  expired(block: Block): boolean {
    if (block.held) return false;
    const age = this.now().valueOf() - block.fetchedAt.valueOf();
    if ((age < 0n ? -age : age) < this.staleThreshold.valueOf()) return false;
    return !this.signaled(block);
  }

  // A sealed block fetched while the stream carried the history signals stays valid
  // until a signal names it. The signals name only the channels stored on the node, and
  // never an append.
  private signaled(block: Block): boolean {
    const node = this.node();
    return (
      block.sealed &&
      this.signalsSince != null &&
      node != null &&
      this.leaseholders.get(block.channel) === node &&
      !block.fetchedAt.before(this.signalsSince)
    );
  }

  /** @returns true when the channel is calculated. */
  isCalculated(ch: channel.Key): boolean {
    return this.calculated.has(ch);
  }

  /** Records the channel, which decides whether history signals name it. */
  setChannel(p: channel.Payload): void {
    if (calculated(p)) this.calculated.add(p.key);
    else this.calculated.delete(p.key);
    if (p.virtual) this.leaseholders.delete(p.key);
    else this.leaseholders.set(p.key, p.leaseholder);
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

  /**
   * Drops the cached samples of the channel over the time range, or of every channel
   * when the channel is null.
   */
  invalidate(ch: channel.Key | null = null, tr: TimeRange = TimeRange.MAX): void {
    const touched = [...this.entries]
      .map(([key, { block }]) => [key, block] as const)
      .filter(([, b]) => (ch == null || b.channel === ch) && b.range.overlapsWith(tr));
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
   * @throws {UnexpectedError} if another kind of block holds the raw block's key.
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
   * @throws {UnexpectedError} if another kind of block holds the raw block's key.
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
   * @returns the index of the newest raw block of the channel that a read or the stream
   * filled with samples, or -Infinity when none did.
   */
  newestBlock(ch: channel.Key): number {
    return this.newest.get(ch) ?? -Infinity;
  }

  /**
   * @returns a count that moves each time a raw block of the channel gains or loses a
   * series. A stream that grows the open block's buffer in place leaves it alone.
   */
  epoch(ch: channel.Key): number {
    return this.epochs.get(ch) ?? 0;
  }

  /** @returns the block the stream is writing the channel's samples into, if any. */
  open(ch: channel.Key): RawBlock | null {
    return this.live.get(ch)?.block ?? null;
  }

  /**
   * Opens the block for the stream's block-aligned samples of its channel. The block it
   * replaces closes, complete when the stream moved past it. The store keeps an open
   * block.
   */
  join(block: RawBlock): void {
    const live = this.liveOf(block.channel);
    if (live.block === block) return;
    if (live.block != null) this.seal(live, live.block.index < block.index);
    block.buffer = live.buffer;
    live.block = block;
    this.set(rawKey(block.channel, block.index), block);
  }

  /**
   * Appends a block-aligned piece to the stream's buffer of the block's channel, and
   * extends the block's coverage to the end of the piece. The block opens as by
   * {@link join}. The series of one run touch in time, as a read's series do, and the
   * series of two runs do not.
   * @param block - The block that holds the piece.
   * @param p - The piece.
   * @param newRun - True when the piece holds the first sample a writer wrote.
   */
  append(block: RawBlock, p: Series, newRun = false): void {
    const live = this.liveOf(block.channel);
    const opened = live.block !== block;
    if (opened) this.join(block);
    if (newRun) {
      flushInto(block, live.buffer);
      live.buffer.startRun();
    } else if (live.buffer.leadingBuffer == null) {
      if (block.fetched.length === 0)
        this.findRaw(block.channel, block.index - 1)?.cutNewest();
      p = block.continued(p);
    }
    const res = live.buffer.write(new MultiSeries([p]));
    const flushed = res.flushed.series.length > 0;
    if (flushed) block.fill(res.flushed.series, block.coveredEnd, block.coveredCount);
    block.extend(p.timeRange.end, offsetOf(p.alignment) + p.length);
    if (opened || flushed || res.allocated.series.length > 0)
      this.set(rawKey(block.channel, block.index), block);
  }

  /**
   * Closes the channel's open block, stamping the end of its buffer. Call it when the
   * stream stops delivering the channel, so reads fetch the span it will not deliver.
   */
  flush(ch: channel.Key): void {
    const live = this.live.get(ch);
    if (live != null) this.seal(live);
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
    for (const [key, { block }] of this.entries)
      if (block.channel === ch) this.delete(key);
    this.newest.delete(ch);
    const live = this.live.get(ch);
    if (live == null) return true;
    live.buffer.retainDataType(dataType);
    if (live.block != null) live.block.buffer = null;
    live.block = null;
    return true;
  }

  /** Drops every block and stream buffer. */
  close(): void {
    this.live.forEach(({ buffer }) => buffer.close());
    this.live.clear();
    this.invalidateHandlers.clear();
    this.entries.clear();
    this.dataTypes.clear();
    this.newest.clear();
    this.bytes = 0;
  }

  // Closes the open block. It keeps its streamed samples as fetched ones, and is
  // complete when the stream moved past it.
  private seal(live: Live, complete = false): void {
    const { block } = live;
    if (block == null) return;
    flushInto(block, live.buffer);
    if (complete) block.extend(block.range.end, block.coveredCount);
    block.buffer = null;
    live.block = null;
    this.set(rawKey(block.channel, block.index), block);
  }

  private changed(ch: channel.Key): void {
    this.epochs.set(ch, this.epoch(ch) + 1);
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

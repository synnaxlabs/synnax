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
  rawIndex,
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
 * leaves first. Every block of a channel holds one data type. Streamed samples go into
 * the raw block of their time, which stays open until the stream moves past it.
 * Fetched blocks stay valid until a history change names them while the history
 * signals stream, and expire otherwise.
 */
export class Store {
  private readonly budget: number;
  private readonly staleThreshold: TimeSpan;
  private readonly dynamicBufferSize: number | TimeSpan;
  private readonly instrumentation: alamos.Instrumentation;
  private readonly now: () => TimeStamp;
  private readonly node: () => number | undefined;
  // Map order is recency order: a get moves its block to the end.
  private readonly blocks = new Map<string, Block>();
  // counted holds the bytes each block counted against the budget at its last set.
  private readonly counted = new Map<string, number>();
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private readonly live = new Map<channel.Key, Live>();
  private bytes = 0;
  private readonly calculated = new Set<channel.Key>();
  // leaseholders holds the node of each stored channel.
  private readonly leaseholders = new Map<channel.Key, number>();
  private signalKey: channel.Key | null = null;
  // signalsSince is when the stream last acked the history signals, or null when it
  // does not carry them.
  private signalsSince: TimeStamp | null = null;
  // versions counts the invalidations of each channel, and of every channel under null.
  private readonly versions = new Map<channel.Key | null, number>();
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
  // signal names it. The signals name only the channels stored on the node.
  private signaled(block: Block): boolean {
    const node = this.node();
    return (
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
   * @returns a count that grows with each invalidation of the channel's samples. A
   * fetch that spans a change of it may hold samples from before the change.
   */
  version(ch: channel.Key): number {
    return (this.versions.get(null) ?? 0) + (this.versions.get(ch) ?? 0);
  }

  /**
   * Drops the cached samples of the channel over the time range, or of every channel
   * when the channel is null.
   */
  invalidate(ch: channel.Key | null = null, tr: TimeRange = TimeRange.MAX): void {
    this.versions.set(ch, (this.versions.get(ch) ?? 0) + 1);
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

  /** @returns the block the stream is writing the channel's samples into, if any. */
  open(ch: channel.Key): RawBlock | null {
    return this.live.get(ch)?.block ?? null;
  }

  /**
   * Appends a block-aligned piece to the stream's buffer of the block's channel, and
   * extends the block's coverage to the end of the piece. The block becomes the open
   * one, and the block it replaces closes, complete when the stream moved past it.
   */
  append(block: RawBlock, p: Series): void {
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
    block.extend(p.timeRange.end, offsetOf(p.alignment) + p.length);
    this.set(rawKey(block.channel, block.index), block);
  }

  /**
   * Writes streamed series of the channel into the raw blocks of their start times. A
   * series with no time range goes into the block of the wall clock.
   */
  write(ch: channel.Key, series: MultiSeries): void {
    for (const s of series.series) {
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
      if (opened || res.flushed.series.length > 0 || res.allocated.series.length > 0)
        this.set(rawKey(ch, index), block);
    }
  }

  /**
   * Closes the channel's open block, stamping the end of its buffer. Call it when the
   * stream stops delivering the channel, so reads fetch the span it will not deliver.
   */
  flush(ch: channel.Key): void {
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

  /** Drops every block and stream buffer. */
  close(): void {
    this.live.forEach(({ buffer }) => buffer.close());
    this.live.clear();
    this.invalidateHandlers.clear();
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

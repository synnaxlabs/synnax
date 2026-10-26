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
  MultiSeries,
  type Series,
  Size,
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
}

const DEFAULT_BUDGET = Size.megabytes(128);
const DEFAULT_STALE_THRESHOLD = TimeSpan.minutes(10);

// ENTRY_OVERHEAD estimates the memory of a block apart from its samples, so that blocks
// with no samples count against the budget too.
const ENTRY_OVERHEAD = 256;

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
 * Caches blocks of channel samples under one byte budget. The least recently used block
 * leaves first. Every block of a channel holds one data type. Streamed samples go into
 * the raw block of their time, which stays open until the stream moves past it.
 */
export class Store {
  private readonly budget: number;
  private readonly staleThreshold: TimeSpan;
  private readonly dynamicBufferSize: number | TimeSpan;
  private readonly instrumentation: alamos.Instrumentation;
  private readonly now: () => TimeStamp;
  // Map order is recency order: a get moves its entry to the end.
  private readonly entries = new Map<string, Entry>();
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private readonly live = new Map<channel.Key, Live>();
  private bytes = 0;

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
    const bytes = block.bytes + ENTRY_OVERHEAD;
    this.bytes += bytes - (this.entries.get(key)?.bytes ?? 0);
    this.entries.delete(key);
    this.entries.set(key, { block, bytes });
    this.evict(key);
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
    this.entries.delete(key);
    this.bytes -= entry.bytes;
  }

  /**
   * @returns true when the block's fetched samples are too old to answer reads. A held
   * block never expires.
   */
  expired(block: Block): boolean {
    if (block.held) return false;
    const age = this.now().valueOf() - block.fetchedAt.valueOf();
    return (age < 0n ? -age : age) >= this.staleThreshold.valueOf();
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
   * Writes streamed series of the channel into the raw blocks of their start times. A
   * series with no time range goes into the block of the wall clock.
   * @returns the buffers the write allocated. A write that fits the open buffer appends
   * in place and returns none.
   */
  write(ch: channel.Key, series: MultiSeries): MultiSeries {
    const allocated = new MultiSeries([]);
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
      allocated.push(res.allocated);
      if (opened || res.flushed.series.length > 0 || res.allocated.series.length > 0)
        this.set(rawKey(ch, index), block);
    }
    return allocated;
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
    for (const [key, { block }] of this.entries)
      if (block.channel === ch) this.delete(key);
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
    this.entries.clear();
    this.dataTypes.clear();
    this.bytes = 0;
  }

  private seal(live: Live): void {
    const { block } = live;
    if (block == null) return;
    const flushed = live.buffer.flush();
    if (flushed != null && flushed.length > 0) block.stream([flushed]);
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

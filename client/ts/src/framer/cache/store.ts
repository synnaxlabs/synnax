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
import { RAW_LEVEL, RawBlock, rawIndex, rawKey } from "@/framer/cache/block";
import { Dynamic, type DynamicProps } from "@/framer/cache/dynamic";
import { tileSpan } from "@/framer/cache/grid";

/** A cached part of one channel's samples. */
export interface Block {
  readonly channel: channel.Key;
  /** When the oldest fetch the block answers reads with landed. */
  readonly fetchedAt: TimeStamp;
  /** The bytes of the block's samples. */
  readonly bytes: number;
  /** True while the block must stay cached, as when a caller holds its series. */
  readonly held: boolean;
}

export interface StoreProps extends Partial<Pick<DynamicProps, "dynamicBufferSize">> {
  /**
   * Byte size the store evicts down to, least recently used first. Each block counts
   * its samples and a small fixed overhead. A held block is never evicted, so the store
   * can exceed the budget.
   * @default Size.megabytes(128)
   */
  budget?: Size;
  /**
   * How long a fetched block answers reads before it is fetched again, so that writes
   * into the past show up.
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
 * Caches blocks of channel samples under one byte budget. The least recently used block
 * leaves first. Every block of a channel holds one data type. Streamed samples go into
 * the raw block of their time, which stays open until the stream moves past it.
 */
export class Store {
  private readonly budget: number;
  private readonly staleThreshold: TimeSpan;
  private readonly dynamicBufferSize: number | TimeSpan;
  private readonly instrumentation: alamos.Instrumentation;
  // Map order is recency order: a get moves its block to the end.
  private readonly blocks = new Map<string, Block>();
  // counted holds the bytes each block counted against the budget at its last set.
  private readonly counted = new Map<string, number>();
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private readonly live = new Map<channel.Key, Live>();
  private bytes = 0;

  constructor({
    budget = DEFAULT_BUDGET,
    staleThreshold = DEFAULT_STALE_THRESHOLD,
    dynamicBufferSize = tileSpan(RAW_LEVEL),
    instrumentation = alamos.NOOP,
  }: StoreProps = {}) {
    this.budget = budget.valueOf();
    this.staleThreshold = staleThreshold;
    this.dynamicBufferSize = dynamicBufferSize;
    this.instrumentation = instrumentation;
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
   * there. Call it again after the block changes, so its new size counts against the
   * budget.
   */
  set(key: string, block: Block): void {
    const bytes = block.bytes + ENTRY_OVERHEAD;
    this.bytes += bytes - (this.counted.get(key) ?? 0);
    this.counted.set(key, bytes);
    this.blocks.delete(key);
    this.blocks.set(key, block);
    for (const [k, b] of this.blocks) {
      if (this.bytes <= this.budget) break;
      if (!b.held) this.delete(k);
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

  /** @returns true when the block's fetched samples are too old to answer reads. */
  expired(block: Block): boolean {
    return !TimeStamp.since(block.fetchedAt).lessThan(this.staleThreshold);
  }

  /**
   * @returns the channel's raw block at the index, cached empty when absent. An expired
   * block comes back without its fetched samples.
   */
  raw(ch: channel.Key, index: number): RawBlock {
    const key = rawKey(ch, index);
    const cached = this.get(key);
    if (cached instanceof RawBlock) {
      if (this.expired(cached) && cached.expire()) this.set(key, cached);
      return cached;
    }
    const block = new RawBlock(ch, index);
    this.set(key, block);
    return block;
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
      const index = rawIndex(s.timeRange.isZero ? TimeStamp.now() : s.timeRange.start);
      if (live.block != null && live.block.index !== index) this.seal(live);
      let block = live.block;
      const opened = block == null;
      if (block == null) {
        block = this.raw(ch, index);
        block.open = live.buffer;
        live.block = block;
      }
      const res = live.buffer.write(new MultiSeries([s]));
      block.streamed.push(...res.flushed.series);
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
   */
  retainDataType(ch: channel.Key, dataType: DataType): void {
    const held = this.dataTypes.get(ch);
    this.dataTypes.set(ch, dataType);
    if (held == null || held.equals(dataType)) return;
    this.instrumentation.L.warn("channel data type changed, dropping its blocks", {
      channel: ch,
      prev: held.toString(),
      next: dataType.toString(),
    });
    for (const [key, block] of this.blocks) if (block.channel === ch) this.delete(key);
    const live = this.live.get(ch);
    if (live == null) return;
    live.buffer.retainDataType(dataType);
    if (live.block != null) live.block.open = null;
    live.block = null;
  }

  /** Drops every block and stream buffer. */
  close(): void {
    this.live.forEach(({ buffer }) => buffer.close());
    this.live.clear();
    this.blocks.clear();
    this.counted.clear();
    this.dataTypes.clear();
    this.bytes = 0;
  }

  private seal(live: Live): void {
    const { block } = live;
    if (block == null) return;
    const flushed = live.buffer.flush();
    if (flushed != null && flushed.length > 0) block.streamed.push(flushed);
    block.open = null;
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
        }),
        block: null,
      };
      this.live.set(ch, live);
    }
    return live;
  }
}

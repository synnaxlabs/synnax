// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type DataType, Size, TimeSpan, TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";

/** A cached part of one channel's samples. */
export interface Block {
  readonly channel: channel.Key;
  /** When the oldest fetch the block answers reads with landed. */
  readonly fetchedAt: TimeStamp;
  /** The bytes of the block's samples. */
  readonly bytes: number;
  /** True while the block must stay cached, as when a caller holds one of its series. */
  readonly held: boolean;
}

export interface StoreProps {
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
}

const DEFAULT_BUDGET = Size.megabytes(128);
const DEFAULT_STALE_THRESHOLD = TimeSpan.minutes(10);

// ENTRY_OVERHEAD estimates the memory of a block apart from its samples, so that blocks
// with no samples count against the budget too.
const ENTRY_OVERHEAD = 256;

/**
 * Caches blocks of channel samples under one byte budget. The least recently used block
 * leaves first. Every block of a channel holds one data type.
 */
export class Store {
  private readonly budget: number;
  private readonly staleThreshold: TimeSpan;
  // Map order is recency order: a get moves its block to the end.
  private readonly blocks = new Map<string, Block>();
  // counted holds the bytes each block counted against the budget at its last set.
  private readonly counted = new Map<string, number>();
  private readonly dataTypes = new Map<channel.Key, DataType>();
  private bytes = 0;

  constructor({
    budget = DEFAULT_BUDGET,
    staleThreshold = DEFAULT_STALE_THRESHOLD,
  }: StoreProps = {}) {
    this.budget = budget.valueOf();
    this.staleThreshold = staleThreshold;
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
   * Caches the block under the key as the most recently used, replacing any block there.
   * Call it again after the block changes, so its new size counts against the budget.
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
   * Records the channel's data type. A channel's data type can change, and series of two
   * data types cannot be read together, so a new data type drops every block of the
   * channel.
   */
  retainDataType(ch: channel.Key, dataType: DataType): void {
    const held = this.dataTypes.get(ch);
    this.dataTypes.set(ch, dataType);
    if (held == null || held.equals(dataType)) return;
    for (const [key, block] of this.blocks) if (block.channel === ch) this.delete(key);
  }

  /** Drops every block. */
  close(): void {
    this.blocks.clear();
    this.counted.clear();
    this.dataTypes.clear();
    this.bytes = 0;
  }
}

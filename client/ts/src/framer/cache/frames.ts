// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type breaker,
  MultiSeries,
  type Series,
  sync,
  TimeRange,
  type TimeStamp,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { NotFoundError, ValidationError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { type Block, FRAMES_LEVEL } from "@/framer/cache/block";
import { indexes } from "@/framer/cache/grid";
import { type Reader } from "@/framer/cache/reader";
import { LOADING, READ_FAILED, SUCCESS } from "@/framer/cache/status";
import { type Store } from "@/framer/cache/store";
import { createBreaker, isDefinitive } from "@/framer/retry";
import { type Snapshot } from "@/framer/subscription";
import { status } from "@/status";

/**
 * Finds the keyframe of a video channel at or before a time.
 * @returns the time of the keyframe, or null when the channel has none at or before
 * the time.
 */
export interface KeyframeFinder {
  (key: channel.Key, ts: TimeStamp): Promise<TimeStamp | null>;
}

export interface FramesProps {
  /** Holds the blocks the reader fills, and reports history changes. */
  store: Store;
  /** Fills raw blocks at FRAMES_LEVEL. */
  frames: Reader;
  findKeyframe: KeyframeFinder;
  retrieveChannels: ChannelRetriever;
  /** Retry behavior for failed reads. */
  breaker?: breaker.Config;
}

/**
 * Serves every stored sample of one video channel and its index over a time range,
 * from the keyframe at or before the range start. A history change over the samples
 * reads them again.
 */
export class Frames {
  private index: channel.Key | null = null;
  // range is the span the pinned blocks cover, from the keyframe, once a read lands.
  private range: TimeRange | null = null;
  // loaded is false while a read of the current range runs.
  private loaded = false;
  private pins = new Set<Block>();
  private failure: status.Status | null = null;
  private generation = 0;
  private snapshot: Snapshot | null = null;
  private readonly notifier = new sync.Notifier();
  private readonly breaker: breaker.Breaker;
  private readonly stopInvalidations: () => void;
  private readonly statuses = {
    loading: status.create(LOADING),
    success: status.create(SUCCESS),
  };

  constructor(
    private readonly props: FramesProps,
    readonly key: channel.Key,
    private window: TimeRange,
    private readonly notify: () => void,
  ) {
    this.breaker = createBreaker(this.notifier, props.breaker);
    this.stopInvalidations = props.store.onInvalidate((ch, tr) => {
      if (ch != null && ch !== key && ch !== this.index) return;
      if (this.range?.overlapsWith(tr) !== true) return;
      void this.load();
    });
    void this.load();
  }

  /** Replaces the time range, reading the samples it lacks. An equal range does nothing. */
  update(window: TimeRange): void {
    if (window.equals(this.window)) return;
    this.window = window;
    void this.load();
  }

  /**
   * @returns the samples of the range, which may reach past it to the edges of the
   * blocks that hold it. The samples held stand in while a new range loads.
   */
  value(): Snapshot {
    if (this.snapshot != null) return this.snapshot;
    const { range, index, failure, loaded, statuses } = this;
    const st = failure ?? (loaded ? statuses.success : statuses.loading);
    if (range == null || index == null) {
      this.snapshot = { series: new MultiSeries([]), status: st };
      return this.snapshot;
    }
    this.snapshot = {
      series: new MultiSeries(this.seriesOf(this.key, range)),
      timestamps: new MultiSeries(this.seriesOf(index, range)),
      status: st,
    };
    return this.snapshot;
  }

  /** Releases every block the channel holds. */
  close(): void {
    this.generation++;
    this.notifier.notify();
    this.stopInvalidations();
    this.props.store.repin(this.pins, new Set());
    this.pins.clear();
  }

  // Finds the keyframe before the range, then fills and pins the blocks from it to the
  // range end. A failure that is definitive or outlasts the breaker stops the read.
  private async load(): Promise<void> {
    const generation = ++this.generation;
    const { store, frames, findKeyframe, retrieveChannels } = this.props;
    this.loaded = false;
    this.changed();
    while (generation === this.generation)
      try {
        if (this.index == null) {
          const [payload] = await retrieveChannels([this.key]);
          if (payload == null) throw new NotFoundError(`channel ${this.key} not found`);
          if (payload.virtual)
            throw new ValidationError(
              `virtual channel ${payload.name} has no samples to play`,
            );
          store.setChannel(payload);
          this.index = payload.index;
        }
        const { start, end } = this.window;
        const keyframe = await findKeyframe(this.key, start);
        if (generation !== this.generation) return;
        const range = new TimeRange(keyframe ?? start, end);
        await frames.fill(range, this.key);
        if (generation !== this.generation) return;
        this.pin(range);
        this.range = range;
        this.loaded = true;
        this.failure = null;
        this.breaker.reset();
        return this.changed();
      } catch (e) {
        if (generation !== this.generation) return;
        this.failure = status.fromException(e, READ_FAILED);
        this.changed();
        if (isDefinitive(e) || !(await this.breaker.wait())) return;
      }
  }

  private pin(range: TimeRange): void {
    const { store } = this.props;
    const next = new Set<Block>();
    for (const key of [this.key, this.index!])
      for (const i of indexes(range, FRAMES_LEVEL)) {
        const block = store.findRaw(key, i, FRAMES_LEVEL);
        if (block != null) next.add(block);
      }
    store.repin(this.pins, next);
    this.pins = next;
  }

  private seriesOf(key: channel.Key, range: TimeRange): Series[] {
    const { store } = this.props;
    return indexes(range, FRAMES_LEVEL).flatMap(
      (i) => store.findRaw(key, i, FRAMES_LEVEL)?.seriesIn(range) ?? [],
    );
  }

  private changed(): void {
    this.snapshot = null;
    this.notify();
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Aggregation,
  type breaker,
  deep,
  type destructor,
  errors,
  MultiSeries,
  resolveWindow,
  type Series,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { channel } from "@/channel";
import { NotFoundError, ValidationError } from "@/errors";
import { type ChannelRetriever } from "@/framer/adapter";
import { type Block, RAW_LEVEL, type RawBlock, rawIndex } from "@/framer/cache/block";
import {
  choose,
  index as tileIndex,
  indexes,
  level as tileLevel,
  pointLimit as tilePointLimit,
  type Position,
  positionKey,
  tileRange,
  tileSpan,
} from "@/framer/cache/grid";
import { type Joins } from "@/framer/cache/join";
import { type Joiner } from "@/framer/cache/joiner";
import { assemble, COMMIT_MARGIN, type Reader } from "@/framer/cache/reader";
import { LOADING, READ_FAILED, SUCCESS } from "@/framer/cache/status";
import { type Store } from "@/framer/cache/store";
import { type Demand, type MultiplexedStreamer } from "@/framer/cache/streamer";
import {
  isCut,
  tileKey,
  type TileReader,
  tileRequestKey,
  type TileSpec,
} from "@/framer/cache/tile";
import { createBreaker, isDefinitive } from "@/framer/retry";
import { type PointsLimit, type Snapshot } from "@/framer/subscription";
import { status } from "@/status";

export interface PointsProps {
  store: Store;
  joins: Joins;
  reader: Reader;
  tiles: TileReader;
  streamer: MultiplexedStreamer;
  joiner: Joiner;
  retrieveChannels: ChannelRetriever;
  now: () => TimeStamp;
  /** Retry behavior for failed reads. */
  breaker?: breaker.Config;
}

interface Held {
  spec: TileSpec;
  series: MultiSeries;
  // The cached tile, when the read cached one.
  block?: Block;
  // True once a history change named the tile. It draws until it reads again.
  stale: boolean;
}

interface Memo {
  snapshot: Snapshot;
  // The store's epoch of the channel when the snapshot was built.
  epoch: number;
  // When the clock moves the window across a grid edge.
  validUntil: bigint;
}

// The stream and join of a window that reaches past the commit point.
interface Live {
  demand: Demand;
  leave: destructor.Destructor;
  keys: channel.Key[];
}

// Returns the start of the tile of the level after the one that holds ts.
const nextEdge = (ts: TimeStamp, level: number): bigint =>
  tileRange({ level, index: tileIndex(ts, level) + 1 }).start.valueOf();

/**
 * Serves one channel under a points limit: reduced tiles across the window, and raw
 * samples at its edge when it reaches the commit point. The edge streams through a join
 * until the window's end commits, so it holds each sample once.
 */
export class Points {
  private index: channel.Key | null = null;
  // parked holds the failure that stops the channel for good.
  private parked: status.Status | null = null;
  private readFailure: status.Status | null = null;
  // rejection holds the failure of the last tile a definitive failure stopped.
  private rejection: status.Status | null = null;
  private live: Live | null = null;
  private readonly held = new Map<string, Held>();
  private readonly pending = new Set<string>();
  // retrying holds failed tiles that wait out the breaker. rejected holds tiles that a
  // definitive failure stopped.
  private readonly retrying = new Set<string>();
  private readonly rejected = new Set<string>();
  private readonly stopInvalidations: destructor.Destructor;
  // boundary is where tiles end and raw samples begin. It trails rawFrom until the
  // tiles that replace the samples before it land, so no span goes blank.
  private boundary = TimeStamp.ZERO;
  // raw holds the raw samples last served, which stand in for blocks being read again.
  private raw: Series[] = [];
  // rawRead is reading while a raw read or its backoff runs, and stopped once the read
  // outlasts the breaker.
  private rawRead: "idle" | "reading" | "stopped" = "idle";
  private loading = true;
  private pins = new Set<Block>();
  private generation = 0;
  // abort ends the wait of a raw read for the stream's ack.
  private abort = new AbortController();
  private readonly notifier = new sync.Notifier();
  private readonly breaker: breaker.Breaker;
  private readonly tileBreaker: breaker.Breaker;
  private readonly statuses = {
    loading: status.create(LOADING),
    success: status.create(SUCCESS),
  };
  // memo holds the value last served, or null after a change. It also goes out of date
  // when the store changes the series of the channel's raw blocks, or when the clock
  // reaches its validUntil.
  private memo: Memo | null = null;
  // grown is true when the stream grew the newest raw series since the last value.
  private grown = false;

  constructor(
    private readonly props: PointsProps,
    readonly key: channel.Key,
    private window: Window,
    private limit: PointsLimit,
    private readonly notify: () => void,
  ) {
    this.breaker = createBreaker(this.notifier, props.breaker);
    this.tileBreaker = createBreaker(this.notifier, props.breaker);
    this.stopInvalidations = props.store.onInvalidate((ch, tr) => {
      if (ch != null && ch !== this.key) return;
      this.held.forEach((h) => {
        if (tileRange(h.spec).overlapsWith(tr)) h.stale = true;
      });
      this.changed();
    });
    void this.resolve();
  }

  private get dynamic(): boolean {
    return this.window.variant === "dynamic";
  }

  // True while the window reaches past the commit point, where only the stream serves
  // it.
  private get streaming(): boolean {
    if (this.dynamic) return true;
    return this.home().end.after(this.props.now().sub(COMMIT_MARGIN));
  }

  private get rawOnly(): boolean {
    return this.limit.count === Infinity;
  }

  /**
   * Replaces the window and limit. Held tiles stand in until the new ones land. Tiles a
   * definitive failure stopped read again only under a new aggregation or point count.
   * A view that moves inside the tiles it draws keeps the snapshot.
   */
  update(window: Window, limit: PointsLimit): void {
    const moved = !deep.equal(window, this.window);
    if (!moved && deep.equal(limit, this.limit)) return;
    const prev = this.limit;
    const slid = !moved && deep.equal({ ...prev, view: limit.view }, limit);
    const tiles = slid ? this.tilesOf(prev) : null;
    this.window = window;
    this.limit = limit;
    if (tiles != null && tiles === this.tilesOf(limit)) return;
    this.memo = null;
    if (
      limit.count !== prev.count ||
      limit.aggregation !== prev.aggregation ||
      limit.extremesKept !== prev.extremesKept
    ) {
      this.rejected.clear();
      this.rejection = null;
    }
    if (this.index == null) return;
    this.stream(this.index);
    if (moved || this.rawOnly) this.boundary = this.rawFrom(this.home());
  }

  /**
   * @returns the samples to draw for the current window and limit. The snapshot stays
   * the same object until its samples or status change. A stream that only grows the
   * newest series keeps the snapshot's series and moves its time.
   */
  value(): Snapshot {
    const now = this.props.now();
    const { memo } = this;
    if (
      memo == null ||
      memo.epoch !== this.props.store.epoch(this.key) ||
      now.valueOf() >= memo.validUntil
    )
      return this.rebuild(now);
    const status = this.status();
    if (!this.grown && status === memo.snapshot.status) return memo.snapshot;
    this.grown = false;
    memo.snapshot = { series: memo.snapshot.series, status, time: this.time() };
    return memo.snapshot;
  }

  // Reconciles the tiles, raw samples, reads, and pins with the window as of now, and
  // caches the snapshot they make.
  private rebuild(now: TimeStamp): Snapshot {
    this.grown = false;
    const epoch = this.props.store.epoch(this.key);
    if (this.index == null) {
      const snapshot = { series: new MultiSeries([]), status: this.status() };
      this.memo = { snapshot, epoch, validUntil: TimeStamp.MAX.valueOf() };
      return snapshot;
    }
    this.stream(this.index);
    const home = this.home();
    const view = this.view(home);
    const next = this.advance(view, home);
    const extremesKept = this.limit.extremesKept === true;
    const targets = this.targets(view, home, extremesKept);
    targets.forEach((spec) => this.request(spec));
    const tiles = this.select(targets, home);
    const { raw, tail, filling } = this.rawEdge(home);
    this.prune(targets, tiles, next, home, extremesKept);
    let series = [...tiles.flatMap((t) => t.series.series), ...raw, ...tail];
    const dataType = this.props.store.dataType(this.key);
    const retyped =
      dataType != null && series.some((s) => !s.dataType.equals(dataType));
    if (retyped) {
      // Series of two data types cannot share a snapshot.
      this.restart();
      series = series.filter((s) => s.dataType.equals(dataType));
      this.raw = this.raw.filter((s) => s.dataType.equals(dataType));
    }
    this.pin(home);
    const loaded = targets.every((spec) => this.held.has(tileRequestKey(spec)));
    if (this.loading && !filling && loaded) this.loading = false;
    const snapshot = {
      series: new MultiSeries(series),
      status: this.status(),
      time: this.time(tail),
    };
    // A restart reads every tile again, so the next value rebuilds.
    this.memo = retyped
      ? null
      : {
          snapshot,
          epoch: this.props.store.epoch(this.key),
          validUntil: this.edgeAfter(now, home),
        };
    return snapshot;
  }

  // Returns when the clock next moves the window across a grid edge that changes its
  // raw blocks, its tiles, or the commit point.
  private edgeAfter(now: TimeStamp, home: TimeRange): bigint {
    const margin = COMMIT_MARGIN.valueOf();
    const committed = now.sub(COMMIT_MARGIN);
    let until = nextEdge(now, RAW_LEVEL);
    const commit = nextEdge(committed, RAW_LEVEL) + margin;
    if (commit < until) until = commit;
    if (!this.dynamic) {
      const end = home.end.valueOf() + margin;
      return home.end.after(committed) && end < until ? end : until;
    }
    const span = home.span.valueOf();
    const level = this.rawOnly ? RAW_LEVEL : tileLevel(home);
    const start = nextEdge(home.start, level) + span;
    if (start < until) until = start;
    // A raw window drops its first series once the window starts after it.
    const first = this.raw[0]?.timeRange.end.valueOf();
    if (this.rawOnly && first != null && first + span < until) until = first + span;
    return until;
  }

  // Marks the snapshot out of date and tells the consumer.
  private changed(): void {
    this.memo = null;
    this.notify();
  }

  /** Releases every tile, block, and stream the channel holds. */
  close(): void {
    this.generation++;
    this.abort.abort();
    this.notifier.notify();
    this.stopInvalidations();
    this.props.store.repin(this.pins, new Set());
    this.pins.clear();
    this.held.clear();
    this.raw = [];
    this.live?.demand.close();
    this.live?.leave();
    this.live = null;
  }

  private home(): TimeRange {
    return resolveWindow(this.window, this.props.now());
  }

  private view(home: TimeRange, limit: PointsLimit = this.limit): TimeRange {
    const { view } = limit;
    if (view == null || !view.start.before(view.end)) return home;
    return view;
  }

  // Returns the keys of the tiles a rebuild draws and advances to under the view of the
  // limit, which is all a rebuild takes from the view.
  private tilesOf(limit: PointsLimit): string {
    const home = this.home();
    const view = this.view(home, limit);
    const extremesKept = this.limit.extremesKept === true;
    return [
      ...this.targets(view, home, extremesKept),
      ...this.targets(view, home, false, this.rawFrom(home)),
    ]
      .map((spec) => tileRequestKey(spec))
      .join();
  }

  // Resolves the channel's index, then streams and joins the window's edge. A failure
  // that is definitive or outlasts the breaker stops the channel for good.
  private async resolve(): Promise<void> {
    const generation = this.generation;
    while (generation === this.generation)
      try {
        const [payload] = await this.props.retrieveChannels([this.key]);
        if (generation !== this.generation) return;
        if (payload == null) throw new NotFoundError(`channel ${this.key} not found`);
        if (payload.virtual && !channel.isCalculated(payload))
          throw new ValidationError(
            `virtual channel ${payload.name} has no samples to draw`,
          );
        const index = payload.isIndex ? payload.key : payload.index;
        if (index === 0)
          throw new ValidationError(`channel ${payload.name} has no index to draw on`);
        this.props.store.setChannel(payload);
        this.index = index;
        this.boundary = this.rawFrom(this.home());
        this.stream(index);
        this.breaker.reset();
        this.readFailure = null;
        this.changed();
        return;
      } catch (e) {
        if (generation !== this.generation) return;
        this.fail(e);
        if (isDefinitive(e) || !(await this.breaker.wait())) {
          this.parked = this.readFailure;
          return;
        }
      }
  }

  // Streams and joins the channel while the window streams, and releases both once it
  // does not.
  private stream(index: channel.Key): void {
    if (this.streaming === (this.live != null)) return;
    if (this.live != null) {
      this.live.demand.close();
      this.live.leave();
      this.live = null;
      this.abort.abort();
      this.abort = new AbortController();
      return;
    }
    const keys = [...new Set([this.key, index])];
    const demand = this.props.streamer.demand(keys, (frame) => {
      if (!frame.has(this.key)) return;
      // A waiting join holds the frame, which adds to the held samples drawn.
      if (this.props.joins.joined(this.key)) this.grown = true;
      else this.memo = null;
      this.notify();
    });
    demand.onStatusChange(() => this.notify());
    const leave = this.props.joiner.join(this.key, index, () => this.changed());
    this.live = { demand, leave, keys };
  }

  // rawFrom is where raw samples start. In a dynamic window, it sits on a tile edge of
  // the window's level, so no tile of that level or finer crosses it. In a static
  // window, it sits on the raw block edge before the commit point, and at the window's
  // end once the end commits.
  private rawFrom(home: TimeRange): TimeStamp {
    if (this.rawOnly) return home.start;
    if (this.dynamic) {
      const level = tileLevel(home);
      return tileRange({ level, index: tileIndex(home.start, level) }).start;
    }
    const committed = this.props.now().sub(COMMIT_MARGIN);
    if (!home.end.after(committed)) return home.end;
    const edge = tileRange({ level: RAW_LEVEL, index: rawIndex(committed) }).start;
    return TimeStamp.max(home.start, edge);
  }

  // Moves the boundary to at most one span of rawFrom's grid behind rawFrom, so a
  // window that never draws stops growing.
  private cutoff(home: TimeRange): void {
    const level = this.dynamic ? tileLevel(home) : RAW_LEVEL;
    const floor = this.rawOnly ? home.start : this.rawFrom(home).sub(tileSpan(level));
    this.boundary = TimeStamp.max(this.boundary, floor);
  }

  // Returns the span raw samples serve: from the boundary to the end of a static
  // window, or on for a dynamic one.
  private rawRange(home: TimeRange): TimeRange {
    return new TimeRange(this.boundary, this.dynamic ? TimeStamp.MAX : home.end);
  }

  // tilesEnd is where tiles stop for the boundary: the boundary, or the start of a raw
  // series that crosses it, so no span draws twice.
  private tilesEnd(boundary: TimeStamp): TimeStamp {
    const first = this.raw.find((s) => s.timeRange.end.after(boundary));
    return first == null ? boundary : TimeStamp.min(boundary, first.timeRange.start);
  }

  private targets(
    view: TimeRange,
    home: TimeRange,
    extremesKept: boolean,
    boundary: TimeStamp = this.boundary,
  ): TileSpec[] {
    if (this.rawOnly) return [];
    const end = this.tilesEnd(boundary);
    const aggregation: Aggregation = extremesKept ? "min_max" : this.limit.aggregation;
    // Tiles are cut to [start, end): from the start of a static window to the start of
    // raw samples.
    const start = this.dynamic ? undefined : home.start;
    const tiled = new TimeRange(
      start == null ? view.start : TimeStamp.max(view.start, start),
      TimeStamp.min(view.end, end),
    );
    if (!tiled.start.before(tiled.end)) return [];
    const level = tileLevel(view);
    const pointLimit = tilePointLimit({
      level,
      view,
      points: this.limit.count,
      drawn: this.limit.aggregation,
      aggregation,
    });
    return indexes(tiled, level).map((index) => {
      const spec: TileSpec = { key: this.key, level, index, pointLimit, aggregation };
      const tr = tileRange(spec);
      if (start != null && tr.start.before(start)) spec.start = start;
      if (tr.end.after(end)) spec.end = end;
      return spec;
    });
  }

  // Moves the boundary to rawFrom once the tiles that replace the raw samples before
  // rawFrom arrive. Returns the keys of those tiles while any is missing.
  private advance(view: TimeRange, home: TimeRange): string[] {
    const rawFrom = this.rawFrom(home);
    if (!rawFrom.after(this.boundary)) return [];
    const next = this.targets(view, home, false, rawFrom);
    const keys = next.map((spec) => tileRequestKey(spec));
    if (keys.every((k) => this.held.has(k))) {
      this.boundary = rawFrom;
      return [];
    }
    next.forEach((spec) => this.request(spec));
    return keys;
  }

  // Returns the held tile at the position: the target, then one of the aggregation,
  // then one of another.
  private find(
    p: Position,
    preferred: Map<string, string>,
    aggregation: Aggregation,
    tiles: Iterable<Held> = this.held.values(),
  ): Held | undefined {
    const want = preferred.get(positionKey(p));
    if (want != null) {
      const exact = this.held.get(want);
      if (exact != null) return exact;
    }
    let other: Held | undefined;
    for (const tile of tiles) {
      if (tile.spec.level !== p.level || tile.spec.index !== p.index) continue;
      if (tile.spec.aggregation === aggregation) return tile;
      other ??= tile;
    }
    return other;
  }

  // Chooses the held tiles that draw the targets, standing in for the missing ones.
  private select(targets: TileSpec[], home: TimeRange): Held[] {
    if (targets.length === 0) return [];
    const preferred = new Map(targets.map((t) => [positionKey(t), tileRequestKey(t)]));
    // A held tile that reaches past tilesEnd covers raw samples the window still draws.
    const raw = this.rawRange(home);
    const end = raw.start.before(raw.end) ? this.tilesEnd(this.boundary) : null;
    const drawable = [...this.held.values()].filter(
      ({ spec }) => end == null || !tileRange(spec).end.after(end),
    );
    return choose(
      targets[0].level,
      targets.map((t) => t.index),
      drawable.map((t) => t.spec),
    ).map((p) => this.find(p, preferred, targets[0].aggregation, drawable)!);
  }

  // Releases the tiles the window no longer draws. The tiles of the window keep, and so
  // do the tiles that stand in for targets until they arrive.
  private prune(
    targets: TileSpec[],
    chosen: Held[],
    next: string[],
    home: TimeRange,
    extremesKept: boolean,
  ): void {
    const keep = new Set([
      ...chosen.map((t) => tileRequestKey(t.spec)),
      ...targets.map((t) => tileRequestKey(t)),
      ...next,
      ...this.homeKeys(home, extremesKept),
    ]);
    for (const key of this.held.keys()) if (!keep.has(key)) this.held.delete(key);
  }

  // Returns the keys of the window's tiles and of the held tiles standing in for them.
  private homeKeys(home: TimeRange, extremesKept: boolean): string[] {
    const targets = this.targets(home, home, extremesKept);
    const preferred = new Map(targets.map((t) => [positionKey(t), tileRequestKey(t)]));
    return [
      ...preferred.values(),
      ...targets.flatMap((t) => {
        const found = this.find(t, preferred, t.aggregation);
        return found == null ? [] : [tileRequestKey(found.spec)];
      }),
    ];
  }

  // A failed tile waits out the breaker before a later value call requests it again. A
  // definitive rejection stops it until the channel restarts.
  private request(spec: TileSpec): void {
    const key = tileRequestKey(spec);
    if (
      this.held.get(key)?.stale === false ||
      this.pending.has(key) ||
      this.retrying.has(key) ||
      this.rejected.has(key)
    )
      return;
    const generation = this.generation;
    const { tiles, store } = this.props;
    this.pending.add(key);
    tiles
      .read(spec, store.dataType(this.key))
      .then((series) => {
        if (generation !== this.generation) return;
        this.pending.delete(key);
        const block = isCut(spec) ? undefined : store.get(tileKey(spec));
        this.held.set(key, { spec, series, block, stale: false });
        this.tileBreaker.reset();
        this.readFailure = null;
        this.changed();
      })
      .catch((e: unknown) => {
        if (generation !== this.generation) return;
        this.pending.delete(key);
        this.fail(e);
        if (isDefinitive(e)) {
          this.rejected.add(key);
          this.rejection = this.readFailure;
          return;
        }
        const waiting = this.retrying.size > 0;
        this.retrying.add(key);
        if (!waiting) void this.retryTiles(generation);
      });
  }

  private async retryTiles(generation: number): Promise<void> {
    if (!(await this.tileBreaker.wait()) || generation !== this.generation) return;
    this.retrying.clear();
    this.changed();
  }

  // Returns the last raw block of the raw range tr: the block of now, or of the newest
  // stored sample in a static window that ends later.
  private lastBlock(tr: TimeRange): number {
    const now = rawIndex(this.props.now());
    if (this.dynamic) return now;
    const end = rawIndex(tr.end.sub(TimeSpan.nanoseconds(1)));
    return Math.min(end, Math.max(now, this.props.store.newestBlock(this.key)));
  }

  // Returns the samples of the raw range, the tail of samples a waiting join holds in
  // it, and whether a block of it waits for its samples. While the window streams, the
  // join fills the blocks from the commit point on, and reads fill the rest. A block
  // being filled shows the samples it showed before, and the tail shows the stream
  // until the join catches up.
  private rawEdge(home: TimeRange): {
    raw: Series[];
    tail: Series[];
    filling: boolean;
  } {
    if (this.parked != null) return { raw: [], tail: [], filling: false };
    this.cutoff(home);
    const tr = this.rawRange(home);
    if (!tr.start.before(tr.end)) {
      this.raw = [];
      return { raw: [], tail: [], filling: false };
    }
    const { store } = this.props;
    const now = this.props.now();
    const last = this.lastBlock(tr);
    const joinFrom = this.streaming ? rawIndex(now.sub(COMMIT_MARGIN)) : Infinity;
    const blocks: RawBlock[] = [];
    for (let i = rawIndex(tr.start); i <= last; i++)
      blocks.push(store.raw(this.key, i));
    const joined = this.props.joins.joined(this.key);
    const filling = new Set(
      blocks
        .filter((b) => (b.index < joinFrom ? !b.covers(tr) : !joined))
        .map((b) => b.index),
    );
    const unread = blocks.filter((b) => b.index < joinFrom && !b.covers(tr));
    if (unread.length > 0)
      this.readRaw(new TimeRange(unread[0].range.start, unread.at(-1)!.range.end));
    const fresh = assemble(
      tr,
      blocks.filter((b) => !filling.has(b.index)),
    );
    const standIns = this.raw.filter(
      (s) =>
        filling.has(rawIndex(s.timeRange.start)) &&
        !fresh.some((f) => f.timeRange.overlapsWith(s.timeRange)),
    );
    this.raw = [...fresh, ...standIns]
      .filter((s) => s.timeRange.overlapsWith(tr))
      .sort((a, b) => TimeRange.sort(a.timeRange, b.timeRange));
    const tail =
      this.live == null
        ? []
        : this.props.joins
            .held(this.key)
            .filter((s) => s.timeRange.isZero || s.timeRange.overlapsWith(tr))
            // A writer can fill an earlier span after later samples streamed.
            .sort((a, b) => TimeRange.sort(a.timeRange, b.timeRange));
    return { raw: this.raw, tail, filling: filling.size > 0 };
  }

  // While the window streams, the read waits for the stream's ack: a read before it can
  // miss samples that a lagging writer stamps before the commit point. Once the window
  // stops streaming, its samples are committed, so the read stops waiting.
  private readRaw(tr: TimeRange): void {
    if (this.rawRead !== "idle") return;
    const generation = this.generation;
    const { reader, streamer } = this.props;
    const { live, abort } = this;
    this.rawRead = "reading";
    let after: "idle" | "stopped" = "idle";
    const read = async (): Promise<void> => {
      if (live != null)
        try {
          await streamer.waitAcked(live.keys, abort.signal);
        } catch (e) {
          if (!abort.signal.aborted) throw errors.fromUnknown(e);
        }
      if (generation === this.generation) await reader.read(tr, this.key);
    };
    read()
      .then(() => {
        if (generation !== this.generation) return;
        this.breaker.reset();
        this.readFailure = null;
      })
      .catch(async (e: unknown) => {
        if (generation !== this.generation) return;
        this.fail(e);
        if (isDefinitive(e)) return void (this.parked = this.readFailure);
        if (!(await this.breaker.wait())) after = "stopped";
      })
      .finally(() => {
        if (generation !== this.generation) return;
        this.rawRead = after;
        this.changed();
      });
  }

  // Pins the tiles and raw blocks the snapshot serves, and unpins the rest.
  private pin(home: TimeRange): void {
    const next = new Set<Block>();
    this.held.forEach(({ block }) => {
      if (block != null) next.add(block);
    });
    const raw = this.rawRange(home);
    if (raw.start.before(raw.end)) {
      const { store } = this.props;
      for (let i = rawIndex(raw.start), last = this.lastBlock(raw); i <= last; i++) {
        const b = store.findRaw(this.key, i);
        if (b != null) next.add(b);
      }
    }
    this.props.store.repin(this.pins, next);
    this.pins = next;
  }

  // Returns the end of the newest drawn sample's span.
  private time(tail: Series[] = []): TimeStamp | undefined {
    const newest = tail.findLast((s) => !s.timeRange.isZero);
    if (newest != null) return newest.timeRange.end;
    const last = this.raw.at(-1);
    if (last == null) return undefined;
    const buffer = this.props.store.open(this.key)?.buffer;
    if (last !== buffer?.leadingBuffer) return last.timeRange.end;
    return buffer.dataTimeRange?.end;
  }

  private status(): status.Status {
    const { live } = this;
    const worst = status.worst(
      this.parked,
      this.rejection,
      this.readFailure,
      this.props.joiner.status(this.key),
      ...(live?.keys.map((k) => live.demand.status(k)) ?? []),
    );
    if (worst != null && worst.variant !== "success") return worst;
    const loading = this.loading || this.pending.size > 0;
    return loading ? this.statuses.loading : this.statuses.success;
  }

  private fail(e: unknown): void {
    this.readFailure = status.fromException(e, READ_FAILED);
    this.loading = false;
    this.changed();
  }

  // Drops every tile and raw sample the channel holds, as when its data type changes.
  private restart(): void {
    this.generation++;
    this.abort.abort();
    this.abort = new AbortController();
    this.notifier.notify();
    this.held.clear();
    this.pending.clear();
    this.retrying.clear();
    this.rejected.clear();
    this.rejection = null;
    this.rawRead = "idle";
    this.loading = true;
    if (this.index != null) this.boundary = this.rawFrom(this.home());
    this.changed();
  }
}

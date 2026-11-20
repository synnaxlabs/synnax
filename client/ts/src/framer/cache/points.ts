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
  unique,
  type Window,
} from "@synnaxlabs/x";

import type { channel } from "@/channel";
import { NotFoundError, UnsupportedError, ValidationError } from "@/errors";
import type { ChannelRetriever } from "@/framer/adapter";
import { type Block, RAW_LEVEL, type RawBlock, rawIndex } from "@/framer/cache/block";
import {
  choose,
  cover,
  index as tileIndex,
  indexes,
  level as tileLevel,
  pointLimit as tilePointLimit,
  type Position,
  positionKey,
  tileRange,
  tileSpan,
} from "@/framer/cache/grid";
import type { Joins } from "@/framer/cache/join";
import type { Joiner } from "@/framer/cache/joiner";
import { COMMIT_MARGIN, type Reader } from "@/framer/cache/reader";
import { LOADING, READ_FAILED, SUCCESS } from "@/framer/cache/status";
import { calculated, type Floor, type Store } from "@/framer/cache/store";
import type { Demand, MultiplexedStreamer } from "@/framer/cache/streamer";
import {
  isCut,
  tileKey,
  type TileReader,
  tileRequestKey,
  type TileSpec,
} from "@/framer/cache/tile";
import type { Frame } from "@/framer/frame";
import { createBreaker, isDefinitive } from "@/framer/retry";
import type { PointsLimit, Snapshot } from "@/framer/subscription";
import { status } from "@/status";

export interface PointsProps {
  store: Store;
  joins: Joins;
  reader: Reader;
  tiles: TileReader;
  streamer: MultiplexedStreamer;
  joiner: Joiner;
  retrieveChannels: ChannelRetriever;
  /** Reads the newest count stored samples of each channel in the time range. */
  readLatest: (keys: channel.Key[], count: number, tr: TimeRange) => Promise<Frame>;
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

// placeless returns the limit without its view and bound, which move a subscription
// inside the tiles it draws.
const placeless = ({ view, bound, ...rest }: PointsLimit): Partial<PointsLimit> => rest;

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
  // steps holds each rawFrom past the boundary whose tiles a rebuild asked for, oldest
  // first.
  private steps: TimeStamp[] = [];
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
  // stored is the end of the newest stored sample's span for a window that streams: of
  // any sample for a dynamic window, and of one in the window for a static one. It is
  // null when there is none, and undefined until the first read lands. A read
  // supersedes the reads before it.
  private stored?: TimeStamp | null;
  private storedRead = 0;
  // storedFailure counts only while the window streams, as stored does.
  private storedFailure: status.Status | null = null;
  // tileKeys holds what tilesOf gave for the limit since the last rebuild or boundary
  // move, so a view that slides works out its tiles once per update.
  private tileKeys: string | null = null;

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
      if (this.streaming && this.index != null) void this.readStored();
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

  /**
   * Replaces the window and limit. Held tiles stand in until the new ones land. Tiles a
   * definitive failure stopped read again only under a new aggregation or point count.
   * A view that moves inside the tiles it draws keeps the snapshot.
   */
  update(window: Window, limit: PointsLimit): void {
    const moved = !deep.equal(window, this.window);
    const wasDynamic = this.dynamic;
    if (!moved && deep.equal(limit, this.limit)) return;
    const prev = this.limit;
    const slid = !moved && deep.equal(placeless(prev), placeless(limit));
    const prevKeys = slid ? (this.tileKeys ?? this.tilesOf(prev)) : null;
    this.window = window;
    this.limit = limit;
    this.tileKeys = slid ? this.tilesOf(limit) : null;
    if (prevKeys != null && prevKeys === this.tileKeys) return;
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
    if (moved) this.moveBoundary();
    // A dynamic window reads the newest sample of any time, which a new span keeps.
    if (moved && this.streaming && !(wasDynamic && this.dynamic)) {
      this.stored = undefined;
      this.storedFailure = null;
      void this.readStored();
    }
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
    const time = this.time();
    // The rebuild cuts the stream's buffer at a static window's end.
    if (!this.dynamic && time?.after(this.home().end)) return this.rebuild(now);
    this.grown = false;
    const { series, streaming } = memo.snapshot;
    memo.snapshot = { series, status, time, streaming };
    return memo.snapshot;
  }

  // Reconciles the tiles, raw samples, reads, and pins with the window as of now, and
  // caches the snapshot they make.
  private rebuild(now: TimeStamp): Snapshot {
    this.grown = false;
    this.tileKeys = null;
    const epoch = this.props.store.epoch(this.key);
    if (this.index == null) {
      const snapshot = {
        series: new MultiSeries([]),
        status: this.status(),
        streaming: this.streaming,
      };
      this.memo = { snapshot, epoch, validUntil: TimeStamp.MAX.valueOf() };
      return snapshot;
    }
    this.stream(this.index);
    const home = this.home();
    this.cutoff(home);
    const view = this.view(home);
    // advance finds where tiles end from the raw samples, so they update first.
    const fillingBlocks = this.refreshRaw(home, this.index);
    const next = this.advance(view, home);
    const extremesKept = this.limit.extremesKept === true;
    const targets = this.targets(view, home, extremesKept);
    targets.forEach((spec) => this.request(spec));
    const tiles = this.select(targets, home);
    const { raw, tail, filling } = this.rawEdge(home, fillingBlocks);
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
      streaming: this.streaming,
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
    const start = nextEdge(home.start, tileLevel(home)) + span;
    if (start < until) until = start;
    // A window drops its first raw series once the window starts after it.
    const first = this.raw[0]?.timeRange.end.valueOf();
    if (first != null && first + span < until) until = first + span;
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
    this.storedRead++;
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

  // Returns the view of the limit, or the home view when it has none. Tiles stop at
  // the commit point, so a static window that streams sizes them by its span up to
  // now.
  private view(home: TimeRange, limit: PointsLimit = this.limit): TimeRange {
    const { view } = limit;
    if (view != null && view.start.before(view.end)) return view;
    if (this.dynamic || !this.streaming) return home;
    const now = this.props.now();
    if (!home.start.before(now)) return home;
    return new TimeRange(home.start, TimeStamp.min(home.end, now));
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
        if (payload.virtual && !calculated(payload))
          throw new ValidationError(
            `virtual channel ${payload.name} has no samples to draw`,
          );
        const index = payload.isIndex ? payload.key : payload.index;
        if (index === 0)
          throw new ValidationError(`channel ${payload.name} has no index to draw on`);
        this.props.store.setChannel(payload);
        this.index = index;
        this.moveBoundary();
        this.stream(index);
        if (this.streaming) void this.readStored();
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

  // Reads the end of the newest stored sample's span, of any time for a dynamic window
  // and in the window for a static one. The read stops once a newer one starts or the
  // window stops streaming. A failure that is definitive or outlasts the breaker stops
  // the channel for good.
  private async readStored(): Promise<void> {
    const read = ++this.storedRead;
    const current = (): boolean => read === this.storedRead && this.streaming;
    const brk = createBreaker(this.notifier, this.props.breaker);
    const tr = this.dynamic ? TimeRange.MAX : this.home();
    while (current())
      try {
        const frame = await this.props.readLatest([this.key], 1, tr);
        if (!current()) return;
        const newest = frame.get(this.key).series.findLast((s) => !s.timeRange.isZero);
        this.stored = newest?.timeRange.end ?? null;
        this.storedFailure = null;
        this.changed();
        return;
      } catch (e) {
        if (!current()) return;
        // The Core stores no samples for a calculation over virtual channels.
        if (UnsupportedError.matches(e)) {
          this.stored = null;
          this.storedFailure = null;
          this.changed();
          return;
        }
        this.storedFailure = status.fromException(e, READ_FAILED);
        this.changed();
        if (isDefinitive(e) || !(await brk.wait())) {
          this.parked = this.storedFailure;
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
    const keys = unique.unique([this.key, index]);
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

  // rawFrom is where raw samples start. A dynamic window no wider than two raw blocks
  // draws only raw samples, from the edge of its level's tile before its start. Other
  // windows draw them from the raw block edge before the commit point, or from the end
  // of a static window once the end commits. Raw samples never start before the floor
  // of the store.
  private rawFrom(home: TimeRange): TimeStamp {
    const committed = this.props.now().sub(COMMIT_MARGIN);
    if (!this.dynamic && !home.end.after(committed)) return home.end;
    const floor = this.floor()?.start ?? TimeStamp.MIN;
    const level = tileLevel(home);
    if (this.dynamic && level <= RAW_LEVEL)
      return TimeStamp.max(
        tileRange({ level, index: tileIndex(home.start, level) }).start,
        floor,
      );
    const edge = tileRange({ level: RAW_LEVEL, index: rawIndex(committed) }).start;
    return TimeStamp.max(home.start, edge, floor);
  }

  private floor(): Floor | undefined {
    return this.index == null ? undefined : this.props.store.floor(this.index);
  }

  private moveBoundary(): void {
    this.boundary = this.rawFrom(this.home());
    this.steps = [];
    this.tileKeys = null;
  }

  // Moves the boundary to at most one raw block behind rawFrom, so a window that never
  // draws stops growing.
  private cutoff(home: TimeRange): void {
    const floor = this.rawFrom(home).sub(tileSpan(RAW_LEVEL));
    this.boundary = TimeStamp.max(this.boundary, floor);
  }

  // Returns the span raw samples serve: from the boundary to the end of a static
  // window, or on for a dynamic one.
  private rawRange(home: TimeRange, boundary: TimeStamp = this.boundary): TimeRange {
    return new TimeRange(boundary, this.dynamic ? TimeStamp.MAX : home.end);
  }

  // tilesEnd is where tiles stop for the boundary: the boundary, or the start of a raw
  // series that crosses it, so no span draws twice.
  private tilesEnd(boundary: TimeStamp): TimeStamp {
    // A raw series ends within its block, though an open one's range ends at MAX.
    const first = this.raw.find((s) => {
      const { end } = tileRange({
        level: RAW_LEVEL,
        index: rawIndex(s.timeRange.start),
      });
      return TimeStamp.min(s.timeRange.end, end).after(boundary);
    });
    return first == null ? boundary : TimeStamp.min(boundary, first.timeRange.start);
  }

  private targets(
    view: TimeRange,
    home: TimeRange,
    extremesKept: boolean,
    boundary: TimeStamp = this.boundary,
  ): TileSpec[] {
    const aggregation: Aggregation = extremesKept ? "min_max" : this.limit.aggregation;
    // Tiles are cut to [start, end): from the start of a static window to the start of
    // raw samples. A view may reach past a static window, as on a zoom out, so it cuts
    // tiles only where raw samples start. End moves while the window streams, so the
    // tile it cuts splits into whole finer tiles, which each read once. When end passes
    // the edge of a coarser tile, that tile reads once and replaces them.
    const viewed = this.limit.view != null;
    const start = this.dynamic || viewed ? undefined : home.start;
    const raw = this.rawRange(home, boundary);
    const end =
      viewed && !raw.start.before(raw.end) ? undefined : this.tilesEnd(boundary);
    const { bound } = this.limit;
    const tiled = new TimeRange(
      TimeStamp.max(view.start, start ?? view.start, bound?.start ?? view.start),
      TimeStamp.min(view.end, end ?? view.end, bound?.end ?? view.end),
    );
    if (!tiled.start.before(tiled.end)) return [];
    const level = tileLevel(view);
    const floor = this.streaming ? RAW_LEVEL : level;
    return indexes(tiled, level)
      .flatMap((index) =>
        end == null ? [{ level, index }] : cover({ level, index }, end, floor),
      )
      .filter((p) => start == null || tileRange(p).end.after(start))
      .map((p) => {
        const tile =
          start != null && tileRange(p).start.before(start) ? { ...p, start } : p;
        return {
          ...tile,
          key: this.key,
          pointLimit: tilePointLimit({
            tile,
            view,
            points: this.limit.count,
            drawn: this.limit.aggregation,
            aggregation,
          }),
          aggregation,
        };
      });
  }

  // Moves the boundary to the newest rawFrom it reached whose tiles arrived, so a
  // floor that moves faster than its tiles land still moves it. Returns the keys of the
  // tiles of each newer rawFrom.
  private advance(view: TimeRange, home: TimeRange): string[] {
    const rawFrom = this.rawFrom(home);
    if (rawFrom.after(this.steps.at(-1) ?? this.boundary)) this.steps.push(rawFrom);
    this.steps = this.steps.filter((step) => step.after(this.boundary));
    const keys: string[] = [];
    for (let i = this.steps.length - 1; i >= 0; i--) {
      const next = this.targets(view, home, false, this.steps[i]);
      const stepKeys = next.map((spec) => tileRequestKey(spec));
      if (stepKeys.every((k) => this.held.has(k))) {
        this.boundary = this.steps[i];
        this.steps = this.steps.slice(i + 1);
        break;
      }
      next.forEach((spec) => this.request(spec));
      keys.push(...stepKeys);
    }
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
      targets,
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
      this.limit.settling === true ||
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
  // stored sample when a writer's clock runs ahead of the Core's. A static window stops
  // at its end.
  private lastBlock(tr: TimeRange): number {
    const now = rawIndex(this.props.now());
    const last = Math.max(now, this.props.store.newestBlock(this.key));
    if (this.dynamic) return last;
    return Math.min(rawIndex(tr.end.sub(TimeSpan.nanoseconds(1))), last);
  }

  // Sets raw to the samples of the raw range, and returns the blocks of it that wait
  // for their samples. While the window streams, the join fills the blocks from the
  // commit point on, and reads fill the rest. A block being filled shows the samples it
  // showed before, and so does a span the store dropped until its tiles land.
  private refreshRaw(home: TimeRange, index: channel.Key): Set<number> {
    if (this.parked != null) return new Set();
    const tr = this.rawRange(home);
    if (!tr.start.before(tr.end)) {
      this.raw = [];
      return new Set();
    }
    const { store } = this.props;
    const now = this.props.now();
    const last = this.lastBlock(tr);
    const kept = this.floor()?.kept ?? TimeStamp.MIN;
    const stored = new TimeRange(TimeStamp.max(tr.start, kept), tr.end);
    // A waiting join's catch-up read fills the blocks from its first held sample on.
    const joinFrom = this.streaming
      ? this.props.joins.catchUpFrom(this.key, now.sub(COMMIT_MARGIN))
      : Infinity;
    const current = rawIndex(now);
    const blocks: RawBlock[] = [];
    // A block after the one of now only draws once it holds samples.
    for (let i = rawIndex(stored.start); i <= last; i++) {
      const b = i <= current ? store.raw(this.key, i) : store.findRaw(this.key, i);
      if (b != null) blocks.push(b);
    }
    const joined = this.props.joins.joined(this.key);
    const filling = new Set(
      blocks
        .filter((b) => (b.index < joinFrom ? !b.covers(stored) : !joined))
        .map((b) => b.index),
    );
    const unread = blocks.filter((b) => b.index < joinFrom && !b.covers(stored));
    if (unread.length > 0)
      this.readRaw(new TimeRange(unread[0].range.start, unread.at(-1)!.range.end));
    const fresh = blocks
      .filter((b) => !filling.has(b.index))
      .flatMap((b) => b.seriesIn(stored, store.findRaw(index, b.index)));
    const standIns = this.raw.filter(
      (s) =>
        (filling.has(rawIndex(s.timeRange.start)) || s.timeRange.start.before(kept)) &&
        !fresh.some((f) => f.timeRange.overlapsWith(s.timeRange)),
    );
    this.raw = [...fresh, ...standIns]
      .filter((s) => s.timeRange.overlapsWith(tr))
      .sort((a, b) => TimeRange.sort(a.timeRange, b.timeRange));
    return filling;
  }

  // Returns the raw samples of the raw range, the tail of samples a waiting join holds
  // in it, and whether a block of it waits for its samples. The tail shows the stream
  // until the join catches up.
  private rawEdge(
    home: TimeRange,
    fillingBlocks: Set<number>,
  ): { raw: Series[]; tail: Series[]; filling: boolean } {
    if (this.parked != null) return { raw: [], tail: [], filling: false };
    const tr = this.rawRange(home);
    if (!tr.start.before(tr.end)) {
      this.raw = [];
      return { raw: [], tail: [], filling: false };
    }
    this.raw = this.raw.filter((s) => s.timeRange.overlapsWith(tr));
    const first = rawIndex(tr.start);
    const filling = [...fillingBlocks].some((i) => i >= first);
    const tail =
      this.live == null
        ? []
        : this.props.joins
            .held(this.key, tr.end)
            .filter((s) => s.timeRange.isZero || s.timeRange.overlapsWith(tr))
            // A writer can fill an earlier span after later samples streamed.
            .sort((a, b) => TimeRange.sort(a.timeRange, b.timeRange));
    return { raw: this.raw, tail, filling };
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
      if (generation === this.generation) await reader.fill(tr, this.key);
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
      // The index's blocks find where the window's end cuts the channel's samples.
      const keys = unique.unique([this.key, this.index ?? this.key]);
      for (let i = rawIndex(raw.start), last = this.lastBlock(raw); i <= last; i++)
        keys.forEach((k) => {
          const b = store.findRaw(k, i);
          if (b != null) next.add(b);
        });
    }
    this.props.store.repin(this.pins, next);
    this.pins = next;
  }

  // Returns the end of the newest sample's span: the newest drawn, or the newest stored
  // when a window that streams has a newer one.
  private time(tail: Series[] = []): TimeStamp | undefined {
    const drawn = this.drawnTime(tail);
    const stored = this.streaming ? this.stored : null;
    if (stored == null) return drawn;
    return drawn == null ? stored : TimeStamp.max(drawn, stored);
  }

  private drawnTime(tail: Series[]): TimeStamp | undefined {
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
      this.streaming ? this.storedFailure : null,
      this.props.joiner.status(this.key),
      ...(live?.keys.map((k) => live.demand.status(k)) ?? []),
    );
    if (worst != null && worst.variant !== "success") return worst;
    const loading =
      this.loading ||
      this.pending.size > 0 ||
      (this.streaming && this.stored === undefined);
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
    if (this.index != null) this.moveBoundary();
    this.changed();
  }
}

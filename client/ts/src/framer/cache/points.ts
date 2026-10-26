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
  MultiSeries,
  resolveWindow,
  type Series,
  sync,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
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
  tileRange,
  tileSpan,
} from "@/framer/cache/grid";
import { type Joins } from "@/framer/cache/join";
import { type Joiner } from "@/framer/cache/joiner";
import { assemble, COMMIT_MARGIN, type Reader } from "@/framer/cache/reader";
import {
  isDefinitive,
  LOADING,
  newBreaker,
  READ_FAILED,
  SUCCESS,
} from "@/framer/cache/status";
import { calculated, type Store } from "@/framer/cache/store";
import { type Demand, type MultiplexedStreamer } from "@/framer/cache/streamer";
import {
  isCut,
  tileKey,
  type TileReader,
  tileRequestKey,
  type TileSpec,
} from "@/framer/cache/tile";
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
}

// The number of distinct target arguments Points remembers the targets of.
const TARGETS_KEPT = 8;

const positionKey = ({ level, index }: Position): string => `${level}/${index}`;

/**
 * Serves one channel under a points limit: reduced tiles across the window, and raw
 * samples at its edge when it reaches the commit point. The edge streams through a join
 * until the window's end commits, so it holds each sample once.
 */
export class Points {
  private index: channel.Key | null = null;
  private resolving = false;
  // parked holds the failure that stops the channel for good.
  private parked: status.Status | null = null;
  private readFailure: status.Status | null = null;
  // rejection holds the failure of the last tile a definitive failure stopped.
  private rejection: status.Status | null = null;
  private demand: Demand | null = null;
  private leave: destructor.Destructor | null = null;
  private readonly held = new Map<string, Held>();
  private readonly pending = new Set<string>();
  // retrying holds failed tiles that wait out the breaker. rejected holds tiles that a
  // definitive failure stopped.
  private readonly retrying = new Set<string>();
  private readonly rejected = new Set<string>();
  // stale holds the held tiles a history change named. They draw until they read again.
  private readonly stale = new Set<string>();
  private readonly stopInvalidations: destructor.Destructor;
  // boundary is where tiles end and raw samples begin. It trails rawFrom until the
  // tiles that replace the samples before it land, so no span goes blank.
  private boundary = TimeStamp.ZERO;
  // raw holds the raw samples last served, which stand in for blocks being read again.
  private raw: Series[] = [];
  private rawReading = false;
  private rawRetrying = false;
  // rawFilling is true while a block of the raw edge waits for its samples.
  private rawFilling = false;
  private loading = true;
  private pins = new Set<Block>();
  private generation = 0;
  private readonly notifier = new sync.Notifier();
  private readonly breaker: breaker.Breaker;
  private readonly tileBreaker: breaker.Breaker;
  private readonly statuses = {
    loading: status.create(LOADING),
    success: status.create(SUCCESS),
  };
  // targetsOf holds the targets of recent arguments, which repeat until the home, view,
  // limit, or boundary moves.
  private readonly targetsOf = new Map<string, TileSpec[]>();

  constructor(
    private readonly props: PointsProps,
    readonly key: channel.Key,
    private window: Window,
    private limit: PointsLimit,
    private readonly notify: () => void,
  ) {
    this.breaker = newBreaker(this.notifier, props.breaker);
    this.tileBreaker = newBreaker(this.notifier, props.breaker);
    this.stopInvalidations = props.store.onInvalidate((ch, tr) => {
      if (ch != null && ch !== this.key) return;
      this.held.forEach(({ spec }, key) => {
        if (tileRange(spec).overlapsWith(tr)) this.stale.add(key);
      });
      this.notify();
    });
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
   * Replaces the window and limit. Held tiles stand in until the new ones land, and
   * tiles a definitive failure stopped read again.
   */
  update(window: Window, limit: PointsLimit): void {
    const moved = !deep.equal(window, this.window);
    if (!moved && deep.equal(limit, this.limit)) return;
    this.window = window;
    this.limit = limit;
    this.targetsOf.clear();
    this.rejected.clear();
    this.rejection = null;
    if (this.index == null) return;
    this.stream(this.index);
    if (moved || this.rawOnly) this.boundary = this.rawFrom(this.home());
  }

  /** @returns the samples to draw for the current window and limit. */
  value(): Snapshot {
    if (this.index == null) {
      void this.resolve();
      return { series: new MultiSeries([]), status: this.status() };
    }
    this.stream(this.index);
    const home = this.home();
    const view = this.view(home);
    const next = this.advance(view, home);
    const extremesKept = this.limit.extremesKept === true;
    const targets = this.targets(view, home, extremesKept);
    targets.forEach((spec) => this.request(spec));
    const tiles = this.select(targets, home);
    const raw = this.rawEdge(home);
    this.prune(targets, tiles, next, home, extremesKept);
    let series = [...tiles.flatMap((t) => t.series.series), ...raw];
    const dataType = this.props.store.dataType(this.key);
    if (dataType != null && series.some((s) => !s.dataType.equals(dataType))) {
      // Series of two data types cannot share a snapshot.
      this.restart();
      series = series.filter((s) => s.dataType.equals(dataType));
      this.raw = this.raw.filter((s) => s.dataType.equals(dataType));
    }
    this.pin(home);
    if (this.loading && this.loaded(targets)) this.loading = false;
    return {
      series: new MultiSeries(series),
      status: this.status(),
      time: this.time(),
    };
  }

  /** Releases every tile, block, and stream the channel holds. */
  close(): void {
    this.generation++;
    this.notifier.notify();
    this.stopInvalidations();
    this.pins.forEach((b) => b.pins--);
    this.pins.clear();
    this.held.clear();
    this.raw = [];
    this.demand?.close();
    this.demand = null;
    this.leave?.();
    this.leave = null;
  }

  private home(): TimeRange {
    return resolveWindow(this.window, this.props.now());
  }

  private view(home: TimeRange): TimeRange {
    const { view } = this.limit;
    if (view == null || !view.start.before(view.end)) return home;
    return view;
  }

  // Resolves the channel's index, then streams and joins the window's edge.
  private async resolve(): Promise<void> {
    if (this.resolving || this.parked != null) return;
    this.resolving = true;
    const generation = this.generation;
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
      this.boundary = this.rawFrom(this.home());
      this.stream(index);
      this.breaker.reset();
      this.readFailure = null;
    } catch (e) {
      if (generation !== this.generation) return;
      this.fail(e);
      if (isDefinitive(e)) this.parked = this.readFailure;
      else if (!(await this.breaker.wait()) || generation !== this.generation) return;
    } finally {
      if (generation === this.generation) this.resolving = false;
    }
    this.notify();
  }

  // Streams and joins the channel while the window streams, and releases both once it
  // does not.
  private stream(index: channel.Key): void {
    if (this.streaming === (this.demand != null)) return;
    if (this.demand != null) {
      this.demand.close();
      this.leave?.();
      this.demand = null;
      this.leave = null;
      return;
    }
    const keys = [...new Set([this.key, index])];
    this.demand = this.props.streamer.demand(keys, (frame) => {
      if (frame.has(this.key)) this.notify();
    });
    this.demand.onStatusChange(() => this.notify());
    this.leave = this.props.joiner.join(this.key, index, () => this.notify());
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
    const tilesEnd = this.tilesEnd(boundary);
    const key = [view.start, view.end, home.start, home.end, tilesEnd, extremesKept]
      .map((v) => String(v?.valueOf()))
      .join("/");
    let specs = this.targetsOf.get(key);
    if (specs == null) {
      if (this.targetsOf.size >= TARGETS_KEPT) this.targetsOf.clear();
      specs = this.computeTargets(view, home, extremesKept, tilesEnd);
      this.targetsOf.set(key, specs);
    }
    return specs;
  }

  private computeTargets(
    view: TimeRange,
    home: TimeRange,
    extremesKept: boolean,
    end: TimeStamp,
  ): TileSpec[] {
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
    for (const key of this.held.keys())
      if (!keep.has(key)) {
        this.held.delete(key);
        this.stale.delete(key);
      }
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
      (this.held.has(key) && !this.stale.has(key)) ||
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
        this.stale.delete(key);
        const block = isCut(spec) ? undefined : store.get(tileKey(spec));
        this.held.set(key, { spec, series, block });
        this.tileBreaker.reset();
        this.readFailure = null;
        this.notify();
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
    this.notify();
  }

  // Returns the samples of the raw range. While the window streams, the join fills the
  // blocks from the commit point on, and reads fill the rest. A block being filled
  // shows the samples it showed before.
  private rawEdge(home: TimeRange): Series[] {
    this.rawFilling = false;
    if (this.parked != null) return [];
    this.cutoff(home);
    const tr = this.rawRange(home);
    if (!tr.start.before(tr.end)) return (this.raw = []);
    const { store } = this.props;
    const now = this.props.now();
    const last = rawIndex(TimeStamp.min(now, tr.end.sub(TimeSpan.nanoseconds(1))));
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
    this.rawFilling = filling.size > 0;
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
    return this.raw;
  }

  private readRaw(tr: TimeRange): void {
    if (this.rawReading || this.rawRetrying) return;
    const generation = this.generation;
    this.rawReading = true;
    this.props.reader
      .read(tr, this.key)
      .then(() => {
        if (generation !== this.generation) return;
        this.breaker.reset();
        this.readFailure = null;
      })
      .catch(async (e: unknown) => {
        if (generation !== this.generation) return;
        this.fail(e);
        if (isDefinitive(e)) return void (this.parked = this.readFailure);
        this.rawRetrying = true;
        if (!(await this.breaker.wait()) || generation !== this.generation) return;
        this.rawRetrying = false;
      })
      .finally(() => {
        if (generation !== this.generation) return;
        this.rawReading = false;
        this.notify();
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
      const end = TimeStamp.min(this.props.now(), raw.end);
      for (let i = rawIndex(raw.start); i <= rawIndex(end); i++) {
        const b = store.findRaw(this.key, i);
        if (b != null) next.add(b);
      }
    }
    this.pins.forEach((b) => {
      if (!next.has(b)) b.pins--;
    });
    next.forEach((b) => {
      if (!this.pins.has(b)) b.pins++;
    });
    this.pins = next;
  }

  private loaded(targets: TileSpec[]): boolean {
    if (this.rawFilling) return false;
    return targets.every((spec) => this.held.has(tileRequestKey(spec)));
  }

  // Returns the end of the newest raw sample's span.
  private time(): TimeStamp | undefined {
    const last = this.raw.at(-1);
    if (last == null) return undefined;
    const live = this.props.store.leadingBuffer(this.key);
    if (last !== live) return last.timeRange.end;
    return this.props.store.findRaw(this.key, rawIndex(last.timeRange.start))
      ?.coveredEnd;
  }

  private status(): status.Status {
    const candidates = [
      this.parked,
      this.rejection,
      this.readFailure,
      this.props.joiner.status(this.key),
      this.demand?.status(this.key),
      this.index == null || this.index === this.key
        ? undefined
        : this.demand?.status(this.index),
    ];
    let worst: status.Status | undefined;
    for (const s of candidates)
      if (s != null && (worst == null || status.moreSevere(s.variant, worst.variant)))
        worst = s;
    if (worst != null && worst.variant !== "success") return worst;
    const loading = this.loading || this.pending.size > 0;
    return loading ? this.statuses.loading : this.statuses.success;
  }

  private fail(e: unknown): void {
    this.readFailure = status.fromException(e, READ_FAILED);
    this.loading = false;
    this.notify();
  }

  // Drops every tile and raw sample the channel holds, as when its data type changes.
  private restart(): void {
    this.generation++;
    this.notifier.notify();
    this.held.clear();
    this.pending.clear();
    this.retrying.clear();
    this.rejected.clear();
    this.rejection = null;
    this.stale.clear();
    this.targetsOf.clear();
    this.rawReading = false;
    this.rawRetrying = false;
    this.loading = true;
    if (this.index != null) this.boundary = this.rawFrom(this.home());
    this.notify();
  }
}

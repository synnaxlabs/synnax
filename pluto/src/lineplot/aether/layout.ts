// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, scale, TimeSpan } from "@synnaxlabs/x";

import type { TickType } from "@/vis/axis/ticks";
import type { line } from "@/vis/line/aether";

/** The window of a line on an x axis, in axis values. */
export interface Trace extends line.DataBounds {
  /** The key of the line. */
  key: string;
  /** Lines of one group split the time between them on a time axis. */
  group?: string;
}

/** A span of x axis values that maps to positions with one affine scale. */
export interface Segment {
  /** The values that the segment holds. */
  bounds: bounds.Bounds;
  /** Maps a value to its decimal position across the viewport. */
  scale: scale.Scale;
}

/**
 * Layout maps the values of an x axis to decimal positions across its home view and
 * across its viewport. Its segments sit side by side in value order and share one scale
 * per unit of value. The values between two segments map to the position where the
 * segments meet, and the first and last segments extend past their bounds.
 */
export class Layout {
  /** The segments, in value order. */
  readonly segments: readonly Segment[];
  /** The values from the lower bound of the first segment to the upper of the last. */
  readonly bounds: bounds.Bounds;
  // homes maps a value of each segment to a decimal position across the home view, and
  // positions holds the home position bounds of each segment.
  private readonly homes: readonly scale.Scale[];
  private readonly reverseHomes: readonly scale.Scale[];
  private readonly positions: readonly bounds.Bounds[];
  private readonly lowers: readonly number[];
  private readonly positionLowers: readonly number[];
  private readonly viewport: scale.Scale;
  private readonly reverseViewport: scale.Scale;
  private readonly spans: ReadonlyMap<string, bounds.Bounds>;

  /**
   * @param segments - the value bounds of each segment, in value order. None may
   * overlap, and each must have a span above zero.
   * @param viewport - the viewport, in decimal space across the home view.
   * @param spans - the span of each line that has one, in axis values.
   */
  constructor(
    segments: bounds.Bounds[],
    viewport: box.Box,
    spans: ReadonlyMap<string, bounds.Bounds> = new Map(),
  ) {
    this.spans = spans;
    const total = segments.reduce((sum, b) => sum + bounds.span(b), 0);
    let before = 0;
    this.positions = segments.map((b) => {
      const lower = before / total;
      before += bounds.span(b);
      return { lower, upper: before / total };
    });
    this.homes = segments.map((b, i) => scale.Scale.scale(b).scale(this.positions[i]));
    this.reverseHomes = this.homes.map((s) => s.reverse());
    const translate = -box.root(viewport).x;
    const magnify = 1 / box.width(viewport);
    this.viewport = scale.Scale.translate(translate).magnify(magnify);
    this.reverseViewport = this.viewport.reverse();
    this.segments = segments.map((b, i) => ({
      bounds: b,
      scale: this.homes[i].translate(translate).magnify(magnify),
    }));
    this.bounds = {
      lower: segments[0].lower,
      upper: segments[segments.length - 1].upper,
    };
    this.lowers = segments.map((b) => b.lower);
    this.positionLowers = this.positions.map((b) => b.lower);
  }

  /**
   * @returns the span of values that the line of key reads and draws. A line with no
   * span spans every value.
   */
  span(key: string): bounds.Bounds {
    return this.spans.get(key) ?? bounds.INFINITE;
  }

  /** @returns the decimal position of value across the viewport. */
  pos(value: number): number {
    return this.viewport.pos(this.home(value));
  }

  /** @returns the value at a decimal position across the viewport. */
  reverse(position: number): number {
    return this.reverseHome(this.reverseViewport.pos(position));
  }

  /** @returns the decimal position of value across the home view. */
  home(value: number): number {
    const i = lastAtOrBelow(this.lowers, value);
    const { upper } = this.segments[i].bounds;
    if (i < this.segments.length - 1 && value > upper) return this.positions[i].upper;
    return this.homes[i].pos(value);
  }

  /**
   * @returns the value at a decimal position across the home view. A position where
   * two segments meet gives the lower bound of the later one.
   */
  reverseHome(position: number): number {
    const i = lastAtOrBelow(this.positionLowers, position);
    return this.reverseHomes[i].pos(position);
  }
}

// A relative line spans its window and a margin on each side: a share of the window's
// span, and at least the floor.
const MARGIN_SHARE = 0.05;
const MARGIN_FLOOR = Number(TimeSpan.SECOND.valueOf());

const margin = (b: bounds.Bounds): number =>
  Math.max(bounds.span(b) * MARGIN_SHARE, MARGIN_FLOOR);

// A span that holds no value, for a line whose time the other lines of its group draw.
const EMPTY: bounds.Bounds = { lower: 0, upper: 0 };

/**
 * @returns the span of each trace that has one, in axis values. On a relative axis each
 * trace spans its window and a margin on each side, with no margin after a window that
 * ends at the newest sample. On a time axis the traces of one group split the time, so
 * each time draws from one trace. A linear axis gives no spans.
 */
export const spans = (traces: Trace[], type: TickType): Map<string, bounds.Bounds> => {
  const finite = traces.filter((t) => bounds.isFinite(t.bounds));
  if (type === "relative") return new Map(finite.map((t) => [t.key, relativeSpan(t)]));
  const result = new Map<string, bounds.Bounds>();
  if (type === "linear") return result;
  const groups = new Map<string, Trace[]>();
  finite.forEach((t) => {
    if (t.group == null) return;
    const members = groups.get(t.group);
    if (members == null) groups.set(t.group, [t]);
    else members.push(t);
  });
  groups.forEach((members) => split(members).forEach((b, key) => result.set(key, b)));
  return result;
};

const relativeSpan = ({ bounds: b, edge }: Trace): bounds.Bounds => {
  if (edge == null) return { lower: b.lower - margin(b), upper: b.upper + margin(b) };
  // A growing window keeps its start, so the floor keeps its span from moving.
  const lower = edge === "growing" ? MARGIN_FLOOR : margin(b);
  return { lower: b.lower - lower, upper: Infinity };
};

// split gives each trace the time of its window that no trace before it takes. Fixed
// windows go first, in order of start, and a rolling window takes the newest part that
// is left. The time between two windows goes half to each, and the time before the
// first window and after the last goes to the outer traces.
const split = (traces: Trace[]): Map<string, bounds.Bounds> => {
  const ordered = [...traces].sort(
    (a, b) =>
      Number(a.edge === "rolling") - Number(b.edge === "rolling") ||
      a.bounds.lower - b.bounds.lower,
  );
  const result = new Map<string, bounds.Bounds>();
  const owned: Array<[string, bounds.Bounds]> = [];
  const taken: bounds.Bounds[] = [];
  ordered.forEach(({ key, bounds: b }) => {
    const piece = newestLeft(b, taken);
    if (piece == null) {
      result.set(key, EMPTY);
      return;
    }
    owned.push([key, piece]);
    taken.push(piece);
  });
  owned.sort(([, a], [, b]) => a.lower - b.lower);
  owned.forEach(([key, piece], i) => {
    const before = owned[i - 1]?.[1];
    const after = owned[i + 1]?.[1];
    result.set(key, {
      lower: before == null ? -Infinity : (before.upper + piece.lower) / 2,
      upper: after == null ? Infinity : (piece.upper + after.lower) / 2,
    });
  });
  return result;
};

// newestLeft returns the newest part of b that no bound of taken holds, or null when
// they hold all of it.
const newestLeft = (b: bounds.Bounds, taken: bounds.Bounds[]): bounds.Bounds | null => {
  let left = [b];
  taken.forEach((t) => {
    left = left.flatMap((l) =>
      [
        { lower: l.lower, upper: Math.min(l.upper, t.lower) },
        { lower: Math.max(l.lower, t.upper), upper: l.upper },
      ].filter((p) => p.lower < p.upper),
    );
  });
  return left.at(-1) ?? null;
};

// lastAtOrBelow returns the index of the last of the ascending keys that is at or below
// target, or 0 when every key is above it.
const lastAtOrBelow = (keys: readonly number[], target: number): number => {
  let [lo, hi] = [0, keys.length - 1];
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (keys[mid] <= target) lo = mid;
    else hi = mid - 1;
  }
  return lo;
};

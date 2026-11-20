// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, scale } from "@synnaxlabs/x";

/** A span of x axis values that maps to positions with one affine scale. */
export interface Segment {
  /** The values that the segment holds. */
  bounds: bounds.Bounds;
  /** Maps a value to its decimal position across the viewport. */
  scale: scale.Scale;
}

/**
 * Layout maps the values of an x axis to decimal positions across its home view and
 * across its viewport. Its segments sit side by side in value order and share one
 * scale per unit of value. The values between two segments map to the position where
 * the segments meet, and the first and last segments extend past their bounds.
 */
export class Layout {
  /** The segments, in value order. */
  readonly segments: readonly Segment[];
  // homes maps a value of each segment to a decimal position across the home view, and
  // positions holds the home position bounds of each segment.
  private readonly homes: readonly scale.Scale[];
  private readonly reverseHomes: readonly scale.Scale[];
  private readonly positions: readonly bounds.Bounds[];
  private readonly viewport: scale.Scale;

  /**
   * @param segments - the value bounds of each segment, in value order. None may
   * overlap, and each must have a span above zero.
   * @param viewport - the viewport, in decimal space across the home view.
   */
  constructor(segments: bounds.Bounds[], viewport: box.Box) {
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
    this.segments = segments.map((b, i) => ({
      bounds: b,
      scale: this.homes[i].translate(translate).magnify(magnify),
    }));
  }

  /** The values from the lower bound of the first segment to the upper of the last. */
  get bounds(): bounds.Bounds {
    return {
      lower: this.segments[0].bounds.lower,
      upper: this.segments[this.segments.length - 1].bounds.upper,
    };
  }

  /** @returns the decimal position of value across the viewport. */
  pos(value: number): number {
    return this.viewport.pos(this.home(value));
  }

  /** @returns the value at a decimal position across the viewport. */
  reverse(position: number): number {
    return this.reverseHome(this.viewport.reverse().pos(position));
  }

  /** @returns the decimal position of value across the home view. */
  home(value: number): number {
    const i = lastAtOrBelow(
      this.segments.length,
      (j) => this.segments[j].bounds.lower,
      value,
    );
    const { upper } = this.segments[i].bounds;
    if (i < this.segments.length - 1 && value > upper) return this.positions[i].upper;
    return this.homes[i].pos(value);
  }

  /**
   * @returns the value at a decimal position across the home view. A position where
   * two segments meet gives the lower bound of the later one.
   */
  reverseHome(position: number): number {
    const i = lastAtOrBelow(
      this.positions.length,
      (j) => this.positions[j].lower,
      position,
    );
    return this.reverseHomes[i].pos(position);
  }
}

// lastAtOrBelow returns the last index in [0, length) whose ascending key is at or
// below target, or 0 when every key is above it.
const lastAtOrBelow = (
  length: number,
  key: (i: number) => number,
  target: number,
): number => {
  let [lo, hi] = [0, length - 1];
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (key(mid) <= target) lo = mid;
    else hi = mid - 1;
  }
  return lo;
};

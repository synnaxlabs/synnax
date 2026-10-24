// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import REGIONS from "@/components/console/tour/regions.json";
import { type Step, STEPS } from "@/components/console/tour/steps";

/**
 * The CDN id under which each still is `<ID>/<still>`. It holds a hash of the stills,
 * so new stills never share an address with a cached old one.
 */
export const ID = `console/ui-overview/tour/${REGIONS.version}`;

/** The still the tour shows while no step is active. */
export const BASE = "mosaic";

/** The size of the Console window in every still, in CSS pixels. */
export const SIZE = { width: REGIONS.width, height: REGIONS.height };

interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

const STILL_REGIONS: Record<string, Record<string, Rect>> = REGIONS.stills;

/** The names of the stills. */
export const STILLS = Object.keys(STILL_REGIONS);

/** A box on a still, in percent of the still's width and height. */
export interface Box {
  left: number;
  top: number;
  width: number;
  height: number;
}

const regionsOf = (still: string): Record<string, Rect> => {
  const regions = STILL_REGIONS[still];
  if (regions == null) throw new Error(`the Console tour has no still "${still}"`);
  return regions;
};

const toBox = (rects: Rect[]): Box => {
  const left = Math.min(...rects.map((r) => r.x));
  const top = Math.min(...rects.map((r) => r.y));
  const right = Math.max(...rects.map((r) => r.x + r.width));
  const bottom = Math.max(...rects.map((r) => r.y + r.height));
  return {
    left: (left / SIZE.width) * 100,
    top: (top / SIZE.height) * 100,
    width: ((right - left) / SIZE.width) * 100,
    height: ((bottom - top) / SIZE.height) * 100,
  };
};

/**
 * Returns one box around the named regions of a still.
 * @throws {Error} if the still or one of the regions does not exist.
 */
export const box = (still: string, names: string[]): Box => {
  const regions = regionsOf(still);
  return toBox(
    names.map((name) => {
      const rect = regions[name];
      if (rect == null)
        throw new Error(`still "${still}" of the Console tour has no region "${name}"`);
      return rect;
    }),
  );
};

/** A region of a still that a reader can select. */
export interface Hotspot {
  box: Box;
  /** The step the region opens. A region with no step blocks the ones under it. */
  step?: Step;
}

/** Returns the hotspots of a still, largest first, so smaller ones sit on top. */
export const hotspots = (still: string): Hotspot[] =>
  Object.entries(regionsOf(still))
    .map(([name, rect]) => ({
      box: toBox([rect]),
      step: STEPS.find((s) => s.key === name),
    }))
    .sort((a, b) => b.box.width * b.box.height - a.box.width * a.box.height);

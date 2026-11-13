// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { theme } from "@synnaxlabs/lyra/theme";
import { type bounds, box, color, location, xy } from "@synnaxlabs/x";

import { legible, withLightness } from "@/lineplot/range/aether/legible";
import { type Draw2D } from "@/vis/draw2d";

/** A range's bar in the annotation strip, in canvas pixels. */
export interface Bar {
  range: ranger.Range;
  color: color.Color;
  /** color with its lightness clamped to read on the theme. */
  display: color.Color;
  /** The range's true start and end, past the plot edges. */
  start: number;
  end: number;
  /** The drawn bounds, clamped to the plot. */
  x0: number;
  x1: number;
  lane: number;
  /** True when the range holds another range, so a fill would tint its children. */
  enclosing: boolean;
}

const MIN_WIDTH = 4;
const LANE_HEIGHT = 20;
const THIN_LANE_HEIGHT = 4;
const LANE_GAP = 2;
const PADDING = 4;
const NAMED_LANES = 3;
const THIN_LANES = 4;
const LABEL_PADDING = 6;
const MIN_LABEL_WIDTH = 16;
const LABEL_FADE = 13;
const DIMMED_ALPHA = 0.5;
const EDGE_ALPHA = 0.45;
const HOVERED_EDGE_ALPHA = 0.9;
const HOVERED_FILL_SCALE = 1.6;
const TOOLTIP_NAME_WIDTH = 240;
const SWATCH_SIZE = 8;
const SWATCH_INDENT = 14;

interface Shades {
  fill: number;
  bar: number;
  hoveredBar: number;
  stroke: number;
  label: number;
}

const DARK: Shades = {
  fill: 0.08,
  bar: 0.17,
  hoveredBar: 0.28,
  stroke: 0.6,
  label: 0.9,
};
const LIGHT: Shades = {
  fill: 0.06,
  bar: 0.12,
  hoveredBar: 0.21,
  stroke: 0.7,
  label: 0.34,
};

const shades = (t: theme.Theme): Shades =>
  color.isDark(t.colors.gray.l0) ? DARK : LIGHT;

/** A thin lane has no name, so it reads only on hover. */
const isThin = (lane: number): boolean => lane >= NAMED_LANES;

const laneOffset = (lane: number): number =>
  Math.min(lane, NAMED_LANES) * (LANE_HEIGHT + LANE_GAP) +
  Math.max(0, lane - NAMED_LANES) * (THIN_LANE_HEIGHT + LANE_GAP);

/** @returns the height of a strip with the given number of lanes. */
export const stripHeight = (lanes: number): number =>
  lanes === 0 ? 0 : 2 * PADDING + laneOffset(lanes) - LANE_GAP;

const widen = (lo: number, hi: number): [number, number] => {
  if (hi - lo >= MIN_WIDTH) return [lo, hi];
  const mid = (lo + hi) / 2;
  return [mid - MIN_WIDTH / 2, mid + MIN_WIDTH / 2];
};

const span = (b: Bar): number => b.end - b.start;

export interface Layout {
  bars: Bar[];
  lanes: number;
}

/**
 * Places each range in the first lane that is free at its start and below every range
 * that holds it, so a parent sits above its children. Lanes pack in time, so a zoom
 * never moves a range to another lane. Ranges past the last thin lane share it. Ranges
 * with no color get no bar.
 * @param position - Maps a timestamp in nanoseconds to a canvas x position.
 */
export const layout = (
  ranges: ranger.Range[],
  region: box.Box,
  position: (value: number) => number,
  t: theme.Theme,
): Layout => {
  const bars = ranges.flatMap((range): Bar[] => {
    const parsed = color.colorZ.safeParse(range.color);
    if (!parsed.success) return [];
    const c = parsed.data;
    const start = position(Number(range.timeRange.start.valueOf()));
    const end = position(Number(range.timeRange.end.valueOf()));
    const [x0, x1] = widen(
      Math.max(start, box.left(region)),
      Math.min(end, box.right(region)),
    );
    return [
      {
        range,
        color: c,
        display: legible(c, t),
        start,
        end,
        x0,
        x1,
        lane: 0,
        enclosing: false,
      },
    ];
  });
  bars.sort((a, b) => a.start - b.start || span(b) - span(a));
  // The time bounds of the last range in each lane.
  const last: bounds.Bounds[] = [];
  bars.forEach((bar, i) => {
    const lower = Number(bar.range.timeRange.start.valueOf());
    const upper = Number(bar.range.timeRange.end.valueOf());
    const below = last.findLastIndex((l) => l.lower <= lower && l.upper >= upper) + 1;
    let lane = last.findIndex((l, j) => j >= below && l.upper <= lower);
    if (lane === -1) lane = last.length;
    last[lane] = { lower, upper };
    bar.lane = Math.min(lane, NAMED_LANES + THIN_LANES - 1);
    for (let j = i + 1; j < bars.length && bars[j].start <= bar.end; j++)
      if (bars[j].end <= bar.end && span(bars[j]) < span(bar)) {
        bar.enclosing = true;
        break;
      }
  });
  return { bars, lanes: Math.min(last.length, NAMED_LANES + THIN_LANES) };
};

const barBox = (bar: Bar, region: box.Box, lanes: number): box.Box => {
  const y = box.top(region) - stripHeight(lanes) + PADDING + laneOffset(bar.lane);
  const height = isThin(bar.lane) ? THIN_LANE_HEIGHT : LANE_HEIGHT;
  return box.construct({ x: bar.x0, y }, { x: bar.x1, y: y + height });
};

/** @returns the narrowest bar under the cursor, or null. */
export const hitTest = (
  bars: Bar[],
  region: box.Box,
  lanes: number,
  cursor: xy.XY,
): Bar | null => {
  let hit: Bar | null = null;
  for (const bar of bars) {
    const b = barBox(bar, region, lanes);
    const target = box.construct(
      { x: box.left(b) - 2, y: box.top(b) - LANE_GAP / 2 },
      { x: box.right(b) + 2, y: box.bottom(b) + LANE_GAP / 2 },
    );
    if (
      box.contains(target, cursor) &&
      (hit == null || bar.x1 - bar.x0 < hit.x1 - hit.x0)
    )
      hit = bar;
  }
  return hit;
};

/**
 * Draws each range's edges and fill in the plot. A range that holds another range has
 * no fill. A hovered range is the only one drawn, and it always fills.
 */
export const drawFills = (
  draw: Draw2D,
  bars: Bar[],
  region: box.Box,
  hovered: Bar | null,
): void => {
  const { fill } = shades(draw.theme);
  const top = box.top(region);
  const bottom = box.bottom(region);
  const shown = hovered == null ? bars : [hovered];
  shown.forEach((bar) => {
    if (bar !== hovered && bar.enclosing) return;
    draw.container({
      region: box.construct({ x: bar.x0, y: top }, { x: bar.x1, y: bottom }),
      backgroundColor: color.setOpacity(
        bar.display,
        bar === hovered ? fill * HOVERED_FILL_SCALE : fill,
      ),
      bordered: false,
      rounded: false,
    });
  });
  shown.forEach((bar) => {
    const stroke = color.setOpacity(
      bar.display,
      bar === hovered ? HOVERED_EDGE_ALPHA : EDGE_ALPHA,
    );
    [bar.start, bar.end].forEach((x) => {
      if (x < box.left(region) || x > box.right(region)) return;
      const crisp = Math.round(x) + 0.5;
      draw.line({
        stroke,
        lineWidth: 1,
        lineDash: 0,
        start: { x: crisp, y: top },
        end: { x: crisp, y: bottom },
      });
    });
  });
};

/** Draws the strip's bars above the plot. A hovered bar dims the others. */
export const drawBars = (
  draw: Draw2D,
  bars: Bar[],
  region: box.Box,
  lanes: number,
  hovered: Bar | null,
): void => {
  const t = draw.theme;
  const ctx = draw.canvas;
  const s = shades(t);
  const radius = Math.round(t.sizes.border.radius.tiny * t.sizes.base);
  // Widest first, so a narrow bar that shares a thin lane with a wide one stays on top.
  const order = [...bars].sort((a, b) => b.x1 - b.x0 - (a.x1 - a.x0));
  order.forEach((bar) => {
    const b = barBox(bar, region, lanes);
    ctx.globalAlpha = hovered != null && hovered !== bar ? DIMMED_ALPHA : 1;
    if (isThin(bar.lane)) {
      ctx.fillStyle = color.hex(bar.display);
      ctx.fillRect(box.left(b), box.top(b), box.width(b), box.height(b));
      return;
    }
    // A side cut by the plot edge gets square corners.
    const left = bar.start < box.left(region) ? 0 : radius;
    const right = bar.end > box.right(region) ? 0 : radius;
    const radii = [left, right, right, left];
    ctx.beginPath();
    ctx.roundRect(box.left(b), box.top(b), box.width(b), box.height(b), radii);
    const tint = hovered === bar ? s.hoveredBar : s.bar;
    ctx.fillStyle = color.hex(color.over(bar.color, t.colors.gray.l0, tint));
    ctx.fill();
    ctx.beginPath();
    ctx.roundRect(
      box.left(b) + 0.5,
      box.top(b) + 0.5,
      box.width(b) - 1,
      box.height(b) - 1,
      radii,
    );
    ctx.strokeStyle = color.hex(color.setOpacity(bar.display, s.stroke));
    ctx.lineWidth = 1;
    ctx.setLineDash([]);
    ctx.stroke();
    const labelWidth = box.width(b) - 2 * LABEL_PADDING;
    if (labelWidth < MIN_LABEL_WIDTH) return;
    draw.text({
      text: bar.range.name,
      position: { x: box.left(b) + LABEL_PADDING, y: box.center(b).y },
      level: "small",
      align: "center",
      color: withLightness(bar.display, s.label),
      maxWidth: labelWidth,
      fade: LABEL_FADE,
    });
  });
  ctx.globalAlpha = 1;
};

const DAY = 86_400_000_000_000n;

/** Draws the hovered range's name, start, end, and duration next to the cursor. */
export const drawTooltip = (
  draw: Draw2D,
  bar: Bar,
  cursor: xy.XY,
  region: box.Box,
): void => {
  const t = draw.theme;
  const ctx = draw.canvas;
  const { name, timeRange } = bar.range;
  const format = timeRange.span.valueOf() > DAY ? "dateTime" : "preciseTime";
  const start = timeRange.start.toString(format, "local");
  const end = timeRange.end.toString(format, "local");
  const details = [`${start} to ${end}`, timeRange.span.toString()];
  ctx.font = theme.fontString(t, { level: "small", weight: 400, code: true });
  const detailWidth = Math.max(...details.map((d) => ctx.measureText(d).width));
  ctx.font = theme.fontString(t, { level: "small", weight: 500 });
  const nameWidth = ctx.measureText(name).width + SWATCH_INDENT;
  const width = Math.ceil(
    Math.max(detailWidth, Math.min(nameWidth, TOOLTIP_NAME_WIDTH)),
  );
  const names = draw.wrap({
    text: name,
    width: width - SWATCH_INDENT,
    level: "small",
    weight: 500,
  });
  const lines = [...names, ...details];
  const root = { ...location.BOTTOM_LEFT };
  if (cursor.x > box.left(region) + box.width(region) * 0.6) root.x = "right";
  draw.list({
    root,
    offset: xy.construct(12),
    length: lines.length,
    padding: xy.construct(6),
    itemHeight: 14,
    spacing: 3,
    width,
    position: cursor,
    draw: (i, b) => {
      const isName = i < names.length;
      const middle = box.center(b).y;
      if (i === 0)
        draw.container({
          region: box.construct(
            { x: box.left(b), y: middle - SWATCH_SIZE / 2 },
            SWATCH_SIZE,
            SWATCH_SIZE,
          ),
          backgroundColor: bar.color,
          bordered: false,
          borderRadius: 2,
        });
      draw.text({
        text: lines[i],
        position: { x: box.left(b) + (isName ? SWATCH_INDENT : 0), y: middle },
        level: "small",
        align: "center",
        weight: isName ? 500 : 400,
        shade: isName ? 11 : 9,
        code: !isName,
      });
    },
  });
};

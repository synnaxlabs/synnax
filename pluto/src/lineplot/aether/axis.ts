// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme as baseTheme } from "@synnaxlabs/lyra/theme";
import {
  bounds,
  type box,
  direction,
  type scale,
  scheduler,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { collapse } from "@/lineplot/aether/layout";
import { theming } from "@/theming/aether";
import { axis } from "@/vis/axis";
import type { TickSegment, TickType } from "@/vis/axis/ticks";
import { Draw2D } from "@/vis/draw2d";
import { grid } from "@/vis/grid";
import type { line } from "@/vis/line/aether";
import { render } from "@/vis/render";

export const baseAxisStateZ = axis.axisStateZ
  .extend({
    axisKey: z.string().optional(),
    bounds: bounds.boundsZ().optional(),
    manualBounds: z
      .object({
        lower: z.boolean().default(false),
        upper: z.boolean().default(false),
      })
      .prefault({}),
    autoBoundPadding: z.number().optional(),
    autoBoundUpdateInterval: TimeSpan.z.default(TimeSpan.seconds(10)),
    size: z.number().default(0),
    label: z.string().default(""),
    labelSize: z.number().default(0),
  })
  .partial({ color: true, font: true, gridColor: true });

const AXIS_SIZE_UPDATE_UPPER_THRESHOLD = 2; // px;
const AXIS_SIZE_UPDATE_LOWER_THRESHOLD = 7; // px;

export const withinSizeThreshold = (prev: number, next: number): boolean =>
  bounds.contains(
    {
      lower: prev - AXIS_SIZE_UPDATE_LOWER_THRESHOLD,
      upper: prev + AXIS_SIZE_UPDATE_UPPER_THRESHOLD,
    },
    next,
  );

const EMPTY_LINEAR_BOUNDS = bounds.DECIMAL;

// Computed per call so an empty time axis tracks the present instead of app start.
const emptyBounds = (type: TickType): bounds.Bounds => {
  if (type === "linear") return EMPTY_LINEAR_BOUNDS;
  if (type === "relative")
    return { lower: 0, upper: Number(TimeSpan.MINUTE.valueOf()) };
  const now = TimeStamp.now();
  return {
    lower: Number(now.valueOf()),
    upper: Number(now.add(TimeSpan.HOUR).valueOf()),
  };
};

export interface AxisRenderProps {
  grid: grid.Grid;
  plot: box.Box;
  viewport: box.Box;
  container: box.Box;
  canvases: render.CanvasVariant[];
  hold: boolean;
  /** Set when the gaps between the windows of a time axis collapse into breaks. */
  gapsCollapsed: boolean;
}

/** @returns the newest sample of the bounds with an edge, or undefined if none. */
export const newestSample = (b: line.DataBounds[]): number | undefined => {
  const edges = b
    .filter((d) => d.edge != null && bounds.isFinite(d.bounds))
    .map((d) => d.bounds.upper);
  return edges.length === 0 ? undefined : Math.max(...edges);
};

/**
 * @returns b with each finite bound that has an edge moved to end at the newest sample
 * of them all, so a line whose samples stopped does not stretch the axis.
 */
export const align = <B extends line.DataBounds>(b: B[]): B[] => {
  const newest = newestSample(b);
  if (newest == null) return b;
  return b.map((d) => {
    const { lower, upper } = d.bounds;
    if (!bounds.isFinite(d.bounds) || d.edge == null) return d;
    if (d.edge === "rolling")
      return { ...d, bounds: { lower: lower + newest - upper, upper: newest } };
    return { ...d, bounds: { lower, upper: newest } };
  });
};

const autoBounds = (
  b: line.DataBounds[],
  padding: number = 0.1,
  type: TickType,
): bounds.Bounds => {
  const m = bounds.union(
    align(b)
      .filter((d) => bounds.isFinite(d.bounds))
      .map((d) => d.bounds),
  );
  if (!bounds.isFinite(m)) return emptyBounds(type);
  const { lower, upper } = m;
  if (upper === lower) return { lower: lower - 1, upper: upper + 1 };
  const _padding = (upper - lower) * padding;
  return { lower: lower - _padding, upper: upper + _padding };
};

interface InternalState {
  render: render.Context;
  draw: Draw2D;
  base: axis.Axis;
  // A hold keeps a snapshot of the segments, so the plot draws in the same place while
  // the data changes.
  segmentSnapshot?: bounds.Bounds[];
  updateBounds?: (bounds: bounds.Bounds) => void;
}

const DEFAULT_X_BOUND_PADDING = 0.01;
const DEFAULT_Y_BOUND_PADDING = 0.1;

export abstract class BaseAxis<
  S extends typeof baseAxisStateZ,
  C extends aether.Component = aether.Component,
> extends aether.Composite<S, InternalState, C> {
  override afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const { location, autoBoundUpdateInterval } = this.state;
    i.render = render.Context.use(ctx);
    // A held view in one tick type means nothing in another.
    if (this.state.type !== this.prevState.type) i.segmentSnapshot = undefined;
    const theme = theming.use(ctx);
    i.draw = new Draw2D(i.render.upper2d, theme);
    this.state.autoBoundPadding ??=
      direction.swap(location) === "x"
        ? DEFAULT_X_BOUND_PADDING
        : DEFAULT_Y_BOUND_PADDING;
    i.base = axis.newCanvas(location, i.render, {
      color: theme.colors.gray.l10,
      font: baseTheme.fontString(theme, { level: "small", code: true }),
      gridColor: theme.colors.gray.l1,
      ...this.state,
    });
    render.request(ctx, "layout");
    i.updateBounds ??= scheduler.throttle(
      (b) => this.setState((p) => ({ ...p, bounds: b })),
      autoBoundUpdateInterval,
    );
  }

  override afterDelete(ctx: aether.Context): void {
    render.request(ctx, "layout");
  }

  renderAxis(
    props: AxisRenderProps,
    decimalToDataScale: scale.Scale,
    segments?: TickSegment[],
  ): void {
    if (!props.canvases.includes("lower2d")) return;
    const { base } = this.internal;
    const { grid: g, container } = props;
    const position = grid.position(`${this.type}-${this.key}`, g, container);
    const p = {
      ...props,
      position,
      decimalToDataScale,
      segments,
      size: this.state.size + this.state.labelSize,
    };
    const { size } = base.render(p);
    if (!withinSizeThreshold(this.state.size, size))
      this.setState((p) => ({ ...p, size }));
  }

  protected iBounds(
    hold: boolean,
    fetchDataBounds: () => line.DataBounds[],
  ): [bounds.Bounds, Error | null] {
    const [segments, err] = this.iSegments(hold, fetchDataBounds, false);
    return [segments[0], err];
  }

  // iSegments returns the segments of the axis with the error that its data bounds met.
  // With collapsed false it returns one segment. Manual bounds clip the outer segments.
  protected iSegments(
    hold: boolean,
    fetchDataBounds: () => line.DataBounds[],
    collapsed: boolean,
  ): [bounds.Bounds[], Error | null] {
    const { internal: i } = this;
    if (hold && i.segmentSnapshot != null) return [i.segmentSnapshot, null];
    const { manualBounds, bounds: manual, autoBoundPadding, type } = this.state;
    const autoLower = !manualBounds.lower || manual == null;
    const autoUpper = !manualBounds.upper || manual == null;
    if (!collapsed && !autoLower && !autoUpper) {
      i.segmentSnapshot = [manual];
      return [[manual], null];
    }
    let auto: bounds.Bounds[];
    let err: Error | null = null;
    let loading = false;
    try {
      const dataBounds = fetchDataBounds();
      loading = dataBounds.some((b) => b.loading);
      auto = collapsed
        ? collapse(align(dataBounds), autoBoundPadding ?? 0)
        : [autoBounds(dataBounds, autoBoundPadding, type)];
      if (auto.length === 0) auto = [emptyBounds(type)];
    } catch (err_) {
      auto = [emptyBounds(type)];
      err = err_ as Error;
    }
    const clipped = clip(
      auto,
      autoLower ? undefined : manual?.lower,
      autoUpper ? undefined : manual?.upper,
    );
    // A hold keeps segments of loaded data, never of data still loading.
    if (!loading) i.segmentSnapshot = clipped;
    const outer = { lower: clipped[0].lower, upper: clipped[clipped.length - 1].upper };
    if (
      manual == null ||
      (autoLower && manual.lower !== outer.lower) ||
      (autoUpper && manual.upper !== outer.upper)
    )
      i.updateBounds?.(outer);
    return [clipped, err];
  }
}

// clip drops the segments wholly outside lower and upper, and moves the outer edges to
// them. When no segment is left inside, the nearest one takes both edges.
const clip = (
  segments: bounds.Bounds[],
  lower?: number,
  upper?: number,
): bounds.Bounds[] => {
  let s = segments;
  if (lower != null) {
    const inside = s.filter((b) => b.upper > lower);
    s = inside.length > 0 ? inside : s.slice(-1);
    s = [{ ...s[0], lower }, ...s.slice(1)];
  }
  if (upper != null) {
    const inside = s.filter((b) => b.lower < upper);
    s = inside.length > 0 ? inside : s.slice(0, 1);
    s = [...s.slice(0, -1), { ...s[s.length - 1], upper }];
  }
  return s;
};

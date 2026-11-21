// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, bounds, box, TimeRange, TimeStamp } from "@synnaxlabs/x";
import { z } from "zod";

import {
  align,
  type AxisRenderProps,
  BaseAxis,
  baseAxisStateZ,
  newestSample,
} from "@/lineplot/aether/axis";
import { join, Layout, spans, type Trace } from "@/lineplot/aether/layout";
import { YAxis, type YAxisProps } from "@/lineplot/aether/YAxis";
import { range } from "@/lineplot/range/aether";
import { axis } from "@/vis/axis";
import type { line } from "@/vis/line/aether";

export const xAxisStateZ = baseAxisStateZ;

/** Where a click lands: a click on the plot finds only closed breaks. */
const breakSourceZ = z.enum(["plot", "axis"]);
export type BreakSource = z.infer<typeof breakSourceZ>;

/** Opens or closes the break under a click, or opens or closes every break. */
const breakActionZ = z.discriminatedUnion("variant", [
  z.object({
    variant: z.literal("toggle"),
    /** The x of the click, in pixels from the left of the canvas. */
    x: z.number(),
    source: breakSourceZ,
  }),
  z.object({ variant: z.literal("all"), open: z.boolean() }),
]);
export type BreakAction = z.infer<typeof breakActionZ>;

export const breaksRequestZ = z.object({
  action: breakActionZ,
  /** The viewport the action applies to. */
  viewport: box.box,
  /** Views the plot holds to return to, in the same space as the viewport. */
  held: z.array(box.box),
  /** The selection, in the same space as the viewport. */
  selection: box.box,
});
export type BreaksRequest = z.infer<typeof breaksRequestZ>;

/** The open breaks after a break action, and the boxes of its request mapped so each
 * holds the same values. */
export const breaksChangeZ = breaksRequestZ
  .omit({ action: true })
  .extend({ openBreaks: z.array(z.number()) });
export type BreaksChange = z.infer<typeof breaksChangeZ>;

// A click opens or closes a break within this many pixels of its mark.
const BREAK_HIT_DISTANCE = axis.BREAK_WIDTH / 2 + 2;

/** The plot state that lays out an x axis. */
export type LayoutProps = Pick<
  AxisRenderProps,
  "hold" | "viewport" | "gapsCollapsed" | "openBreaks"
>;

export class XAxis extends BaseAxis<typeof baseAxisStateZ, YAxis | range.Provider> {
  static readonly TYPE = "XAxis";
  override schema = baseAxisStateZ;

  /**
   * Draws the axis, its y axes, and its ranges. Each y axis adds a draw of each of its
   * raised lines to raised.
   * @returns the layout of the draw, or null for a deleted axis.
   */
  render(props: AxisRenderProps, raised: Array<() => void>): Layout | null {
    if (this.deleted) return null;
    const [layout, err] = this.iLayout(props);
    const segments = layout.segments.map(({ position, scale }) => ({
      position,
      decimalToDataScale: scale.reverse(),
    }));
    const openBreaks = layout.gaps.filter((g) => g.open).map((g) => layout.mark(g));
    this.renderAxis(props, segments[0].decimalToDataScale, segments, openBreaks);
    this.renderYAxes({ ...props, layout }, raised);
    this.renderRanges(props, layout);
    this.renderBreaks(props, layout);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
    return layout;
  }

  findByXDecimal(
    props: Omit<AxisRenderProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    return this.findByXValue(props, this.layout(props).reverse(target));
  }

  findByXValue(
    props: Omit<AxisRenderProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const p = this.yAxisProps(props);
    return this.yAxes.map((el) => el.findByXValue(p, target)).flat();
  }

  /**
   * @returns the point each line drew at its last render whose x is nearest target, a
   * decimal position across the axis.
   */
  findDrawnByXDecimal(
    props: Omit<AxisRenderProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const p = this.yAxisProps(props);
    const value = p.layout.reverse(target);
    return this.yAxes.map((el) => el.findDrawnByXValue(p, value)).flat();
  }

  /**
   * @returns the open breaks after the action of req, and the boxes of req mapped so
   * each holds the same values, or null when the action changes nothing.
   * @param plot - the plot region, in pixels from the top left of the canvas.
   */
  breaks(props: LayoutProps, req: BreaksRequest, plot: box.Box): BreaksChange | null {
    const { action, viewport } = req;
    const before = this.layout({ ...props, viewport });
    const shown = before.gaps.filter((g) => g.open).map((g) => g.bounds.lower);
    let openBreaks: number[];
    if (action.variant === "all")
      openBreaks = action.open ? before.gaps.map((g) => g.bounds.lower) : [];
    else {
      const width = box.width(plot);
      const position = (action.x - box.left(plot)) / width;
      const hit = BREAK_HIT_DISTANCE / width;
      const gap = before.gapAt(position, hit, action.source === "plot");
      if (gap == null) return null;
      const { lower } = gap.bounds;
      openBreaks = gap.open ? shown.filter((l) => l !== lower) : [...shown, lower];
    }
    if (array.equalAsSets(openBreaks, props.openBreaks)) return null;
    const after = this.layout({ ...props, openBreaks, viewport: box.DECIMAL });
    const view = (b: box.Box) => remap(before, after, b, true);
    return {
      openBreaks,
      viewport: view(viewport),
      held: req.held.map(view),
      selection: remap(before, after, req.selection, false),
    };
  }

  /**
   * @returns the values that the closed break of layout at a decimal position across
   * the plot hides, or null.
   */
  hiddenAt(layout: Layout, position: number, plot: box.Box): bounds.Bounds | null {
    const hit = BREAK_HIT_DISTANCE / box.width(plot);
    return layout.gapAt(position, hit, true)?.bounds ?? null;
  }

  private yAxisProps(
    props: Omit<AxisRenderProps, "canvases">,
  ): Omit<YAxisProps, "canvases"> {
    return { ...props, layout: this.layout(props) };
  }

  private renderYAxes(props: YAxisProps, raised: Array<() => void>): void {
    this.yAxes.forEach((el) => el.render(props, raised));
  }

  get yAxes(): readonly YAxis[] {
    return this.childrenOfType<YAxis>(YAxis.TYPE);
  }

  get ranges(): readonly range.Provider[] {
    return this.childrenOfType<range.Provider>(range.Provider.TYPE);
  }

  get loading(): boolean {
    return this.yAxes.some((el) => el.loading);
  }

  get fetching(): boolean {
    return this.yAxes.some((el) => el.fetching);
  }

  bounds(hold: boolean): bounds.Bounds {
    const [bound, err] = this.iBounds(hold, this.traces.bind(this));
    if (err != null) throw err;
    return bound;
  }

  /** @returns the layout of the axis for the viewport. */
  layout(props: LayoutProps): Layout {
    const [layout, err] = this.iLayout(props);
    if (err != null) throw err;
    return layout;
  }

  // iLayout returns the layout with the error that its bounds met, so a render can draw
  // the axis before it throws.
  private iLayout({
    hold,
    viewport,
    gapsCollapsed,
    openBreaks,
  }: LayoutProps): [Layout, Error | null] {
    const { type } = this.state;
    const traces = this.traces();
    const collapsed = gapsCollapsed && type === "time";
    const [pieces, err] = this.iSegments(hold, () => traces, collapsed);
    const { segments, gaps } = join(pieces, openBreaks);
    const s = spans(align(traces), type, segments);
    return [new Layout(segments, viewport, s, gaps), err];
  }

  private renderRanges(props: AxisRenderProps, layout: Layout): void {
    const clampedBounds = bounds.intersection([
      layout.bounds,
      TimeRange.MAX.numericBounds,
    ]);
    const timeRange = new TimeRange(clampedBounds.lower, clampedBounds.upper);
    const newest = newestSample(this.traces());
    this.ranges.forEach((el) =>
      el.render({
        layout,
        region: props.plot,
        timeRange,
        newest: newest == null ? undefined : new TimeStamp(newest),
      }),
    );
  }

  // renderBreaks draws a band over each seam in view, through the plot and the range
  // strip, so a line stops short of the seam on each side.
  private renderBreaks({ plot, canvases }: AxisRenderProps, layout: Layout): void {
    if (!canvases.includes("upper2d") || layout.segments.length < 2) return;
    const { draw } = this.internal;
    const { theme } = draw;
    const strip = Math.max(
      0,
      ...this.ranges.filter((r) => r.state.visible).map((r) => r.state.height),
    );
    const top = box.top(plot) - strip;
    layout.segments.slice(1).forEach(({ position }) => {
      const x = box.left(plot) + position.lower * box.width(plot);
      if (x <= box.left(plot) || x >= box.right(plot)) return;
      const lower = x - axis.BREAK_WIDTH / 2;
      const upper = x + axis.BREAK_WIDTH / 2;
      draw.container({
        region: box.construct({ x: lower, y: top }, { x: upper, y: box.bottom(plot) }),
        backgroundColor: theme.colors.gray.l0,
        bordered: false,
        rounded: false,
      });
      [lower, upper].forEach((edge) => {
        const crisp = Math.round(edge) + 0.5;
        draw.line({
          stroke: theme.colors.gray.l3,
          lineWidth: 1,
          lineDash: 3,
          start: { x: crisp, y: top },
          end: { x: crisp, y: box.bottom(plot) },
        });
      });
    });
  }

  private traces(): Trace[] {
    return this.yAxes.flatMap((el) => el.traces());
  }
}

// remap returns the box across the home view of after that holds the x values of b
// across the home view of before. A view that falls inside a closed gap keeps its
// width, centered on the seam.
const remap = (before: Layout, after: Layout, b: box.Box, view: boolean): box.Box => {
  const x = box.xBounds(b);
  const values = before.reverseHomeSpan(x);
  let lower = after.home(values.lower);
  let upper = after.home(values.upper);
  if (view && upper <= lower) {
    const half = bounds.span(x) / 2;
    lower -= half;
    upper += half;
  }
  const { one, two, root } = box.construct(b);
  return { root, one: { x: lower, y: one.y }, two: { x: upper, y: two.y } };
};

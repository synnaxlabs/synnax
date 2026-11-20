// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, TimeRange, TimeStamp } from "@synnaxlabs/x";

import {
  align,
  type AxisRenderProps,
  BaseAxis,
  baseAxisStateZ,
  newestSample,
} from "@/lineplot/aether/axis";
import { Layout, spans, type Trace } from "@/lineplot/aether/layout";
import { YAxis, type YAxisProps } from "@/lineplot/aether/YAxis";
import { range } from "@/lineplot/range/aether";
import { axis } from "@/vis/axis";
import type { line } from "@/vis/line/aether";

export const xAxisStateZ = baseAxisStateZ;

/** The plot state that lays out an x axis. */
export type LayoutProps = Pick<AxisRenderProps, "hold" | "viewport" | "gapsCollapsed">;

export class XAxis extends BaseAxis<typeof baseAxisStateZ, YAxis | range.Provider> {
  static readonly TYPE = "XAxis";
  override schema = baseAxisStateZ;

  /**
   * Draws the axis, its y axes, and its ranges. Each y axis adds a draw of each of its
   * raised lines to raised.
   */
  render(props: AxisRenderProps, raised: Array<() => void>): void {
    if (this.deleted) return;
    const [layout, err] = this.iLayout(props);
    const segments = layout.segments.map(({ position, scale }) => ({
      position,
      decimalToDataScale: scale.reverse(),
    }));
    this.renderAxis(props, segments[0].decimalToDataScale, segments);
    this.renderYAxes({ ...props, layout }, raised);
    this.renderRanges(props, layout);
    this.renderBreaks(props, layout);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
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
  }: LayoutProps): [Layout, Error | null] {
    const { type } = this.state;
    const traces = this.traces();
    const collapsed = gapsCollapsed && type === "time";
    const [segments, err] = this.iSegments(hold, () => traces, collapsed);
    const s = spans(align(traces), type, segments);
    return [new Layout(segments, viewport, s), err];
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

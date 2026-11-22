// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Instrumentation } from "@synnaxlabs/alamos";
import { type bounds, box, type destructor, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import type { Gap, Layout } from "@/lineplot/aether/layout";
import {
  type BreaksChange,
  breaksChangeZ,
  type BreaksRequest,
  breaksRequestZ,
  type LayoutProps,
  XAxis,
} from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { measure } from "@/lineplot/measure/aether";
import { rule } from "@/lineplot/rule/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { status } from "@/status/aether";
import { grid } from "@/vis/grid";
import type { line } from "@/vis/line/aether";
import { render } from "@/vis/render";

export type AxesBounds = Record<string, bounds.Bounds>;

const breakCountsZ = z.object({ open: z.number(), closed: z.number() });
export type BreakCounts = z.infer<typeof breakCountsZ>;

const numericBoundsZ = z.object({ lower: z.number(), upper: z.number() });

export const linePlotStateZ = z.object({
  container: box.box,
  viewport: box.box,
  hold: z.boolean().default(false),
  /** Set when x1 shows its gaps at their true width, with no breaks. */
  breaksHidden: z.boolean().default(false),
  /** The lower bounds of the gaps of x1 that show in place of a break. */
  openBreaks: z.array(z.number()).default([]),
  /**
   * The number of open and of closed breaks on x1, which the plot sets. It counts them
   * while they are hidden too.
   */
  breaks: breakCountsZ.default({ open: 0, closed: 0 }),
  grid: z.record(z.string(), grid.regionZ),
  visible: z.boolean().default(true),
  clearOverScan: xy.crudeZ.default(xy.ZERO),
  loading: z.boolean().default(false),
  fetching: z.boolean().default(false),
});

const axesBoundsZ = z.record(z.string(), numericBoundsZ);

export const linePlotMethodsZ = {
  /**
   * Returns, for each x axis, the values that a span of decimal positions across the
   * home view holds.
   */
  getXBounds: z.function({ input: z.tuple([numericBoundsZ]), output: axesBoundsZ }),
  /**
   * Returns, for each x axis, the decimal positions across the home view of a span of
   * values. It is the inverse of getXBounds.
   */
  getXPositions: z.function({ input: z.tuple([numericBoundsZ]), output: axesBoundsZ }),
  /**
   * Returns the open breaks of x1 after a break action, and the boxes of the request
   * mapped so each holds the same values, or null when the action changes nothing. It
   * sets nothing: the caller sets the open breaks and the boxes.
   */
  getBreakChange: z.function({
    input: z.tuple([breaksRequestZ]),
    output: breaksChangeZ.nullable(),
  }),
  /** Returns the decimal positions across the home view of the seams of x1. */
  getSeams: z.function({ input: z.tuple([]), output: z.array(z.number()) }),
};

interface InternalState {
  instrumentation: Instrumentation;
  handleError: status.ErrorHandler;
  renderCtx: render.Context;
  stopListeningRestore?: destructor.Destructor;
}

type Children = XAxis | tooltip.Tooltip | measure.Measure;

const RENDER_CANVASES: render.CanvasVariant[] = ["upper2d", "lower2d", "gl"] as const;
const TOOL_RENDER_CANVASES: render.CanvasVariant[] = ["upper2d"];

// Only the x axis keyed x1 collapses its gaps: the x axes share one viewport, and a
// break change maps it through one layout.
const isX1 = (a: XAxis): boolean => (a.state.axisKey ?? a.key) === "x1";

// axisProps returns the props that the x axis a lays out with.
const axisProps = <P extends LayoutProps>(a: XAxis, props: P): P =>
  isX1(a) ? props : { ...props, breaksHidden: true };

export class LinePlot
  extends aether.Composite<
    typeof linePlotStateZ,
    InternalState,
    Children,
    typeof linePlotMethodsZ
  >
  implements aether.HandlersFromSchema<typeof linePlotMethodsZ>
{
  static readonly TYPE: string = "LinePlot";

  override schema = linePlotStateZ;
  override methods = linePlotMethodsZ;

  override afterUpdate(ctx: aether.Context): void {
    this.internal.instrumentation = alamos.useInstrumentation(ctx, "lineplot");
    this.internal.handleError = status.useErrorHandler(ctx);
    this.internal.renderCtx = render.Context.use(ctx);
    render.control(ctx, (r) => {
      if (!this.state.visible) return;
      this.requestRender("low", r);
    });
    this.internal.stopListeningRestore?.();
    this.internal.stopListeningRestore = this.internal.renderCtx.onRestore(() => {
      if (this.state.visible) this.requestRender("high", "layout");
    });
    if (!this.state.visible && !this.prevState.visible) return;
    this.requestRender("high", "layout");
  }

  override afterDelete(ctx: aether.Context): void {
    this.internal.stopListeningRestore?.();
    this.internal.renderCtx = render.Context.use(ctx);
    this.requestRender("high", "layout");
  }

  findByXDecimal(x: number): line.FindResult[] {
    const props = { ...this.state, plot: this.calculatePlot() };
    return this.axes
      .flatMap((xAxis) => xAxis.findByXDecimal(axisProps(xAxis, props), x))
      .flat();
  }

  private findDrawnByXDecimal(x: number): line.FindResult[] {
    const props = { ...this.state, plot: this.calculatePlot() };
    return this.axes.flatMap((a) => a.findDrawnByXDecimal(axisProps(a, props), x));
  }

  findByXValue(x: number): line.FindResult[] {
    const props = { ...this.state, plot: this.calculatePlot() };
    return this.axes.flatMap((a) => a.findByXValue(axisProps(a, props), x)).flat();
  }

  private get axes(): readonly XAxis[] {
    return this.childrenOfType<XAxis>(XAxis.TYPE);
  }

  private get tooltips(): readonly tooltip.Tooltip[] {
    return this.childrenOfType<tooltip.Tooltip>(tooltip.Tooltip.TYPE);
  }

  private get measures(): readonly measure.Measure[] {
    return this.childrenOfType<measure.Measure>(measure.Measure.TYPE);
  }

  private get loading(): boolean {
    return this.axes.some((a) => a.loading);
  }

  private get fetching(): boolean {
    return this.axes.some((a) => a.fetching);
  }

  // Lines record their loading state as they render, so it syncs after a render.
  private syncLoading(): void {
    const { loading, fetching } = this;
    if (loading !== this.state.loading || fetching !== this.state.fetching)
      this.setState((p) => ({ ...p, loading, fetching }));
  }

  private syncBreaks(gaps: readonly Gap[]): void {
    const open = gaps.filter((g) => g.open).length;
    const breaks = { open, closed: gaps.length - open };
    const prev = this.state.breaks;
    if (prev.open !== breaks.open || prev.closed !== breaks.closed)
      this.setState((p) => ({ ...p, breaks }));
  }

  // renderAxes draws each x axis and returns the layout of x1, or null.
  private renderAxes(plot: box.Box, canvases: render.CanvasVariant[]): Layout | null {
    const p = { ...this.state, plot, canvases };
    // Raised lines draw after the lines of every axis, so they draw over all of them.
    const raised: Array<() => void> = [];
    try {
      const drawn = this.axes.map((xAxis) => xAxis.render(axisProps(xAxis, p), raised));
      const i = this.axes.findIndex(isX1);
      const [x1, gaps] = (i < 0 ? null : drawn[i]) ?? [null, []];
      this.syncBreaks(gaps);
      const err = drawn.find((d) => d?.[2] != null)?.[2];
      if (err != null) throw err;
      return x1;
    } finally {
      raised.forEach((draw) => draw());
    }
  }

  getSeams(): number[] {
    return this.axes.find(isX1)?.layout(this.state).seams ?? [];
  }

  getBreakChange(req: BreaksRequest): BreaksChange | null {
    const x1 = this.axes.find(isX1);
    if (x1 == null) return null;
    return x1.breaks(this.state, req, this.calculatePlot());
  }

  private renderTooltips(
    region: box.Box,
    canvases: render.CanvasVariant[],
    x1: Layout | null,
  ): void {
    const findByXDecimal = this.findDrawnByXDecimal.bind(this);
    const hiddenAt = (position: number) =>
      x1 == null ? null : XAxis.hiddenAt(x1, position, region);
    const p = { findByXDecimal, hiddenAt, region, canvases };
    this.tooltips.forEach((t) => t.render(p));
  }

  getXBounds(positions: bounds.Bounds): AxesBounds {
    return Object.fromEntries(
      this.axes.map((a) => [
        a.state.axisKey ?? a.key,
        a.layout(axisProps(a, this.state)).reverseHomeBounds(positions),
      ]),
    );
  }

  getXPositions(values: bounds.Bounds): AxesBounds {
    return Object.fromEntries(
      this.axes.map((a) => {
        const layout = a.layout(axisProps(a, this.state));
        const positions = {
          lower: layout.home(values.lower),
          upper: layout.home(values.upper),
        };
        return [a.state.axisKey ?? a.key, positions];
      }),
    );
  }

  private renderMeasures(region: box.Box, x1: Layout | null): void {
    const p: measure.MeasureProps = {
      findByXDecimal: this.findByXDecimal.bind(this),
      findByXValue: this.findByXValue.bind(this),
      // Only the lines of x1 break.
      closedBetween: (a, b) =>
        x1 != null &&
        x1.hasSpan(a.key) &&
        x1.hasSpan(b.key) &&
        x1.closedBetween(a.value.x, b.value.x),
      region,
    };
    this.measures.forEach((m) => m.render(p));
  }

  private calculatePlot(): box.Box {
    return grid.visualizationBox(this.state.grid, this.state.container);
  }

  private render(canvases: render.CanvasVariant[]): render.Cleanup | undefined {
    const { instrumentation: ins, renderCtx, handleError } = this.internal;
    if (this.deleted) {
      ins.L.debug("deleted, skipping render", { key: this.key });
      return;
    }
    if (!this.state.visible) {
      ins.L.debug("not visible, skipping render", { key: this.key });
      this.syncLoading();
      return ({ canvases }) =>
        renderCtx.erase(this.state.container, this.state.clearOverScan, ...canvases);
    }

    const plot = this.calculatePlot();

    ins.L.debug("rendering", {
      key: this.key,
      viewport: this.state.viewport,
      container: this.state.container,
      grid: this.state.grid,
      plot,
      canvases,
    });

    const os = xy.construct(this.state.clearOverScan);

    const removeCanvasScissor = renderCtx.scissor(
      this.state.container,
      os,
      canvases.filter((c) => c !== "gl"),
    );
    const removeGLScissor = renderCtx.scissor(
      plot,
      xy.ZERO,
      canvases.filter((c) => c === "gl"),
    );

    try {
      const x1 = this.renderAxes(plot, canvases);
      this.renderTooltips(plot, canvases, x1);
      this.renderMeasures(plot, x1);
    } catch (e) {
      handleError(e, "Failed to render line plot");
    } finally {
      removeCanvasScissor();
      removeGLScissor();
    }
    this.syncLoading();
    ins.L.debug("rendered", { key: this.key });
    const eraseRegion = box.copy(this.state.container);

    return ({ canvases }) =>
      renderCtx.erase(eraseRegion, this.state.clearOverScan, ...canvases);
  }

  requestRender(priority: render.Priority, reason: string): void {
    const { renderCtx: ctx } = this.internal;
    let canvases = RENDER_CANVASES;
    // Optimization for tooltips, measures and other utilities. In this case, we only
    // need to render the upper2d canvas.
    if (reason === "tool") canvases = TOOL_RENDER_CANVASES;
    ctx.loop.set({
      key: `${this.type}-${this.key}`,
      render: () => this.render(canvases),
      priority,
      canvases,
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [LinePlot.TYPE]: LinePlot,
  [XAxis.TYPE]: XAxis,
  [YAxis.TYPE]: YAxis,
  ...measure.REGISTRY,
  ...rule.REGISTRY,
  ...tooltip.REGISTRY,
};

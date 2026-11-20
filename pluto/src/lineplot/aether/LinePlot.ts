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
import type { Layout } from "@/lineplot/aether/layout";
import {
  type BreaksChange,
  breaksChangeZ,
  type BreaksRequest,
  breaksRequestZ,
  XAxis,
} from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { measure } from "@/lineplot/measure/aether";
import { rule } from "@/lineplot/rule/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { status } from "@/status/aether";
import { grid } from "@/vis/grid";
import type { FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

export type AxesBounds = Record<string, bounds.Bounds>;

const breakCountsZ = z.object({ open: z.number(), closed: z.number() });
export type BreakCounts = z.infer<typeof breakCountsZ>;

const numericBoundsZ = z.object({ lower: z.number(), upper: z.number() });

export const linePlotStateZ = z.object({
  container: box.box,
  viewport: box.box,
  hold: z.boolean().default(false),
  gapsCollapsed: z.boolean().default(false),
  /** The lower bounds of the gaps of x1 that show in place of a break. */
  openBreaks: z.array(z.number()).default([]),
  /** The number of open and of closed breaks on x1, which the plot sets. */
  breaks: breakCountsZ.default({ open: 0, closed: 0 }),
  grid: z.record(z.string(), grid.regionZ),
  visible: z.boolean().default(true),
  clearOverScan: xy.crudeZ.default(xy.ZERO),
  loading: z.boolean().default(false),
  fetching: z.boolean().default(false),
});

const axesBoundsZ = z.record(z.string(), numericBoundsZ);

export const linePlotMethodsZ = {
  getBounds: z.function({ input: z.tuple([]), output: axesBoundsZ }),
  /**
   * Returns, for each x axis, the values that a span of decimal positions across the
   * home view holds.
   */
  getXBounds: z.function({ input: z.tuple([numericBoundsZ]), output: axesBoundsZ }),
  /**
   * Applies a break action to x1, and returns the open breaks and the boxes of the
   * request mapped so each holds the same values, or null when the action changes
   * nothing. The caller sets the open breaks and the boxes.
   */
  setBreaks: z.function({
    input: z.tuple([breaksRequestZ]),
    output: breaksChangeZ.nullable(),
  }),
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

  findByXDecimal(x: number): FindResult[] {
    const props = { ...this.state, plot: this.calculatePlot() };
    return this.axes.flatMap((xAxis) => xAxis.findByXDecimal(props, x)).flat();
  }

  private findDrawnByXDecimal(x: number): FindResult[] {
    const props = { ...this.state, plot: this.calculatePlot() };
    return this.axes.flatMap((a) => a.findDrawnByXDecimal(props, x));
  }

  findByXValue(x: number): FindResult[] {
    const props = { ...this.state, plot: this.calculatePlot() };
    return this.axes.flatMap((a) => a.findByXValue(props, x)).flat();
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

  private syncBreaks(layout: Layout | null): void {
    const gaps = layout?.gaps ?? [];
    const open = gaps.filter((g) => g.open).length;
    const breaks = { open, closed: gaps.length - open };
    const prev = this.state.breaks;
    if (prev.open !== breaks.open || prev.closed !== breaks.closed)
      this.setState((p) => ({ ...p, breaks }));
  }

  private renderAxes(plot: box.Box, canvases: render.CanvasVariant[]): void {
    const p = { ...this.state, plot, canvases };
    // Raised lines draw after the lines of every axis, so they draw over all of them.
    const raised: Array<() => void> = [];
    try {
      const layouts = this.axes.map((xAxis) => xAxis.render(p, raised));
      this.syncBreaks(layouts[0] ?? null);
    } finally {
      raised.forEach((draw) => draw());
    }
  }

  setBreaks(req: BreaksRequest): BreaksChange | null {
    const x1 = this.axes.at(0);
    if (x1 == null) return null;
    return x1.breaks(this.state, req, this.calculatePlot());
  }

  private renderTooltips(region: box.Box, canvases: render.CanvasVariant[]): void {
    const findByXDecimal = this.findDrawnByXDecimal.bind(this);
    const p = { findByXDecimal, region, canvases };
    this.tooltips.forEach((t) => t.render(p));
  }

  getBounds(): AxesBounds {
    const bounds: AxesBounds = {};
    this.axes.forEach((v) => {
      const axisKey = v.state.axisKey ?? v.key;
      const layout = v.layout(this.state);
      bounds[axisKey] = layout.bounds;
      v.yAxes.forEach((y) => {
        const yAxisKey = y.state.axisKey ?? y.key;
        bounds[yAxisKey] = y.bounds(this.state.hold, layout);
      });
    });
    return bounds;
  }

  getXBounds(positions: bounds.Bounds): AxesBounds {
    return Object.fromEntries(
      this.axes.map((a) => [a.state.axisKey ?? a.key, a.values(this.state, positions)]),
    );
  }

  private renderMeasures(region: box.Box): void {
    const p: measure.MeasureProps = {
      findByXDecimal: this.findByXDecimal.bind(this),
      findByXValue: this.findByXValue.bind(this),
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
      this.renderAxes(plot, canvases);
      this.renderTooltips(plot, canvases);
      this.renderMeasures(plot);
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

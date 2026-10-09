// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { type bounds, box, deep, stats, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { XAxis } from "@/lineplot/aether/XAxis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { measure } from "@/lineplot/measure/aether";
import { rule } from "@/lineplot/rule/aether";
import { tooltip } from "@/lineplot/tooltip/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { grid } from "@/vis/grid";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

export type AxesBounds = Record<string, bounds.Bounds>;

const statisticZ = z.number().nullable();

/** Statistics of one line's samples inside the measurement window. */
export const measurementZ = z.object({
  line: z.string(),
  label: z.string().optional(),
  units: z.string().optional(),
  min: statisticZ,
  max: statisticZ,
  /** The x value at the maximum sample. */
  peakX: statisticZ,
  peakToPeak: statisticZ,
  mean: statisticZ,
  rms: statisticZ,
  /** Hz when x is time, otherwise cycles per x unit. */
  frequency: statisticZ,
  riseTime: statisticZ,
});

export type Measurement = z.infer<typeof measurementZ>;

export const linePlotStateZ = z.object({
  container: box.box,
  viewport: box.box,
  hold: z.boolean().default(false),
  grid: z.record(z.string(), grid.regionZ),
  visible: z.boolean().default(true),
  clearOverScan: xy.crudeZ.default(xy.ZERO),
  loading: z.boolean().default(false),
  /** Computes measurements on every render while set. */
  measuring: z.boolean().default(false),
  measurements: z.array(measurementZ).default([]),
});

const MEASURE_INTERVAL = 250; // ms

const statistic = (v: number): number | null => (Number.isFinite(v) ? v : null);

const peakX = (x: Float64Array, y: Float64Array): number => {
  let at = -1;
  for (let i = 0; i < y.length; i++) if (at === -1 || y[i] > y[at]) at = i;
  return at === -1 ? NaN : x[at];
};

const axesBoundsZ = z.record(
  z.string(),
  z.object({ lower: z.number(), upper: z.number() }),
);

export const linePlotMethodsZ = {
  getBounds: z.function({ input: z.tuple([]), output: axesBoundsZ }),
};

interface InternalState {
  instrumentation: Instrumentation;
  handleError: status.ErrorHandler;
  renderCtx: render.Context;
  measuredAt: number;
}

type Children = XAxis | tooltip.Tooltip | measure.Measure;

const calculateExposure = (viewport: box.Box, region: box.Box): number => {
  const vpArea = box.width(viewport) * Math.sqrt(box.height(viewport));
  const regArea = box.width(region) * Math.sqrt(box.height(region));
  return vpArea / regArea;
};

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

  schema = linePlotStateZ;
  methods = linePlotMethodsZ;

  afterUpdate(ctx: aether.Context): void {
    this.internal.instrumentation = alamos.useInstrumentation(ctx, "lineplot");
    this.internal.handleError = status.useErrorHandler(ctx);
    this.internal.renderCtx = render.Context.use(ctx);
    this.internal.measuredAt ??= 0;
    if (
      this.state.hold !== this.prevState.hold ||
      !ctx.wasSetPreviously(telem.HOLD_CONTEXT_KEY)
    )
      telem.setHold(ctx, this.state.hold);
    render.control(ctx, (r) => {
      if (!this.state.visible) return;
      this.requestRender("low", r);
    });
    if (!this.state.visible && !this.prevState.visible) return;
    this.requestRender("high", "layout");
  }

  afterDelete(ctx: aether.Context): void {
    this.internal.renderCtx = render.Context.use(ctx);
    this.requestRender("high", "layout");
  }

  findByXDecimal(x: number): FindResult[] {
    const props = {
      ...this.state,
      plot: this.calculatePlot(),
      exposure: this.exposure,
    };
    return this.axes.flatMap((xAxis) => xAxis.findByXDecimal(props, x)).flat();
  }

  findByXValue(x: number): FindResult[] {
    const props = {
      ...this.state,
      plot: this.calculatePlot(),
      exposure: this.exposure,
    };
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

  private get exposure(): number {
    return calculateExposure(this.state.viewport, this.state.container);
  }

  private get loading(): boolean {
    return this.axes.some((a) => a.loading);
  }

  private renderAxes(plot: box.Box, canvases: render.CanvasVariant[]): void {
    const p = { ...this.state, plot, canvases, exposure: this.exposure };
    this.axes.forEach((xAxis) => xAxis.render(p));
  }

  private renderTooltips(region: box.Box, canvases: render.CanvasVariant[]): void {
    const p = { findByXDecimal: this.findByXDecimal.bind(this), region, canvases };
    this.tooltips.forEach((t) => t.render(p));
  }

  getBounds(): AxesBounds {
    const bounds: AxesBounds = {};
    this.axes.forEach((v) => {
      const axisKey = v.state.axisKey ?? v.key;
      const xBounds = v.bounds(this.state.hold);
      bounds[axisKey] = xBounds;
      v.yAxes.forEach((y) => {
        const yAxisKey = y.state.axisKey ?? y.key;
        bounds[yAxisKey] = y.bounds(this.state.hold, xBounds);
      });
    });
    return bounds;
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

  // The window is the span between the measure points, or the visible x range before
  // both are placed. Results reach React through state, throttled and only on change.
  private measure(plot: box.Box): void {
    const now = performance.now();
    if (now - this.internal.measuredAt < MEASURE_INTERVAL) return;
    this.internal.measuredAt = now;
    const props = { ...this.state, plot, exposure: this.exposure };
    const pointWindow = this.measures.map((m) => m.window()).find((w) => w != null);
    const measurements: Measurement[] = [];
    for (const xAxis of this.axes) {
      const window = pointWindow ?? xAxis.visibleBounds(props);
      const time = xAxis.state.type === "time";
      for (const yAxis of xAxis.yAxes)
        for (const line of yAxis.visibleLines) {
          const { x, y } = line.samplesIn(window);
          const xs = time ? x.map((v) => v / 1e9) : x;
          measurements.push({
            line: line.key,
            label: line.state.label,
            units: yAxis.state.label,
            min: statistic(stats.min(y)),
            max: statistic(stats.max(y)),
            peakX: statistic(time ? peakX(x, y) : peakX(xs, y)),
            peakToPeak: statistic(stats.peakToPeak(y)),
            mean: statistic(stats.mean(y)),
            rms: statistic(stats.rms(y)),
            frequency: statistic(stats.frequency(xs, y)),
            riseTime: statistic(stats.riseTime(xs, y)),
          });
        }
    }
    if (deep.equal(measurements, this.state.measurements)) return;
    this.setState((p) => ({ ...p, measurements }));
  }

  private render(canvases: render.CanvasVariant[]): render.Cleanup | undefined {
    const { instrumentation: ins, renderCtx, handleError } = this.internal;
    if (this.deleted) {
      ins.L.debug("deleted, skipping render", { key: this.key });
      return;
    }
    // Skips draws while loading to free the worker and avoid autoscaling partial data.
    const loading = this.loading;
    if (loading !== this.state.loading) this.setState((p) => ({ ...p, loading }));
    const skip = !this.state.visible ? "not visible" : loading ? "loading" : null;
    if (skip != null) {
      ins.L.debug(`${skip}, skipping render`, { key: this.key });
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
      if (this.state.measuring) this.measure(plot);
    } catch (e) {
      handleError(e, "Failed to render line plot");
    } finally {
      removeCanvasScissor();
      removeGLScissor();
    }
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

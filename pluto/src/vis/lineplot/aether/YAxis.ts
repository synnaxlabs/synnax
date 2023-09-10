// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, bounds, scale, location, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/core/fontString";
import { axis } from "@/vis/axis";
import { line } from "@/vis/line/aether";
import {
  type GridPositionSpec,
  calculateGridPosition,
  autoBounds,
  withinSizeThreshold,
  emptyBounds,
} from "@/vis/lineplot/aether/grid";
import { render } from "@/vis/render";
import { rule } from "@/vis/rule/aether";

export const yAxisStateZ = axis.axisStateZ
  .extend({
    location: location.x.optional().default("left"),
    bounds: bounds.bounds.optional(),
    autoBoundPadding: z.number().optional().default(0.05),
    size: z.number().optional().default(0),
    labelSize: z.number().optional().default(0),
    label: z.string().optional().default(""),
  })
  .partial({
    color: true,
    gridColor: true,
    font: true,
  });

export interface YAxisProps {
  grid: GridPositionSpec[];
  plot: box.Box;
  viewport: box.Box;
  container: box.Box;
  xDataToDecimalScale: scale.Scale;
  canvases: render.CanvasVariant[];
  hold: boolean;
}

interface InternalState {
  render: render.Context;
  core: axis.Axis;
  // In the case where we're in a hold, we want to keep a snapshot of the hold bounds
  // so that we can rerender the plot in the same position even if the data changes.
  // changes.
  boundSnapshot?: bounds.Bounds;
}

type Children = line.Line | rule.Rule;

export class YAxis extends aether.Composite<
  typeof yAxisStateZ,
  InternalState,
  Children
> {
  static readonly TYPE = CSS.BE("line-plot", "y-axis");
  schema = yAxisStateZ;

  afterUpdate(): void {
    this.internal.render = render.Context.use(this.ctx);
    const theme = theming.use(this.ctx);
    this.internal.core = new axis.Canvas(this.internal.render, {
      color: theme.colors.gray.p1,
      font: fontString(theme, "small"),
      gridColor: theme.colors.gray.m2,
      ...this.state,
      size: this.state.size + this.state.labelSize,
    });
    render.Controller.requestRender(this.ctx, render.REASON_LAYOUT);
  }

  async xBounds(): Promise<bounds.Bounds> {
    return bounds.max(
      await Promise.all(this.lines.map(async (el) => await el.xBounds())),
    );
  }

  async render(props: YAxisProps): Promise<void> {
    const [dataToDecimalScale, error] = await this.dataToDecimalScale(
      props.viewport,
      props.hold,
    );
    // We need to invert scale because the y-axis is inverted in decimal space.
    const decimalToDataScale = dataToDecimalScale.invert().reverse();
    this.renderAxis(props, decimalToDataScale);
    // Throw the error we encounter here so that the user still has a visible axis.
    if (error != null) throw error;
    await this.renderLines(props, dataToDecimalScale);
    await this.renderRules(props, decimalToDataScale);
  }

  private renderAxis(
    { grid, plot, container, canvases }: YAxisProps,
    decimalToDataScale: scale.Scale,
  ): void {
    if (!canvases.includes("lower2d")) return;
    const { core } = this.internal;
    const position = calculateGridPosition(this.key, grid, container);
    const { size: currentSize } = this.state;
    const props = { plot, position, decimalToDataScale };
    const { size: nextSize } = core.render(props);
    if (!withinSizeThreshold(currentSize, nextSize))
      this.setState((p) => ({ ...p, size: nextSize }));
  }

  private async renderLines(
    { xDataToDecimalScale: xScale, plot, canvases }: YAxisProps,
    yScale: scale.Scale,
  ): Promise<void> {
    if (!canvases.includes("gl")) return;
    const props: line.LineProps = {
      region: plot,
      dataToDecimalScale: new scale.XY(xScale, yScale),
    };
    await Promise.all(this.lines.map(async (el) => await el.render(props)));
  }

  private async renderRules(
    { container, plot, canvases }: YAxisProps,
    decimalToDataScale: scale.Scale,
  ): Promise<void> {
    if (!canvases.includes("upper2d")) return;
    const { location } = this.state;
    const { render } = this.internal;
    const scissor = box.construct(
      box.left(container),
      box.top(plot),
      box.width(container),
      box.height(plot),
    );
    const clearScissor = render.scissor(scissor, xy.ZERO, ["upper2d"]);
    const props = { container, plot, decimalToDataScale, location };
    await Promise.all(this.rules.map(async (el) => await el.render(props)));
    clearScissor();
  }

  async findByXValue(
    { xDataToDecimalScale, plot, viewport, hold }: Omit<YAxisProps, "canvases">,
    target: number,
  ): Promise<line.FindResult[]> {
    const [yDataToDecimalScale, error] = await this.dataToDecimalScale(viewport, hold);
    if (error != null) throw error;
    const dataToDecimalScale = new scale.XY(xDataToDecimalScale, yDataToDecimalScale);
    const props: line.LineProps = { region: plot, dataToDecimalScale };
    return (
      await Promise.all(
        this.lines.map(async (el) => await el.findByXValue(props, target)),
      )
    ).map((v) => ({ ...v, units: this.state.label }));
  }

  private async yBounds(hold: boolean): Promise<[bounds.Bounds, Error | null]> {
    if (hold && this.internal.boundSnapshot != null)
      return [this.internal.boundSnapshot, null];
    if (this.state.bounds != null && !bounds.isZero(this.state.bounds))
      return [this.state.bounds, null];
    try {
      const bounds = await Promise.all(
        this.lines.map(async (el) => await el.yBounds()),
      );
      const ab = autoBounds(bounds, this.state.autoBoundPadding, this.state.type);
      this.internal.boundSnapshot = ab;
      return [ab, null];
    } catch (err) {
      return [emptyBounds(this.state.type), err as Error];
    }
  }

  private async dataToDecimalScale(
    viewport: box.Box,
    hold: boolean,
  ): Promise<[scale.Scale, Error | null]> {
    const [bounds, err] = await this.yBounds(hold);
    return [
      scale.Scale.scale(bounds)
        .scale(1)
        .translate(-box.y(viewport))
        .magnify(1 / box.height(viewport)),
      err,
    ];
  }

  private get lines(): readonly line.Line[] {
    return this.childrenOfType(line.Line.TYPE);
  }

  private get rules(): readonly rule.Rule[] {
    return this.childrenOfType(rule.Rule.TYPE);
  }
}

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, Box, Location, Scale, XYScale } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { ThemeContext } from "@/core/theming/aether";
import { fontString } from "@/core/theming/fontString";
import { Axis, AxisCanvas } from "@/core/vis/Axis";
import { axisState } from "@/core/vis/Axis/core";
import { AetherLine, LineProps, FindResult } from "@/core/vis/Line/aether";
import {
  GridPositionMeta,
  calculateGridPosition as gridPosition,
  autoBounds,
  withinSizeThreshold,
} from "@/core/vis/LinePlot/aether/grid";
import { RenderContext } from "@/core/vis/render";
import { AetherRule } from "@/core/vis/Rule/aether";

const stateZ = axisState
  .extend({
    location: Location.strictXZ.optional().default("left"),
    bounds: Bounds.looseZ.optional(),
    autoBoundPadding: z.number().optional().default(0.05),
    size: z.number().optional().default(0),
    labelSize: z.number().optional().default(0),
  })
  .partial({
    color: true,
    gridColor: true,
    font: true,
  });

export interface YAxisProps {
  grid: GridPositionMeta[];
  plot: Box;
  viewport: Box;
  container: Box;
  xDataToDecimalScale: Scale;
}

interface InternalState {
  render: RenderContext;
  core: Axis;
}

type Children = AetherLine | AetherRule;

export class AetherYAxis extends AetherComposite<
  typeof stateZ,
  InternalState,
  Children
> {
  static readonly TYPE = CSS.BE("line-plot", "y-axis");
  static readonly stateZ = stateZ;
  schema = AetherYAxis.stateZ;

  afterUpdate(): void {
    this.internal.render = RenderContext.use(this.ctx);
    const theme = ThemeContext.use(this.ctx);
    this.internal.core = new AxisCanvas(this.internal.render, {
      color: theme.colors.gray.p2,
      font: fontString(theme, "small"),
      gridColor: theme.colors.gray.m2,
      ...this.state,
      size: this.state.size + this.state.labelSize,
    });
  }

  async xBounds(): Promise<Bounds> {
    return Bounds.max(
      await Promise.all(this.lines.map(async (el) => await el.xBounds()))
    );
  }

  async render(props: YAxisProps): Promise<void> {
    const dataToDecimalScale = await this.dataToDecimalScale(props.viewport);
    // We need to invert scale because the y-axis is inverted in decimal space.
    const decimalToDataScale = dataToDecimalScale.invert().reverse();
    this.renderAxis(props, decimalToDataScale);
    await this.renderLines(props, dataToDecimalScale);
    await this.renderRules(props, decimalToDataScale);
  }

  private renderAxis({ grid, plot }: YAxisProps, decimalToDataScale: Scale): void {
    const { core } = this.internal;
    const position = gridPosition(this.key, grid, plot);
    const { size: currentSize } = this.state;
    const props = { plot, position, decimalToDataScale };
    const { size: nextSize } = core.render(props);
    if (!withinSizeThreshold(currentSize, nextSize))
      this.setState((p) => ({ ...p, size: nextSize }));
  }

  private async renderLines(
    { xDataToDecimalScale: xScale, plot }: YAxisProps,
    yScale: Scale
  ): Promise<void> {
    const props: LineProps = {
      region: plot,
      dataToDecimalScale: new XYScale(xScale, yScale),
    };
    await Promise.all(this.lines.map(async (el) => await el.render(props)));
  }

  private async renderRules(
    { container, plot }: YAxisProps,
    decimalToDataScale: Scale
  ): Promise<void> {
    const { location } = this.state;
    const { render } = this.internal;
    const scissor = new Box(container.left, plot.top, container.width, plot.height);
    const clearScissor = render.scissorCanvas(scissor);
    const props = { container, plot, decimalToDataScale, location };
    await Promise.all(this.rules.map(async (el) => await el.render(props)));
    clearScissor();
  }

  async findByXValue(
    { xDataToDecimalScale, plot, viewport }: YAxisProps,
    target: number
  ): Promise<FindResult[]> {
    const yDataToDecimalScale = await this.dataToDecimalScale(viewport);
    const dataToDecimalScale = new XYScale(xDataToDecimalScale, yDataToDecimalScale);
    const props: LineProps = { region: plot, dataToDecimalScale };
    return await Promise.all(
      this.lines.map(async (el) => await el.findByXValue(props, target))
    );
  }

  private async yBounds(): Promise<Bounds> {
    if (this.state.bounds != null && !this.state.bounds.isZero)
      return this.state.bounds;
    const bounds = await Promise.all(this.lines.map(async (el) => await el.yBounds()));
    return autoBounds(bounds, this.state.autoBoundPadding, this.state.type);
  }

  private async dataToDecimalScale(viewport: Box): Promise<Scale> {
    const bounds = await this.yBounds();
    return Scale.scale(bounds)
      .scale(1)
      .translate(-viewport.y)
      .magnify(1 / viewport.height);
  }

  private get lines(): readonly AetherLine[] {
    return this.childrenOfType(AetherLine.TYPE);
  }

  private get rules(): readonly AetherRule[] {
    return this.childrenOfType(AetherRule.TYPE);
  }
}

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location, bounds, box, scale } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/core/fontString";
import { axis } from "@/vis/axis";
import { Canvas } from "@/vis/axis/canvas";
import { type FindResult } from "@/vis/line/aether/line";
import {
  calculateGridPosition,
  autoBounds,
  withinSizeThreshold,
  emptyBounds,
} from "@/vis/lineplot/aether/grid";
import { type YAxis, type YAxisProps } from "@/vis/lineplot/aether/YAxis";
import { render } from "@/vis/render";

export const xAxisStateZ = axis.axisStateZ
  .extend({
    location: location.y.optional().default("bottom"),
    bounds: bounds.bounds.optional(),
    autoBoundPadding: z.number().optional().default(0.01),
    size: z.number().optional().default(0),
    labelSize: z.number().optional().default(0),
  })
  .partial({
    color: true,
    font: true,
    gridColor: true,
  });

export interface XAxisProps extends Omit<YAxisProps, "xDataToDecimalScale"> {
  viewport: box.Box;
}

interface InternalState {
  ctx: render.Context;
  core: axis.Axis;
  // In the case where we're in a hold, we want to keep a snapshot of the hold bounds
  // so that we can rerender the plot in the same position even if the data changes.
  boundSnapshot?: bounds.Bounds;
}

export class XAxis extends aether.Composite<typeof xAxisStateZ, InternalState, YAxis> {
  static readonly TYPE = CSS.BE("line-plot", "x-axis");
  schema = xAxisStateZ;

  afterUpdate(): void {
    this.internal.ctx = render.Context.use(this.ctx);
    const theme = theming.use(this.ctx);
    this.internal.core = new Canvas(this.internal.ctx, {
      color: theme.colors.gray.p1,
      font: fontString(theme, "small"),
      gridColor: theme.colors.gray.m2,
      ...this.state,
      size: this.state.size + this.state.labelSize,
    });
    render.Controller.requestRender(this.ctx, render.REASON_LAYOUT);
  }

  async render(props: XAxisProps): Promise<void> {
    const [dataToDecimal, err] = await this.dataToDecimalScale(
      props.viewport,
      props.hold,
    );
    await this.renderAxis(props, dataToDecimal.reverse());
    await this.renderYAxes(props, dataToDecimal);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
  }

  async findByXDecimal(
    props: Omit<XAxisProps, "canvases">,
    target: number,
  ): Promise<FindResult[]> {
    const [scale, err] = await this.dataToDecimalScale(props.viewport, props.hold);
    if (err != null) throw err;
    return await this.findByXValue(props, scale.reverse().pos(target));
  }

  async findByXValue(
    props: Omit<XAxisProps, "canvases">,
    target: number,
  ): Promise<FindResult[]> {
    const [xDataToDecimalScale, error] = await this.dataToDecimalScale(
      props.viewport,
      props.hold,
    );
    if (error != null) throw error;
    const p = { ...props, xDataToDecimalScale };
    const prom = this.children.map(async (el) => await el.findByXValue(p, target));
    return (await Promise.all(prom)).flat();
  }

  private async renderAxis(
    props: XAxisProps,
    decimalToDataScale: scale.Scale,
  ): Promise<void> {
    if (!props.canvases.includes("lower2d")) return;
    const { core } = this.internal;
    const { grid, container } = props;
    const position = calculateGridPosition(this.key, grid, container);
    const p = { ...props, position, decimalToDataScale };
    const { size } = core.render(p);
    if (!withinSizeThreshold(this.state.size, size))
      this.setState((p) => ({ ...p, size }));
  }

  private async renderYAxes(
    props: XAxisProps,
    xDataToDecimalScale: scale.Scale,
  ): Promise<void> {
    const p = { ...props, xDataToDecimalScale };
    await Promise.all(this.children.map(async (el) => await el.render(p)));
  }

  private async xBounds(hold: boolean): Promise<[bounds.Bounds, Error | null]> {
    if (hold && this.internal.boundSnapshot != null)
      return [this.internal.boundSnapshot, null];
    if (this.state.bounds != null && !bounds.isZero(this.state.bounds))
      return [this.state.bounds, null];
    try {
      const b = (
        await Promise.all(this.children.map(async (el) => await el.xBounds()))
      ).filter((b) => bounds.isFinite(b));
      const ab = autoBounds(b, this.state.autoBoundPadding, this.state.type);
      this.internal.boundSnapshot = ab;
      return [autoBounds(b, this.state.autoBoundPadding, this.state.type), null];
    } catch (err) {
      return [emptyBounds(this.state.type), err as Error];
    }
  }

  private async dataToDecimalScale(
    viewport: box.Box,
    hold: boolean,
  ): Promise<[scale.Scale, Error | null]> {
    const [bounds, error] = await this.xBounds(hold);
    return [
      scale.Scale.scale(bounds)
        .scale(1)
        .translate(-box.x(viewport))
        .magnify(1 / box.width(viewport)),
      error,
    ];
  }
}

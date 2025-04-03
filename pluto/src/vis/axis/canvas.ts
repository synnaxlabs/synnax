// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, dimensions, xy } from "@synnaxlabs/x";

import { dimensions as textDimensions } from "@/text/dimensions";
import { prettyParse } from "@/util/zod";
import {
  type Axis,
  type AxisProps,
  type AxisState,
  axisStateZ,
  type ParsedAxisState,
  type RenderResult,
} from "@/vis/axis/axis";
import { newTickFactory, type Tick, type TickFactory } from "@/vis/axis/ticks";
import { type render } from "@/vis/render";

const TICK_LINE_SIZE = 4;

class TickTextDimensions {
  private readonly numberDims: dimensions.Dimensions;
  private readonly negativeWidth: number;
  private readonly periodWidth: number;
  private readonly colonWidth: number;

  constructor(canvas: OffscreenCanvasRenderingContext2D, font: string) {
    this.numberDims = textDimensions("0", font, canvas);
    this.negativeWidth = textDimensions("-", font, canvas).width;
    this.periodWidth = textDimensions(".", font, canvas).width;
    this.colonWidth = textDimensions(":", font, canvas).width;
  }

  get(label: string): dimensions.Dimensions {
    const dimensions: dimensions.Dimensions = {
      width: 0,
      height: this.numberDims.height,
    };
    let count = label.length;
    if (label.includes(".")) {
      dimensions.width += this.periodWidth;
      count -= 1;
    }
    if (label.startsWith("-")) {
      dimensions.width += this.negativeWidth;
      count -= 1;
    }
    if (label.includes(":")) {
      dimensions.width += this.colonWidth;
      count -= 1;
    }
    dimensions.width += count * this.numberDims.width;
    return dimensions;
  }
}

export class Canvas implements Axis {
  renderCtx: render.Context;
  state: ParsedAxisState;
  tickFactory: TickFactory;
  dimensions: TickTextDimensions;

  constructor(ctx: render.Context, state: ParsedAxisState) {
    this.renderCtx = ctx;
    this.state = state;
    this.tickFactory = newTickFactory(this.state);
    this.dimensions = new TickTextDimensions(ctx.lower2d, state.font);
  }

  setState(state: AxisState): void {
    this.state = prettyParse(axisStateZ, state);
    this.tickFactory = newTickFactory(state);
    this.dimensions = new TickTextDimensions(this.renderCtx.lower2d, this.state.font);
  }

  render(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    canvas.font = this.state.font;
    canvas.fillStyle = this.state.color.hex;
    canvas.lineWidth = 1;

    switch (this.state.location) {
      case "left":
        return this.drawLeft(props);
      case "right":
        return this.drawRight(props);
      case "top":
        return this.drawTop(props);
      default:
        return this.drawBottom(props);
    }
  }

  drawBottom(ctx: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = ctx;
    const size = box.width(plottingRegion);
    const gridSize = box.height(plottingRegion);
    const p = ctx.position;
    const ticks = this.tickFactory.create({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "x", tick.position),
      xy.translate(p, { x: tick.position, y: -gridSize }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, xy.translate(p, "x", size));
    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y + TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y + 10 + d.height,
      );
    });
    // Add some extra padding to the bottom of the axis.
    return { size: maxTickDims.height + TICK_LINE_SIZE + 6 };
  }

  drawTop(ctx: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = ctx;
    const size = box.width(plottingRegion);
    const gridSize = box.height(plottingRegion);
    const p = xy.translate(ctx.position, "y", ctx.size);
    const ticks = this.tickFactory.create({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "x", tick.position),
      xy.translate(p, { x: tick.position, y: gridSize }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, xy.translate(p, "x", size));
    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y - TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y - 5 - d.height,
      );
    });

    return { size: maxTickDims.height + TICK_LINE_SIZE };
  }

  drawLeft(ctx: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = ctx;
    const size = box.height(plottingRegion);
    const gridSize = box.width(plottingRegion);
    const p = xy.translate(ctx.position, "x", ctx.size);
    const ticks = this.tickFactory.create({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "y", tick.position),
      xy.translate(p, { x: gridSize, y: tick.position }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, xy.translate(p, "y", size));
    const maxTickSize = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x - TICK_LINE_SIZE, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x - d.width - TICK_LINE_SIZE * 2,
        p.y + tick.position + d.height / 3,
      );
    });

    return { size: maxTickSize.width + TICK_LINE_SIZE * 2 };
  }

  drawRight(ctx: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = ctx;
    const size = box.height(plottingRegion);
    const gridSize = box.width(plottingRegion);
    const p = ctx.position;
    const ticks = this.tickFactory.create({ ...ctx, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "y", tick.position),
      xy.translate(p, { x: -gridSize, y: tick.position }),
    ]);
    canvas.strokeStyle = this.state.color.hex;
    this.drawLine(p, xy.translate(p, "y", size));
    const maxTickSize = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x, p.y + tick.position);
      canvas.lineTo(p.x + TICK_LINE_SIZE, p.y + tick.position);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + TICK_LINE_SIZE + 2,
        p.y + tick.position + d.height / 3,
      );
    });
    return { size: maxTickSize.width + TICK_LINE_SIZE * 2 };
  }

  private drawLine(start: xy.XY, end: xy.XY): void {
    const { lower2d: canvas } = this.renderCtx;
    canvas.beginPath();
    canvas.moveTo(...xy.couple(start));
    canvas.lineTo(...xy.couple(end));
    canvas.stroke();
  }

  private drawTicks(
    ticks: Tick[],
    f: (textDimensions: dimensions.Dimensions, tick: Tick) => void,
  ): dimensions.Dimensions {
    let maxDimensions = dimensions.ZERO;
    ticks.forEach((tick) => {
      const d = this.dimensions.get(tick.label);
      maxDimensions = dimensions.max([maxDimensions, d]);
      f(d, tick);
    });
    return maxDimensions;
  }

  private maybeDrawGrid(
    size: number,
    ticks: Tick[],
    f: (tick: Tick) => [xy.XY, xy.XY],
  ): void {
    const { showGrid, gridColor } = this.state;
    if (showGrid) {
      const startBound = bounds.construct(-1, 1);
      const endBound = bounds.construct(size - 1, size + 1);
      this.renderCtx.lower2d.strokeStyle = gridColor.hex;
      ticks
        .filter(
          ({ position }) =>
            !bounds.contains(startBound, position) &&
            !bounds.contains(endBound, position),
        )
        .forEach((tick) => this.drawLine(...f(tick)));
    }
  }
}

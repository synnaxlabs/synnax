// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, color, dimensions, xy } from "@synnaxlabs/x";

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
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { type render } from "@/vis/render";

const TICK_LINE_SIZE = 5;
const TICK_PADDING = 6;
const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

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

  render(args: AxisProps): RenderResult {
    switch (this.state.location) {
      case "left":
        return this.drawLeft(args);
      case "right":
        return this.drawRight(args);
      case "top":
        return this.drawTop(args);
      default:
        return this.drawBottom(args);
    }
  }

  drawBottom(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = props;
    const size = box.width(plottingRegion);
    const gridSize = box.height(plottingRegion);
    const pos = props.position;
    const ticks = this.tickFactory.create({ ...props, size });
    canvas.beginPath();
    canvas.strokeStyle = color.hex(this.state.color);
    canvas.fillStyle = color.hex(this.state.color);
    this.drawLine(pos, xy.translate(pos, "x", size));
    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      this.drawLine(
        xy.translateX(pos, tick.position),
        xy.translate(pos, { x: tick.position, y: TICK_LINE_SIZE }),
      );
      canvas.fillText(
        tick.label,
        pos.x + tick.position - d.width / 2,
        pos.y + TICK_LINE_SIZE + d.height + TICK_PADDING,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });
    canvas.stroke();
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(pos, "x", tick.position),
      xy.translate(pos, { x: tick.position, y: -gridSize }),
    ]);
    // Add some extra padding to the bottom of the axis.
    return { size: maxTickDims.height + TICK_LINE_SIZE + TICK_PADDING };
  }

  drawTop(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = props;
    const size = box.width(plottingRegion);
    const gridSize = box.height(plottingRegion);
    const p = xy.translate(props.position, "y", props.size);
    const ticks = this.tickFactory.create({ ...props, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "x", tick.position),
      xy.translate(p, { x: tick.position, y: gridSize }),
    ]);
    canvas.strokeStyle = color.hex(this.state.color);
    canvas.fillStyle = color.hex(this.state.color);
    this.drawLine(p, xy.translate(p, "x", size));
    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(p.x + tick.position, p.y);
      canvas.lineTo(p.x + tick.position, p.y - TICK_LINE_SIZE);
      canvas.stroke();
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y - TICK_LINE_SIZE - d.height - TICK_PADDING,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });

    return { size: maxTickDims.height + TICK_LINE_SIZE };
  }

  drawLeft(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = props;
    const size = box.height(plottingRegion);
    const gridSize = box.width(plottingRegion);
    const p = xy.translate(props.position, "x", props.size);
    const ticks = this.tickFactory.create({ ...props, size });
    canvas.beginPath();
    canvas.strokeStyle = color.hex(this.state.color);
    canvas.fillStyle = color.hex(this.state.color);
    this.drawLine(p, xy.translate(p, "y", size));
    const maxTickSize = this.drawTicks(ticks, (d, tick) => {
      this.drawLine(
        xy.translateY(p, tick.position),
        xy.translate(p, { x: -TICK_LINE_SIZE, y: tick.position }),
      );
      canvas.fillText(
        tick.label,
        p.x - d.width - TICK_LINE_SIZE * 2,
        p.y + tick.position + d.height / 2,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });
    canvas.stroke();
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "y", tick.position),
      xy.translate(p, { x: gridSize, y: tick.position }),
    ]);
    return { size: maxTickSize.width + TICK_LINE_SIZE * 2 };
  }

  drawRight(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = props;
    const size = box.height(plottingRegion);
    const gridSize = box.width(plottingRegion);
    const pos = props.position;
    const ticks = this.tickFactory.create({ ...props, size });
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(pos, "y", tick.position),
      xy.translate(pos, { x: -gridSize, y: tick.position }),
    ]);
    canvas.strokeStyle = color.hex(this.state.color);
    canvas.fillStyle = color.hex(this.state.color);
    this.drawLine(pos, xy.translate(pos, "y", size));
    const maxTickSize = this.drawTicks(ticks, (d, tick) => {
      canvas.moveTo(pos.x, pos.y + tick.position);
      canvas.lineTo(pos.x + TICK_LINE_SIZE, pos.y + tick.position);
      canvas.fillText(
        tick.label,
        pos.x + TICK_LINE_SIZE + TICK_PADDING,
        pos.y + tick.position + d.height / 2,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });
    canvas.stroke();
    return { size: maxTickSize.width + TICK_LINE_SIZE + TICK_PADDING };
  }

  private drawLine(start: xy.XY, end: xy.XY): void {
    const { lower2d: canvas } = this.renderCtx;
    canvas.moveTo(...xy.couple(start));
    canvas.lineTo(...xy.couple(end));
  }

  private drawTicks(
    ticks: Tick[],
    f: (textDimensions: dimensions.Dimensions, tick: Tick) => void,
  ): dimensions.Dimensions {
    let maxDimensions = dimensions.ZERO;
    ticks.forEach((tick) => {
      const d = this.renderCtx.lower2d.textDimensions(tick.label, FILL_TEXT_OPTIONS);
      maxDimensions = dimensions.max([maxDimensions, d]);
      f(d, tick);
    });
    return maxDimensions;
  }

  private static START_BOUND = bounds.construct(-1, 1);

  private maybeDrawGrid(
    size: number,
    ticks: Tick[],
    f: (tick: Tick) => [xy.XY, xy.XY],
  ): void {
    const { showGrid, gridColor } = this.state;
    if (showGrid) {
      const startBound = Canvas.START_BOUND;
      const endBound = bounds.construct(size - 1, size + 1);
      const { lower2d: canvas } = this.renderCtx;
      canvas.beginPath();
      canvas.strokeStyle = color.hex(gridColor);
      ticks.forEach((tick) => {
        if (
          bounds.contains(startBound, tick.position) ||
          bounds.contains(endBound, tick.position)
        )
          return;
        this.drawLine(...f(tick));
      });
      canvas.stroke();
    }
  }
}

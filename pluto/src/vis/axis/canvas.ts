// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, color, dimensions, type location, xy } from "@synnaxlabs/x";

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

const AXES: Record<
  location.Outer,
  (ctx: render.Context, state: ParsedAxisState) => Axis
> = {
  bottom: (ctx, state) => new Bottom(ctx, state),
  top: (ctx, state) => new Top(ctx, state),
  left: (ctx, state) => new LeftCanvas(ctx, state),
  right: (ctx, state) => new Right(ctx, state),
};

export const newCanvas = (
  loc: location.Outer,
  ctx: render.Context,
  state: ParsedAxisState,
): Axis => AXES[loc](ctx, state);

export class Base {
  renderCtx: render.Context;
  state: ParsedAxisState;
  tickFactory: TickFactory;

  constructor(ctx: render.Context, state: ParsedAxisState) {
    this.renderCtx = ctx;
    this.state = state;
    this.tickFactory = newTickFactory(this.state);
  }

  setState(state: AxisState): void {
    this.state = prettyParse(axisStateZ, state);
    this.tickFactory = newTickFactory(state);
  }

  protected drawLine(start: xy.XY, end: xy.XY): void {
    const { lower2d: canvas } = this.renderCtx;
    canvas.moveTo(...xy.couple(start));
    canvas.lineTo(...xy.couple(end));
  }

  protected drawTicks(
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

  protected maybeDrawGrid(
    size: number,
    ticks: Tick[],
    f: (tick: Tick) => [xy.XY, xy.XY],
  ): void {
    const { showGrid, gridColor } = this.state;
    if (showGrid) {
      const startBound = Base.START_BOUND;
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

  protected setColor(colorValue: color.Color): void {
    const hexColor = color.hex(colorValue);
    this.renderCtx.lower2d.strokeStyle = hexColor;
    this.renderCtx.lower2d.fillStyle = hexColor;
    this.renderCtx.lower2d.font = this.state.font;
  }
}

export class Bottom extends Base implements Axis {
  render(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion, position: pos } = props;
    const { width: size, height: gridSize } = box.dims(plottingRegion);

    const ticks = this.tickFactory.create({ ...props, size });

    canvas.beginPath();
    this.setColor(this.state.color);
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
}

export class Top extends Base implements Axis {
  render(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion } = props;
    const { width: size, height: gridSize } = box.dims(plottingRegion);

    const p = xy.translate(props.position, "y", props.size);
    const ticks = this.tickFactory.create({ ...props, size });

    canvas.beginPath();
    this.setColor(this.state.color);
    this.drawLine(p, xy.translate(p, "x", size));

    const maxTickDims = this.drawTicks(ticks, (d, tick) => {
      this.drawLine(
        xy.translateX(p, tick.position),
        xy.translate(p, { x: tick.position, y: -TICK_LINE_SIZE }),
      );
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y - TICK_LINE_SIZE - d.height - TICK_PADDING,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });
    canvas.stroke();
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "x", tick.position),
      xy.translate(p, { x: tick.position, y: gridSize }),
    ]);

    return { size: maxTickDims.height + TICK_LINE_SIZE };
  }
}

class LeftCanvas extends Base implements Axis {
  render(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion, position: pos } = props;

    const { height: size, width: gridSize } = box.dims(plottingRegion);
    const p = xy.translate(pos, "x", props.size);

    const ticks = this.tickFactory.create({ ...props, size });

    canvas.beginPath();
    this.setColor(this.state.color);
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
}

export class Right extends Base implements Axis {
  render(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion, position: pos } = props;
    const { height: size, width: gridSize } = box.dims(plottingRegion);

    const ticks = this.tickFactory.create({ ...props, size });

    canvas.beginPath();
    this.setColor(this.state.color);
    this.drawLine(pos, xy.translateY(pos, size));

    const maxTickSize = this.drawTicks(ticks, (d, tick) => {
      this.drawLine(
        xy.translateY(pos, tick.position),
        xy.translate(pos, { x: TICK_LINE_SIZE, y: tick.position }),
      );
      canvas.fillText(
        tick.label,
        pos.x + TICK_LINE_SIZE + TICK_PADDING,
        pos.y + tick.position + d.height / 2,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });

    canvas.stroke();

    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translateY(pos, tick.position),
      xy.translate(pos, { x: -gridSize, y: tick.position }),
    ]);

    return { size: maxTickSize.width + TICK_LINE_SIZE + TICK_PADDING };
  }
}

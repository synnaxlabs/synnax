// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, color, dimensions, type location, xy, zod } from "@synnaxlabs/x";

import {
  type Axis,
  type AxisProps,
  type AxisState,
  axisStateZ,
  BREAK_WIDTH,
  type ParsedAxisState,
  type RenderResult,
} from "@/vis/axis/axis";
import {
  newTickFactory,
  type Tick,
  type TickFactory,
  type TickSegment,
} from "@/vis/axis/ticks";
import type { FillTextOptions } from "@/vis/draw2d/canvas";
import type { render } from "@/vis/render";

const TICK_LINE_SIZE = 5;
const TICK_PADDING = 6;
// The space in pixels that a label keeps from a seam or an anchor.
const LABEL_GAP = 8;
// The offset in pixels of each stroke of a break's mark from its seam, and the half
// size of a stroke.
const BREAK_STROKES = [-2, 2];
const BREAK_STROKE_SIZE = 4;
const OPEN_BREAK_ALPHA = 0.35;
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
    this.state = zod.parse(axisStateZ, state, { label: "axis" });
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
    const { lower2d: canvas } = this.renderCtx;
    canvas.textAlign = "left";
    canvas.textBaseline = "alphabetic";
    let maxDimensions = dimensions.ZERO;
    ticks.forEach((tick) => {
      const d = canvas.textDimensions(tick.label, FILL_TEXT_OPTIONS);
      maxDimensions = dimensions.max([maxDimensions, d]);
      f(d, tick);
    });
    return maxDimensions;
  }

  // drawBreakAxis draws the axis line from start along x with a gap at each seam, and a
  // mark of two strokes in each gap.
  protected drawBreakAxis(start: xy.XY, size: number, seams: number[]): void {
    let from = 0;
    [...seams, size + BREAK_WIDTH / 2].forEach((seam) => {
      const to = seam - BREAK_WIDTH / 2;
      if (to > from)
        this.drawLine(xy.translateX(start, from), xy.translateX(start, to));
      from = seam + BREAK_WIDTH / 2;
    });
    seams.forEach((seam) => this.drawBreakMark(start, seam));
  }

  private drawBreakMark(start: xy.XY, position: number): void {
    BREAK_STROKES.forEach((offset) => {
      const x = start.x + position + offset;
      this.drawLine(
        { x: x - BREAK_STROKE_SIZE / 2, y: start.y + BREAK_STROKE_SIZE },
        { x: x + BREAK_STROKE_SIZE / 2, y: start.y - BREAK_STROKE_SIZE },
      );
    });
  }

  // drawOpenBreaks strokes a faint mark at each decimal position of an open break.
  protected drawOpenBreaks(start: xy.XY, size: number, positions: number[] = []): void {
    const shown = positions.map((p) => p * size).filter((p) => p > 0 && p < size);
    if (shown.length === 0) return;
    const { lower2d: canvas } = this.renderCtx;
    canvas.beginPath();
    canvas.globalAlpha = OPEN_BREAK_ALPHA;
    shown.forEach((p) => this.drawBreakMark(start, p));
    canvas.stroke();
    canvas.globalAlpha = 1;
  }

  // labels returns the label that each tick draws. With no seams every tick draws its
  // label. Else the first tick of each segment whose anchor label is clear of the seams
  // and of the anchors before it draws its anchor label. Any other tick after the
  // anchor of its segment draws its label when it is clear of the seams and anchors.
  protected labels(ticks: Tick[], seams: number[]): Map<Tick, string> {
    if (seams.length === 0) return new Map(ticks.map((t) => [t, t.label]));
    const { lower2d: canvas } = this.renderCtx;
    const extent = (position: number, label: string): bounds.Bounds => {
      const half = canvas.textDimensions(label, FILL_TEXT_OPTIONS).width / 2;
      return { lower: position - half, upper: position + half };
    };
    const taken = seams.map((s) => ({
      lower: s - BREAK_WIDTH / 2,
      upper: s + BREAK_WIDTH / 2,
    }));
    const clear = (e: bounds.Bounds): boolean =>
      taken.every(
        (o) => e.upper + LABEL_GAP <= o.lower || e.lower >= o.upper + LABEL_GAP,
      );
    const labels = new Map<Tick, string>();
    // The position of the anchor of each segment.
    const anchors = new Map<number, number>();
    ticks.forEach((tick) => {
      const { segment, anchorLabel, position } = tick;
      if (segment == null || anchorLabel == null || anchors.has(segment)) return;
      const e = extent(position, anchorLabel);
      if (!clear(e)) return;
      taken.push(e);
      anchors.set(segment, position);
      labels.set(tick, anchorLabel);
    });
    ticks.forEach((tick) => {
      const anchor = tick.segment == null ? undefined : anchors.get(tick.segment);
      if (labels.has(tick) || (anchor != null && tick.position < anchor)) return;
      if (clear(extent(tick.position, tick.label))) labels.set(tick, tick.label);
    });
    return labels;
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

// seamPositions returns the position in pixels of each seam between two segments that
// the axis shows.
const seamPositions = (segments: TickSegment[] | undefined, size: number): number[] =>
  (segments ?? [])
    .slice(1)
    .map(({ position }) => position.lower * size)
    .filter((p) => p > 0 && p < size);

// nearSeam is true when position is inside the mark of a seam.
const nearSeam = (seams: number[], position: number): boolean =>
  seams.some((s) => Math.abs(s - position) < BREAK_WIDTH / 2);

export class Bottom extends Base implements Axis {
  render(props: AxisProps): RenderResult {
    const { lower2d: canvas } = this.renderCtx;
    const { plot: plottingRegion, position: pos } = props;
    const { width: size, height: gridSize } = box.dims(plottingRegion);

    const seams = seamPositions(props.segments, size);
    const ticks = this.tickFactory
      .create({ ...props, size })
      .filter((tick) => !nearSeam(seams, tick.position));
    const labels = this.labels(ticks, seams);
    const shown = ticks.map((t) => ({ ...t, label: labels.get(t) ?? "" }));

    canvas.beginPath();
    this.setColor(this.state.color);
    this.drawBreakAxis(pos, size, seams);

    const maxTickDims = this.drawTicks(shown, (d, tick) => {
      this.drawLine(
        xy.translateX(pos, tick.position),
        xy.translate(pos, { x: tick.position, y: TICK_LINE_SIZE }),
      );
      if (tick.label === "") return;
      canvas.fillText(
        tick.label,
        pos.x + tick.position - d.width / 2,
        pos.y + TICK_LINE_SIZE + d.height + TICK_PADDING,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });
    canvas.stroke();
    this.drawOpenBreaks(pos, size, props.openBreaks);
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
    const seams = seamPositions(props.segments, size);
    const ticks = this.tickFactory
      .create({ ...props, size })
      .filter((tick) => !nearSeam(seams, tick.position));
    const labels = this.labels(ticks, seams);
    const shown = ticks.map((t) => ({ ...t, label: labels.get(t) ?? "" }));

    canvas.beginPath();
    this.setColor(this.state.color);
    this.drawBreakAxis(p, size, seams);

    const maxTickDims = this.drawTicks(shown, (d, tick) => {
      this.drawLine(
        xy.translateX(p, tick.position),
        xy.translate(p, { x: tick.position, y: -TICK_LINE_SIZE }),
      );
      if (tick.label === "") return;
      canvas.fillText(
        tick.label,
        p.x + tick.position - d.width / 2,
        p.y - TICK_LINE_SIZE - TICK_PADDING,
        undefined,
        FILL_TEXT_OPTIONS,
      );
    });
    canvas.stroke();
    this.drawOpenBreaks(p, size, props.openBreaks);
    this.maybeDrawGrid(size, ticks, (tick) => [
      xy.translate(p, "x", tick.position),
      xy.translate(p, { x: tick.position, y: gridSize }),
    ]);

    return { size: maxTickDims.height + TICK_LINE_SIZE + TICK_PADDING };
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

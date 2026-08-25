// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  border,
  bounds,
  box,
  color,
  type dimensions,
  direction,
  location,
  notation,
  scale,
  text,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/base/fontString";
import { newTickFactory, type Tick, type TickFactory } from "@/vis/axis/ticks";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { Draw2D } from "@/vis/draw2d";
import { domRadii } from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";
import { staleness } from "@/vis/staleness/aether";

export const scaleStateZ = staleness.configZ.extend({
  box: box.box,
  telem: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
  bounds: bounds.boundsZ().default(bounds.construct(0, 100)),
  color: color.colorZ.default(color.ZERO),
  // Colors the bar outline, the spine, and the tick marks. Zero uses the theme.
  axisColor: color.colorZ.default(color.ZERO),
  // Colors the tick labels. Zero uses the theme.
  textColor: color.colorZ.default(color.ZERO),
  direction: direction.directionZ.default("y"),
  // Fills the bar up to the value. Independent of the caret, so a symbol can show
  // either, both, or neither.
  showFill: z.boolean().default(true),
  // Draws a caret at the value against the scale, with a readout box beside it.
  showCaret: z.boolean().default(true),
  showScale: z.boolean().default(true),
  // The side of the bar on which tick marks and labels are drawn.
  side: location.outerZ.default("right"),
  // The direction the readout is offset from the value. A side along the bar has no edge
  // to clear, so the readout sits inside the bar beside the fill edge.
  caretSide: location.outerZ.default("right"),
  // Appended to the readout, after the value.
  units: z.string().default(""),
  // Colors the fill, the caret, and the readout once the source stops sending. Zero
  // uses the theme.
  stalenessColor: color.colorZ.default(color.ZERO),
  // Formatting of the readout value.
  notation: notation.notationZ.default("standard"),
  precision: z.number().default(2),
  // When true the scale is drawn outside box, beyond a small gap (used by symbols that
  // provide their own container, like a tank). When false a gutter is reserved inside
  // box alongside the bar for the scale.
  externalScale: z.boolean().default(false),
  borderRadius: z.number().default(2),
  // Per-corner pixel radii used to clip the fill to a rounded container (a tank).
  // When unset the fill uses the uniform borderRadius above.
  cornerRadii: border.radiusZ.optional(),
  level: text.levelZ.default("small"),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];
const GUTTER = 26;

/**
 * Pixels reserved alongside the bar for the ticks and their labels. A symbol sizing
 * itself around a scale takes this on top of the bar it wants.
 */
export const gutter = ({
  showScale,
  externalScale = false,
}: {
  showScale: boolean;
  externalScale?: boolean;
}): number => (showScale && !externalScale ? GUTTER : 0);

const TICK_LENGTH = 5;
// Minor ticks are denser, unlabeled markers drawn shorter than the labeled major ticks.
const MINOR_TICK_LENGTH = 3;
const MINOR_TICK_SPACING = 14;
const TICK_LABEL_GAP = 3;
const CARET_SIZE = 5;
// Space between the value and its units in the readout.
const UNITS_GAP = 4;
// Gap between the outer box edge and the external scale's spine (a tank), so the scale
// reads as its own axis separate from the container.
const EXTERNAL_GAP = 10;

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.NumberSource;
  staleness: staleness.Registration;
  stale: boolean;
  stopListening?: () => void;
  requestRender: render.Requestor | null;
  fillColor: color.Color;
  axisColor: color.Color;
  textColor: color.Color;
  tickFactory: TickFactory;
  minorTickFactory: TickFactory;
  ticks: Tick[];
  minorTicks: Tick[];
  tickLevel: text.Level;
}

export class Scale
  extends aether.Leaf<typeof scaleStateZ, InternalState>
  implements Element
{
  static readonly TYPE = "scale";
  static readonly z = scaleStateZ;
  schema = Scale.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    i.telem = telem.useSource(ctx, this.state.telem, i.telem);
    i.staleness = staleness.useInternalRegistration(
      ctx,
      i.staleness,
      this,
      i.telem,
      () => this.requestRender(),
    );
    i.stopListening?.();
    i.stopListening = i.telem.onChange(() => {
      i.staleness.received();
      this.requestRender();
    });
    i.requestRender = render.useOptionalRequestor(ctx);

    i.fillColor = color.isZero(this.state.color)
      ? i.theme.colors.visualization.palettes.default[0]
      : this.state.color;
    const { axisColor, textColor } = this.state;
    i.axisColor = color.isZero(axisColor) ? i.theme.colors.gray.l8 : axisColor;
    i.textColor = color.isZero(textColor) ? i.theme.colors.gray.l10 : textColor;
    i.tickLevel = this.state.level;

    const { lower, upper } = this.state.bounds;
    const decimalToDataScale = scale.Scale.scale<number>(0, 1).scale(lower, upper);
    i.tickFactory ??= newTickFactory({ type: "linear" });
    i.minorTickFactory ??= newTickFactory({
      type: "linear",
      tickSpacing: MINOR_TICK_SPACING,
    });
    i.ticks = i.tickFactory.create({ decimalToDataScale, size: this.alongLength });
    i.minorTicks = i.minorTickFactory.create({
      decimalToDataScale,
      size: this.alongLength,
    });

    this.requestRender();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.staleness?.cleanup();
    i.telem.cleanup?.();
    if (i.requestRender == null)
      i.render.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender("layout");
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender("layout");
    else void this.render({});
  }

  /** Length of the bar along the value axis in pixels. */
  private get alongLength(): number {
    const b = this.state.box;
    return this.state.direction === "y" ? box.height(b) : box.width(b);
  }

  /**
   * The side the ticks sit on. A side parallel to the bar has no room for them, so it
   * moves to the matching side of the other axis.
   */
  private get side(): location.Outer {
    const { side, direction: d } = this.state;
    return direction.construct(side) === d ? location.swapAxis(side) : side;
  }

  /** The axis the readout is offset on. */
  private get caretAxis(): direction.Direction {
    return direction.construct(this.state.caretSide);
  }

  /** True when the readout is offset along the bar rather than across it. */
  private get caretAlongBar(): boolean {
    return this.caretAxis === this.state.direction;
  }

  /** -1 when the fill grows up the screen, 1 when it grows right. */
  private get fillGrowth(): number {
    return this.state.direction === "y" ? -1 : 1;
  }

  /** Screen coordinate of the bar edge the fill grows from. */
  private zeroEdge(bar: box.Box): number {
    return this.state.direction === "y" ? box.bottom(bar) : box.left(bar);
  }

  /** Screen coordinate a pixel distance along the value axis from the zero edge. */
  private alongCoord(bar: box.Box, distance: number): number {
    return this.zeroEdge(bar) + this.fillGrowth * distance;
  }

  /** Maps a point in bar space (along the value axis, then across) to the screen. */
  private point(along: number, across: number): xy.XY {
    return this.state.direction === "y"
      ? xy.construct(across, along)
      : xy.construct(along, across);
  }

  /** True when the bar is outlined. A caller drawing its own container passes false. */
  private get showTrack(): boolean {
    return this.state.showFill && !this.state.externalScale;
  }

  /** -1 when the side points left or up of the bar, 1 when it points right or down. */
  private outwardOf(side: location.Outer): number {
    return side === "left" || side === "top" ? -1 : 1;
  }

  /** The region occupied by the bar (fill + track), excluding the scale gutter. */
  private get barRegion(): box.Box {
    const { box: b } = this.state;
    const { side } = this;
    // The bar stops at zero rather than going negative, which would mirror it onto the
    // gutter and read as a scale on the wrong side.
    const shrink = (size: number): number => Math.max(0, size - gutter(this.state));
    if (direction.construct(side) === "x") {
      const width = shrink(box.width(b));
      const left = box.left(b) + (side === "left" ? box.width(b) - width : 0);
      return box.construct(xy.construct(left, box.top(b)), {
        width,
        height: box.height(b),
      });
    }
    const height = shrink(box.height(b));
    const top = box.top(b) + (side === "top" ? box.height(b) - height : 0);
    return box.construct(xy.construct(box.left(b), top), {
      width: box.width(b),
      height,
    });
  }

  /**
   * Coordinate of the given side of the bar, where the ticks or the caret sit. In
   * external mode it is anchored to the outer box edge plus a gap so the scale reads as
   * its own axis clear of the container; otherwise it sits at the bar edge.
   */
  private edgeOf(bar: box.Box, side: location.Outer): number {
    const { externalScale } = this.state;
    const region = externalScale ? this.state.box : bar;
    const edge =
      direction.construct(side) === "x"
        ? side === "left"
          ? box.left(region)
          : box.right(region)
        : side === "top"
          ? box.top(region)
          : box.bottom(region);
    return externalScale ? edge + this.outwardOf(side) * EXTERNAL_GAP : edge;
  }

  render({ viewportScale = scale.XY.IDENTITY }): void {
    const { internal: i } = this;
    const draw = new Draw2D(i.render.upper2d.applyScale(viewportScale), i.theme);

    const bar = this.barRegion;
    const value = i.telem.value();
    const { lower, upper } = this.state.bounds;
    const range = upper - lower;
    const ratio =
      range === 0 || Number.isNaN(value)
        ? 0
        : (bounds.clamp(this.state.bounds, value) - lower) / range;

    if (this.state.showFill) {
      // The fill goes on the lower canvas so a symbol's own container paints over it.
      // Both edges then quantize independently without opening a seam, which is only
      // visible at a device pixel ratio of 1.
      const lowerDraw = new Draw2D(i.render.lower2d.applyScale(viewportScale), i.theme);
      this.renderFill(lowerDraw, bar, ratio);
    }
    if (this.showTrack)
      draw.border({
        region: bar,
        color: i.axisColor,
        radius: this.state.borderRadius,
        width: 1,
      });
    if (this.state.showScale) this.renderTicks(draw, bar);
    if (this.state.showCaret) this.renderValue(draw, bar, ratio, this.valueText(value));
  }

  private renderFill(draw: Draw2D, bar: box.Box, ratio: number): void {
    const { borderRadius } = this.state;
    const fillColor = this.valueColor;
    const zero = this.zeroEdge(bar);
    const value = this.valuePosition(bar, ratio);
    const start = Math.min(zero, value);
    const length = Math.abs(value - zero);
    const region =
      this.state.direction === "y"
        ? box.construct(xy.construct(box.left(bar), start), {
            width: box.width(bar),
            height: length,
          })
        : box.construct(xy.construct(start, box.top(bar)), {
            width: length,
            height: box.height(bar),
          });
    const { cornerRadii } = this.state;
    if (cornerRadii != null) {
      const ctx = draw.canvas;
      // Scissor to the filled rectangle so the liquid surface is flat, then fill the
      // full rounded bar so the tank's rounded corners are preserved at the fill edges.
      const restore = ctx.scissor(region);
      ctx.beginPath();
      ctx.roundRect(
        box.left(bar),
        box.top(bar),
        box.width(bar),
        box.height(bar),
        domRadii(cornerRadii),
      );
      ctx.fillStyle = color.hex(fillColor);
      ctx.fill();
      restore();
      return;
    }
    draw.container({
      region,
      bordered: false,
      rounded: borderRadius > 0,
      borderRadius,
      backgroundColor: fillColor,
    });
  }

  /**
   * Color of the parts that carry the value: the fill, the caret, and the readout. It
   * turns to the staleness color once the source stops sending.
   */
  private get valueColor(): color.Color {
    const { fillColor, stale, theme } = this.internal;
    return stale ? staleness.resolveColor(this.state.stalenessColor, theme) : fillColor;
  }

  /**
   * Color of the caret. A caret offset toward the origin sits on the fill, so it takes
   * the readout's background to stay legible against it.
   */
  private get caretColor(): color.Color {
    const { showFill, caretSide } = this.state;
    const onFill =
      showFill && this.caretAlongBar && this.outwardOf(caretSide) === -this.fillGrowth;
    return onFill ? this.internal.theme.colors.gray.l1 : this.valueColor;
  }

  /** The caret readout. Empty when the source has no value yet. */
  private valueText(value: number): string {
    if (Number.isNaN(value)) return "";
    const { precision } = this.state;
    return notation.stringifyNumber(value, precision, this.state.notation);
  }

  /**
   * Draws a caret at the value with an altimeter-style readout box beyond it. The caret
   * sits against the bar edge it is offset from, or on the bar's center line when it is
   * offset along the bar and so has no edge to sit against.
   */
  private renderValue(
    draw: Draw2D,
    bar: box.Box,
    ratio: number,
    valueText: string,
  ): void {
    const { x, y } = box.center(bar);
    const acrossCenter = this.state.direction === "y" ? x : y;
    const across = this.caretAlongBar
      ? acrossCenter
      : this.edgeOf(bar, this.state.caretSide);
    const tip = this.point(this.valuePosition(bar, ratio), across);
    this.drawCaret(draw, tip);
    this.renderValueBox(draw, bar, tip, valueText);
  }

  /**
   * Fills a triangle with its point at tip, opening away from it along the axis the
   * readout is offset on, so the caret always aims back at the value.
   */
  private drawCaret(draw: Draw2D, tip: xy.XY): void {
    const ctx = draw.canvas;
    const dir = this.outwardOf(this.state.caretSide);
    const horizontal = this.caretAxis === "x";
    const base = (horizontal ? tip.x : tip.y) + dir * CARET_SIZE;
    ctx.beginPath();
    ctx.moveTo(tip.x, tip.y);
    if (horizontal) {
      ctx.lineTo(base, tip.y - CARET_SIZE);
      ctx.lineTo(base, tip.y + CARET_SIZE);
    } else {
      ctx.lineTo(tip.x - CARET_SIZE, base);
      ctx.lineTo(tip.x + CARET_SIZE, base);
    }
    ctx.closePath();
    ctx.fillStyle = color.hex(this.caretColor);
    ctx.fill();
  }

  /**
   * Places the readout just beyond the caret and centered on it. A readout offset along
   * the bar is held inside it so a value near either end stays legible.
   */
  private valueBoxRegion(
    bar: box.Box,
    tip: xy.XY,
    { width, height }: dimensions.Dimensions,
  ): box.Box {
    const horizontal = this.caretAxis === "x";
    // Extents of the readout along the axis it is offset on, and across it.
    const [depth, breadth] = horizontal ? [width, height] : [height, width];
    const dir = this.outwardOf(this.state.caretSide);
    let start = (horizontal ? tip.x : tip.y) + dir * CARET_SIZE - (dir < 0 ? depth : 0);
    if (this.caretAlongBar) {
      const limit = horizontal
        ? bounds.construct(box.left(bar), box.right(bar) - depth)
        : bounds.construct(box.top(bar), box.bottom(bar) - depth);
      start = bounds.clamp(limit, start);
    }
    const center = (horizontal ? tip.y : tip.x) - breadth / 2;
    return box.construct(
      horizontal ? xy.construct(start, center) : xy.construct(center, start),
      { width, height },
    );
  }

  private renderValueBox(
    draw: Draw2D,
    bar: box.Box,
    tip: xy.XY,
    valueText: string,
  ): void {
    if (valueText.length === 0) return;
    const { units } = this.state;
    const { theme, tickLevel, stale } = this.internal;
    const valueColor = this.valueColor;
    const ctx = draw.canvas;
    ctx.font = fontString(theme, { level: tickLevel, code: true });
    const value = ctx.textDimensions(valueText, { useAtlas: true });
    // The units are drawn separately so they sit tighter than a monospace space.
    const unitsWidth =
      units.length === 0
        ? 0
        : ctx.textDimensions(units, { useAtlas: true }).width + UNITS_GAP;
    const region = this.valueBoxRegion(bar, tip, {
      width: value.width + unitsWidth + 8,
      height: value.height + 6,
    });
    draw.container({
      region,
      rounded: true,
      borderRadius: 2,
      borderColor: valueColor,
      borderWidth: 1,
      backgroundColor: theme.colors.gray.l1,
    });
    const center = xy.translateY(box.center(region), 1);
    const left = center.x - (value.width + unitsWidth) / 2;
    const text = (text: string, x: number): void =>
      draw.text({
        text,
        position: xy.construct(x, center.y),
        level: tickLevel,
        justify: "left",
        align: "middle",
        shade: 11,
        color: stale ? valueColor : undefined,
        code: true,
        useAtlas: true,
      });
    text(valueText, left);
    if (units.length > 0) text(units, left + value.width + UNITS_GAP);
  }

  /** Screen coordinate of the value along the value axis. */
  private valuePosition(bar: box.Box, ratio: number): number {
    const length = this.state.direction === "y" ? box.height(bar) : box.width(bar);
    return this.alongCoord(bar, length * ratio);
  }

  private renderTicks(draw: Draw2D, bar: box.Box): void {
    const { ticks, minorTicks, tickLevel, axisColor, textColor } = this.internal;
    const { direction: d } = this.state;
    const { side } = this;
    const dir = this.outwardOf(side);
    const edge = this.edgeOf(bar, side);
    const vertical = d === "y";
    const line = (start: xy.XY, end: xy.XY): void =>
      draw.line({ stroke: axisColor, lineWidth: 1, lineDash: 0, start, end });
    const tick = (position: number, length: number): void => {
      const along = this.alongCoord(bar, position);
      line(this.point(along, edge), this.point(along, edge + dir * length));
    };
    // Without a track, the ticks need a spine of their own to sit against.
    if (!this.showTrack)
      line(
        this.point(vertical ? box.top(bar) : box.left(bar), edge),
        this.point(vertical ? box.bottom(bar) : box.right(bar), edge),
      );
    minorTicks.forEach(({ position }) => tick(position, MINOR_TICK_LENGTH));
    ticks.forEach(({ position, label }) => {
      tick(position, TICK_LENGTH);
      draw.text({
        text: label,
        position: this.point(
          this.alongCoord(bar, position),
          edge + dir * (TICK_LENGTH + TICK_LABEL_GAP),
        ),
        level: tickLevel,
        color: textColor,
        align: vertical ? "middle" : dir > 0 ? "top" : "bottom",
        justify: vertical ? (dir > 0 ? "left" : "right") : "center",
        code: true,
        useAtlas: true,
      });
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Scale.TYPE]: Scale };

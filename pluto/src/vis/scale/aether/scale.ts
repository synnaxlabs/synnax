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

  /** True when the bar is outlined. A caller drawing its own container passes false. */
  private get showTrack(): boolean {
    return this.state.showFill && !this.state.externalScale;
  }

  /** -1 when ticks and the caret point left or up of the bar, 1 when right or down. */
  private get outward(): number {
    return this.state.side === "left" || this.state.side === "top" ? -1 : 1;
  }

  /** The region occupied by the bar (fill + track), excluding the scale gutter. */
  private get barRegion(): box.Box {
    const { box: b, side, showScale, externalScale } = this.state;
    const gutter = showScale && !externalScale ? GUTTER : 0;
    let left = box.left(b);
    let top = box.top(b);
    let width = box.width(b);
    let height = box.height(b);
    if (side === "right") width -= gutter;
    else if (side === "left") {
      left += gutter;
      width -= gutter;
    } else if (side === "bottom") height -= gutter;
    else {
      top += gutter;
      height -= gutter;
    }
    return box.construct(xy.construct(left, top), { width, height });
  }

  /**
   * Across-axis coordinate of the scale spine, where the ticks and the caret sit. In
   * external mode it is anchored to the outer box edge plus a gap so the scale reads as
   * its own axis clear of the container; otherwise it sits at the bar edge.
   */
  private scaleEdge(bar: box.Box): number {
    const { direction: d, side, externalScale } = this.state;
    const region = externalScale ? this.state.box : bar;
    const edge =
      d === "y"
        ? side === "left"
          ? box.left(region)
          : box.right(region)
        : side === "top"
          ? box.top(region)
          : box.bottom(region);
    return externalScale ? edge + this.outward * EXTERNAL_GAP : edge;
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
    if (this.state.showCaret) this.renderCaret(draw, bar, ratio, this.valueText(value));
  }

  private renderFill(draw: Draw2D, bar: box.Box, ratio: number): void {
    const { borderRadius } = this.state;
    const fillColor = this.valueColor;
    let region: box.Box;
    if (this.state.direction === "y") {
      const height = box.height(bar) * ratio;
      region = box.construct(xy.construct(box.left(bar), box.bottom(bar) - height), {
        width: box.width(bar),
        height,
      });
    } else
      region = box.construct(box.topLeft(bar), {
        width: box.width(bar) * ratio,
        height: box.height(bar),
      });
    const { cornerRadii } = this.state;
    if (cornerRadii != null) {
      const ctx = draw.canvas;
      // Scissor to the filled rectangle so the liquid surface is flat, then fill the
      // full rounded bar so the tank's rounded corners are preserved at the fill edges.
      const restore = ctx.scissor(region);
      ctx.beginPath();
      ctx.roundRect(box.left(bar), box.top(bar), box.width(bar), box.height(bar), [
        cornerRadii.topLeft,
        cornerRadii.topRight,
        cornerRadii.bottomRight,
        cornerRadii.bottomLeft,
      ]);
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

  /** The caret readout. Empty when the source has no value yet. */
  private valueText(value: number): string {
    if (Number.isNaN(value)) return "";
    const { precision } = this.state;
    return notation.stringifyNumber(value, precision, this.state.notation);
  }

  // Draws a caret at the value on the tick side of the spine with its point aimed at
  // the spine, and an altimeter-style readout box beyond it.
  private renderCaret(
    draw: Draw2D,
    bar: box.Box,
    ratio: number,
    valueText: string,
  ): void {
    const ctx = draw.canvas;
    const { outward: dir } = this;
    const edge = this.scaleEdge(bar);
    const pos = this.valuePosition(bar, ratio);
    ctx.beginPath();
    if (this.state.direction === "y") {
      ctx.moveTo(edge, pos);
      ctx.lineTo(edge + dir * CARET_SIZE, pos - CARET_SIZE);
      ctx.lineTo(edge + dir * CARET_SIZE, pos + CARET_SIZE);
    } else {
      ctx.moveTo(pos, edge);
      ctx.lineTo(pos - CARET_SIZE, edge + dir * CARET_SIZE);
      ctx.lineTo(pos + CARET_SIZE, edge + dir * CARET_SIZE);
    }
    ctx.closePath();
    ctx.fillStyle = color.hex(this.valueColor);
    ctx.fill();
    this.renderValueBox(draw, edge, pos, valueText);
  }

  private renderValueBox(
    draw: Draw2D,
    edge: number,
    pos: number,
    valueText: string,
  ): void {
    if (valueText.length === 0) return;
    const { units, direction: d } = this.state;
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
    const width = value.width + unitsWidth + 8;
    const height = value.height + 6;
    const { outward: dir } = this;
    const inner = edge + dir * CARET_SIZE;
    const region =
      d === "y"
        ? box.construct(
            xy.construct(dir < 0 ? inner - width : inner, pos - height / 2),
            { width, height },
          )
        : box.construct(
            xy.construct(pos - width / 2, dir < 0 ? inner - height : inner),
            { width, height },
          );
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
    if (this.state.direction === "y") return box.bottom(bar) - box.height(bar) * ratio;
    return box.left(bar) + box.width(bar) * ratio;
  }

  private renderTicks(draw: Draw2D, bar: box.Box): void {
    const { ticks, minorTicks, tickLevel, axisColor, textColor } = this.internal;
    const { direction: d } = this.state;
    const { outward: dir } = this;
    const edge = this.scaleEdge(bar);
    const line = (start: xy.XY, end: xy.XY): void =>
      draw.line({ stroke: axisColor, lineWidth: 1, lineDash: 0, start, end });
    // Without a track, the ticks need a spine of their own to sit against.
    const showSpine = !this.showTrack;
    if (d === "y") {
      if (showSpine)
        line(xy.construct(edge, box.top(bar)), xy.construct(edge, box.bottom(bar)));
      minorTicks.forEach(({ position }) => {
        const y = box.bottom(bar) - position;
        line(xy.construct(edge, y), xy.construct(edge + dir * MINOR_TICK_LENGTH, y));
      });
      ticks.forEach(({ position, label }) => {
        const y = box.bottom(bar) - position;
        line(xy.construct(edge, y), xy.construct(edge + dir * TICK_LENGTH, y));
        draw.text({
          text: label,
          position: xy.construct(edge + dir * (TICK_LENGTH + TICK_LABEL_GAP), y),
          level: tickLevel,
          color: textColor,
          align: "middle",
          justify: dir > 0 ? "left" : "right",
          code: true,
          useAtlas: true,
        });
      });
    } else {
      if (showSpine)
        line(xy.construct(box.left(bar), edge), xy.construct(box.right(bar), edge));
      minorTicks.forEach(({ position }) => {
        const x = box.left(bar) + position;
        line(xy.construct(x, edge), xy.construct(x, edge + dir * MINOR_TICK_LENGTH));
      });
      ticks.forEach(({ position, label }) => {
        const x = box.left(bar) + position;
        line(xy.construct(x, edge), xy.construct(x, edge + dir * TICK_LENGTH));
        draw.text({
          text: label,
          position: xy.construct(x, edge + dir * (TICK_LENGTH + TICK_LABEL_GAP)),
          level: tickLevel,
          color: textColor,
          align: dir > 0 ? "top" : "bottom",
          justify: "center",
          code: true,
          useAtlas: true,
        });
      });
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Scale.TYPE]: Scale };

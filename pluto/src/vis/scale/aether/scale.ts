// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, color, direction, scale, text, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { fontString } from "@/theming/base/fontString";
import { newTickFactory, type Tick, type TickFactory } from "@/vis/axis/ticks";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

/** The way the live value is indicated against the scale. */
export const styleZ = z.enum(["fill", "caret"]);
export type Style = z.infer<typeof styleZ>;

/** The side of the bar on which tick marks and labels are drawn. */
export const sideZ = z.enum(["left", "right", "top", "bottom"]);
export type Side = z.infer<typeof sideZ>;

/** Per-corner elliptical radii in pixels, mirroring CSS border-radius. */
export const cornerRadiiZ = z.object({
  topLeft: xy.xyZ,
  topRight: xy.xyZ,
  bottomLeft: xy.xyZ,
  bottomRight: xy.xyZ,
});
export type CornerRadii = z.infer<typeof cornerRadiiZ>;

export const scaleStateZ = z.object({
  box: box.box,
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  bounds: bounds.boundsZ().default(bounds.construct(0, 100)),
  color: color.colorZ.default(color.ZERO),
  direction: direction.directionZ.default("y"),
  style: styleZ.default("fill"),
  showScale: z.boolean().default(true),
  side: sideZ.default("right"),
  // When true the scale is drawn outside box, beyond a small gap (used by symbols that
  // provide their own container, like a tank). When false a gutter is reserved inside
  // box alongside the bar for the scale.
  externalScale: z.boolean().default(false),
  // When true a rounded outline is drawn around the bar. Symbols that already render
  // their own container (a tank) pass false.
  showTrack: z.boolean().default(true),
  borderRadius: z.number().default(2),
  // Inset of the fill region within box, used to keep the fill inside a stroked wall.
  inset: z.number().default(0),
  // Per-corner radii used to clip the fill to a rounded container (a tank). When unset
  // the fill uses the uniform borderRadius above.
  cornerRadii: cornerRadiiZ.optional(),
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
// Gap between the outer box edge and the external scale's spine (a tank), so the scale
// reads as its own axis separate from the container.
const EXTERNAL_GAP = 10;

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.StringSource;
  stopListening?: () => void;
  requestRender: render.Requestor | null;
  fillColor: color.Color;
  trackColor: color.Color;
  tickColor: color.Color;
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
    i.stopListening?.();
    i.stopListening = i.telem.onChange(() => this.requestRender());
    i.requestRender = render.useOptionalRequestor(ctx);

    i.fillColor = color.isZero(this.state.color)
      ? i.theme.colors.visualization.palettes.default[0]
      : this.state.color;
    i.trackColor = i.theme.colors.gray.l5;
    i.tickColor = i.theme.colors.gray.l8;
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
    const len = this.state.direction === "y" ? box.height(b) : box.width(b);
    return len - this.state.inset * 2;
  }

  /** The region occupied by the bar (fill + track), excluding the scale gutter. */
  private get barRegion(): box.Box {
    const { box: b, side, showScale, externalScale, inset } = this.state;
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
    return box.construct(xy.construct(left + inset, top + inset), {
      width: width - inset * 2,
      height: height - inset * 2,
    });
  }

  render({ viewportScale = scale.XY.IDENTITY }): void {
    const { internal: i } = this;
    const region = i.render.upper2d.applyScale(viewportScale);
    const draw = new Draw2D(region, i.theme);

    const bar = this.barRegion;
    const valueText = i.telem.value();
    const value = Number(valueText);
    const clamped = bounds.clamp(this.state.bounds, value);
    const { lower, upper } = this.state.bounds;
    const range = upper - lower;
    const ratio = range === 0 ? 0 : (clamped - lower) / range;

    if (this.state.showTrack && !this.state.externalScale)
      draw.border({
        region: bar,
        color: i.trackColor,
        radius: this.state.borderRadius,
        width: 1,
      });

    if (this.state.style === "fill") this.renderFill(draw, bar, ratio);
    else this.renderCaret(draw, bar, ratio);

    if (this.state.showScale) {
      this.renderTicks(draw, bar);
      if (this.state.externalScale) {
        this.renderScaleCaret(draw, bar, ratio);
        this.renderValueBox(draw, bar, ratio, valueText);
      }
    }
  }

  private renderFill(draw: Draw2D, bar: box.Box, ratio: number): void {
    const { fillColor } = this.internal;
    const { borderRadius } = this.state;
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

  private renderCaret(draw: Draw2D, bar: box.Box, ratio: number): void {
    const ctx = draw.canvas;
    const { side } = this.state;
    const pos = this.valuePosition(bar, ratio);
    ctx.beginPath();
    if (this.state.direction === "y") {
      const x = side === "left" ? box.left(bar) : box.right(bar);
      const dir = side === "left" ? -1 : 1;
      ctx.moveTo(x, pos);
      ctx.lineTo(x + dir * CARET_SIZE, pos - CARET_SIZE);
      ctx.lineTo(x + dir * CARET_SIZE, pos + CARET_SIZE);
    } else {
      const y = side === "top" ? box.top(bar) : box.bottom(bar);
      const dir = side === "top" ? -1 : 1;
      ctx.moveTo(pos, y);
      ctx.lineTo(pos - CARET_SIZE, y + dir * CARET_SIZE);
      ctx.lineTo(pos + CARET_SIZE, y + dir * CARET_SIZE);
    }
    ctx.closePath();
    ctx.fillStyle = color.hex(this.internal.fillColor);
    ctx.fill();
  }

  // Draws a caret on the external scale at the current value, sitting on the tick side of
  // the spine with its point aimed at the spine.
  private renderScaleCaret(draw: Draw2D, bar: box.Box, ratio: number): void {
    const { side, direction: d } = this.state;
    const outer = this.state.box;
    const ctx = draw.canvas;
    const pos = this.valuePosition(bar, ratio);
    ctx.beginPath();
    if (d === "y") {
      const dir = side === "left" ? -1 : 1;
      const edge =
        (side === "left" ? box.left(outer) : box.right(outer)) + dir * EXTERNAL_GAP;
      ctx.moveTo(edge, pos);
      ctx.lineTo(edge + dir * CARET_SIZE, pos - CARET_SIZE);
      ctx.lineTo(edge + dir * CARET_SIZE, pos + CARET_SIZE);
    } else {
      const dir = side === "top" ? -1 : 1;
      const edge =
        (side === "top" ? box.top(outer) : box.bottom(outer)) + dir * EXTERNAL_GAP;
      ctx.moveTo(pos, edge);
      ctx.lineTo(pos - CARET_SIZE, edge + dir * CARET_SIZE);
      ctx.lineTo(pos + CARET_SIZE, edge + dir * CARET_SIZE);
    }
    ctx.closePath();
    ctx.fillStyle = color.hex(this.internal.fillColor);
    ctx.fill();
  }

  // Draws a readout box with the current value next to the caret, on the outer side of
  // the scale (altimeter style).
  private renderValueBox(
    draw: Draw2D,
    bar: box.Box,
    ratio: number,
    valueText: string,
  ): void {
    if (valueText.length === 0) return;
    const { side, direction: d } = this.state;
    const { theme, tickLevel, fillColor } = this.internal;
    const outer = this.state.box;
    const pos = this.valuePosition(bar, ratio);
    const ctx = draw.canvas;
    ctx.font = fontString(theme, { level: tickLevel, code: true });
    const td = ctx.textDimensions(valueText, { useAtlas: true });
    const width = td.width + 8;
    const height = td.height + 4;
    const dir = side === "left" || side === "top" ? -1 : 1;
    let region: box.Box;
    if (d === "y") {
      const edge =
        (side === "left" ? box.left(outer) : box.right(outer)) + dir * EXTERNAL_GAP;
      const inner = edge + dir * CARET_SIZE;
      region = box.construct(
        xy.construct(dir < 0 ? inner - width : inner, pos - height / 2),
        { width, height },
      );
    } else {
      const edge =
        (side === "top" ? box.top(outer) : box.bottom(outer)) + dir * EXTERNAL_GAP;
      const inner = edge + dir * CARET_SIZE;
      region = box.construct(
        xy.construct(pos - width / 2, dir < 0 ? inner - height : inner),
        { width, height },
      );
    }
    draw.container({
      region,
      rounded: true,
      borderRadius: 2,
      borderColor: fillColor,
      borderWidth: 1,
      backgroundColor: theme.colors.gray.l1,
    });
    draw.text({
      text: valueText,
      position: xy.translateY(box.center(region), 1),
      level: tickLevel,
      justify: "center",
      align: "middle",
      shade: 11,
      code: true,
      useAtlas: true,
    });
  }

  /** Screen coordinate of the value along the value axis. */
  private valuePosition(bar: box.Box, ratio: number): number {
    if (this.state.direction === "y") return box.bottom(bar) - box.height(bar) * ratio;
    return box.left(bar) + box.width(bar) * ratio;
  }

  private renderTicks(draw: Draw2D, bar: box.Box): void {
    const { ticks, minorTicks, tickLevel, tickColor } = this.internal;
    const { side, externalScale, direction: d } = this.state;
    const outer = this.state.box;
    const line = (start: xy.XY, end: xy.XY): void =>
      draw.line({ stroke: tickColor, lineWidth: 1, lineDash: 0, start, end });
    // Ticks always point outward (away from the bar). In external mode the scale is
    // anchored to the outer box edge plus a gap so it sits clear of the container as its
    // own axis, joined by a spine; otherwise it sits at the bar edge in the gutter.
    if (d === "y") {
      const dir = side === "left" ? -1 : 1;
      const edge = externalScale
        ? (side === "left" ? box.left(outer) : box.right(outer)) + dir * EXTERNAL_GAP
        : side === "left"
          ? box.left(bar)
          : box.right(bar);
      if (externalScale)
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
          shade: 10,
          align: "middle",
          justify: dir > 0 ? "left" : "right",
          code: true,
          useAtlas: true,
        });
      });
    } else {
      const dir = side === "top" ? -1 : 1;
      const edge = externalScale
        ? (side === "top" ? box.top(outer) : box.bottom(outer)) + dir * EXTERNAL_GAP
        : side === "top"
          ? box.top(bar)
          : box.bottom(bar);
      if (externalScale)
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
          shade: 10,
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

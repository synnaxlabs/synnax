// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { border, box, color, notation, scale, spatial, text, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { noopColorSourceSpec } from "@/telem/aether/noop";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";
import { staleness } from "@/vis/staleness/aether";

const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

// Below this contrast against the background a color is illegible and gets swapped for
// a legible gray. Rough guard, tune later.
const MIN_LEGIBLE_CONTRAST = 1.1;

// How far left of the first digit the negative sign sits, as a multiple of the font
// height. The draw and the clamp that keeps the sign in the box must use the same one.
const SIGN_OFFSET = 0.6;

// Space the text keeps from each edge of the box, as a multiple of the font height.
const PADDING = 0.3;

// Fills the box in place of a value too wide for it, as spreadsheets do.
const OVERFLOW = "#";

const MAX_PRECISION = 20;

// Fewest decimal places that show the value exactly, capped at MAX_PRECISION.
const exactPrecision = (value: number, n: notation.Notation): number => {
  if (!isFinite(value)) return 0;
  for (let p = 0; p < MAX_PRECISION; p++) {
    const text = notation.stringifyNumber(value, p, n).replace("ᴇ", "e");
    if (Number(text) === value) return p;
  }
  return MAX_PRECISION;
};

// Text for the value that fits, or null when none does. An absent precision drops
// decimals until the value fits; an explicit one either fits or does not.
const fit = (
  value: number,
  precision: number | undefined,
  n: notation.Notation,
  fits: (text: string) => boolean,
): string | null => {
  const min = precision ?? 0;
  for (let p = precision ?? exactPrecision(value, n); p >= min; p--) {
    const text = notation.stringifyNumber(value, p, n);
    if (fits(text)) return text;
  }
  return null;
};

const valueState = staleness.configZ.extend({
  box: box.box,
  telem: telem.numberSourceSpecZ.default(telem.noopNumericSourceSpec),
  // precision is the decimal places shown. When absent, the value shows as many as fit.
  precision: z.number().optional(),
  notation: notation.notationZ.default("standard"),
  backgroundTelem: telem.colorSourceSpecZ.default(telem.noopColorSourceSpec),
  level: text.levelZ,
  color: color.colorZ.default(color.ZERO),
  stalenessColor: color.colorZ.optional(),
  location: spatial.xCenterLocationZ.default("left"),
  // borderRadius rounds the clip region, in px. Set it when the host has rounded
  // corners, so the background fill does not square them off.
  borderRadius: border.crudeRadiusZ.optional(),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

interface InternalState {
  theme: theme.Theme;
  renderCtx: render.Context;
  telem: telem.NumberSource;
  stopListening?: () => void;
  backgroundTelem: telem.ColorSource;
  stopListeningBackground?: () => void;
  requestRender: render.Requestor | null;
  fontString: string;
  staleness: staleness.Registration;
  // Staleness stays on the worker here, which draws the value itself.
  stale: boolean;
}

export class Value
  extends aether.Leaf<typeof valueState, InternalState>
  implements Element
{
  static readonly TYPE = "value";
  static readonly z = valueState;
  schema = Value.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
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
    i.fontString = theme.fontString(i.theme, { level: this.state.level, code: true });
    i.backgroundTelem = telem.useSource(
      ctx,
      this.state.backgroundTelem,
      i.backgroundTelem,
    );
    i.stopListeningBackground?.();
    i.stopListeningBackground = i.backgroundTelem.onChange(() => this.requestRender());
    i.requestRender = render.useOptionalRequestor(ctx);
    this.requestRender();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.stopListeningBackground?.();
    i.staleness?.cleanup();
    i.telem.cleanup?.();
    i.backgroundTelem.cleanup?.();
    if (i.requestRender == null)
      i.renderCtx.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender("layout");
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender("layout");
    else this.render({});
  }

  get box(): box.Box {
    return this.state.box;
  }

  private get fontHeight(): number {
    const { theme } = this.internal;
    return theme.typography[this.state.level].size * theme.sizes.base;
  }

  // Color the value draws in, given the color it draws on top of. Pass ZERO when no
  // background is filled, which leaves the value on the host's own surface.
  private getTextColor(background: color.Color): color.Color {
    const { theme } = this.internal;
    if (this.internal.stale)
      return staleness.resolveColor(this.state.stalenessColor, theme);

    // A redline paints any color under the value, so the legible fallback is whichever
    // end of the gray scale stands out against what is actually there.
    const { l0, l11 } = theme.colors.gray;
    const surface = color.isZero(background) ? l0 : background;
    const legible = color.pickByContrast(surface, l11, l0);
    // Honor an explicit color unless it's illegible against the surface.
    if (color.isZero(this.state.color)) return legible;
    if (color.contrast(surface, this.state.color) < MIN_LEGIBLE_CONTRAST)
      return legible;
    return this.state.color;
  }

  render({ viewportScale = scale.XY.IDENTITY }): void {
    const { renderCtx, telem, backgroundTelem, fontString, requestRender } =
      this.internal;
    const { location, box: b, precision, notation: n } = this.state;
    if (box.areaIsZero(b)) return;
    const bTopLeft = box.topLeft(b);
    const bWidth = box.width(b);
    const bHeight = box.height(b);
    const canvas = renderCtx.upper2d.applyScale(viewportScale);
    const raw = telem.value();
    canvas.font = fontString;
    canvas.textAlign = "left";
    canvas.textBaseline = "alphabetic";
    const fontHeight = this.fontHeight;
    let isNegative = raw < 0;

    if (requestRender == null) renderCtx.erase(box.construct(this.prevState.box));

    // The leftmost the text may start: enough room for the sign, and the inset a
    // left-located value already sits at.
    const inset = 6 + fontHeight * 0.75;
    const padding = fontHeight * PADDING;
    const start =
      location === "left"
        ? inset
        : padding + (isNegative ? fontHeight * SIGN_OFFSET : 0);
    const available = bWidth - start - padding;
    const measure = (text: string): number =>
      canvas.textDimensions(text, FILL_TEXT_OPTIONS).width;
    let value = isNaN(raw)
      ? ""
      : fit(Math.abs(raw), precision, n, (text) => measure(text) <= available);
    if (value == null) {
      isNegative = false;
      value = OVERFLOW.repeat(Math.max(1, Math.floor(available / measure(OVERFLOW))));
    }
    const dims = canvas.textDimensions(value, FILL_TEXT_OPTIONS);

    const labelOffset = {
      x:
        location === "left"
          ? inset
          : location === "center"
            ? bWidth / 2 - dims.width / 2
            : bWidth - dims.width - inset,
      y: bHeight / 2 + dims.height / 2,
    };
    // A value that fits only without its inset must still keep its sign in the box.
    labelOffset.x = Math.max(labelOffset.x, start);

    const labelPosition = xy.translate(bTopLeft, labelOffset);

    const background =
      this.state.backgroundTelem.type != noopColorSourceSpec.type
        ? backgroundTelem.value()
        : color.ZERO;

    const undoClip = canvas.scissor(b, xy.ZERO, this.state.borderRadius);
    try {
      if (!color.isZero(background)) {
        canvas.fillStyle = color.hex(background);
        canvas.fillRect(...xy.couple(bTopLeft), bWidth, bHeight);
      }

      canvas.fillStyle = color.hex(this.getTextColor(background));

      // If the value is negative, chop of the negative sign and draw it separately so
      // that the first digit always stays in the same position, regardless of the sign.
      if (isNegative)
        canvas.fillText(
          "-",
          ...xy.couple(xy.translateX(labelPosition, -fontHeight * SIGN_OFFSET)),
          undefined,
          FILL_TEXT_OPTIONS,
        );
      canvas.fillText(value, ...xy.couple(labelPosition), undefined, FILL_TEXT_OPTIONS);
    } finally {
      undoClip();
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Value.TYPE]: Value };

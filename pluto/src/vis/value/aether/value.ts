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
  box,
  color,
  type dimensions,
  location,
  scale,
  text,
  xy,
} from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { noopColorSourceSpec } from "@/telem/aether/noop";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import {
  type FillTextOptions,
  type SugaredOffscreenCanvasRenderingContext2D,
} from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";
import { staleness } from "@/vis/staleness/aether";

const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

// Below this contrast against the background a color is illegible and gets
// swapped for a legible gray. Rough guard, tune later.
const MIN_LEGIBLE_CONTRAST = 1.1;

// How far left of the first digit the negative sign sits, as a multiple of the font
// height. The draw and the clamp that keeps the sign in the box must use the same one.
const SIGN_OFFSET = 0.6;

const ELLIPSIS = "\u2026";

const valueState = staleness.configZ.extend({
  box: box.box,
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  backgroundTelem: telem.colorSourceSpecZ.default(telem.noopColorSourceSpec),
  level: text.levelZ.default("p"),
  color: color.colorZ.default(color.ZERO),
  stalenessColor: color.colorZ.default(color.ZERO),
  location: location.xy.default({ x: "left", y: "center" }),
  // borderRadius rounds the clip region, in px. Set it when the host has rounded
  // corners, so the background fill does not square them off.
  borderRadius: border.crudeRadiusZ.optional(),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

interface InternalState {
  theme: theming.Theme;
  renderCtx: render.Context;
  telem: telem.StringSource;
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
    i.fontString = theming.fontString(i.theme, { level: this.state.level, code: true });
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

  // Longest head of the value that fits in available, with an ellipsis standing in for
  // what was cut. Returns the value unchanged when it already fits, and the bare
  // ellipsis when not even one digit does, so a cut reading is never mistaken for a
  // whole one. The value font is monospaced, so one advance estimates the fit and a
  // single remeasure confirms it.
  private ellipsize(
    canvas: SugaredOffscreenCanvasRenderingContext2D,
    value: string,
    available: number,
    dims: dimensions.Dimensions,
  ): string {
    if (dims.width <= available || value.length < 2) return value;
    const advance = dims.width / value.length;
    let head = Math.max(0, Math.floor(available / advance) - 1);
    let fitted = `${value.slice(0, head)}${ELLIPSIS}`;
    while (
      head > 0 &&
      canvas.textDimensions(fitted, FILL_TEXT_OPTIONS).width > available
    ) {
      head -= 1;
      fitted = `${value.slice(0, head)}${ELLIPSIS}`;
    }
    return fitted;
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
    const { location, box: b } = this.state;
    if (box.areaIsZero(b)) return;
    const bTopLeft = box.topLeft(b);
    const bWidth = box.width(b);
    const bHeight = box.height(b);
    const canvas = renderCtx.upper2d.applyScale(viewportScale);
    let value = telem.value();
    canvas.font = fontString;
    canvas.textAlign = "left";
    canvas.textBaseline = "alphabetic";
    const fontHeight = this.fontHeight;
    const isNegative = value[0] == "-";
    if (isNegative) value = value.slice(1);

    if (requestRender == null) renderCtx.erase(box.construct(this.prevState.box));

    // The leftmost the text may start: enough room for the sign, and the inset a
    // left-located value already sits at.
    const inset = 6 + fontHeight * 0.75;
    const start =
      location.x === "left" ? inset : isNegative ? fontHeight * SIGN_OFFSET : 0;
    let dims = canvas.textDimensions(value, FILL_TEXT_OPTIONS);
    const fitted = this.ellipsize(canvas, value, bWidth - start, dims);
    if (fitted !== value) {
      value = fitted;
      dims = canvas.textDimensions(value, FILL_TEXT_OPTIONS);
    }

    const labelOffset = { ...xy.ZERO };
    if (location.x === "left") labelOffset.x = inset;
    else if (location.x === "center") labelOffset.x = bWidth / 2 - dims.width / 2;
    else labelOffset.x = bWidth - dims.width - inset;
    if (location.y === "center") labelOffset.y = bHeight / 2 + dims.height / 2;
    else if (location.y === "bottom") labelOffset.y = bHeight;
    // Overflow must never eat the sign or the leading digits: both change what the
    // value reads as, and neither loss is visible. Pinning the start keeps the cut at
    // the right end, where the ellipsis shows it.
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

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { border, box, color, location, scale, text, xy } from "@synnaxlabs/x";
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

// Below this contrast against the background a color is illegible and gets
// swapped for a legible gray. Rough guard, tune later.
const MIN_LEGIBLE_CONTRAST = 1.1;

// How far left of the first digit the negative sign sits, as a multiple of the font
// height. The draw and the clamp that keeps the sign in the box must use the same one.
const SIGN_OFFSET = 0.6;

const valueState = staleness.configZ.extend({
  box: box.box,
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  backgroundTelem: telem.colorSourceSpecZ.default(telem.noopColorSourceSpec),
  level: text.levelZ.default("p"),
  color: color.colorZ.default(color.ZERO),
  stalenessColor: color.colorZ.default(color.ZERO),
  location: location.xy.default({ x: "left", y: "center" }),
  valueBackgroundShift: xy.xyZ.default(xy.ZERO),
  valueBackgroundOverScan: xy.xyZ.default(xy.ZERO),
  // clip restricts canvas drawing to the configured box. Use when the
  // host can't grow to fit the natural text width (e.g. a table cell);
  // overflow gets truncated at the cell edge instead of bleeding past.
  clip: z.boolean().default(false),
  // borderRadius rounds the clip region, in px. Set it when the host has rounded
  // corners, so the background fill does not square them off.
  borderRadius: border.crudeRadiusZ.optional(),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

export interface ValueProps {
  scale?: scale.XY;
}

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

  private getTextColor(): color.Color {
    const { theme } = this.internal;
    if (this.internal.stale)
      return staleness.resolveColor(this.state.stalenessColor, theme);

    // gray.l0 is the background the text renders on; gray.l11 is the
    // high-contrast end of the scale, legible against it in both themes.
    const background = theme.colors.gray.l0;
    const legible = theme.colors.gray.l11;
    // Honor an explicit color unless it's illegible against the background.
    if (color.isZero(this.state.color)) return legible;
    if (color.contrast(background, this.state.color) < MIN_LEGIBLE_CONTRAST)
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

    const dims = canvas.textDimensions(value, FILL_TEXT_OPTIONS);
    if (requestRender == null) renderCtx.erase(box.construct(this.prevState.box));

    const labelOffset = { ...xy.ZERO };
    const inset = 6 + fontHeight * 0.75;
    if (location.x === "left") labelOffset.x = inset;
    else if (location.x === "center") labelOffset.x = bWidth / 2 - dims.width / 2;
    else labelOffset.x = bWidth - dims.width - inset;
    if (location.y === "center") labelOffset.y = bHeight / 2 + dims.height / 2;
    else if (location.y === "bottom") labelOffset.y = bHeight;
    // The sign hangs to the left of the first digit, so an overflowing value would clip
    // it and show a negative number as a positive one. Losing a digit on the right is
    // visible; losing the sign is not.
    if (isNegative) labelOffset.x = Math.max(labelOffset.x, fontHeight * SIGN_OFFSET);

    const labelPosition = xy.translate(bTopLeft, labelOffset);

    const undoClip = this.state.clip
      ? canvas.scissor(b, xy.ZERO, this.state.borderRadius)
      : null;
    try {
      if (this.state.backgroundTelem.type != noopColorSourceSpec.type) {
        const colorValue = backgroundTelem.value();
        const isZero = color.isZero(colorValue);
        if (!isZero) {
          canvas.fillStyle = color.hex(colorValue);
          canvas.fillRect(
            ...xy.couple(xy.translate(bTopLeft, this.state.valueBackgroundShift)),
            bWidth + this.state.valueBackgroundOverScan.x,
            bHeight + this.state.valueBackgroundOverScan.y,
          );
        }
      }

      const textColor = this.getTextColor();
      canvas.fillStyle = color.hex(textColor);

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
      undoClip?.();
    }
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};

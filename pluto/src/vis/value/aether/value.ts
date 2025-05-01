// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, notation } from "@synnaxlabs/x";
import { box, location, scale, xy } from "@synnaxlabs/x/spatial";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { noopColorSourceSpec } from "@/telem/aether/noop";
import { text } from "@/text/core";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
import { render } from "@/vis/render";

const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

const valueState = z.object({
  box: box.box,
  telem: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  backgroundTelem: telem.colorSourceSpecZ.optional().default(telem.noopColorSourceSpec),
  level: text.levelZ.optional().default("p"),
  color: color.colorZ.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  minWidth: z.number().optional().default(60),
  width: z.number().optional(),
  notation: notation.notationZ.optional().default("standard"),
  location: location.xy.optional().default({ x: "left", y: "center" }),
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
  requestRender: render.RequestF | null;
  textColor: color.Color;
  fontString: string;
}

export class Value
  extends aether.Leaf<typeof valueState, InternalState>
  implements Element
{
  static readonly TYPE = "value";
  static readonly z = valueState;
  schema = Value.z;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    if (color.isZero(this.state.color))
      this.internal.textColor = i.theme.colors.gray.l10;
    else i.textColor = this.state.color;
    i.telem = await telem.useSource(ctx, this.state.telem, i.telem);
    i.stopListening?.();
    i.stopListening = this.internal.telem.onChange(() => {
      this.requestRender();
    });
    i.fontString = theming.fontString(i.theme, {
      level: this.state.level,
      code: true,
    });
    i.backgroundTelem = await telem.useSource(
      ctx,
      this.state.backgroundTelem,
      i.backgroundTelem,
    );
    i.stopListeningBackground?.();
    i.stopListeningBackground = this.internal.backgroundTelem.onChange(() =>
      this.requestRender(),
    );
    this.internal.requestRender = render.Controller.useOptionalRequest(ctx);
    this.requestRender();
  }

  async afterDelete(): Promise<void> {
    const { internal: i } = this;
    i.stopListening?.();
    i.stopListeningBackground?.();
    await i.telem.cleanup?.();
    await i.backgroundTelem.cleanup?.();
    if (i.requestRender == null)
      i.renderCtx.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender(render.REASON_LAYOUT);
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender(render.REASON_LAYOUT);
    else void this.render({});
  }

  private get fontHeight(): number {
    const { theme } = this.internal;
    return theme.typography[this.state.level].size * theme.sizes.base;
  }

  private maybeUpdateWidth(width: number) {
    const { theme } = this.internal;
    const requiredWidth = width + theme.sizes.base + this.fontHeight;
    if (
      this.state.width == null ||
      this.state.width + this.fontHeight * 0.5 < requiredWidth ||
      (this.state.minWidth > requiredWidth && this.state.width !== this.state.minWidth)
    )
      this.setState((p) => ({ ...p, width: Math.max(requiredWidth, p.minWidth) }));
    else if (this.state.width - this.fontHeight > requiredWidth)
      this.setState((p) => ({ ...p, width: Math.max(requiredWidth, p.minWidth) }));
  }

  async render({ viewportScale = scale.XY.IDENTITY }): Promise<void> {
    const { renderCtx, telem, backgroundTelem, fontString, requestRender } =
      this.internal;
    const { location, box: b } = this.state;
    if (box.areaIsZero(b)) return;
    const bTopLeft = box.topLeft(b);
    const bWidth = box.width(b);
    const bHeight = box.height(b);
    const canvas = renderCtx.lower2d.applyScale(viewportScale);
    let value = await telem.value();
    canvas.font = fontString;
    const fontHeight = this.fontHeight;
    const isNegative = value[0] == "-";
    if (isNegative) value = value.slice(1);

    const { theme } = this.internal;
    const dims = canvas.textDimensions(value, FILL_TEXT_OPTIONS);
    const width = dims.width + theme.sizes.base;
    const height = dims.height;
    if (requestRender == null) renderCtx.erase(box.construct(this.prevState.box));

    this.maybeUpdateWidth(width);
    const labelOffset = { ...xy.ZERO };
    if (location.x === "left") labelOffset.x = 6 + fontHeight * 0.75;
    if (location.x === "center") labelOffset.x = bWidth / 2 - width / 2;
    if (location.y === "center") labelOffset.y = bHeight / 2 + height / 2;

    // canvas.fillStyle = "red";
    // canvas.rect(...xy.couple(box.topLeft(b)), box.width(b), box.height(b));
    // canvas.fill();

    const labelPosition = xy.translate(bTopLeft, labelOffset);

    let setDefaultFillStyle = true;
    if (this.state.backgroundTelem.type != noopColorSourceSpec.type) {
      const colorValue = await backgroundTelem.value();
      const isZero = color.isZero(colorValue);
      setDefaultFillStyle = isZero;
      if (!isZero) {
        const lower2d = renderCtx.lower2d.applyScale(viewportScale);
        lower2d.fillStyle = color.hex(colorValue);
        lower2d.rect(...xy.couple(bTopLeft), bWidth, bHeight);
        lower2d.fill();
        canvas.fillStyle = color.hex(
          color.pickByContrast(colorValue, theme.colors.gray.l0, theme.colors.gray.l11),
        );
      }
    }
    if (setDefaultFillStyle) canvas.fillStyle = color.hex(this.internal.textColor);

    // If the value is negative, chop of the negative sign and draw it separately
    // so that the first digit always stays in the same position, regardless of the sign.
    if (isNegative)
      canvas.fillText(
        "-",
        // 0.55 is a multiplier of the font height that seems to keep the sign in
        // the right place.
        ...xy.couple(xy.translateX(labelPosition, -fontHeight * 0.6)),
        undefined,
        FILL_TEXT_OPTIONS,
      );
    canvas.fillText(value, ...xy.couple(labelPosition), undefined, FILL_TEXT_OPTIONS);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Value.TYPE]: Value,
};

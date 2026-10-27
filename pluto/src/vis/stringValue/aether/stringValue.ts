// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { box, color, type destructor, scale, text, xy } from "@synnaxlabs/x";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { type diagram } from "@/vis/diagram/aether";
import { render } from "@/vis/render";
import { staleness } from "@/vis/staleness/aether";

export const stateZ = staleness.configZ.extend({
  // box is the region the text fills. The text starts at its left edge, centers on it
  // vertically, and fades out where it would overflow.
  box: box.box,
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  level: text.levelZ.default("p"),
  // color is the text color. Absent, the text takes the theme's text color.
  color: color.colorZ.optional(),
  stalenessColor: color.colorZ.optional(),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

// Lyra fades overflowing text over its last 2.25rem.
const FADE_REM = 2.25;

interface InternalState {
  theme: theme.Theme;
  renderCtx: render.Context;
  source: telem.StringSource;
  stopListening?: destructor.Destructor;
  requestRender: render.Requestor | null;
  fontString: string;
  staleness: staleness.Registration;
  stale: boolean;
}

/**
 * StringValue draws the latest value of a string source on the canvas, so arriving
 * samples never reach the main thread. It uses the canvas's own text rendering instead
 * of a glyph atlas, so the text can hold any character.
 */
export class StringValue
  extends aether.Leaf<typeof stateZ, InternalState>
  implements diagram.Element
{
  static readonly TYPE = "StringValue";
  static readonly z = stateZ;

  schema = StringValue.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.renderCtx = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    i.fontString = theme.fontString(i.theme, { level: this.state.level, code: true });
    i.source = telem.useSource(ctx, this.state.telem, i.source);
    i.staleness = staleness.useInternalRegistration(
      ctx,
      i.staleness,
      this,
      i.source,
      () => this.requestRender(),
    );
    i.stopListening?.();
    i.stopListening = i.source.onChange(() => {
      i.staleness.received();
      this.requestRender();
    });
    i.requestRender = render.useOptionalRequestor(ctx);
    this.requestRender();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.staleness?.cleanup();
    i.source.cleanup?.();
    if (i.requestRender == null)
      i.renderCtx.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender("layout");
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender("layout");
    else this.render({});
  }

  private get textColor(): color.Color {
    const { theme, stale } = this.internal;
    if (stale) return staleness.resolveColor(this.state.stalenessColor, theme);
    return this.state.color ?? theme.colors.text;
  }

  render({ viewportScale = scale.XY.IDENTITY }: { viewportScale?: scale.XY }): void {
    const { renderCtx, source, fontString, requestRender, theme } = this.internal;
    const { box: b } = this.state;
    if (requestRender == null) renderCtx.erase(box.construct(this.prevState.box));
    if (box.areaIsZero(b)) return;
    const canvas = renderCtx.upper2d.applyScale(viewportScale);
    canvas.font = fontString;
    canvas.textAlign = "left";
    canvas.textBaseline = "middle";
    canvas.fillStyle = color.hex(this.textColor);
    const value = source.value();
    if (value.length === 0) return;
    const position = xy.construct(box.left(b), box.center(b).y);
    const width = box.width(b);
    if (canvas.textDimensions(value).width <= width) {
      canvas.fillText(value, position.x, position.y);
      return;
    }
    // Color glyphs such as emoji ignore the fade, so the box clips them.
    const removeScissor = canvas.scissor(b);
    canvas.fillText(value, position.x, position.y, undefined, {
      fade: { end: width, width: FADE_REM * theme.sizes.base },
    });
    removeScissor();
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [StringValue.TYPE]: StringValue,
};

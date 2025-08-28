// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, location, scale, xy } from "@synnaxlabs/x/spatial";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { notationZ } from "@/notation/notation";
import { telem } from "@/telem/aether";
import { text } from "@/text/core";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

const gaugeState = z.object({
  box: box.box,
  telem: telem.stringSourceSpecZ.optional().default(telem.noopStringSourceSpec),
  backgroundTelem: telem.colorSourceSpecZ.optional().default(telem.noopColorSourceSpec),
  level: text.levelZ.optional().default("p"),
  color: color.Color.z.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  minWidth: z.number().optional().default(60),
  width: z.number().optional(),
  notation: notationZ.optional().default("standard"),
  location: location.xy.optional().default({ x: "left", y: "center" }),
  units: z.string().optional().default("RPM"),
  max: z.number().optional().default(100),
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

export interface GaugeProps {
  scale?: scale.XY;
}

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.StringSource;
  stopListening?: () => void;
  backgroundTelem: telem.ColorSource;
  stopListeningBackground?: () => void;
  requestRender: render.RequestF | null;
  textColor: color.Color;
}

export class Gauge
  extends aether.Leaf<typeof gaugeState, InternalState>
  implements Element
{
  static readonly TYPE = "gauge";
  static readonly z = gaugeState;
  schema = Gauge.z;

  afterUpdate(): void {
    const { internal: i } = this;
    i.render = render.Context.use(this.ctx);
    i.theme = theming.use(this.ctx);
    if (this.state.color.isZero) this.internal.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = telem.useSource(this.ctx, this.state.telem, i.telem);
    i.stopListening?.();
    i.stopListening = this.internal.telem.onChange(() => this.requestRender());
    i.backgroundTelem = telem.useSource(
      this.ctx,
      this.state.backgroundTelem,
      i.backgroundTelem,
    );
    i.stopListeningBackground?.();
    i.stopListeningBackground = this.internal.backgroundTelem.onChange(() =>
      this.requestRender(),
    );
    this.internal.requestRender = render.Controller.useOptionalRequest(this.ctx);
    this.requestRender();
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.stopListening?.();
    i.stopListeningBackground?.();
    await i.telem.cleanup?.();
    await i.backgroundTelem.cleanup?.();
    if (i.requestRender == null)
      i.render.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender(render.REASON_LAYOUT);
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender(render.REASON_LAYOUT);
    else void this.render({});
  }

  async render({ viewportScale = scale.XY.IDENTITY }): Promise<void> {
    const { render: renderCtx, theme } = this.internal;
    const upper2d = renderCtx.upper2d.applyScale(viewportScale);
    const draw2d = new Draw2D(upper2d, theme);
    const b = this.state.box;
    const baseRadius = box.width(b) / 2;
    const value = await this.internal.telem.value();
    draw2d.text({
      text: value,
      position: xy.translateY(box.center(b), -6),
      shade: 9,
      level: "h2",
      align: "center",
      baseline: "middle",
      code: true,
    });
    draw2d.text({
      text: this.state.units,
      position: xy.translateY(box.center(b), 27),
      shade: 7,
      level: "p",
      align: "center",
      baseline: "middle",
    });
    draw2d.circle({
      fill: this.internal.theme.colors.gray.l5,
      radius: { lower: baseRadius - 12, upper: baseRadius },
      position: box.center(b),
      angle: { lower: 0 * Math.PI, upper: 2 * Math.PI },
    });
    draw2d.circle({
      fill: this.internal.theme.colors.visualization.palettes.default[0],
      radius: { lower: baseRadius - 12, upper: baseRadius },
      position: box.center(b),
      angle: {
        lower: 1 * Math.PI,
        upper: (1 + (Number(value) / this.state.max) * 2) * Math.PI,
      },
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Gauge.TYPE]: Gauge,
};

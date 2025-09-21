// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, location, notation, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
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
  color: color.colorZ.optional().default(color.ZERO),
  precision: z.number().optional().default(2),
  minWidth: z.number().optional().default(60),
  width: z.number().optional(),
  notation: notation.notationZ.optional().default("standard"),
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
  requestRender: render.Requestor | null;
  textColor: color.Color;
}

export class Gauge
  extends aether.Leaf<typeof gaugeState, InternalState>
  implements Element
{
  static readonly TYPE = "gauge";
  static readonly z = gaugeState;
  schema = Gauge.z;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.render = render.Context.use(ctx);
    i.theme = theming.use(ctx);
    if (color.isZero(this.state.color)) i.textColor = i.theme.colors.gray.l8;
    else i.textColor = this.state.color;
    i.telem = telem.useSource(ctx, this.state.telem, i.telem);
    i.stopListening?.();
    i.stopListening = i.telem.onChange(() => this.requestRender());
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
    i.telem.cleanup?.();
    i.backgroundTelem.cleanup?.();
    if (i.requestRender == null)
      i.render.erase(box.construct(this.state.box), xy.ZERO, ...CANVAS_VARIANTS);
    else i.requestRender("layout");
  }

  private requestRender(): void {
    const { requestRender } = this.internal;
    if (requestRender != null) requestRender("layout");
    else void this.render({});
  }

  render({ viewportScale = scale.XY.IDENTITY }): void {
    const { render: renderCtx, theme } = this.internal;
    const upper2d = renderCtx.upper2d.applyScale(viewportScale);
    const draw2d = new Draw2D(upper2d, theme);
    const b = this.state.box;
    const baseRadius = box.width(b) / 2;
    const value = this.internal.telem.value();
    draw2d.text({
      text: value,
      position: xy.translateY(box.center(b), -6),
      shade: 9,
      level: "h2",
      align: "middle",
      // baseline: "middle",
      code: true,
    });
    draw2d.text({
      text: this.state.units,
      position: xy.translateY(box.center(b), 27),
      shade: 7,
      level: "p",
      align: "middle",
    });
    draw2d.circle({
      fill: this.internal.theme.colors.gray.l5,
      radius: baseRadius - 12,
      position: box.center(b),
      // angle: { lower: 0 * Math.PI, upper: 2 * Math.PI },
    });
    draw2d.circle({
      fill: this.internal.theme.colors.visualization.palettes.default[0],
      radius: baseRadius - 12,
      position: box.center(b),
      // angle: {
      //   lower: 1 * Math.PI,
      //   upper: (1 + (Number(value) / this.state.max) * 2) * Math.PI,
      // },
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Gauge.TYPE]: Gauge };

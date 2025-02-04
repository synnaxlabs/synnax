// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, scale, TimeStamp, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

export const tooltipStateZ = z.object({
  position: xy.xy.or(z.null()),
  textColor: color.Color.z.optional().default(color.ZERO),
  backgroundColor: color.Color.z.optional().default(color.ZERO),
  borderColor: color.Color.z.optional().default(color.ZERO),
  ruleColor: color.Color.z.optional().default(color.ZERO),
  ruleStrokeWidth: z.number().optional().default(1),
  ruleStrokeDash: z.number().default(0),
});

interface InternalState {
  render: render.Context;
  draw: Draw2D;
  dotColor: color.Color;
  dotColorContrast: color.Color;
}

export interface TooltipProps {
  findByXDecimal: (position: number) => Promise<FindResult[]>;
  region: box.Box;
}

export class Tooltip extends aether.Leaf<typeof tooltipStateZ, InternalState> {
  static readonly TYPE = "tooltip";
  schema = tooltipStateZ;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    const theme = theming.use(ctx);
    if (this.state.textColor.isZero) this.state.textColor = theme.colors.text;
    if (this.state.backgroundColor.isZero)
      this.state.backgroundColor = theme.colors.gray.l1;
    if (this.state.borderColor.isZero) this.state.borderColor = theme.colors.border;
    if (this.state.ruleColor.isZero) this.state.ruleColor = theme.colors.gray.l5;
    this.internal.dotColor = theme.colors.text;
    this.internal.dotColorContrast = theme.colors.textInverted;

    this.internal.render = render.Context.use(ctx);
    this.internal.draw = new Draw2D(this.internal.render.upper2d, theme);
    render.Controller.requestRender(ctx, render.REASON_TOOL);
  }

  async afterDelete(ctx: aether.Context): Promise<void> {
    render.Controller.requestRender(ctx, render.REASON_TOOL);
  }

  async render(props: TooltipProps): Promise<void> {
    if (this.deleted || this.state.position == null) return;
    const { region } = props;
    const scale_ = scale.XY.scale(box.DECIMAL).scale(region);
    const reverseScale = scale.XY.scale(region).scale(box.DECIMAL);
    const values = await props.findByXDecimal(
      reverseScale.x.pos(this.state.position.x),
    );
    const validValues = values.filter((c) => xy.isFinite(c.value));
    const { draw } = this.internal;

    const avgXPosition =
      validValues.reduce((p, c) => p + c.position.x, 0) / validValues.length;
    const avgXValue = new TimeStamp(
      validValues.reduce((p, c) => p + c.value.x, 0) / validValues.length,
    );

    const rulePosition = scale_.x.pos(avgXPosition);
    if (!bounds.contains(box.xBounds(region), rulePosition)) return;

    draw.rule({
      stroke: this.state.ruleColor,
      lineWidth: this.state.ruleStrokeWidth,
      lineDash: this.state.ruleStrokeDash,
      direction: "y",
      region,
      position: rulePosition,
    });

    validValues.forEach((r) => {
      const position = scale_.pos(r.position);
      draw.circle({ fill: r.color.setAlpha(0.5), radius: 8, position });
      draw.circle({ fill: r.color.setAlpha(0.8), radius: 5, position });
      draw.circle({
        fill: r.color.pickByContrast(
          this.internal.dotColor,
          this.internal.dotColorContrast,
        ),
        radius: 2,
        position,
      });
    });

    const text = values.map((r) => `${r.label ?? ""}: ${r.value.y.toFixed(2)}`);
    text.unshift(`Time: ${avgXValue.fString("preciseDate", "local")}`);

    const relativePosition = reverseScale.pos(this.state.position);

    draw.textContainer({
      text,
      backgroundColor: this.state.backgroundColor,
      borderColor: this.state.borderColor,
      position: this.state.position,
      direction: "y",
      level: "small",
      spacing: 0.5,
      offset: { x: 12, y: 12 },
      root: {
        x: relativePosition.x > 0.8 ? "right" : "left",
        y: relativePosition.y > 0.8 ? "top" : "bottom",
      },
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Tooltip.TYPE]: Tooltip,
};

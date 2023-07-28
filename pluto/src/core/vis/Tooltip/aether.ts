// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Direction, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { Draw2D } from "../draw2d";

import { AetherLeaf } from "@/core/aether/worker";
import { Color } from "@/core/color";
import { ThemeContext } from "@/core/theming/aether";
import { LookupResult } from "@/core/vis/Line/core";
import { RenderContext, RenderController } from "@/core/vis/render";

export const tooltipState = z.object({
  position: XY.z.or(z.null()),
  textColor: Color.z.optional().default(Color.ZERO),
  backgroundColor: Color.z.optional().default(Color.ZERO),
  borderColor: Color.z.optional().default(Color.ZERO),
  ruleColor: Color.z.optional().default(Color.ZERO),
  ruleStrokeWidth: z.number().optional().default(1),
  ruleStrokeDash: z.number().default(0),
});

interface Derived {
  renderCtx: RenderContext;
  draw: Draw2D;
}

export interface TooltipProps {
  lookupX: (position: number) => Promise<LookupResult[]>;
  region: Box;
}

export class AetherTooltip extends AetherLeaf<typeof tooltipState, Derived> {
  static readonly TYPE = "tooltip";
  static readonly stateZ = tooltipState;
  schema = AetherTooltip.stateZ;

  derive(): Derived {
    const theme = ThemeContext.use(this.ctx);
    if (this.state.textColor.isZero) this.state.textColor = theme.colors.text;
    if (this.state.backgroundColor.isZero)
      this.state.backgroundColor = theme.colors.gray.m2;
    if (this.state.borderColor.isZero) this.state.borderColor = theme.colors.border;
    if (this.state.ruleColor.isZero) this.state.ruleColor = theme.colors.gray.m1;

    const ctx = RenderContext.use(this.ctx);
    return {
      renderCtx: ctx,
      draw: new Draw2D(ctx.upper2d, theme),
    };
  }

  afterUpdate(): void {
    RenderController.requestRender(this.ctx);
  }

  afterDelete(): void {
    RenderController.requestRender(this.ctx);
  }

  async render(props: TooltipProps): Promise<void> {
    if (this.deleted || this.state.position == null) return;
    const pos = this.state.position.translate(props.region.topLeft.scale(-1));

    const values = await props.lookupX(pos.x);
    const { region } = props;

    const { draw } = this.derived;

    const x = values.reduce((p, c) => p + c.position.x, 0) / values.length;

    draw.rule({
      stroke: this.state.ruleColor,
      lineWidth: this.state.ruleStrokeWidth,
      lineDash: this.state.ruleStrokeDash,
      direction: Direction.Y,
      region,
      position: region.left + x,
    });

    values.forEach((r) => {
      const position = r.position.translate(region.topLeft);
      draw.circle({ fill: r.color.setAlpha(0.5), radius: 8, position });
      draw.circle({ fill: r.color.setAlpha(0.8), radius: 5, position });
      draw.circle({ fill: this.state.textColor, radius: 2, position });
    });

    const text = values.map((r) => `${r.label ?? ""}: ${r.value.toFixed(2)}`);

    draw.textContainer({
      text,
      position: this.state.position.translate([10, 10]),
      direction: Direction.Y,
      level: "small",
      spacing: 1,
    });
  }
}

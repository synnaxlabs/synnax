// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, location, notation, scale, TimeStamp, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

const TOOLTIP_LIST_OFFSET: xy.XY = xy.construct(12);
const TOOLTIP_LIST_SPACING: number = 3;
const TOOLTIP_LIST_ITEM_HEIGHT: number = 14;
const TOOLTIP_PADDING: xy.XY = xy.construct(6);

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
    if (this.state.ruleColor.isZero) this.state.ruleColor = theme.colors.gray.l7;
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

    const relativePosition = reverseScale.pos(this.state.position);

    const root = { ...location.TOP_LEFT };
    if (relativePosition.x > 0.6) root.x = "right";
    if (relativePosition.y > 0.6) root.y = "bottom";

    let maxLabelLength = values.reduce((p, c) => Math.max(p, c.label?.length ?? 0), 0);
    const timeValueLength = avgXValue.fString("preciseDate", "local").length;
    if (timeValueLength > maxLabelLength) maxLabelLength = timeValueLength;

    draw.list({
      root,
      offset: TOOLTIP_LIST_OFFSET,
      length: values.length + 1,
      padding: TOOLTIP_PADDING,
      itemHeight: TOOLTIP_LIST_ITEM_HEIGHT,
      spacing: TOOLTIP_LIST_SPACING,
      width: maxLabelLength * 7 + 48,
      position: this.state.position,
      draw: (i, b) => {
        let label = "";
        let value = "";
        let color = this.state.textColor;
        if (i === 0) {
          label = "Time";
          value = avgXValue.fString("preciseDate", "local");
        } else {
          const v = values[i - 1];
          label = v.label ?? "";
          value = notation.roundSmart(v.value.y, v.bounds).toString();
          color = v.color;
        }
        draw.text({
          position: box.topLeft(b),
          text: label,
          level: "small",
          weight: 500,
          color,
        });
        draw.text({
          position: xy.translateY(box.topRight(b), -1),
          text: value,
          level: "small",
          justify: "right",
          code: true,
          shade: 10,
        });
      },
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Tooltip.TYPE]: Tooltip,
};

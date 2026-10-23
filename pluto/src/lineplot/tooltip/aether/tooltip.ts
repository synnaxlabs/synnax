// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  box,
  color,
  location,
  math,
  scale,
  TimeStamp,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

const TOOLTIP_LIST_OFFSET: xy.XY = xy.construct(12);
const TOOLTIP_LIST_SPACING: number = 3;
const TOOLTIP_LIST_ITEM_HEIGHT: number = 14;
const TOOLTIP_PADDING: xy.XY = xy.construct(6);
const TOOLTIP_COLUMN_GAP: number = 24;
const LABEL_FONT = { level: "small", weight: 500 } as const;
const VALUE_FONT = { level: "small", code: true } as const;

export const tooltipStateZ = z.object({
  position: xy.xyZ.or(z.null()),
  textColor: color.colorZ.default(color.ZERO),
  backgroundColor: color.colorZ.default(color.ZERO),
  borderColor: color.colorZ.default(color.ZERO),
  ruleColor: color.colorZ.default(color.ZERO),
  ruleStrokeWidth: z.number().default(1),
  ruleStrokeDash: z.number().default(0),
});

interface InternalState {
  render: render.Context;
  draw: Draw2D;
  dotColor: color.Color;
  dotColorContrast: color.Color;
  textColor: color.Color;
  ruleColor: color.Color;
}

export interface TooltipProps {
  findByXDecimal: (position: number) => FindResult[];
  region: box.Box;
}

export class Tooltip extends aether.Leaf<typeof tooltipStateZ, InternalState> {
  static readonly TYPE = "tooltip";
  schema = tooltipStateZ;

  afterUpdate(ctx: aether.Context): void {
    const theme = theming.use(ctx);
    const { textColor, ruleColor } = this.state;
    this.internal.textColor = color.isZero(textColor) ? theme.colors.text : textColor;
    this.internal.ruleColor = color.isZero(ruleColor)
      ? theme.colors.gray.l7
      : ruleColor;
    this.internal.dotColor = theme.colors.text;
    this.internal.dotColorContrast = theme.colors.textInverted;

    this.internal.render = render.Context.use(ctx);
    this.internal.draw = new Draw2D(this.internal.render.upper2d, theme);
    render.request(ctx, "tool");
  }

  afterDelete(ctx: aether.Context): void {
    render.request(ctx, "tool");
  }

  render(props: TooltipProps): void {
    if (this.deleted || this.state.position == null) return;
    const { region } = props;
    const scale_ = scale.XY.scale(box.DECIMAL).scale(region);
    const reverseScale = scale.XY.scale(region).scale(box.DECIMAL);
    const values = props.findByXDecimal(reverseScale.x.pos(this.state.position.x));
    const validValues = values.filter((c) => Number.isFinite(c.value.x));
    const { draw } = this.internal;

    const avgXPosition =
      validValues.reduce((p, c) => p + c.position.x, 0) / validValues.length;
    const avgXValue = new TimeStamp(
      validValues.reduce((p, c) => p + c.value.x, 0) / validValues.length,
    );

    const rulePosition = scale_.x.pos(avgXPosition);
    if (!bounds.contains(box.xBounds(region), rulePosition)) return;

    draw.rule({
      stroke: this.internal.ruleColor,
      lineWidth: this.state.ruleStrokeWidth,
      lineDash: this.state.ruleStrokeDash,
      direction: "y",
      region,
      position: rulePosition,
    });

    validValues.forEach((r) => {
      if (!Number.isFinite(r.value.y)) return;
      const position = scale_.pos(r.position);
      draw.circle({ fill: color.setAlpha(r.color, 0.5), radius: 8, position });
      draw.circle({ fill: color.setAlpha(r.color, 0.8), radius: 5, position });
      draw.circle({
        fill: color.pickByContrast(
          r.color,
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

    const time = avgXValue.toString("preciseDate", "local");
    const rows = [
      { label: "Time", value: time, color: this.internal.textColor },
      ...validValues.map((v) => ({
        label: v.label ?? "",
        value: formatValue(v),
        color: v.color,
      })),
    ];
    const width = Math.max(
      ...rows.map(
        ({ label, value }) =>
          draw.textWidth({ text: label, ...LABEL_FONT }) +
          TOOLTIP_COLUMN_GAP +
          draw.textWidth({ text: value, ...VALUE_FONT }),
      ),
    );

    draw.list({
      root,
      offset: TOOLTIP_LIST_OFFSET,
      length: rows.length,
      padding: TOOLTIP_PADDING,
      itemHeight: TOOLTIP_LIST_ITEM_HEIGHT,
      spacing: TOOLTIP_LIST_SPACING,
      width,
      position: this.state.position,
      draw: (i, b) => {
        const { label, value, color } = rows[i];
        draw.text({ position: box.topLeft(b), text: label, color, ...LABEL_FONT });
        draw.text({
          position: xy.translateY(box.topRight(b), -1),
          text: value,
          justify: "right",
          shade: 10,
          ...VALUE_FONT,
        });
      },
    });
  }
}

const formatValue = ({ value: { y }, bounds }: FindResult): string => {
  if (y === Infinity) return "∞";
  if (y === -Infinity) return "-∞";
  return math.smartRound(y, bounds).toString();
};

export const REGISTRY: aether.ComponentRegistry = {
  [Tooltip.TYPE]: Tooltip,
};

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
import { formatRelativeTime } from "@/vis/axis/relative";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

const TOOLTIP_LIST_OFFSET: xy.XY = xy.construct(12);
const TOOLTIP_LIST_SPACING: number = 3;
const TOOLTIP_LIST_ITEM_HEIGHT: number = 14;
const TOOLTIP_PADDING: xy.XY = xy.construct(6);

export const tooltipStateZ = z.object({
  position: xy.xyZ.or(z.null()),
  textColor: color.colorZ.optional(),
  backgroundColor: color.colorZ.optional(),
  borderColor: color.colorZ.optional(),
  ruleColor: color.colorZ.optional(),
  ruleStrokeWidth: z.number().default(1),
  ruleStrokeDash: z.number().default(0),
  /** True when x is a span from a zero, shown as T+ time. */
  relative: z.boolean().default(false),
  /** The line nearest the cursor and the x of its value, set by the tooltip. */
  hovered: z.object({ key: z.string(), x: z.number() }).nullable().default(null),
});

export type Hovered = NonNullable<z.infer<typeof tooltipStateZ>["hovered"]>;

// A line is hovered when the cursor is within this many pixels of its value.
const HOVER_DISTANCE = 12;

// Row is one row of the tooltip list: a heading, or an entry with a value.
interface Row {
  label: string;
  value?: string;
  color?: color.Color;
  bold?: boolean;
}

// byValue sorts high to low, with NaN last.
const byValue = (
  { value: { y: a } }: FindResult,
  { value: { y: b } }: FindResult,
): number => (a === b ? 0 : Number.isNaN(a) ? 1 : Number.isNaN(b) ? -1 : b - a);

// rows lists the results of each tooltip group under a heading, sorted by value.
// Results with no group keep their order.
const rows = (results: FindResult[], hovered: Hovered | null): Row[] => {
  const entry = (r: FindResult, label: string | undefined): Row => ({
    label: label ?? "",
    value: formatValue(r),
    color: r.color,
    bold: r.key === hovered?.key,
  });
  if (!results.some((r) => r.tooltipGroup != null))
    return results.map((r) => entry(r, r.label));
  const groups = new Map<string, FindResult[]>();
  results.forEach((r) => {
    const key = r.tooltipGroup ?? r.key;
    groups.set(key, [...(groups.get(key) ?? []), r]);
  });
  return [...groups.values()].flatMap((members) => [
    { label: members[0].label ?? "", bold: true },
    ...[...members].sort(byValue).map((r) => entry(r, r.group)),
  ]);
};

// fit is how many list items fit in height.
const fit = (height: number): number =>
  Math.floor(
    (height - TOOLTIP_PADDING.y * 2 + TOOLTIP_LIST_SPACING) /
      (TOOLTIP_LIST_ITEM_HEIGHT + TOOLTIP_LIST_SPACING),
  );

// capped returns at most max rows, ending with "and N more" when it drops any.
const capped = (all: Row[], max: number): Row[] => {
  if (all.length <= max) return all;
  const kept = all.slice(0, max - 1);
  const dropped = all.slice(max - 1).filter((r) => r.value != null).length;
  return [...kept, { label: `and ${dropped} more` }];
};

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
  override schema = tooltipStateZ;

  override afterUpdate(ctx: aether.Context): void {
    const theme = theming.use(ctx);
    const { textColor, ruleColor } = this.state;
    this.internal.textColor = textColor ?? theme.colors.text;
    this.internal.ruleColor = ruleColor ?? theme.colors.gray.l7;
    this.internal.dotColor = theme.colors.text;
    this.internal.dotColorContrast = theme.colors.textInverted;

    this.internal.render = render.Context.use(ctx);
    this.internal.draw = new Draw2D(this.internal.render.upper2d, theme);
    if (this.state.position == null && this.state.hovered != null)
      this.setState((p) => ({ ...p, hovered: null }));
    render.request(ctx, "tool");
  }

  override afterDelete(ctx: aether.Context): void {
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
    const avgX = validValues.reduce((p, c) => p + c.value.x, 0) / validValues.length;
    const timeValue = this.state.relative
      ? formatRelativeTime(avgX, MILLISECOND)
      : new TimeStamp(avgX).toString("preciseDate", "local");
    this.updateHovered(validValues, scale_);

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
      draw.circle({ fill: color.setOpacity(r.color, 0.5), radius: 8, position });
      draw.circle({ fill: color.setOpacity(r.color, 0.8), radius: 5, position });
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

    // The time heading takes one item, and "and N more" always fits.
    const maxRows = Math.max(1, fit(box.height(region)) - 1);
    const listed = capped(rows(validValues, this.state.hovered), maxRows);
    const maxLabelLength = Math.max(
      timeValue.length,
      ...listed.map((r) => r.label.length + (r.value?.length ?? 0)),
    );

    draw.list({
      root,
      offset: TOOLTIP_LIST_OFFSET,
      length: listed.length + 1,
      padding: TOOLTIP_PADDING,
      itemHeight: TOOLTIP_LIST_ITEM_HEIGHT,
      spacing: TOOLTIP_LIST_SPACING,
      width: maxLabelLength * 7 + 48,
      position: this.state.position,
      bound: region,
      draw: (i, b) => {
        const row: Row = i === 0 ? { label: "Time", value: timeValue } : listed[i - 1];
        draw.text({
          position: box.topLeft(b),
          text: row.label,
          level: "small",
          weight: row.bold === true ? 700 : 500,
          color: row.color ?? this.internal.textColor,
        });
        if (row.value == null) return;
        draw.text({
          position: xy.translateY(box.topRight(b), -1),
          text: row.value,
          level: "small",
          justify: "right",
          code: true,
          shade: 10,
          weight: row.bold === true ? 700 : undefined,
        });
      },
    });
  }

  // updateHovered sets hovered to the key of the line whose value is nearest the
  // cursor, or null when no value is within HOVER_DISTANCE.
  private updateHovered(results: FindResult[], toPixels: scale.XY): void {
    const { position } = this.state;
    if (position == null) return;
    let nearest: FindResult | null = null;
    let distance = HOVER_DISTANCE;
    for (const r of results) {
      if (!Number.isFinite(r.value.y)) continue;
      const d = Math.abs(toPixels.pos(r.position).y - position.y);
      if (d >= distance) continue;
      nearest = r;
      distance = d;
    }
    const hovered = nearest == null ? null : { key: nearest.key, x: nearest.value.x };
    const prev = this.state.hovered;
    if (prev?.key === hovered?.key && prev?.x === hovered?.x) return;
    this.setState((p) => ({ ...p, hovered }));
  }
}

const MILLISECOND = 1e6;

const formatValue = ({ value: { y }, bounds }: FindResult): string => {
  if (y === Infinity) return "∞";
  if (y === -Infinity) return "-∞";
  return math.smartRound(y, bounds).toString();
};

export const REGISTRY: aether.ComponentRegistry = {
  [Tooltip.TYPE]: Tooltip,
};

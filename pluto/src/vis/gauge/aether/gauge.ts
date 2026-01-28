// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, color, location, notation, scale, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { telem } from "@/telem/aether";
import { text } from "@/text/base";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { Draw2D } from "@/vis/draw2d";
import { render } from "@/vis/render";

// Define gauge size presets
export const GAUGE_SIZES = {
  small: 80,
  medium: 120,
  large: 160,
  huge: 200,
} as const;

export type GaugeSize = keyof typeof GAUGE_SIZES;
export const gaugeSizeZ = z.enum(["small", "medium", "large", "huge"]);

const gaugeState = z.object({
  box: box.box,
  telem: telem.stringSourceSpecZ.default(telem.noopStringSourceSpec),
  backgroundTelem: telem.colorSourceSpecZ.default(telem.noopColorSourceSpec),
  level: text.levelZ.default("p"),
  color: color.colorZ.default(color.ZERO),
  precision: z.number().default(2),
  minWidth: z.number().default(60),
  width: z.number().optional(),
  notation: notation.notationZ.default("standard"),
  location: location.xy.default({ x: "left", y: "center" }),
  units: z.string().default("RPM"),
  bounds: bounds.bounds.default(bounds.construct(0, 100)),
  // New gauge configuration properties
  barWidth: z.number().default(12), // Width of the gauge bar in pixels
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

export interface GaugeProps {
  scale?: scale.XY;
}

interface InternalState {
  theme: theming.Theme;
  render: render.Context;
  telem: telem.StringSource;
  draw2d: Draw2D;
  stopListening?: () => void;
  backgroundTelem: telem.ColorSource;
  stopListeningBackground?: () => void;
  requestRender: render.Requestor | null;
  textColor: color.Color;
  strokeColor: color.Color;
  gaugeStartAngle: number;
  gaugeEndAngle: number;
  gaugeAngleRange: number;
  outerRadius: number;
  innerRadius: number;
  labelRadius: number;
  labelInwardShift: number;
  minLabelPos: xy.XY;
  maxLabelPos: xy.XY;
  centerPos: xy.XY;
  valueTextPos: xy.XY;
  unitsTextPos: xy.XY;
  textLevel: text.Level;
  labelLevel: text.Level;
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

    i.strokeColor = color.isZero(this.state.color)
      ? i.theme.colors.visualization.palettes.default[0]
      : this.state.color;

    // Pre-calculate gauge geometry
    const b = this.state.box;
    const baseRadius = box.width(b) / 2;
    const { barWidth } = this.state;

    i.outerRadius = baseRadius - 8;
    i.innerRadius = i.outerRadius - barWidth;

    // Calculate gauge angles (270-degree arc with 90-degree gap at top)
    const gapSize = Math.PI / 2; // 90 degree gap
    i.gaugeStartAngle = (3 * Math.PI) / 4; // Start from top-left (135 degrees)
    i.gaugeAngleRange = 2 * Math.PI - gapSize; // Total arc is 270 degrees
    i.gaugeEndAngle = i.gaugeStartAngle + i.gaugeAngleRange;

    // Calculate label positions
    i.labelRadius = i.outerRadius + 12;
    i.labelInwardShift = 12;
    i.centerPos = box.center(b);

    const labelY = i.centerPos.y + i.labelRadius * Math.sin(i.gaugeStartAngle) + 6;

    i.minLabelPos = xy.construct(
      i.centerPos.x + i.labelRadius * Math.cos(i.gaugeStartAngle) + i.labelInwardShift,
      labelY,
    );

    i.maxLabelPos = xy.construct(
      i.centerPos.x + i.labelRadius * Math.cos(i.gaugeEndAngle) - i.labelInwardShift,
      labelY,
    );

    i.valueTextPos = xy.translateY(i.centerPos, -6);
    i.unitsTextPos = xy.translateY(i.centerPos, box.height(b) / 9);

    i.textLevel = this.state.level;
    i.labelLevel = text.downLevel(text.downLevel(i.textLevel));

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
    const { internal: i } = this;
    const upper2d = i.render.upper2d.applyScale(viewportScale);
    const draw2d = new Draw2D(upper2d, i.theme);
    const value = i.telem.value();

    // Calculate value angle (only thing that changes per render)
    const { lower, upper } = this.state.bounds;
    const valueNum = Number(value);
    const clampedValue = bounds.clamp(this.state.bounds, valueNum);
    const range = upper - lower;
    const valueRatio = range === 0 ? 0 : (clampedValue - lower) / range;
    const valueAngle = i.gaugeStartAngle + valueRatio * i.gaugeAngleRange;

    draw2d.text({
      text: value,
      position: i.valueTextPos,
      shade: 10,
      level: i.textLevel,
      align: "middle",
      justify: "center",
      weight: 450,
      code: true,
      useAtlas: true,
    });
    draw2d.text({
      text: this.state.units,
      position: i.unitsTextPos,
      shade: 8,
      level: text.downLevel(i.textLevel),
      align: "middle",
      justify: "center",
      code: true,
      useAtlas: true,
    });

    // Add min/max labels at the gap endpoints
    draw2d.text({
      text: lower.toString(),
      position: i.minLabelPos,
      shade: 7,
      level: i.labelLevel,
      align: "middle",
      justify: "center",
      code: true,
      useAtlas: true,
    });

    draw2d.text({
      text: upper.toString(),
      position: i.maxLabelPos,
      shade: 7,
      level: i.labelLevel,
      align: "middle",
      justify: "center",
      code: true,
      useAtlas: true,
    });

    draw2d.circle({
      stroke: i.theme.colors.gray.l5,
      radius: { inner: i.innerRadius, outer: i.outerRadius },
      position: i.centerPos,
      angle: { lower: i.gaugeStartAngle, upper: i.gaugeEndAngle },
      lineCap: "round",
    });

    draw2d.circle({
      stroke: i.strokeColor,
      radius: { inner: i.innerRadius, outer: i.outerRadius },
      position: i.centerPos,
      angle: {
        lower: i.gaugeStartAngle,
        upper: valueAngle,
      },
      lineCap: "round",
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Gauge.TYPE]: Gauge };

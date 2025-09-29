// Copyright 2024 Synnax Labs, Inc.
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
import { text } from "@/text/core";
import { theming } from "@/theming/aether";
import { type Element } from "@/vis/diagram/aether/Diagram";
import { Draw2D } from "@/vis/draw2d";
import { type FillTextOptions } from "@/vis/draw2d/canvas";
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
  bounds: bounds.bounds.optional().default(bounds.construct(0, 100)),
  // New gauge configuration properties
  barWidth: z.number().optional().default(12), // Width of the gauge bar in pixels
});

const CANVAS_VARIANTS: render.Canvas2DVariant[] = ["upper2d", "lower2d"];

export interface GaugeProps {
  scale?: scale.XY;
}

const FILL_TEXT_OPTIONS: FillTextOptions = { useAtlas: true };

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
    const { barWidth } = this.state;

    const outerRadius = baseRadius - 8;
    const innerRadius = outerRadius - barWidth;

    // Calculate value ratio based on bounds
    const { lower, upper } = this.state.bounds;
    // Create a gap at the top - gauge spans from top-left to top-right
    // Rotating 90 degrees counter-clockwise from previous position
    // Start at 3π/4 (135 degrees) and end at π/4 (45 degrees) - this leaves a 90 degree gap at top
    const gapSize = Math.PI / 2; // 90 degree gap
    const gaugeStartAngle = (3 * Math.PI) / 4; // Start from top-left (135 degrees)
    const gaugeAngleRange = 2 * Math.PI - gapSize; // Total arc is 270 degrees

    const valueNum = Number(value);
    const clampedValue = bounds.clamp(this.state.bounds, valueNum);
    // Ensure proper handling of negative values
    const range = upper - lower;
    const valueRatio = range === 0 ? 0 : (clampedValue - lower) / range;
    const valueAngle = gaugeStartAngle + valueRatio * gaugeAngleRange;

    const textLevel = this.state.level;
    const labelLevel = text.downLevel(text.downLevel(textLevel));

    // Main value and units
    draw2d.text({
      text: value,
      position: xy.translateY(box.center(b), -6),
      shade: 10,
      level: textLevel,
      align: "middle",
      justify: "center",
      weight: 450,
      code: true,
    });
    draw2d.text({
      text: this.state.units,
      position: xy.translateY(box.center(b), 6),
      shade: 8,
      level: text.downLevel(textLevel),
      align: "top",
      justify: "center",
    });

    // Add min/max labels at the gap endpoints
    const labelRadius = outerRadius + 12;
    const gaugeEndAngle = gaugeStartAngle + gaugeAngleRange;
    const labelInwardShift = 8; // Shift labels inward on x-axis

    // Min value label (at start of gauge arc - top left) - shift right
    const minLabelPos = xy.construct(
      box.center(b).x + labelRadius * Math.cos(gaugeStartAngle) + labelInwardShift,
      box.center(b).y + labelRadius * Math.sin(gaugeStartAngle),
    );
    draw2d.text({
      text: String(lower),
      position: minLabelPos,
      shade: 7,
      level: labelLevel,
      align: "middle",
      justify: "center",
      code: true,
    });

    // Max value label (at end of gauge arc - top right) - shift left
    const maxLabelPos = xy.construct(
      box.center(b).x + labelRadius * Math.cos(gaugeEndAngle) - labelInwardShift,
      box.center(b).y + labelRadius * Math.sin(gaugeEndAngle),
    );
    draw2d.text({
      text: String(upper),
      position: maxLabelPos,
      shade: 7,
      level: labelLevel,
      align: "middle",
      justify: "center",
      code: true,
    });

    // Draw background arc (with gap) using stroke for rounded caps
    draw2d.circle({
      stroke: this.internal.theme.colors.gray.l5,
      radius: { inner: innerRadius, outer: outerRadius },
      position: box.center(b),
      angle: { lower: gaugeStartAngle, upper: gaugeEndAngle },
      lineCap: "round",
    });

    // Draw value arc with rounded caps
    draw2d.circle({
      stroke: color.isZero(this.state.color)
        ? this.internal.theme.colors.visualization.palettes.default[0]
        : this.state.color,
      radius: { inner: innerRadius, outer: outerRadius },
      position: box.center(b),
      angle: {
        lower: gaugeStartAngle,
        upper: valueAngle,
      },
      lineCap: "round",
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = { [Gauge.TYPE]: Gauge };

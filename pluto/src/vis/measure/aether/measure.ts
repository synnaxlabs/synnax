// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, color, scale, TimeSpan, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { type Theme } from "@/theming/core/theme";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

export const modeZ = z.enum(["one", "two", "clear", "empty"]);
export type Mode = z.infer<typeof modeZ>;

export const measureStateZ = z.object({
  one: xy.xy.nullable(),
  two: xy.xy.nullable(),
  hover: xy.xy.nullable(),
  mode: modeZ.optional().default("one"),
  color: z
    .union([
      color.colorZ,
      z.object({
        verticalLine: color.colorZ.optional().default(color.ZERO),
        horizontalLine: color.colorZ.optional().default(color.ZERO),
        obliqueLine: color.colorZ.optional().default(color.ZERO),
      }),
    ])
    .optional()
    .default(color.ZERO),
  strokeWidth: z.number().optional().default(1),
  strokeDash: z.number().optional().default(2),
});

interface InternalState {
  renderCtx: render.Context;
  theme: Theme;
  draw: Draw2D;
  dataOne: xy.XY | null;
  dataTwo: xy.XY | null;
}

const measureModeText = (mode: Mode): string => {
  if (mode === "one") return "1";
  return "2";
};

const xLabelColor = (t: Theme) => t.colors.error.z;
const yLabelColor = (t: Theme) => t.colors.secondary.z;
const slopeLabelColor = (t: Theme) => t.colors.gray.l9;

const LABEL_CONTAINER_PADDING = 6;
const LABEL_CONTAINER_HEIGHT = 14;
const LABEL_CHAR_WIDTH = 8;
const VALUE_CHAR_WIDTH = 8;
const LABEL_VALUE_SPACING = 3;

const HOVER_CIRCLE_OUTER_RADIUS = 9;
const HOVER_CIRCLE_INNER_RADIUS = 8;

const POINT_CIRCLE_RADIUS_OUTER = 8;
const POINT_CIRCLE_RADIUS_MID = 5;
const POINT_CIRCLE_RADIUS_INNER = 2;
const POINT_CIRCLE_ALPHA_OUTER = 0.5;
const POINT_CIRCLE_ALPHA_MID = 0.8;

const PROXIMITY_THRESHOLD = 50;
const LABEL_OFFSET_VERY_CLOSE_X = 80;
const LABEL_OFFSET_VERY_CLOSE_Y = 30;
const LABEL_OFFSET_VERTICAL_Y = 30;
const LABEL_OFFSET_HORIZONTAL_X = 10;
const LABEL_OFFSET_HORIZONTAL_SLOPE = 35;
const LABEL_OFFSET_VERY_CLOSE_X_DOWN = 40;

const LABEL_OVERLAP_X_THRESHOLD = 100;
const LABEL_OVERLAP_Y_THRESHOLD = 30;

export interface MeasureProps {
  findByXDecimal: (target: number) => FindResult[];
  findByXValue: (target: number) => FindResult[];
  region: box.Box;
}

export class Measure extends aether.Leaf<typeof measureStateZ, InternalState> {
  static readonly TYPE = "measure";
  schema = measureStateZ;

  afterUpdate(ctx: aether.Context): void {
    const renderCtx = render.Context.use(ctx);
    this.internal.theme = theming.use(ctx);
    this.internal.renderCtx = renderCtx;
    this.internal.draw = new Draw2D(renderCtx.upper2d, this.internal.theme);
    render.request(ctx, "tool");
  }

  afterDelete(ctx: aether.Context): void {
    render.request(ctx, "layout");
  }

  private get verticalLineColor(): color.Color {
    if (color.isColor(this.state.color)) {
      if (color.isZero(this.state.color)) return this.internal.theme.colors.gray.l8;
      return this.state.color;
    }

    if (color.isZero(this.state.color.verticalLine))
      return this.internal.theme.colors.gray.l8;
    return this.state.color.verticalLine;
  }

  private get horizontalLineColor(): color.Color {
    if (color.isColor(this.state.color)) {
      if (color.isZero(this.state.color)) return this.internal.theme.colors.gray.l8;
      return this.state.color;
    }
    if (color.isZero(this.state.color.horizontalLine))
      return this.internal.theme.colors.gray.l8;
    return this.state.color.horizontalLine;
  }

  private get obliqueLineColor(): color.Color {
    if (color.isColor(this.state.color)) {
      if (color.isZero(this.state.color)) return this.internal.theme.colors.gray.l8;
      return this.state.color;
    }
    if (color.isZero(this.state.color.obliqueLine))
      return this.internal.theme.colors.gray.l8;
    return this.state.color.obliqueLine;
  }

  private find(props: MeasureProps): [FindResult, FindResult] | null {
    const { one, two } = this.state;
    if (one == null || two == null) return null;
    const { one: prevOne, two: prevTwo } = this.prevState;
    const { dataOne, dataTwo } = this.internal;

    let oneResult: FindResult | null = null;
    if (prevOne != null && xy.equals(one, prevOne) && dataOne != null) {
      const results = props.findByXValue(dataOne.x);
      if (results.length > 0)
        oneResult = results.sort(
          (a, b) => xy.distance(dataOne, a.value) - xy.distance(dataOne, b.value),
        )[0];
    }

    let twoResult: FindResult | null = null;
    if (prevTwo != null && xy.equals(two, prevTwo) && dataTwo != null) {
      const results = props.findByXValue(dataTwo.x);
      if (results.length > 0)
        twoResult = results.sort(
          (a, b) => xy.distance(dataTwo, a.value) - xy.distance(dataTwo, b.value),
        )[0];
    }

    const s = scale.XY.scale(props.region).scale(box.DECIMAL);

    if (oneResult == null) {
      const scaledOne = s.pos(one);
      const oneValues = props.findByXDecimal(scaledOne.x);
      if (oneValues.length === 0) return null;
      oneResult = oneValues.sort(
        (a, b) =>
          xy.distance(scaledOne, a.position) - xy.distance(scaledOne, b.position),
      )[0];
      this.internal.dataOne = oneResult.value;
    }

    if (twoResult == null) {
      const scaledTwo = s.pos(two);
      const twoValues = props.findByXDecimal(scaledTwo.x);
      if (twoValues.length === 0) return null;
      twoResult = twoValues.sort(
        (a, b) =>
          xy.distance(scaledTwo, a.position) - xy.distance(scaledTwo, b.position),
      )[0];
      this.internal.dataTwo = twoResult.value;
    }

    return [oneResult, twoResult];
  }

  private drawLabelValue(
    label: string,
    value: string,
    position: xy.XY,
    labelColor: (t: Theme) => color.Color,
  ): void {
    const { draw } = this.internal;
    const padding = xy.construct(LABEL_CONTAINER_PADDING);
    const approxLabelWidth = label.length * LABEL_CHAR_WIDTH;
    const approxValueWidth = value.length * VALUE_CHAR_WIDTH;
    const width = approxLabelWidth + approxValueWidth + padding.x * LABEL_VALUE_SPACING;
    const height = LABEL_CONTAINER_HEIGHT + padding.y * 2;

    const region = box.construct(position, width, height);

    draw.container({
      region,
      backgroundColor: (t) => t.colors.gray.l1,
    });

    draw.text({
      text: label,
      position: xy.translate(box.topLeft(region), padding),
      level: "small",
      weight: 500,
      color: labelColor(this.internal.theme),
    });

    draw.text({
      text: value,
      position: xy.translate(box.topRight(region), [-padding.x - 1, padding.y - 1]),
      level: "small",
      justify: "right",
      code: true,
      shade: 10,
    });
  }

  private drawPointMarker(position: xy.XY, pointColor: color.Color): void {
    const { draw } = this.internal;
    draw.circle({
      fill: color.setAlpha(pointColor, POINT_CIRCLE_ALPHA_OUTER),
      radius: POINT_CIRCLE_RADIUS_OUTER,
      position,
    });
    draw.circle({
      fill: color.setAlpha(pointColor, POINT_CIRCLE_ALPHA_MID),
      radius: POINT_CIRCLE_RADIUS_MID,
      position,
    });
    draw.circle({
      fill: color.WHITE,
      radius: POINT_CIRCLE_RADIUS_INNER,
      position,
    });
  }

  private calculateYLabelPosition(
    basePos: xy.XY,
    isVeryClose: boolean,
    isVertical: boolean,
  ): xy.XY {
    if (isVeryClose) return xy.translateX(basePos, LABEL_OFFSET_VERY_CLOSE_Y);
    if (isVertical) return xy.translateX(basePos, LABEL_OFFSET_VERTICAL_Y);
    return basePos;
  }

  private calculateXLabelPosition(
    basePos: xy.XY,
    isVeryClose: boolean,
    isHorizontal: boolean,
  ): xy.XY {
    if (isVeryClose) return xy.translateY(basePos, LABEL_OFFSET_VERY_CLOSE_X_DOWN);
    if (isHorizontal) return xy.translateY(basePos, LABEL_OFFSET_HORIZONTAL_X);
    return basePos;
  }

  private calculateSlopeLabelPosition(
    basePos: xy.XY,
    yLabelPos: xy.XY,
    isVeryClose: boolean,
    isHorizontal: boolean,
    xPixelDist: number,
    yPixelDist: number,
  ): xy.XY {
    if (isVeryClose) return xy.translateX(basePos, LABEL_OFFSET_VERY_CLOSE_X);
    if (isHorizontal) return xy.translateY(basePos, -LABEL_OFFSET_HORIZONTAL_SLOPE);

    const slopeYLabelOverlap =
      Math.abs(basePos.x - yLabelPos.x) < LABEL_OVERLAP_X_THRESHOLD &&
      Math.abs(basePos.y - yLabelPos.y) < LABEL_OVERLAP_Y_THRESHOLD;
    if (slopeYLabelOverlap) {
      if (xPixelDist < yPixelDist)
        return xy.translateY(basePos, -LABEL_OFFSET_HORIZONTAL_SLOPE);
      return xy.translateX(basePos, LABEL_OFFSET_VERY_CLOSE_X);
    }
    return basePos;
  }

  private renderHover(props: MeasureProps): void {
    if (this.state.hover == null) return;
    const hover: xy.XY = this.state.hover;

    const s = scale.XY.scale(props.region).scale(box.DECIMAL);
    const scaledPos = s.pos(hover);
    const res = props.findByXDecimal(s.pos(hover).x);
    if (res.length === 0) return;
    const v = res.sort(
      (a, b) => xy.distance(scaledPos, a.position) - xy.distance(scaledPos, b.position),
    )[0];
    const { draw } = this.internal;

    const position = s.reverse().pos(v.position);
    draw.circle({
      fill: color.setAlpha(v.color, 1),
      radius: HOVER_CIRCLE_OUTER_RADIUS,
      position,
    });
    draw.circle({
      fill: this.internal.theme.colors.gray.l3,
      radius: HOVER_CIRCLE_INNER_RADIUS,
      position,
    });
    draw.text({
      text: measureModeText(this.state.mode),
      position,
      level: "small",
      align: "middle",
      justify: "center",
    });
  }

  render(props: MeasureProps): void {
    if (this.deleted) return;
    this.renderHover(props);
    const res = this.find(props);
    if (res == null) return;
    const [oneValue, twoValue] = res;
    const { draw } = this.internal;
    const { strokeDash, strokeWidth } = this.state;
    const s = scale.XY.scale(box.DECIMAL).scale(props.region);
    const onePos = s.pos(oneValue.position);
    const twoPos = s.pos(twoValue.position);

    const xDist = new TimeSpan(Math.abs(oneValue.value.x - twoValue.value.x));
    const yDist = Math.abs(oneValue.value.y - twoValue.value.y);
    const slope = yDist / xDist.seconds;

    const xPixelDist = Math.abs(onePos.x - twoPos.x);
    const yPixelDist = Math.abs(onePos.y - twoPos.y);
    const isHorizontal = yPixelDist < PROXIMITY_THRESHOLD;
    const isVertical = xPixelDist < PROXIMITY_THRESHOLD;
    const isVeryClose =
      xPixelDist < PROXIMITY_THRESHOLD && yPixelDist < PROXIMITY_THRESHOLD;

    draw.line({
      start: xy.construct(onePos.x, onePos.y),
      end: xy.construct(onePos.x, twoPos.y),
      stroke: this.verticalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    const yValue = `${yDist.toFixed(2)} ${oneValue.units ?? ""}`;
    const yLabelPos = this.calculateYLabelPosition(
      xy.construct(onePos.x, (onePos.y + twoPos.y) / 2),
      isVeryClose,
      isVertical,
    );
    this.drawLabelValue("Y", yValue, yLabelPos, yLabelColor);

    draw.line({
      start: xy.construct(onePos.x, twoPos.y),
      end: xy.construct(twoPos.x, twoPos.y),
      stroke: this.horizontalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    const trunc = xDist.lessThan(TimeSpan.milliseconds(10))
      ? TimeSpan.MICROSECOND
      : TimeSpan.MILLISECOND;
    const xValue = xDist.truncate(trunc).toString();
    const xLabelPos = this.calculateXLabelPosition(
      xy.construct((onePos.x + twoPos.x) / 2, twoPos.y),
      isVeryClose,
      isHorizontal,
    );
    this.drawLabelValue("X", xValue, xLabelPos, xLabelColor);

    draw.line({
      start: xy.construct(onePos.x, onePos.y),
      end: xy.construct(twoPos.x, twoPos.y),
      stroke: this.obliqueLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    let slopeValue = slope.toFixed(2);
    if (oneValue.units != null && oneValue.units.length > 0)
      slopeValue += ` ${oneValue.units} / S`;
    const slopeLabelPos = this.calculateSlopeLabelPosition(
      xy.construct((onePos.x + twoPos.x) / 2, (onePos.y + twoPos.y) / 2),
      yLabelPos,
      isVeryClose,
      isHorizontal,
      xPixelDist,
      yPixelDist,
    );
    this.drawLabelValue("Slope", slopeValue, slopeLabelPos, slopeLabelColor);

    this.drawPointMarker(onePos, oneValue.color);
    this.drawPointMarker(twoPos, twoValue.color);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Measure.TYPE]: Measure,
};

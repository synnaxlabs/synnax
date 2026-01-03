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
  math,
  scale,
  TimeSpan,
  TimeStamp,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { theming } from "@/theming/aether";
import { type Theme } from "@/theming/core/theme";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

export const modeZ = z.enum(["one", "two"]);
export type Mode = z.infer<typeof modeZ>;

export const measureStateZ = z.object({
  one: xy.xy.nullable(),
  two: xy.xy.nullable(),
  hover: xy.xy.nullable(),
  mode: modeZ.default("one"),
  color: z
    .union([
      color.colorZ,
      z.object({
        verticalLine: color.colorZ.default(color.ZERO),
        horizontalLine: color.colorZ.default(color.ZERO),
        obliqueLine: color.colorZ.default(color.ZERO),
      }),
    ])
    .optional()
    .default(color.ZERO),
  strokeWidth: z.number().default(1),
  strokeDash: z.number().default(2),
});

interface InternalState {
  renderCtx: render.Context;
  theme: Theme;
  draw: Draw2D;
  dataOne: xy.XY | null;
  dataTwo: xy.XY | null;
  clickOne: xy.XY | null;
  clickTwo: xy.XY | null;
  dotColor: color.Color;
  dotColorContrast: color.Color;
}

interface PointLabelParams {
  pointNumber: number;
  position: xy.XY;
  value: xy.XY;
  units: string | null;
  bounds: bounds.Bounds;
  toTop: boolean;
  viewRegion: box.Box;
  xDist: TimeSpan;
}

interface PointLabelRowParams {
  region: box.Box;
  label: string;
  value: string;
  labelColor: color.Color;
  yOffset: number;
  leftOffset: number;
}

interface SlopeLabelPositionParams {
  basePos: xy.XY;
  yLabelPos: xy.XY;
  isVeryClose: boolean;
  isHorizontal: boolean;
  xPixelDist: number;
  yPixelDist: number;
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
const HOVER_CIRCLE_STROKE_WIDTH = 2;
const HOVER_CIRCLE_LINE_DASH = 2;

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

const POINT_LABEL_OFFSET = 15;
const POINT_LABEL_SPACING = 8;

const TIME_FORMAT_THRESHOLD_MS = 10;

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
    this.internal.dotColor = this.internal.theme.colors.text;
    this.internal.dotColorContrast = this.internal.theme.colors.textInverted;
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

  private findPoint(
    props: MeasureProps,
    clickPos: xy.XY,
    cachedData: xy.XY | null,
    cachedClick: xy.XY | null,
    s: scale.XY,
  ): FindResult | null {
    const clickChanged = cachedClick != null && !xy.equals(clickPos, cachedClick);

    if (cachedData != null && !clickChanged) {
      // We have cached data AND click hasn't changed - use cached data only
      const results = props
        .findByXValue(cachedData.x)
        .filter((r) => Number.isFinite(r.value.x) && Number.isFinite(r.value.y));
      if (results.length > 0)
        return results.sort(
          (a, b) => xy.distance(cachedData, a.value) - xy.distance(cachedData, b.value),
        )[0];
      return null;
    }

    // Either no cached data OR click changed - search by decimal position
    const scaled = s.pos(clickPos);
    const values = props
      .findByXDecimal(scaled.x)
      .filter((r) => Number.isFinite(r.value.x) && Number.isFinite(r.value.y));
    if (values.length === 0) return null;
    return values.sort(
      (a, b) => xy.distance(scaled, a.position) - xy.distance(scaled, b.position),
    )[0];
  }

  private find(props: MeasureProps): [FindResult, FindResult] | null {
    const { one, two } = this.state;
    if (one == null || two == null) return null;
    const { dataOne, dataTwo, clickOne, clickTwo } = this.internal;

    const s = scale.XY.scale(props.region).scale(box.DECIMAL);

    const oneResult = this.findPoint(props, one, dataOne, clickOne, s);
    if (oneResult == null) return null;

    // Update cache if we found a new point
    if (clickOne == null || !xy.equals(one, clickOne)) {
      this.internal.dataOne = { ...oneResult.value };
      this.internal.clickOne = { ...one };
    }

    const twoResult = this.findPoint(props, two, dataTwo, clickTwo, s);
    if (twoResult == null) return null;

    // Update cache if we found a new point
    if (clickTwo == null || !xy.equals(two, clickTwo)) {
      this.internal.dataTwo = { ...twoResult.value };
      this.internal.clickTwo = { ...two };
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

  private drawCombinedDeltaLabel(
    position: xy.XY,
    xValue: string,
    yValue: string,
    slopeValue: string,
    viewRegion: box.Box,
    centered: boolean,
  ): void {
    const { draw, theme } = this.internal;
    const padding = xy.construct(LABEL_CONTAINER_PADDING);

    const labelWidth = LABEL_CHAR_WIDTH * 5; // "Slope" is longest at 5 chars
    const maxValueLength = Math.max(xValue.length, yValue.length, slopeValue.length);
    const width =
      labelWidth + maxValueLength * VALUE_CHAR_WIDTH + padding.x * LABEL_VALUE_SPACING;
    const lineHeight = LABEL_CONTAINER_HEIGHT;
    const height = lineHeight * 3 + padding.y * 2;

    let region: box.Box;
    if (centered) {
      // Center the label around the position
      const centeredPos = xy.translate(position, [-width / 2, -height / 2]);
      region = box.construct(centeredPos, width, height);
    } else {
      // Use position as top-left corner for right-side positioning
      region = box.construct(position, width, height);

      // Check if label goes outside the view region on the right, if so flip it to the left
      if (box.right(region) > box.right(viewRegion)) {
        const flippedPos = xy.translateX(
          position,
          -width - LABEL_OFFSET_VERY_CLOSE_X * 2,
        );
        region = box.construct(flippedPos, width, height);
      }
    }

    draw.container({
      region,
      backgroundColor: (t) => t.colors.gray.l1,
    });

    // X row
    this.drawPointLabelRow({
      region,
      label: "ΔX",
      value: xValue,
      labelColor: theme.colors.error.z,
      yOffset: 0,
      leftOffset: 0,
    });

    // Y row
    this.drawPointLabelRow({
      region,
      label: "ΔY",
      value: yValue,
      labelColor: theme.colors.secondary.z,
      yOffset: lineHeight,
      leftOffset: 0,
    });

    // Slope row
    this.drawPointLabelRow({
      region,
      label: "Slope",
      value: slopeValue,
      labelColor: theme.colors.gray.l9,
      yOffset: lineHeight * 2,
      leftOffset: 0,
    });
  }

  private drawPointLabelRow(params: PointLabelRowParams): void {
    const { region, label, value, labelColor, yOffset, leftOffset } = params;
    const { draw } = this.internal;
    const padding = xy.construct(LABEL_CONTAINER_PADDING);

    draw.text({
      text: label,
      position: xy.translate(box.topLeft(region), [
        padding.x + leftOffset,
        padding.y + yOffset,
      ]),
      level: "small",
      weight: 500,
      color: labelColor,
    });
    draw.text({
      text: value,
      position: xy.translate(box.topRight(region), [
        -padding.x - 1,
        padding.y + yOffset - 1,
      ]),
      level: "small",
      justify: "right",
      code: true,
      shade: 10,
    });
  }

  private drawPointLabel(params: PointLabelParams): void {
    const { pointNumber, position, value, units, bounds, toTop, viewRegion, xDist } =
      params;
    const { draw, theme } = this.internal;
    const ts = new TimeStamp(value.x);
    const xValue = ts.toString(ts.formatBySpan(xDist), "local");
    const yValue = `${math.roundBySpan(value.y, bounds)} ${units ?? ""}`;

    const pointText = `${pointNumber}`;
    const pointTextWidth = pointText.length * LABEL_CHAR_WIDTH;

    const padding = xy.construct(LABEL_CONTAINER_PADDING);
    const labelWidth = LABEL_CHAR_WIDTH;
    const maxValueLength = Math.max(xValue.length, yValue.length);
    const width =
      pointTextWidth +
      POINT_LABEL_SPACING +
      labelWidth +
      maxValueLength * VALUE_CHAR_WIDTH +
      padding.x * LABEL_VALUE_SPACING;
    const lineHeight = LABEL_CONTAINER_HEIGHT;
    const height = lineHeight * 2 + padding.y * 2;

    // Calculate and adjust label position
    let yOffset = toTop ? -(height + POINT_LABEL_OFFSET) : POINT_LABEL_OFFSET;
    let region = box.construct(
      xy.translate(position, [-width / 2, yOffset]),
      width,
      height,
    );

    // Flip if outside view region
    if (
      (toTop && box.top(region) < box.top(viewRegion)) ||
      (!toTop && box.bottom(region) > box.bottom(viewRegion))
    ) {
      yOffset = toTop ? POINT_LABEL_OFFSET : -(height + POINT_LABEL_OFFSET);
      region = box.construct(
        xy.translate(position, [-width / 2, yOffset]),
        width,
        height,
      );
    }

    draw.container({
      region,
      backgroundColor: (t) => t.colors.gray.l1,
    });

    draw.text({
      text: pointText,
      position: xy.translate(box.topLeft(region), [padding.x, padding.y]),
      level: "small",
      weight: 500,
      color: theme.colors.gray.l9,
    });

    const labelOffset = pointTextWidth + POINT_LABEL_SPACING;
    this.drawPointLabelRow({
      region,
      label: "X",
      value: xValue,
      labelColor: theme.colors.error.z,
      yOffset: 0,
      leftOffset: labelOffset,
    });
    this.drawPointLabelRow({
      region,
      label: "Y",
      value: yValue,
      labelColor: theme.colors.secondary.z,
      yOffset: lineHeight,
      leftOffset: labelOffset,
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
      fill: color.pickByContrast(
        pointColor,
        this.internal.dotColor,
        this.internal.dotColorContrast,
      ),
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

  private calculateSlopeLabelPosition(params: SlopeLabelPositionParams): xy.XY {
    const { basePos, yLabelPos, isVeryClose, isHorizontal, xPixelDist, yPixelDist } =
      params;
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
      stroke: color.setAlpha(v.color, 1),
      strokeWidth: HOVER_CIRCLE_STROKE_WIDTH,
      lineDash: HOVER_CIRCLE_LINE_DASH,
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

    const xDistRaw = Math.abs(oneValue.value.x - twoValue.value.x);
    const yDistRaw = twoValue.value.y - oneValue.value.y;
    const yDist = Math.abs(yDistRaw);
    if (!Number.isFinite(xDistRaw) || !Number.isFinite(yDist)) return;

    const xDist = new TimeSpan(xDistRaw);
    const slope = yDistRaw / xDist.seconds;

    const xPixelDist = Math.abs(onePos.x - twoPos.x);
    const yPixelDist = Math.abs(onePos.y - twoPos.y);
    const isHorizontal = yPixelDist < PROXIMITY_THRESHOLD;
    const isVertical = xPixelDist < PROXIMITY_THRESHOLD;
    const isVeryClose =
      xPixelDist < PROXIMITY_THRESHOLD || yPixelDist < PROXIMITY_THRESHOLD;

    // Draw all lines first
    draw.line({
      start: xy.construct(onePos.x, onePos.y),
      end: xy.construct(onePos.x, twoPos.y),
      stroke: this.verticalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });

    draw.line({
      start: xy.construct(onePos.x, twoPos.y),
      end: xy.construct(twoPos.x, twoPos.y),
      stroke: this.horizontalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });

    draw.line({
      start: xy.construct(onePos.x, onePos.y),
      end: xy.construct(twoPos.x, twoPos.y),
      stroke: this.obliqueLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });

    this.drawPointMarker(onePos, oneValue.color);
    this.drawPointMarker(twoPos, twoValue.color);

    // Now draw all labels on top
    const yValue = `${math.roundBySpan(yDist, bounds.construct(yDist))} ${oneValue.units ?? ""}`;
    const trunc = xDist.lessThan(TimeSpan.milliseconds(TIME_FORMAT_THRESHOLD_MS))
      ? TimeSpan.MICROSECOND
      : TimeSpan.MILLISECOND;
    const xValue = xDist.truncate(trunc).toString();
    let slopeValue = math
      .roundBySpan(slope, bounds.construct(Math.abs(slope)))
      .toString();
    if (oneValue.units != null && oneValue.units.length > 0)
      slopeValue += ` ${oneValue.units} / s`;

    if (isVeryClose) {
      // Draw combined label when points are very close
      const centerPos = xy.construct(
        (onePos.x + twoPos.x) / 2,
        (onePos.y + twoPos.y) / 2,
      );

      // Calculate combined label dimensions
      const labelWidth = LABEL_CHAR_WIDTH * 6; // "Slope" is longest at 6 chars, "ΔX" and "ΔY" are 2 chars each
      const maxValueLength = Math.max(xValue.length, yValue.length, slopeValue.length);
      const padding = LABEL_CONTAINER_PADDING;
      const combinedWidth =
        labelWidth + maxValueLength * VALUE_CHAR_WIDTH + padding * LABEL_VALUE_SPACING;
      const combinedHeight = LABEL_CONTAINER_HEIGHT * 3 + padding * 2;

      // Calculate centered combined label region
      const centeredCombinedRegion = box.construct(
        xy.translate(centerPos, [-combinedWidth / 2, -combinedHeight / 2]),
        combinedWidth,
        combinedHeight,
      );

      // Calculate point label regions (same logic as drawPointLabel)
      const oneIsTop = onePos.y < twoPos.y;
      const pointLabelWidth = 150; // Approximate conservative estimate
      const pointLabelHeight = LABEL_CONTAINER_HEIGHT * 2 + padding * 2;

      const oneYOffset = oneIsTop
        ? -(pointLabelHeight + POINT_LABEL_OFFSET)
        : POINT_LABEL_OFFSET;
      const oneLabelRegion = box.construct(
        xy.translate(onePos, [-pointLabelWidth / 2, oneYOffset]),
        pointLabelWidth,
        pointLabelHeight,
      );

      const twoYOffset = !oneIsTop
        ? -(pointLabelHeight + POINT_LABEL_OFFSET)
        : POINT_LABEL_OFFSET;
      const twoLabelRegion = box.construct(
        xy.translate(twoPos, [-pointLabelWidth / 2, twoYOffset]),
        pointLabelWidth,
        pointLabelHeight,
      );

      // Check if centered combined label would overlap with point labels
      const overlapsOne =
        box.area(box.intersection(centeredCombinedRegion, oneLabelRegion)) > 0;
      const overlapsTwo =
        box.area(box.intersection(centeredCombinedRegion, twoLabelRegion)) > 0;
      const centerFits = !overlapsOne && !overlapsTwo;

      const combinedPos = centerFits
        ? centerPos
        : xy.construct(
            Math.max(onePos.x, twoPos.x) + LABEL_OFFSET_VERY_CLOSE_X,
            (onePos.y + twoPos.y) / 2,
          );

      this.drawCombinedDeltaLabel(
        combinedPos,
        xValue,
        yValue,
        slopeValue,
        props.region,
        centerFits,
      );
    } else {
      // Draw separate labels when points are spread out
      const yLabelPos = this.calculateYLabelPosition(
        xy.construct(onePos.x, (onePos.y + twoPos.y) / 2),
        isVeryClose,
        isVertical,
      );
      this.drawLabelValue("ΔY", yValue, yLabelPos, yLabelColor);

      const xLabelPos = this.calculateXLabelPosition(
        xy.construct((onePos.x + twoPos.x) / 2, twoPos.y),
        isVeryClose,
        isHorizontal,
      );
      this.drawLabelValue("ΔX", xValue, xy.translateX(xLabelPos, -15), xLabelColor);

      const slopeLabelPos = this.calculateSlopeLabelPosition({
        basePos: xy.construct((onePos.x + twoPos.x) / 2, (onePos.y + twoPos.y) / 2),
        yLabelPos,
        isVeryClose,
        isHorizontal,
        xPixelDist,
        yPixelDist,
      });
      this.drawLabelValue("Slope", slopeValue, slopeLabelPos, slopeLabelColor);
    }

    const oneIsTop = onePos.y < twoPos.y;
    this.drawPointLabel({
      pointNumber: 1,
      position: onePos,
      value: oneValue.value,
      units: oneValue.units ?? null,
      bounds: oneValue.bounds,
      toTop: oneIsTop,
      viewRegion: props.region,
      xDist,
    });
    this.drawPointLabel({
      pointNumber: 2,
      position: twoPos,
      value: twoValue.value,
      units: twoValue.units ?? null,
      bounds: twoValue.bounds,
      toTop: !oneIsTop,
      viewRegion: props.region,
      xDist,
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Measure.TYPE]: Measure,
};

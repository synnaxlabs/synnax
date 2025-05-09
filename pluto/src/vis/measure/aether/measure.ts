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

export const measureStateZ = z.object({
  one: xy.xy.nullable(),
  two: xy.xy.nullable(),
  hover: xy.xy.nullable(),
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
    render.Controller.requestRender(ctx, render.REASON_TOOL);
  }

  afterDelete(ctx: aether.Context): void {
    render.Controller.requestRender(ctx, render.REASON_LAYOUT);
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
    if (
      prevOne != null &&
      xy.equals(one, prevOne) &&
      prevTwo != null &&
      xy.equals(two, prevTwo) &&
      dataOne != null &&
      dataTwo != null
    ) {
      const [one, two] = [props.findByXValue(dataOne.x), props.findByXValue(dataTwo.x)];
      if (one.length === 0 || two.length === 0) return null;
      return [
        one.sort(
          (a, b) => xy.distance(dataOne, a.value) - xy.distance(dataOne, b.value),
        )[0],
        two.sort(
          (a, b) => xy.distance(dataTwo, a.value) - xy.distance(dataTwo, b.value),
        )[0],
      ];
    }
    const s = scale.XY.scale(props.region).scale(box.DECIMAL);
    const [scaledOne, scaledTwo] = [s.pos(one), s.pos(two)];
    const [oneValues, twoValues] = [
      props.findByXDecimal(scaledOne.x),
      props.findByXDecimal(scaledTwo.x),
    ];
    if (oneValues.length === 0 || twoValues.length === 0) return null;
    const [oneValue, twoValue] = [
      oneValues.sort(
        (a, b) =>
          xy.distance(scaledOne, a.position) - xy.distance(scaledOne, b.position),
      )[0],
      twoValues.sort(
        (a, b) =>
          xy.distance(scaledTwo, a.position) - xy.distance(scaledTwo, b.position),
      )[0],
    ];
    this.internal.dataOne = oneValue.value;
    this.internal.dataTwo = twoValue.value;
    return [oneValue, twoValue];
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

    draw.circle({
      fill: color.setAlpha(v.color, 0.5),
      radius: 9,
      position: s.reverse().pos(v.position),
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
    draw.line({
      start: xy.construct(onePos.x, onePos.y),
      end: xy.construct(onePos.x, twoPos.y),
      stroke: this.verticalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    draw.textContainer({
      text: [`${yDist.toFixed(2)} ${oneValue.units ?? ""}`],
      direction: "x",
      position: xy.construct(onePos.x, (onePos.y + twoPos.y) / 2),
      level: "small",
    });
    draw.line({
      start: xy.construct(onePos.x, twoPos.y),
      end: xy.construct(twoPos.x, twoPos.y),
      stroke: this.horizontalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    draw.textContainer({
      text: [xDist.truncate(TimeSpan.MILLISECOND).toString()],
      direction: "x",
      position: xy.construct((onePos.x + twoPos.x) / 2, twoPos.y),
      level: "small",
    });
    draw.line({
      start: xy.construct(onePos.x, onePos.y),
      end: xy.construct(twoPos.x, twoPos.y),
      stroke: this.obliqueLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    draw.textContainer({
      text: [`${slope.toFixed(2)} ${oneValue.units ?? ""} / S`],
      direction: "x",
      position: xy.construct((onePos.x + twoPos.x) / 2, (onePos.y + twoPos.y) / 2),
      level: "small",
    });
    draw.circle({
      fill: color.setAlpha(oneValue.color, 0.5),
      radius: 8,
      position: onePos,
    });
    draw.circle({
      fill: color.setAlpha(oneValue.color, 0.8),
      radius: 5,
      position: onePos,
    });
    draw.circle({ fill: color.construct("#ffffff"), radius: 2, position: onePos });

    draw.circle({
      fill: color.setAlpha(twoValue.color, 0.5),
      radius: 8,
      position: twoPos,
    });
    draw.circle({
      fill: color.setAlpha(twoValue.color, 0.8),
      radius: 5,
      position: twoPos,
    });
    draw.circle({ fill: color.construct("#ffffff"), radius: 2, position: twoPos });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Measure.TYPE]: Measure,
};

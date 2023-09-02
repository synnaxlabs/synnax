// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XYScale, Direction, TimeSpan, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import { theming } from "@/theming/aether";
import { type Theme } from "@/theming/core/theme";
import { Draw2D } from "@/vis/draw2d";
import { type FindResult } from "@/vis/line/aether/line";
import { render } from "@/vis/render";

export const measureStateZ = z.object({
  one: XY.z.nullable(),
  two: XY.z.nullable(),
  hover: XY.z.nullable(),
  color: z
    .union([
      color.Color.z,
      z.object({
        verticalLine: color.Color.z.optional().default(color.ZERO),
        horizontalLine: color.Color.z.optional().default(color.ZERO),
        obliqueLine: color.Color.z.optional().default(color.ZERO),
      }),
    ])
    .optional()
    .default(color.ZERO),
  strokeWidth: z.number().optional().default(1),
  strokeDash: z.number().optional().default(2),
});

interface InternalState {
  render: render.Context;
  theme: Theme;
  draw: Draw2D;
  dataOne: XY | null;
  dataTwo: XY | null;
}

export interface MeasureProps {
  findByXDecimal: (target: number) => Promise<FindResult[]>;
  findByXValue: (target: number) => Promise<FindResult[]>;
  region: Box;
}

export class Measure extends aether.Leaf<typeof measureStateZ, InternalState> {
  static readonly TYPE = "measure";
  schema = measureStateZ;

  afterUpdate(): void {
    const ctx = render.Context.use(this.ctx);
    this.internal.theme = theming.use(this.ctx);
    this.internal.render = ctx;
    this.internal.draw = new Draw2D(ctx.upper2d, this.internal.theme);
    render.Controller.requestRender(this.ctx);
  }

  afterDelete(): void {
    render.Controller.requestRender(this.ctx);
  }

  get verticalLineColor(): color.Color {
    if (this.state.color instanceof color.Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.p0;
    }

    if (!this.state.color.verticalLine.isZero) return this.state.color.verticalLine;
    return this.internal.theme.colors.gray.p0;
  }

  get horizontalLineColor(): color.Color {
    if (this.state.color instanceof color.Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.p0;
    }
    if (!this.state.color.horizontalLine.isZero) return this.state.color.horizontalLine;
    return this.internal.theme.colors.gray.p0;
  }

  get obliqueLineColor(): color.Color {
    if (this.state.color instanceof color.Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.p0;
    }
    if (!this.state.color.obliqueLine.isZero) return this.state.color.obliqueLine;
    return this.internal.theme.colors.gray.p0;
  }

  async find(props: MeasureProps): Promise<[FindResult, FindResult] | null> {
    const { one, two } = this.state;
    if (one == null || two == null) return null;
    const { one: prevOne, two: prevTwo } = this.prevState;
    const { dataOne, dataTwo } = this.internal;
    if (
      prevOne != null &&
      one.equals(prevOne) &&
      prevTwo != null &&
      two.equals(prevTwo) &&
      dataOne != null &&
      dataTwo != null
    ) {
      const [one, two] = [
        await props.findByXValue(dataOne.x),
        await props.findByXValue(dataTwo.x),
      ];
      if (one.length === 0 || two.length === 0) return null;
      return [
        one.sort(
          (a, b) => dataOne.distanceTo(a.value) - dataOne.distanceTo(b.value),
        )[0],
        two.sort(
          (a, b) => dataTwo.distanceTo(a.value) - dataTwo.distanceTo(b.value),
        )[0],
      ];
    }
    const scale = XYScale.scale(props.region).scale(Box.DECIMAL);
    const [scaledOne, scaledTwo] = [scale.pos(one), scale.pos(two)];
    const [oneValues, twoValues] = [
      await props.findByXDecimal(scaledOne.x),
      await props.findByXDecimal(scaledTwo.x),
    ];
    if (oneValues.length === 0 || twoValues.length === 0) return null;
    const [oneValue, twoValue] = [
      oneValues.sort(
        (a, b) => scaledOne.distanceTo(a.position) - scaledOne.distanceTo(b.position),
      )[0],
      twoValues.sort(
        (a, b) => scaledTwo.distanceTo(a.position) - scaledTwo.distanceTo(b.position),
      )[0],
    ];
    this.internal.dataOne = oneValue.value;
    this.internal.dataTwo = twoValue.value;
    return [oneValue, twoValue];
  }

  async renderHover(props: MeasureProps): Promise<void> {
    if (this.state.hover == null) return;
    const hover: XY = this.state.hover;

    const scale = XYScale.scale(props.region).scale(Box.DECIMAL);
    const scaledPos = scale.pos(hover);
    const res = await props.findByXDecimal(scale.pos(hover).x);
    if (res.length === 0) return;
    const v = res.sort(
      (a, b) => scaledPos.distanceTo(a.position) - scaledPos.distanceTo(b.position),
    )[0];
    const { draw } = this.internal;

    draw.circle({
      fill: v.color.setAlpha(0.5),
      radius: 9,
      position: scale.reverse().pos(v.position),
    });
  }

  async render(props: MeasureProps): Promise<void> {
    if (this.deleted) return;
    await this.renderHover(props);
    const res = await this.find(props);
    if (res == null) return;
    const [oneValue, twoValue] = res;
    const { draw } = this.internal;
    const { strokeDash, strokeWidth } = this.state;
    const scale = XYScale.scale(Box.DECIMAL).scale(props.region);
    const onePos = scale.pos(oneValue.position);
    const twoPos = scale.pos(twoValue.position);
    const xDist = new TimeSpan(Math.abs(oneValue.value.x - twoValue.value.x));
    const yDist = Math.abs(oneValue.value.y - twoValue.value.y);
    const slope = yDist / xDist.seconds;
    draw.line({
      start: new XY(onePos.x, onePos.y),
      end: new XY(onePos.x, twoPos.y),
      stroke: this.verticalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    draw.textContainer({
      text: [`${yDist.toFixed(2)} ${oneValue.units ?? ""}`],
      direction: Direction.X,
      position: new XY(onePos.x, (onePos.y + twoPos.y) / 2),
      level: "small",
    });
    draw.line({
      start: new XY(onePos.x, twoPos.y),
      end: new XY(twoPos.x, twoPos.y),
      stroke: this.horizontalLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    draw.textContainer({
      text: [xDist.truncate(TimeSpan.MILLISECOND).toString()],
      direction: Direction.X,
      position: new XY((onePos.x + twoPos.x) / 2, twoPos.y),
      level: "small",
    });
    draw.line({
      start: new XY(onePos.x, onePos.y),
      end: new XY(twoPos.x, twoPos.y),
      stroke: this.obliqueLineColor,
      lineDash: strokeDash,
      lineWidth: strokeWidth,
    });
    draw.textContainer({
      text: [`${slope.toFixed(2)} ${oneValue.units ?? ""} / S`],
      direction: Direction.X,
      position: new XY((onePos.x + twoPos.x) / 2, (onePos.y + twoPos.y) / 2),
      level: "small",
    });
    draw.circle({ fill: oneValue.color.setAlpha(0.5), radius: 8, position: onePos });
    draw.circle({ fill: oneValue.color.setAlpha(0.8), radius: 5, position: onePos });
    draw.circle({ fill: new color.Color("#ffffff"), radius: 2, position: onePos });

    draw.circle({ fill: twoValue.color.setAlpha(0.5), radius: 8, position: twoPos });
    draw.circle({ fill: twoValue.color.setAlpha(0.8), radius: 5, position: twoPos });
    draw.circle({ fill: new color.Color("#ffffff"), radius: 2, position: twoPos });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Measure.TYPE]: Measure,
};

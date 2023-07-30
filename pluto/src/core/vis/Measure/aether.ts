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

import { AetherLeaf } from "@/core/aether/worker";
import { Color } from "@/core/color";
import { ThemeContext } from "@/core/theming/aether";
import { Theme } from "@/core/theming/theme";
import { Draw2D } from "@/core/vis/draw2d";
import { FindResult } from "@/core/vis/Line/aether";
import { RenderContext, RenderController } from "@/core/vis/render";

const measureState = z.object({
  one: XY.z.nullable(),
  two: XY.z.nullable(),
  hover: XY.z.nullable(),
  color: z
    .union([
      Color.z,
      z.object({
        verticalLine: Color.z.optional().default(Color.ZERO),
        horizontalLine: Color.z.optional().default(Color.ZERO),
        obliqueLine: Color.z.optional().default(Color.ZERO),
      }),
    ])
    .optional()
    .default(Color.ZERO),
  strokeWidth: z.number().optional().default(1),
  strokeDash: z.number().optional().default(2),
});

interface InternalState {
  render: RenderContext;
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

export class AetherMeasure extends AetherLeaf<typeof measureState, InternalState> {
  static readonly TYPE = "measure";
  static readonly stateZ = measureState;
  schema = AetherMeasure.stateZ;

  afterUpdate(): void {
    const ctx = RenderContext.use(this.ctx);
    this.internal.theme = ThemeContext.use(this.ctx);
    this.internal.render = ctx;
    this.internal.draw = new Draw2D(ctx.upper2d, this.internal.theme);
    RenderController.requestRender(this.ctx);
  }

  get verticalLineColor(): Color {
    if (this.state.color instanceof Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.p0;
    }

    if (!this.state.color.verticalLine.isZero) return this.state.color.verticalLine;
    return this.internal.theme.colors.gray.p0;
  }

  get horizontalLineColor(): Color {
    if (this.state.color instanceof Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.p0;
    }
    if (!this.state.color.horizontalLine.isZero) return this.state.color.horizontalLine;
    return this.internal.theme.colors.gray.p0;
  }

  get obliqueLineColor(): Color {
    if (this.state.color instanceof Color) {
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
          (a, b) => dataOne.distanceTo(a.value) - dataOne.distanceTo(b.value)
        )[0],
        two.sort(
          (a, b) => dataTwo.distanceTo(a.value) - dataTwo.distanceTo(b.value)
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
        (a, b) => scaledOne.distanceTo(a.position) - scaledOne.distanceTo(b.position)
      )[0],
      twoValues.sort(
        (a, b) => scaledTwo.distanceTo(a.position) - scaledTwo.distanceTo(b.position)
      )[0],
    ];
    this.internal.dataOne = oneValue.value;
    this.internal.dataTwo = twoValue.value;
    return [oneValue, twoValue];
  }

  async render(props: MeasureProps): Promise<void> {
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
      text: [(Math.trunc(yDist * 100) / 100).toString()],
      direction: Direction.Y,
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
      text: [xDist.truncate(TimeSpan.NANOSECOND).toString()],
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
      text: [slope.toString()],
      direction: Direction.X,
      position: new XY((onePos.x + twoPos.x) / 2, (onePos.y + twoPos.y) / 2),
      level: "small",
    });
  }
}

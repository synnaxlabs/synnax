// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, scale, TimeSpan, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
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
  dataOne: xy.XY | null;
  dataTwo: xy.XY | null;
}

export interface MeasureProps {
  findByXDecimal: (target: number) => Promise<FindResult[]>;
  findByXValue: (target: number) => Promise<FindResult[]>;
  region: box.Box;
}

export class Measure extends aether.Leaf<typeof measureStateZ, InternalState> {
  static readonly TYPE = "measure";
  schema = measureStateZ;

  async afterUpdate(): Promise<void> {
    const ctx = render.Context.use(this.ctx);
    this.internal.theme = theming.use(this.ctx);
    this.internal.render = ctx;
    this.internal.draw = new Draw2D(ctx.upper2d, this.internal.theme);
    render.Controller.requestRender(this.ctx, render.REASON_TOOL);
  }

  async afterDelete(): Promise<void> {
    render.Controller.requestRender(this.ctx, render.REASON_LAYOUT);
  }

  private get verticalLineColor(): color.Color {
    if (this.state.color instanceof color.Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.l6;
    }

    if (!this.state.color.verticalLine.isZero) return this.state.color.verticalLine;
    return this.internal.theme.colors.gray.l6;
  }

  private get horizontalLineColor(): color.Color {
    if (this.state.color instanceof color.Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.l6;
    }
    if (!this.state.color.horizontalLine.isZero) return this.state.color.horizontalLine;
    return this.internal.theme.colors.gray.l6;
  }

  private get obliqueLineColor(): color.Color {
    if (this.state.color instanceof color.Color) {
      if (!this.state.color.isZero) return this.state.color;
      return this.internal.theme.colors.gray.l6;
    }
    if (!this.state.color.obliqueLine.isZero) return this.state.color.obliqueLine;
    return this.internal.theme.colors.gray.l6;
  }

  private async find(props: MeasureProps): Promise<[FindResult, FindResult] | null> {
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
      const [one, two] = [
        await props.findByXValue(dataOne.x),
        await props.findByXValue(dataTwo.x),
      ];
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
      await props.findByXDecimal(scaledOne.x),
      await props.findByXDecimal(scaledTwo.x),
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

  private async renderHover(props: MeasureProps): Promise<void> {
    if (this.state.hover == null) return;
    const hover: xy.XY = this.state.hover;

    const s = scale.XY.scale(props.region).scale(box.DECIMAL);
    const scaledPos = s.pos(hover);
    const res = await props.findByXDecimal(s.pos(hover).x);
    if (res.length === 0) return;
    const v = res.sort(
      (a, b) => xy.distance(scaledPos, a.position) - xy.distance(scaledPos, b.position),
    )[0];
    const { draw } = this.internal;

    draw.circle({
      fill: v.color.setAlpha(0.5),
      radius: 9,
      position: s.reverse().pos(v.position),
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

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, location, scale, xy } from "@synnaxlabs/x";

import { line } from "@/vis/line/aether";
import {
  type AxisRenderProps,
  CoreAxis,
  coreAxisStateZ,
} from "@/vis/lineplot/aether/axis";
import { rule } from "@/vis/rule/aether";

export const yAxisStateZ = coreAxisStateZ.extend({
  location: location.x.optional().default("left"),
});

export interface YAxisProps extends AxisRenderProps {
  xDataToDecimalScale: scale.Scale;
  exposure: number;
}

type Children = line.Line | rule.Rule;

const INVALID_SIZE_THRESHOLD = 2; // px;

// There are certain cases where the plot box is too small or completely
// negative. In these cases there is no visual area to render to the user,
// so we can skip rendering the lines.
const invalidArea = (region: box.Box): boolean =>
  box.signedWidth(region) < INVALID_SIZE_THRESHOLD ||
  box.signedHeight(region) < INVALID_SIZE_THRESHOLD;

export class YAxis extends CoreAxis<typeof coreAxisStateZ, Children> {
  static readonly TYPE = "YAxis";
  schema = coreAxisStateZ;

  async xBounds(): Promise<bounds.Bounds> {
    return bounds.max(
      (await Promise.all(this.lines.map(async (el) => await el.xBounds()))).filter(
        (b) => bounds.isFinite(b),
      ),
    );
  }

  async render(props: YAxisProps): Promise<void> {
    if (this.deleted) return;
    const [dataToDecimalScale, error] = await this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    // We need to invert scale because the y-axis is inverted in decimal space.
    const decimalToDataScale = dataToDecimalScale.invert().reverse();
    this.renderAxis(props, decimalToDataScale);
    await this.renderLines(props, dataToDecimalScale);
    await this.renderRules(props, decimalToDataScale);
    // Throw the error we encounter here so that the user still has a visible axis.
    if (error != null) throw error;
  }

  private async renderLines(
    { xDataToDecimalScale: xScale, plot, canvases, exposure }: YAxisProps,
    yScale: scale.Scale,
  ): Promise<void> {
    if (!canvases.includes("gl") || invalidArea(plot)) return;
    const props: line.LineProps = {
      region: plot,
      dataToDecimalScale: new scale.XY(xScale, yScale),
      exposure,
    };
    await Promise.all(this.lines.map(async (el) => await el.render(props)));
  }

  private async renderRules(
    { container, plot, canvases }: YAxisProps,
    decimalToDataScale: scale.Scale,
  ): Promise<void> {
    if (!canvases.includes("upper2d")) return;
    const { location } = this.state;
    const { render } = this.internal;
    const scissor = box.construct(
      box.left(container),
      box.top(plot),
      box.width(container),
      box.height(plot),
    );
    const clearScissor = render.scissor(scissor, xy.ZERO, ["upper2d"]);
    const props = { container, plot, decimalToDataScale, location };
    await Promise.all(this.rules.map(async (el) => await el.render(props)));
    clearScissor();
  }

  async findByXValue(
    {
      xDataToDecimalScale,
      plot,
      viewport,
      hold,
      exposure,
    }: Omit<YAxisProps, "canvases">,
    target: number,
  ): Promise<line.FindResult[]> {
    const [yDataToDecimalScale, error] = await this.dataToDecimalScale(
      hold,
      this.dataBounds.bind(this),
      viewport,
    );
    if (error != null) throw error;
    const dataToDecimalScale = new scale.XY(xDataToDecimalScale, yDataToDecimalScale);
    const props: line.LineProps = { region: plot, dataToDecimalScale, exposure };
    return (
      await Promise.all(
        this.lines.map(async (el) => await el.findByXValue(props, target)),
      )
    ).map((v) => ({ ...v, units: this.state.label }));
  }

  private async dataBounds(): Promise<bounds.Bounds[]> {
    return await Promise.all(this.lines.map(async (el) => await el.yBounds()));
  }

  private get lines(): readonly line.Line[] {
    return this.childrenOfType(line.Line.TYPE);
  }

  private get rules(): readonly rule.Rule[] {
    return this.childrenOfType(rule.Rule.TYPE);
  }
}

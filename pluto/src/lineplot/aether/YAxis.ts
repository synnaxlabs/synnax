// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, box, direction, location, scale, xy } from "@synnaxlabs/x";

import { type AxisRenderProps, BaseAxis, baseAxisStateZ } from "@/lineplot/aether/axis";
import type { Layout } from "@/lineplot/aether/layout";
import { rule } from "@/lineplot/rule/aether";
import { line } from "@/vis/line/aether";

export const yAxisStateZ = baseAxisStateZ.extend({
  location: location.xZ.default("left"),
});

export interface YAxisProps extends AxisRenderProps {
  /** The layout of the parent x axis. Y bounds cover only samples inside its bounds. */
  layout: Layout;
}

type Children = line.Line | rule.Rule;

const INVALID_SIZE_THRESHOLD = 2; // px;

// There are certain cases where the plot box is too small or completely
// negative. In these cases there is no visual area to render to the user,
// so we can skip rendering the lines.
const invalidArea = (region: box.Box): boolean =>
  box.signedWidth(region) < INVALID_SIZE_THRESHOLD ||
  box.signedHeight(region) < INVALID_SIZE_THRESHOLD;

export class YAxis extends BaseAxis<typeof baseAxisStateZ, Children> {
  static readonly TYPE = "YAxis";
  override schema = baseAxisStateZ;

  xBounds(): line.DataBounds[] {
    return this.visibleLines.map((el) => el.xBounds());
  }

  bounds(hold: boolean, xBounds: bounds.Bounds): bounds.Bounds {
    const [bound, err] = this.iBounds(hold, () => this.dataBounds(xBounds));
    if (err != null) throw err;
    return bound;
  }

  /**
   * Draws the axis, its rules, and its lines that are not raised. Adds a draw of each
   * raised line to raised, for the plot to run after every axis draws.
   */
  render(props: YAxisProps, raised: Array<() => void>): void {
    if (this.deleted) return;
    const [dataToDecimalScale, error] = this.dataToDecimalScale(props);
    // We need to invert scale because the y-axis is inverted in decimal space.
    const decimalToDataScale = dataToDecimalScale.invert().reverse();
    this.renderAxis(props, decimalToDataScale);
    this.renderLines(props, dataToDecimalScale, raised);
    this.renderRules(props, decimalToDataScale);

    // Throw the error we encounter here so that the user still has a visible axis.
    if (error != null) throw error;
  }

  private renderLines(
    { layout, plot, canvases }: YAxisProps,
    yScale: scale.Scale,
    raised: Array<() => void>,
  ): void {
    if (!canvases.includes("gl") || invalidArea(plot)) return;
    const props: line.LineProps = {
      region: plot,
      dataToDecimalScale: new scale.XY(layout.segments[0].scale, yScale),
    };
    this.lines.forEach((el) => {
      if (el.raised) raised.push(() => el.render(props));
      else el.render(props);
    });
  }

  private renderRules(
    { container, plot, canvases }: YAxisProps,
    decimalToDataScale: scale.Scale,
  ): void {
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
    this.rules.map((el) => el.render(props));
    clearScissor();
  }

  findByXValue(props: Omit<YAxisProps, "canvases">, target: number): line.FindResult[] {
    const p = this.lineProps(props);
    return this.lines.map((el) => ({
      ...el.findByXValue(p, target),
      units: this.state.label,
    }));
  }

  /** @returns the point each line drew at its last render whose x is nearest target. */
  findDrawnByXValue(
    props: Omit<YAxisProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const p = this.lineProps(props);
    return this.lines.map((el) => ({
      ...el.findDrawnByXValue(p, target),
      units: this.state.label,
    }));
  }

  private lineProps(props: Omit<YAxisProps, "canvases">): line.LineProps {
    const [yDataToDecimalScale, error] = this.dataToDecimalScale(props);
    if (error != null) throw error;
    const xScale = props.layout.segments[0].scale;
    return {
      region: props.plot,
      dataToDecimalScale: new scale.XY(xScale, yDataToDecimalScale),
    };
  }

  // dataToDecimalScale returns the scale from the axis values to decimal positions
  // across the viewport, with the error that the bounds of the axis met.
  private dataToDecimalScale({
    hold,
    layout,
    viewport,
  }: Omit<YAxisProps, "canvases">): [scale.Scale, Error | null] {
    const [b, err] = this.iBounds(hold, () => this.dataBounds(layout.bounds));
    const dir = direction.swap(direction.construct(this.state.location));
    return [
      scale.Scale.scale<number>(b)
        .scale(1)
        .translate(-box.root(viewport)[dir])
        .magnify(1 / box.dim(viewport, dir)),
      err,
    ];
  }

  get loading(): boolean {
    return this.lines.some((el) => el.loading);
  }

  get fetching(): boolean {
    return this.lines.some((el) => el.fetching);
  }

  private dataBounds(xBounds: bounds.Bounds): line.DataBounds[] {
    return this.visibleLines.map((el) => el.yBounds(xBounds));
  }

  private get lines(): readonly line.Line[] {
    return this.childrenOfType(line.Line.TYPE);
  }

  private get visibleLines(): readonly line.Line[] {
    return this.lines.filter((el) => el.state.visible);
  }

  private get rules(): readonly rule.Rule[] {
    return this.childrenOfType(rule.Rule.TYPE);
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, box, direction, location, scale, xy } from "@synnaxlabs/x";

import { type AxisRenderProps, BaseAxis, baseAxisStateZ } from "@/lineplot/aether/axis";
import type { Layout, Trace } from "@/lineplot/aether/layout";
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

// lineProps returns the props of a line by its key. A line draws with the scale of its
// segment.
const lineProps = (
  layout: Layout,
  region: box.Box,
  yScale: scale.Scale,
): ((key: string) => line.LineProps) => {
  const scales = layout.segments.map((s) => new scale.XY(s.scale, yScale));
  return (key) => ({
    region,
    dataToDecimalScale: scales[layout.segments.indexOf(layout.segment(key))],
    span: layout.span(key),
  });
};

export class YAxis extends BaseAxis<typeof baseAxisStateZ, Children> {
  static readonly TYPE = "YAxis";
  override schema = baseAxisStateZ;

  /** @returns the window of each visible line. */
  traces(): Trace[] {
    return this.visibleLines.map((el) => ({
      ...el.xBounds(),
      key: el.key,
      group: el.state.group,
    }));
  }

  bounds(hold: boolean, layout: Layout): bounds.Bounds {
    const [bound, err] = this.iBounds(hold, () => this.dataBounds(layout));
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
    const propsOf = lineProps(layout, plot, yScale);
    this.lines.forEach((el) => {
      const props = propsOf(el.key);
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
    const propsOf = lineProps(props.layout, props.plot, this.yScale(props));
    return this.lines.map((el) => ({
      ...el.findByXValue(propsOf(el.key), target),
      units: this.state.label,
    }));
  }

  /** @returns the point each line drew at its last render whose x is nearest target. */
  findDrawnByXValue(
    props: Omit<YAxisProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const propsOf = lineProps(props.layout, props.plot, this.yScale(props));
    return this.lines.map((el) => ({
      ...el.findDrawnByXValue(propsOf(el.key), target),
      units: this.state.label,
    }));
  }

  private yScale(props: Omit<YAxisProps, "canvases">): scale.Scale {
    const [s, error] = this.dataToDecimalScale(props);
    if (error != null) throw error;
    return s;
  }

  // dataToDecimalScale returns the scale from the axis values to decimal positions
  // across the viewport, with the error that the bounds of the axis met.
  private dataToDecimalScale({
    hold,
    layout,
    viewport,
  }: Omit<YAxisProps, "canvases">): [scale.Scale, Error | null] {
    const [b, err] = this.iBounds(hold, () => this.dataBounds(layout));
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

  // dataBounds returns the y bounds of each visible line over its span inside the
  // bounds of the x axis. A line with no time there gives none.
  private dataBounds(layout: Layout): line.DataBounds[] {
    return this.visibleLines.flatMap((el) => {
      const window = bounds.intersection([layout.span(el.key), layout.bounds]);
      return window.lower < window.upper ? [el.yBounds(window)] : [];
    });
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

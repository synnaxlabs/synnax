// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type box, TimeRange, TimeStamp } from "@synnaxlabs/x";

import {
  type AxisRenderProps,
  BaseAxis,
  baseAxisStateZ,
  newestSample,
} from "@/lineplot/aether/axis";
import { Layout } from "@/lineplot/aether/layout";
import { YAxis, type YAxisProps } from "@/lineplot/aether/YAxis";
import { range } from "@/lineplot/range/aether";
import type { line } from "@/vis/line/aether";

export const xAxisStateZ = baseAxisStateZ;

export class XAxis extends BaseAxis<typeof baseAxisStateZ, YAxis | range.Provider> {
  static readonly TYPE = "XAxis";
  override schema = baseAxisStateZ;

  /**
   * Draws the axis, its y axes, and its ranges. Each y axis adds a draw of each of its
   * raised lines to raised.
   */
  render(props: AxisRenderProps, raised: Array<() => void>): void {
    if (this.deleted) return;
    const [layout, err] = this.iLayout(props.hold, props.viewport);
    this.renderAxis(props, layout.segments[0].scale.reverse());
    this.renderYAxes({ ...props, layout }, raised);
    this.renderRanges(props, layout);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
  }

  findByXDecimal(
    props: Omit<AxisRenderProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const layout = this.layout(props.hold, props.viewport);
    return this.findByXValue(props, layout.reverse(target));
  }

  findByXValue(
    props: Omit<AxisRenderProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const p = this.yAxisProps(props);
    return this.yAxes.map((el) => el.findByXValue(p, target)).flat();
  }

  /**
   * @returns the point each line drew at its last render whose x is nearest target, a
   * decimal position across the axis.
   */
  findDrawnByXDecimal(
    props: Omit<AxisRenderProps, "canvases">,
    target: number,
  ): line.FindResult[] {
    const p = this.yAxisProps(props);
    const value = p.layout.reverse(target);
    return this.yAxes.map((el) => el.findDrawnByXValue(p, value)).flat();
  }

  /**
   * @returns the values that a span of decimal positions across the home view holds.
   */
  values(hold: boolean, viewport: box.Box, positions: bounds.Bounds): bounds.Bounds {
    const layout = this.layout(hold, viewport);
    return {
      lower: layout.reverseHome(positions.lower),
      upper: layout.reverseHome(positions.upper),
    };
  }

  private yAxisProps(
    props: Omit<AxisRenderProps, "canvases">,
  ): Omit<YAxisProps, "canvases"> {
    return { ...props, layout: this.layout(props.hold, props.viewport) };
  }

  private renderYAxes(props: YAxisProps, raised: Array<() => void>): void {
    this.yAxes.forEach((el) => el.render(props, raised));
  }

  get yAxes(): readonly YAxis[] {
    return this.childrenOfType<YAxis>(YAxis.TYPE);
  }

  get ranges(): readonly range.Provider[] {
    return this.childrenOfType<range.Provider>(range.Provider.TYPE);
  }

  get loading(): boolean {
    return this.yAxes.some((el) => el.loading);
  }

  get fetching(): boolean {
    return this.yAxes.some((el) => el.fetching);
  }

  bounds(hold: boolean): bounds.Bounds {
    const [bound, err] = this.iBounds(hold, this.dataBounds.bind(this));
    if (err != null) throw err;
    return bound;
  }

  private layout(hold: boolean, viewport: box.Box): Layout {
    const [layout, err] = this.iLayout(hold, viewport);
    if (err != null) throw err;
    return layout;
  }

  // iLayout returns the layout with the error that its bounds met, so a render can draw
  // the axis before it throws.
  private iLayout(hold: boolean, viewport: box.Box): [Layout, Error | null] {
    const [b, err] = this.iBounds(hold, this.dataBounds.bind(this));
    return [new Layout([b], viewport), err];
  }

  private renderRanges(props: AxisRenderProps, layout: Layout): void {
    const clampedBounds = bounds.intersection([
      layout.bounds,
      TimeRange.MAX.numericBounds,
    ]);
    const timeRange = new TimeRange(clampedBounds.lower, clampedBounds.upper);
    const newest = newestSample(this.dataBounds());
    this.ranges.forEach((el) =>
      el.render({
        layout,
        region: props.plot,
        timeRange,
        newest: newest == null ? undefined : new TimeStamp(newest),
      }),
    );
  }

  private dataBounds(): line.DataBounds[] {
    return this.yAxes.flatMap((el) => el.xBounds());
  }
}

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type scale, TimeRange } from "@synnaxlabs/x";

import { type AxisRenderProps, BaseAxis, baseAxisStateZ } from "@/lineplot/aether/axis";
import { YAxis } from "@/lineplot/aether/YAxis";
import { range } from "@/lineplot/range/aether";
import { type FindResult } from "@/vis/line/aether/line";

export const xAxisStateZ = baseAxisStateZ;

export interface XAxisRenderProps extends AxisRenderProps {
  exposure: number;
}

export class XAxis extends BaseAxis<typeof baseAxisStateZ, YAxis | range.Provider> {
  static readonly TYPE = "XAxis";
  schema = baseAxisStateZ;

  render(props: XAxisRenderProps): void {
    if (this.deleted) return;
    const [dataToDecimal, xBounds, err] = this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    this.renderAxis(props, dataToDecimal.reverse());
    this.renderYAxes(props, dataToDecimal, xBounds);
    this.renderRanges(props, dataToDecimal);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
  }

  findByXDecimal(
    props: Omit<XAxisRenderProps, "canvases">,
    target: number,
  ): FindResult[] {
    const [scale, , err] = this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    if (err != null) throw err;
    return this.findByXValue(props, scale.reverse().pos(target));
  }

  findByXValue(
    props: Omit<XAxisRenderProps, "canvases">,
    target: number,
  ): FindResult[] {
    const [xDataToDecimalScale, xBounds, error] = this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    if (error != null) throw error;
    const p = { ...props, xDataToDecimalScale, xBounds };
    const { type: xType, label: xUnits } = this.state;
    return this.yAxes
      .map((el) => el.findByXValue(p, target))
      .flat()
      .map((r) => ({ ...r, xType, xUnits }));
  }

  /** @returns the x range the viewport shows. */
  visibleBounds(props: Omit<XAxisRenderProps, "canvases">): bounds.Bounds {
    const [scale, , err] = this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    if (err != null) throw err;
    const reverse = scale.reverse();
    return bounds.construct(reverse.pos(0), reverse.pos(1));
  }

  private renderYAxes(
    props: XAxisRenderProps,
    xDataToDecimalScale: scale.Scale,
    xBounds: bounds.Bounds,
  ): void {
    const p = { ...props, xDataToDecimalScale, xBounds };
    this.yAxes.forEach((el) => el.render(p));
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

  bounds(hold: boolean): bounds.Bounds {
    const [bound, err] = this.iBounds(hold, this.dataBounds.bind(this));
    if (err != null) throw err;
    return bound;
  }

  // Ranges are time spans, so they draw only on a time axis.
  private renderRanges(
    props: XAxisRenderProps,
    xDataToDecimalScale: scale.Scale,
  ): void {
    if (this.state.type !== "time") return;
    const bound = this.bounds(props.hold);
    const clampedBounds = bounds.min([bound, TimeRange.MAX.numericBounds]);
    const timeRange = new TimeRange(clampedBounds.lower, clampedBounds.upper);
    this.ranges.forEach((el) =>
      el.render({
        dataToDecimalScale: xDataToDecimalScale,
        region: props.plot,
        viewport: props.viewport,
        timeRange,
      }),
    );
  }

  private dataBounds(): bounds.Bounds[] {
    return this.yAxes.map((el) => el.xBounds());
  }
}

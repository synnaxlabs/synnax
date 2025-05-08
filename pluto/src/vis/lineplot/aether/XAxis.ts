// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, type scale, TimeRange } from "@synnaxlabs/x";

import { type FindResult } from "@/vis/line/aether/line";
import {
  type AxisRenderProps,
  CoreAxis,
  coreAxisStateZ,
} from "@/vis/lineplot/aether/axis";
import { YAxis } from "@/vis/lineplot/aether/YAxis";
import { range } from "@/vis/lineplot/range/aether";

export const xAxisStateZ = coreAxisStateZ;

export interface XAxisRenderProps extends AxisRenderProps {
  exposure: number;
}

export class XAxis extends CoreAxis<typeof coreAxisStateZ, YAxis | range.Provider> {
  static readonly TYPE = "XAxis";
  schema = coreAxisStateZ;

  render(props: XAxisRenderProps): void {
    if (this.deleted) return;
    const [dataToDecimal, err] = this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    this.renderAxis(props, dataToDecimal.reverse());

    this.renderYAxes(props, dataToDecimal);
    this.renderRangeAnnotations(props, dataToDecimal);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
  }

  findByXDecimal(
    props: Omit<XAxisRenderProps, "canvases">,
    target: number,
  ): FindResult[] {
    const [scale, err] = this.dataToDecimalScale(
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
    const [xDataToDecimalScale, error] = this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    if (error != null) throw error;
    const p = { ...props, xDataToDecimalScale };
    return this.yAxes.map((el) => el.findByXValue(p, target)).flat();
  }

  private renderYAxes(props: XAxisRenderProps, xDataToDecimalScale: scale.Scale): void {
    const p = { ...props, xDataToDecimalScale };
    this.yAxes.forEach((el) => el.render(p));
  }

  get yAxes(): readonly YAxis[] {
    return this.childrenOfType<YAxis>(YAxis.TYPE);
  }

  get rangeAnnotations(): readonly range.Provider[] {
    return this.childrenOfType<range.Provider>(range.Provider.TYPE);
  }

  private renderRangeAnnotations(
    props: XAxisRenderProps,
    xDataToDecimalScale: scale.Scale,
  ): void {
    const [bound, err] = this.bounds(props.hold, this.dataBounds.bind(this));
    if (err != null) throw err;
    this.rangeAnnotations.forEach((el) =>
      el.render({
        dataToDecimalScale: xDataToDecimalScale,
        region: props.plot,
        viewport: props.viewport,
        timeRange: new TimeRange(bound.lower, bound.upper),
      }),
    );
  }

  private dataBounds(): bounds.Bounds[] {
    return this.yAxes.map((el) => el.xBounds());
  }
}

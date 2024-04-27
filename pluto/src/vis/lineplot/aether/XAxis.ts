// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, type scale } from "@synnaxlabs/x";

import { type FindResult } from "@/vis/line/aether/line";
import {
  type AxisRenderProps,
  coreAxisStateZ,
  CoreAxis,
} from "@/vis/lineplot/aether/axis";
import { YAxis } from "@/vis/lineplot/aether/YAxis";
import { range } from "@/vis/lineplot/range/aether";

export const xAxisStateZ = coreAxisStateZ;

export interface XAxisRenderProps extends AxisRenderProps {}

export class XAxis extends CoreAxis<typeof coreAxisStateZ, YAxis | range.Annotation> {
  static readonly TYPE = "XAxis";
  schema = coreAxisStateZ;

  async render(props: XAxisRenderProps): Promise<void> {
    if (this.deleted) return;
    const [dataToDecimal, err] = await this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    this.renderAxis(props, dataToDecimal.reverse());
    await this.renderYAxes(props, dataToDecimal);
    await this.renderRangeAnnotations(props, dataToDecimal);
    // Throw the error here to that the user still has a visible axis.
    if (err != null) throw err;
  }

  async findByXDecimal(
    props: Omit<XAxisRenderProps, "canvases">,
    target: number,
  ): Promise<FindResult[]> {
    const [scale, err] = await this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    if (err != null) throw err;
    return await this.findByXValue(props, scale.reverse().pos(target));
  }

  async findByXValue(
    props: Omit<XAxisRenderProps, "canvases">,
    target: number,
  ): Promise<FindResult[]> {
    const [xDataToDecimalScale, error] = await this.dataToDecimalScale(
      props.hold,
      this.dataBounds.bind(this),
      props.viewport,
    );
    if (error != null) throw error;
    const p = { ...props, xDataToDecimalScale };
    const prom = this.yAxes.map(async (el) => await el.findByXValue(p, target));
    return (await Promise.all(prom)).flat();
  }

  private async renderYAxes(
    props: XAxisRenderProps,
    xDataToDecimalScale: scale.Scale,
  ): Promise<void> {
    const p = { ...props, xDataToDecimalScale };
    await Promise.all(this.yAxes.map(async (el) => await el.render(p)));
  }

  get yAxes(): readonly YAxis[] {
    return this.childrenOfType<YAxis>(YAxis.TYPE);
  }

  get rangeAnnotations(): readonly range.Annotation[] {
    return this.childrenOfType<range.Annotation>(
      range.Annotation.TYPE,
      range.Provider.TYPE,
    );
  }

  private async renderRangeAnnotations(
    props: XAxisRenderProps,
    xDataToDecimalScale: scale.Scale,
  ): Promise<void> {
    const p = { ...props, xDataToDecimalScale };
    await Promise.all(
      this.rangeAnnotations.map(
        async (el) =>
          await el.render({
            dataToDecimalScale: xDataToDecimalScale,
            region: props.plot,
          }),
      ),
    );
  }

  private async dataBounds(): Promise<bounds.Bounds[]> {
    return await Promise.all(this.yAxes.map(async (el) => await el.xBounds()));
  }
}

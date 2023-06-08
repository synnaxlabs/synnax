// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Bounds,
  GLBufferControl,
  LazyArray,
  TimeRange,
  generateTimeArray,
} from "@synnaxlabs/x";
import { z } from "zod";

import { Client } from "../client";

import { XYTelemSource } from "@/core/vis/telem";

export const rangeXYTelemProps = z.object({
  timeRange: TimeRange.z,
  x: z.number().optional(),
  y: z.number(),
});

export type RangeXYTelemProps = z.infer<typeof rangeXYTelemProps>;

export class RangeXYTelem implements XYTelemSource {
  key: string;
  type = "xy";
  private readonly props: RangeXYTelemProps;
  private readonly client: Client;
  private _x: LazyArray[] | null = null;
  private _y: LazyArray[] | null = null;

  constructor(key: string, props_: any, client: Client) {
    this.key = key;
    this.client = client;
    this.props = rangeXYTelemProps.parse(props_);
  }

  async x(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (this._x == null) await this.read();
    const x = this._x as LazyArray[];
    if (gl != null) x.forEach((x) => x.updateGLBuffer(gl));
    return x;
  }

  async y(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (this._y == null) await this.read();
    const y = this._y as LazyArray[];
    if (gl != null) y.forEach((y) => y.updateGLBuffer(gl));
    return y;
  }

  private async read(): Promise<void> {
    const { x, y, timeRange } = this.props;
    const toRead = [y];
    if (x == null) {
      const yChan = await this.client.core.channels.retrieve(y);
      // If the channel is indexed, read from its index.
      if (yChan.isIndexed) toRead.push(yChan.index);
    } else toRead.push(x);
    const d = await this.client.read(timeRange, toRead);
    const rate = d[y].channel.rate;
    this._y = d[y].data;
    // We need to generate a time array because the channel is rate based.
    const mustGenerate = toRead.length === 1;
    if (mustGenerate) {
      this._x = this._y.map((arr) =>
        generateTimeArray(rate, timeRange.start, arr.length)
      );
    } else {
      // TODO: Make sure X and Y have the same shape.
      this._x = d[x as number].data;
    }
  }

  async xBound(): Promise<Bounds> {
    const x = await this.x();
    return Bounds.max(x.map((x) => x.bound));
  }

  async yBound(): Promise<Bounds> {
    const y = await this.y();
    return Bounds.max(y.map((y) => y.bound));
  }

  setProps(props_: any): void {}
}

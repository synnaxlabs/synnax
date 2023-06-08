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
  GLBufferController,
  LazyArray,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { Client, StreamHandler } from "../client";

import { DynamicXYTelemSource } from "@/core/vis/telem";

export const dynamicRangeXYTelemProps = z.object({
  span: TimeSpan.z,
  x: z.number(),
  y: z.number(),
});

export type DynamicRangeXYTelemProps = z.infer<typeof dynamicRangeXYTelemProps>;

export class DynamicRangeXYTelem implements DynamicXYTelemSource {
  key: string;
  type = "dynamic-xy";
  private readonly props: DynamicRangeXYTelemProps;
  private handler: (() => void) | null = null;
  private readonly client: Client;
  private _x: LazyArray[] | null = null;
  private _y: LazyArray[] | null = null;
  private streamHandler: StreamHandler | null = null;

  constructor(key: string, props_: any, client: Client) {
    this.client = client;
    this.props = dynamicRangeXYTelemProps.parse(props_);
    this.key = key;
  }

  onChange(f: () => void): void {
    this.handler = f;
  }

  async x(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (this._x == null) await this.read();
    const x = this._x as LazyArray[];
    if (gl != null) x.forEach((a) => a.updateGLBuffer(gl));
    return x;
  }

  async y(gl?: GLBufferControl): Promise<LazyArray[]> {
    if (this._y == null) await this.read();
    const y = this._y as LazyArray[];
    if (gl != null) y.forEach((a) => a.updateGLBuffer(gl));
    return y;
  }

  async read(): Promise<void> {
    const { x, y, span } = this.props;
    const tr = TimeStamp.now().spanRange(span);
    const d = await this.client.read(tr, [x, y]);
    this._x = d[x].data;
    this._y = d[y].data;
    this.streamHandler = (data) => {
      if (data != null) {
        if (x in data) this._x?.push(...data[x].data);
        if (y in data) this._y?.push(...data[y].data);
      }
      this.handler?.();
    };
    this.client.setStreamhandler(this.streamHandler, [x, y]);
  }

  async xBound(): Promise<Bounds> {
    const x = await this.x();
    return Bounds.max(x.map((a) => a.bound));
  }

  async yBound(): Promise<Bounds> {
    const y = await this.y();
    return Bounds.max(y.map((a) => a.bound));
  }
}

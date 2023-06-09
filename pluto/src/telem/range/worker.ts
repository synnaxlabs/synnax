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
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";
import { z } from "zod";

import { DynamicXYTelemSource, XYTelemSource } from "@/core/vis/telem";
import { Client, StreamHandler } from "@/telem/client";
import { TelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";

export class RangeTelemFactory implements TelemFactory {
  private readonly client: Client;
  constructor(client: Client) {
    this.client = client;
  }

  create(key: string, type: string, props: any): ModifiableTelemSourceMeta | null {
    switch (type) {
      case RangeXYTelem.TYPE:
        return new RangeXYTelem(key, this.client, props);
      case DynamicRangeXYTelem.TYPE:
        return new DynamicRangeXYTelem(key, this.client, props);
      default:
        return null;
    }
  }
}

const rangeXYTelemCoreProps = z.object({
  x: z.number().optional(),
  y: z.number(),
});

class RangeXYTelemCore {
  key: string;
  variant = "xy";
  readonly client: Client;
  handler: (() => void) | null = null;
  _x: LazyArray[] | null = null;
  _y: LazyArray[] | null = null;

  constructor(key: string, client: Client) {
    this.key = key;
    this.client = client;
  }

  async x(gl?: GLBufferController): Promise<LazyArray[]> {
    const x = this._x as LazyArray[];
    if (gl != null) x.forEach((x) => x.updateGLBuffer(gl));
    return x;
  }

  async y(gl?: GLBufferController): Promise<LazyArray[]> {
    const y = this._y as LazyArray[];
    if (gl != null) y.forEach((y) => y.updateGLBuffer(gl));
    return y;
  }

  async xBounds(): Promise<Bounds> {
    const x = await this.x();
    return Bounds.max(x.map((x) => x.bounds));
  }

  async yBounds(): Promise<Bounds> {
    const y = await this.y();
    return Bounds.max(y.map((y) => y.bounds));
  }

  async readFixed(tr: TimeRange, y: number, x?: number): Promise<void> {
    const toRead = [y];
    if (x == null) {
      const yChan = await this.client.core.channels.retrieve(y);
      // If the channel is indexed, read from its index.
      if (yChan.isIndexed) toRead.push(yChan.index);
    } else toRead.push(x);
    const d = await this.client.read(tr, toRead);
    const rate = d[y].channel.rate;
    this._y = d[y].data;
    // We need to generate a time array because the channel is rate based.
    const mustGenerate = toRead.length === 1;
    if (mustGenerate) {
      this._x = this._y.map((arr) =>
        LazyArray.generateTimestamps(arr.length, rate, tr.start)
      );
    } else {
      this._x = d[x as number].data;
    }
  }

  onChange(f: () => void): void {
    this.handler = f;
  }
}

export const rangeXYTelemProps = rangeXYTelemCoreProps.extend({
  timeRange: TimeRange.z,
});

export type RangeXYTelemProps = z.infer<typeof rangeXYTelemProps>;

export class RangeXYTelem extends RangeXYTelemCore implements XYTelemSource {
  private props: RangeXYTelemProps;

  static readonly TYPE = "range-xy";

  constructor(key: string, client: Client, props_: any) {
    super(key, client);
    this.key = key;
    this.props = rangeXYTelemProps.parse(props_);
  }

  async read(): Promise<void> {
    const { x, y, timeRange } = this.props;
    await this.readFixed(timeRange, y, x);
  }

  async y(gl?: GLBufferController): Promise<LazyArray[]> {
    if (this._y == null) await this.read();
    return await super.y(gl);
  }

  async x(gl?: GLBufferController): Promise<LazyArray[]> {
    if (this._x == null) await this.read();
    return await super.x(gl);
  }

  setProps(props: any): void {
    this.props = rangeXYTelemProps.parse(props);
    this.handler?.();
  }

  cleanup(): void {
    this.handler = null;
  }
}

export const dynamicRangeXYTelemProps = z.object({
  span: TimeSpan.z,
  x: z.number(),
  y: z.number(),
});

export type DynamicRangeXYTelemProps = z.infer<typeof dynamicRangeXYTelemProps>;

export class DynamicRangeXYTelem
  extends RangeXYTelemCore
  implements DynamicXYTelemSource
{
  private props: DynamicRangeXYTelemProps;

  private streamHandler: StreamHandler | null = null;

  static readonly TYPE = "dynamic-range-xy";

  constructor(key: string, props_: any, client: Client) {
    super(key, client);
    this.props = dynamicRangeXYTelemProps.parse(props_);
    this.key = key;
  }

  async x(gl?: GLBufferController): Promise<LazyArray[]> {
    if (this._x == null) await this.read();
    return await super.x(gl);
  }

  async y(gl?: GLBufferController): Promise<LazyArray[]> {
    if (this._y == null) await this.read();
    return await super.y(gl);
  }

  async read(): Promise<void> {
    const { x, y, span } = this.props;
    const tr = TimeStamp.now().spanRange(span);
    await this.readFixed(tr, y, x);
    this.udpateStreamHandler();
  }

  private udpateStreamHandler(): void {
    if (this.streamHandler != null) this.client.removeStreamHandler(this.streamHandler);
    const { x, y } = this.props;
    this.streamHandler = (data) => {
      if (data != null) {
        if (x in data) this._x?.push(...data[x].data);
        if (y in data) this._y?.push(...data[y].data);
      }
      this.handler?.();
    };
    this.client.setStreamhandler(this.streamHandler, [x, y]);
  }

  setProps(props: any): void {
    this.props = dynamicRangeXYTelemProps.parse(props);
    this._x = null;
    this._y = null;
    this.handler?.();
  }

  cleanup(): void {
    if (this.streamHandler != null) this.client.removeStreamHandler(this.streamHandler);
    this.streamHandler = null;
    this.handler = null;
  }
}

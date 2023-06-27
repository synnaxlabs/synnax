// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel } from "@synnaxlabs/client";
import {
  Bounds,
  GLBufferController,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
  addSamples,
} from "@synnaxlabs/x";
import { z } from "zod";

import { XYTelemSource } from "@/core/vis/telem";
import { NumericTelemSource } from "@/core/vis/telem/TelemSource";
import { Client, StreamHandler } from "@/telem/client/client";
import { TelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";

export class RangeTelemFactory implements TelemFactory {
  type = "range";

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
      case RangePointTelem.TYPE:
        return new RangePointTelem(key, this.client, props);
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
  client: Client;
  valid: boolean = false;
  handler: (() => void) | null = null;
  _x: Series[] = [];
  _y: Series[] = [];

  constructor(key: string, client: Client) {
    this.key = key;
    this.client = client;
  }

  invalidate(): void {
    this.valid = false;
    this.handler?.();
  }

  acquire(gl?: GLBufferController): void {
    this._x?.forEach((x) => x.acquire(gl));
    this._y?.forEach((y) => y.acquire(gl));
  }

  release(gl?: GLBufferController): void {
    // this._x?.forEach((x) => x.release(gl));
    // this._y?.forEach((y) => y.release(gl));
  }

  async x(gl?: GLBufferController): Promise<Series[]> {
    const x = this._x;
    if (gl != null) x.forEach((x) => x.updateGLBuffer(gl));
    return x;
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    const y = this._y;
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

  async retrieveChannels(
    y: number,
    x?: number
  ): Promise<{ y: Channel; x: Channel | null }> {
    const yChan = await this.client.core.channels.retrieve(y);
    if (x == null) x = yChan.index;
    return {
      y: yChan,
      x: x === 0 ? null : await this.client.core.channels.retrieve(x),
    };
  }

  async readFixed(tr: TimeRange, y: number, x?: number): Promise<void> {
    const { x: xChan } = await this.retrieveChannels(y, x);
    const toRead = [y];
    if (xChan != null) toRead.push(xChan.key);
    const d = await this.client.read(tr, toRead);
    const rate = d[y].channel.rate;
    this._y = d[y].data;
    // We need to generate a time array because the channel is rate based.
    const mustGenerate = toRead.length === 1;
    if (mustGenerate) {
      this._x = this._y.map((arr) =>
        Series.generateTimestamps(arr.length, rate, tr.start)
      );
    } else {
      this._x = d[x as number].data;
    }
  }

  onChange(f: () => void): void {
    this.handler = f;
  }

  cleanup(): void {
    this._x = [];
    this._y = [];
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

  async read(gl?: GLBufferController): Promise<void> {
    this.release(gl);
    const { x, y, timeRange } = this.props;
    await this.readFixed(timeRange, y, x);
    this.acquire(gl);
    this.valid = true;
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    if (!this.valid) await this.read(gl);
    return await super.y(gl);
  }

  async x(gl?: GLBufferController): Promise<Series[]> {
    if (!this.valid) await this.read(gl);
    return await super.x(gl);
  }

  setProps(props: any): void {
    this.props = rangeXYTelemProps.parse(props);
    this.handler?.();
  }

  cleanup(): void {
    this.handler = null;
    super.cleanup();
  }
}

export const dynamicRangeXYTelemProps = z.object({
  span: TimeSpan.z,
  x: z.number().optional(),
  y: z.number(),
});

export type DynamicRangeXYTelemProps = z.infer<typeof dynamicRangeXYTelemProps>;

export class DynamicRangeXYTelem extends RangeXYTelemCore implements XYTelemSource {
  private props: DynamicRangeXYTelemProps;

  private streamHandler: StreamHandler | null = null;

  static readonly TYPE = "dynamic-range-xy";

  constructor(key: string, client: Client, props_: any) {
    super(key, client);
    this.props = dynamicRangeXYTelemProps.parse(props_);
    this.key = key;
  }

  async x(gl?: GLBufferController): Promise<Series[]> {
    if (!this.valid) await this.read(gl);
    return await super.x(gl);
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    if (!this.valid) await this.read(gl);
    return await super.y(gl);
  }

  async read(gl?: GLBufferController): Promise<void> {
    this.release(gl);
    const { x, y, span } = this.props;
    const tr = TimeStamp.now().spanRange(-span);
    await this.readFixed(tr, y, x);
    this.acquire(gl);
    await this.udpateStreamHandler();
    this.valid = true;
  }

  private async udpateStreamHandler(): Promise<void> {
    if (this.streamHandler != null) this.client.removeStreamHandler(this.streamHandler);
    const { x, y } = await this.retrieveChannels(this.props.y, this.props.x);
    this.streamHandler = (data) => {
      if (data != null) {
        const yd = data[y.key];
        if (yd == null) return;
        yd.data.forEach((arr) => arr.acquire());
        this._y?.push(...yd.data);
        if (x != null) {
          const xd = data[x.key];
          if (xd == null) return;
          xd.data.forEach((arr) => arr.acquire());
          this._x?.push(...xd.data);
        } else {
          // this._x?.push(
          //   ...yd.map((arr) =>
          //     Series.generateTimestamps(arr.length, y.rate, arr.timeRange.start)
          //   )
          // );
        }
      }
      this.handler?.();
    };
    const toStream = [y.key];
    if (x != null) toStream.push(x.key);
    this.client.setStreamHandler(this.streamHandler, toStream);
  }

  setProps(props: any): void {
    this.props = dynamicRangeXYTelemProps.parse(props);
    this.valid = false;
    this.handler?.();
  }

  cleanup(): void {
    if (this.streamHandler != null) this.client.removeStreamHandler(this.streamHandler);
    this.streamHandler = null;
    this.handler = null;
    super.cleanup();
  }
}

export const rangePointTelemProps = z.object({
  channel: z.number(),
});

export type RangePointTelemProps = z.infer<typeof rangePointTelemProps>;

export class RangePointTelem implements NumericTelemSource, ModifiableTelemSourceMeta {
  variant = "point";
  key: string;

  streamHandler: StreamHandler | null = null;

  static readonly TYPE = "range-point";

  private handler: (() => void) | null = null;
  private valid = false;
  private values: Series | null;
  private readonly client: Client;
  private props: z.infer<typeof rangePointTelemProps>;

  constructor(key: string, client: Client, props: any) {
    this.client = client;
    this.key = key;
    this.values = null;
    this.props = rangePointTelemProps.parse(props);
  }

  cleanup(): void {}
  invalidate(): void {
    this.valid = false;
  }

  release(gl: GLBufferController): void {}

  async value(): Promise<number> {
    if (!this.valid) await this.read();
    if (this.values == null || this.values.length === 0) return 0;
    const v = this.values.data[this.values.length - 1];
    return Number(addSamples(v, this.values.sampleOffset));
  }

  async read(): Promise<void> {
    const { channel } = this.props;
    const now = TimeStamp.now()
      .add(TimeSpan.seconds(10))
      .spanRange(-TimeSpan.seconds(5));
    const d = await this.client.read(now, [channel]);
    this.values = d[channel].data[0];
    await this.updateStreamHandler();
    this.valid = true;
  }

  async updateStreamHandler(): Promise<void> {
    if (this.streamHandler != null) this.client.removeStreamHandler(this.streamHandler);
    const { channel } = this.props;
    this.streamHandler = (data) => {
      if (data != null) {
        const d = data[channel];
        if (d.data.length > 0) this.values = d.data[0];
      }
      this.handler?.();
    };
    this.client.setStreamHandler(this.streamHandler, [channel]);
  }

  onChange(f: () => void): void {
    this.handler = f;
  }

  setProps(props: any): void {
    this.props = rangePointTelemProps.parse(props);
    this.valid = false;
    this.handler?.();
  }
}

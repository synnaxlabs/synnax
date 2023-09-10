// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Channel } from "@synnaxlabs/client";
import {
  type Destructor,
  type GLBufferController,
  Series,
  TimeRange,
  TimeSpan,
  TimeStamp,
  bounds,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type client } from "@/telem/client";
import { type telem } from "@/telem/core";
import { TelemMeta } from "@/telem/core/base";

const xySourceCorePropsZ = z.object({
  x: z.number().optional().default(0),
  y: z.number(),
});

class XYSourceCore<
  P extends z.ZodTypeAny,
  C extends client.StaticClient & client.ChannelClient = client.StaticClient &
    client.ChannelClient,
> extends TelemMeta<P> {
  client: C;
  valid: boolean = false;
  _x: Series[] = [];
  _y: Series[] = [];

  schema: P | undefined = undefined;

  constructor(key: string, client: C) {
    super(key);
    this.client = client;
  }

  release(): void {
    this._x?.forEach((x) => x.release());
    this._y?.forEach((y) => y.release());
  }

  acquire(gl?: GLBufferController): void {
    this._x?.forEach((x) => x.acquire(gl));
    this._y?.forEach((y) => y.acquire(gl));
  }

  async x(gl?: GLBufferController): Promise<Series[]> {
    const x = this._x;
    if (gl != null) this.updateBuffers(gl);
    return x;
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    const y = this._y;
    if (gl != null) this.updateBuffers(gl);
    return y;
  }

  async xBounds(): Promise<bounds.Bounds> {
    const x = await this.x();
    return bounds.max(x.map((x) => x.bounds));
  }

  async yBounds(): Promise<bounds.Bounds> {
    const y = await this.y();
    return bounds.max(y.map((y) => y.bounds));
  }

  updateBuffers(gl: GLBufferController): void {
    this._x.forEach((x) => x.updateGLBuffer(gl));
    this._y.forEach((y) => y.updateGLBuffer(gl));
  }

  async retrieveChannels(
    y: number,
    x: number,
  ): Promise<{ y: Channel; x: Channel | null }> {
    const yChan = await this.client.retrieveChannel(y);
    if (x === 0) x = yChan.index;
    return {
      y: yChan,
      x: x === 0 ? null : await this.client.retrieveChannel(x),
    };
  }

  async readFixed(tr: TimeRange, y: number, x: number): Promise<void> {
    const { x: xChan } = await this.retrieveChannels(y, x);
    const toRead = [y];
    if (xChan != null) toRead.push(xChan.key);
    const d = await this.client.read(tr, toRead);
    const rate = d[y].channel.rate;
    this._y = d[y].data;
    // We need to generate a time array because the channel is rate based.
    const mustGenerate = xChan == null;
    if (mustGenerate) {
      this._x = this._y.map((arr) =>
        Series.generateTimestamps(arr.length, rate, tr.start),
      );
    } else this._x = d[xChan.key].data;
  }

  setProps(props: any): void {
    super.setProps(props);
    if (!this.propsDeepEqual) this.invalidate();
  }

  invalidate(): void {
    this.valid = false;
    this.notify?.();
    this.release();
    this._x = [];
    this._y = [];
  }

  cleanup(): void {
    this.release();
    this.valid = false;
    this._x = [];
    this._y = [];
    super.cleanup();
  }
}

export const xySourcePropsZ = xySourceCorePropsZ.extend({
  timeRange: TimeRange.z,
});

export type XYSourceProps = z.input<typeof xySourcePropsZ>;

export class XYSource
  extends XYSourceCore<typeof xySourcePropsZ>
  implements telem.XYSource
{
  static readonly TYPE = "range-xy";
  schema = xySourcePropsZ;

  async read(gl?: GLBufferController): Promise<void> {
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
}

export const dynamicXYSourceProps = z.object({
  span: TimeSpan.z,
  x: z.number().optional().default(0),
  y: z.number(),
});

export type DynamicXYSourceProps = z.input<typeof dynamicXYSourceProps>;

export class DynamicXYSource
  extends XYSourceCore<typeof dynamicXYSourceProps, client.Client>
  implements telem.XYSource
{
  private stopStreaming: Destructor | null = null;
  schema = dynamicXYSourceProps;

  static readonly TYPE = "dynamic-range-xy";

  async x(gl?: GLBufferController): Promise<Series[]> {
    if (!this.valid) await this.read(gl);
    return await super.x(gl);
  }

  async y(gl?: GLBufferController): Promise<Series[]> {
    if (!this.valid) await this.read(gl);
    return await super.y(gl);
  }

  async read(gl?: GLBufferController): Promise<void> {
    const { x, y, span } = this.props;
    const tr = TimeStamp.now().spanRange(-span);
    await this.readFixed(tr, y, x);
    this.acquire(gl);
    await this.udpateStreamHandler();
    this.valid = true;
  }

  private async udpateStreamHandler(): Promise<void> {
    this.stopStreaming?.();
    const { x, y } = await this.retrieveChannels(this.props.y, this.props.x);
    const handler: client.StreamHandler = (data) => {
      const yd = data[y.key];
      if (yd != null && yd.data.length !== 0) {
        yd.data.forEach((arr) => arr.acquire());
        this._y?.push(...yd.data);
      }
      if (x != null) {
        const xd = data[x.key];
        if (xd != null && xd.data.length !== 0) {
          xd.data.forEach((arr) => arr.acquire());
          this._x?.push(...xd.data);
        }
      } else {
        this._x?.push(
          ...yd.data.map((arr) =>
            Series.generateTimestamps(arr.length, y.rate, arr.timeRange.start),
          ),
        );
      }
      this.notify?.();
    };
    const toStream = [y.key];
    if (x != null) toStream.push(x.key);
    this.stopStreaming = await this.client.stream(handler, toStream);
  }

  cleanup(): void {
    this.stopStreaming?.();
    this.stopStreaming = null;
    super.cleanup();
  }
}

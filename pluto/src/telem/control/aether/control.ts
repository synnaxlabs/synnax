// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Channel,
  ChannelKey,
  ChannelKeys,
  CrudeFrame,
  Series,
  Synnax,
  TimeStamp,
  Writer,
  Frame,
} from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/core";
import { TelemMeta } from "@/telem/core/base";

export const controllerStateZ = z.object({
  authority: z.number(),
});

interface InternalState {
  client: Synnax | null;
  prov: telem.Provider;
}

interface AetherControllerTelem {
  channelKeys: () => Promise<ChannelKeys>;
}

export class Controller
  extends aether.Composite<typeof controllerStateZ, InternalState>
  implements telem.Provider, telem.Factory
{
  registry: Map<AetherControllerTelem, null> = new Map();
  writer?: Writer;

  static readonly TYPE = "Controller";
  schema = controllerStateZ;

  afterUpdate(): void {
    this.internal.client = synnax.use(this.ctx);
    this.internal.prov = telem.get(this.ctx);
    telem.set(this.ctx, this);
  }

  private async channelKeys(): Promise<ChannelKeys> {
    const keys = new Set<ChannelKey>([]);
    for (const telem of this.registry.keys()) {
      const telemKeys = await telem.channelKeys();
      for (const key of telemKeys) keys.add(key);
    }
    return Array.from(keys);
  }

  async acquire(): Promise<void> {
    if (this.internal.client == null) return;

    this.writer = await this.internal.client.telem.newWriter(
      TimeStamp.now(),
      await this.channelKeys()
    );
  }

  async release(): Promise<void> {
    await this.writer?.close();
  }

  async set(frame: CrudeFrame): Promise<void> {
    if (this.writer == null) await this.acquire();
    await this.writer?.write(frame);
  }

  create(key: string, spec: telem.Spec): telem.Telem | null {
    if (spec.type === ControlledNumericTelemSink.TYPE) {
      const sink = new ControlledNumericTelemSink(key, this);
      this.registry.set(sink, null);
      return sink;
    }
    return null;
  }

  use<T>(key: string, spec: telem.Spec): telem.UseResult<T> {
    if (spec.type === ControlledNumericTelemSink.TYPE) {
      const sink = new ControlledNumericTelemSink(key, this);
      this.registry.set(sink, null);
      return [
        sink as unknown as T,
        () => {
          sink.cleanup();
          this.registry.delete(sink);
        },
      ];
    }
    return this.internal.prov.use<T>(key, spec, this);
  }
}

export const controlNumericTelemSinkProps = z.object({
  channel: z.number(),
});

export type ControlNumericTelemSinkProps = z.infer<typeof controlNumericTelemSinkProps>;

export class ControlledNumericTelemSink
  extends TelemMeta<typeof controlNumericTelemSinkProps>
  implements telem.NumericSink
{
  controller: Controller;
  static readonly TYPE = "controlled-numeric-telem-sink";
  channels: Channel[] = [];
  schema = controlNumericTelemSinkProps;

  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  async channelKeys(): Promise<ChannelKeys> {
    if (this.controller.internal.client == null) return [];
    const chan = await this.controller.internal.client?.channels.retrieve(
      this.props.channel
    );
    const keys = [chan.key];
    this.channels = [chan];
    if (chan.index !== 0) {
      keys.push(chan.index);
      this.channels.push(
        await this.controller.internal.client?.channels.retrieve(chan.index)
      );
    }
    return keys;
  }

  invalidate(): void {}

  async set(value: number): Promise<void> {
    if (this.controller.internal.client == null) return;
    const ch = await this.controller.internal.client.channels.retrieve(
      this.props.channel
    );
    const ch2 = await this.controller.internal.client.channels.retrieve(ch.index);
    const frame = new Frame(
      [ch.key, ch2.key],
      [
        // @ts-expect-error
        new Series(new ch.dataType.Array([value])),
        // @ts-expect-error
        new Series(new ch2.dataType.Array([BigInt(TimeStamp.now())])),
      ]
    );
    await this.controller.set(frame);
  }
}

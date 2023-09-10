// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Channel,
  type channel,
  Series,
  type Synnax,
  TimeStamp,
  framer,
} from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/core";
import { TelemMeta } from "@/telem/core/base";

export const STATUSES = ["acquired", "released", "overridden", "failed"] as const;
export const statusZ = z.enum(STATUSES);
export type Status = z.infer<typeof statusZ>;

export const controllerStateZ = z.object({
  name: z.string(),
  authority: z.number(),
  acquireTrigger: z.number(),
  status: statusZ.optional(),
});

interface InternalState {
  client: Synnax | null;
  prov: telem.Provider;
  addStatus: status.Aggregate;
}

interface AetherControllerTelem {
  channelKeys: (client: Synnax) => Promise<channel.Keys>;
}

export class Controller
  extends aether.Composite<typeof controllerStateZ, InternalState>
  implements telem.Provider, telem.Factory
{
  registry = new Map<AetherControllerTelem, null>();
  writer?: framer.Writer;

  static readonly TYPE = "Controller";
  schema = controllerStateZ;

  afterUpdate(): void {
    this.internal.client = synnax.use(this.ctx);
    const t = telem.get(this.ctx);
    if (!(t instanceof Controller)) this.internal.prov = t;
    this.internal.addStatus = status.useAggregate(this.ctx);
    telem.set(this.ctx, this);

    // If the counter has been incremented, we need to acquire control.
    // If the counter has been decremented, we need to release control.
    if (this.state.acquireTrigger > this.prevState.acquireTrigger) {
      void this.acquire();
    } else if (this.state.acquireTrigger < this.prevState.acquireTrigger)
      void this.release();
  }

  private async channelKeys(): Promise<channel.Keys> {
    const keys = new Set<channel.Key>([]);
    for (const telem of this.registry.keys()) {
      const telemKeys = await telem.channelKeys(this.internal.client as Synnax);
      for (const key of telemKeys) keys.add(key);
    }
    return Array.from(keys);
  }

  async acquire(): Promise<void> {
    const { client, addStatus } = this.internal;
    if (client == null)
      return addStatus({
        message: `Cannot acquire control on ${this.state.name} because no cluster has been connected`,
        variant: "warning",
      });

    try {
      const keys = await this.channelKeys();
      if (keys.length === 0)
        return addStatus({
          message: `Cannot acquire control on ${this.state.name} - no channels to control!`,
          variant: "warning",
        });

      this.writer = await client.telem.newWriter(TimeStamp.now(), keys);
      this.setState((p) => ({ ...p, status: "acquired" }));
      addStatus({
        message: `Acquired control on ${this.state.name}`,
        variant: "success",
      });
    } catch (e) {
      this.setState((p) => ({ ...p, status: "failed" }));
      addStatus({
        message: `${this.state.name} failed to acquire control: ${
          (e as Error).message
        }`,
        variant: "error",
      });
    }
  }

  async release(): Promise<void> {
    await this.writer?.close();
    this.writer = undefined;
    this.setState((p) => ({ ...p, status: "released" }));
    this.internal.addStatus({
      message: `Released control on ${this.state.name}.`,
      variant: "success",
    });
  }

  async set(frame: framer.CrudeFrame): Promise<void> {
    if (this.writer == null) await this.acquire();
    await this.writer?.write(frame);
  }

  create(key: string, spec: telem.Spec): telem.Telem | null {
    if (spec.type === NumericSink.TYPE) {
      const sink = new NumericSink(key, this);
      this.registry.set(sink, null);
      return sink;
    }
    return null;
  }

  use<T>(key: string, spec: telem.Spec): telem.UseResult<T> {
    return this.internal.prov.use<T>(key, spec, this);
  }
}

export const numericSinkProps = z.object({
  channel: z.number(),
});

export type NumericSinkProps = z.infer<typeof numericSinkProps>;

export class NumericSink
  extends TelemMeta<typeof numericSinkProps>
  implements telem.NumericSink
{
  controller: Controller;
  static readonly TYPE = "controlled-numeric-telem-sink";
  channels: Channel[] = [];
  schema = numericSinkProps;

  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  async channelKeys(client: Synnax): Promise<channel.Keys> {
    const chan = await client.channels.retrieve(this.props.channel);
    const keys = [chan.key];
    this.channels = [chan];
    if (chan.index !== 0) {
      keys.push(chan.index);
      this.channels.push(await client.channels.retrieve(chan.index));
    }
    return keys;
  }

  invalidate(): void {}

  async set(value: number): Promise<void> {
    if (this.controller.internal.client == null) return;
    const ch = await this.controller.internal.client.channels.retrieve(
      this.props.channel,
    );
    const ch2 = await this.controller.internal.client.channels.retrieve(ch.index);
    const frame = new framer.Frame(
      [ch.key, ch2.key],
      [
        // @ts-expect-error
        new Series(new ch.dataType.Array([value])),
        // @ts-expect-error
        new Series(new ch2.dataType.Array([BigInt(TimeStamp.now())])),
      ],
    );
    await this.controller.set(frame);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Controller.TYPE]: Controller,
};

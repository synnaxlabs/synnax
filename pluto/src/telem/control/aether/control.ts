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
  Authority,
  control,
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
  authority: z.number().default(0),
  acquireTrigger: z.number(),
  status: statusZ.optional(),
});

interface InternalState {
  client: Synnax | null;
  prov: telem.Provider;
  addStatus: status.Aggregate;
}

interface AetherControllerTelem extends telem.Telem {
  channelKeys: (client: Synnax) => Promise<channel.Keys>;
}

export class Controller
  extends aether.Composite<typeof controllerStateZ, InternalState>
  implements telem.Provider, telem.Factory
{
  registry = new Map<AetherControllerTelem, null>();
  writer?: framer.Writer;
  controlState?: control.StateTracker;

  static readonly TYPE = "Controller";
  schema = controllerStateZ;

  afterUpdate(): void {
    const nextClient = synnax.use(this.ctx);
    if (nextClient !== this.internal.client)
      this.registry.forEach((_, telem) => telem.invalidate());
    if (nextClient != null && this.internal.client !== nextClient) {
      control.StateTracker.open(nextClient.telem)
        .then((state) => {
          this.controlState = state;
          this.registry.forEach((_, telem) => telem.invalidate());
        })
        .catch(console.error);
    }
    this.internal.client = nextClient;
    const t = telem.get(this.ctx);
    if (!(t instanceof Controller)) this.internal.prov = t;
    this.internal.addStatus = status.useAggregate(this.ctx);
    telem.set(this.ctx, this);

    // If the counter has been incremented, we need to acquire control.
    // If the counter has been decremented, we need to release control.
    if (this.state.acquireTrigger > this.prevState.acquireTrigger) void this.acquire();
    else if (this.state.acquireTrigger < this.prevState.acquireTrigger)
      void this.release();
  }

  afterDelete(): void {
    void this.release();
  }

  get controlKey(): string {
    return `${this.state.name}-${this.key}`;
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

      this.writer = await client.telem.newWriter(
        TimeStamp.now(),
        keys,
        this.controlKey,
        this.state.authority,
      );
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
    this.controlState?.close();
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

  async setAuthority(channels: channel.Keys, value: Authority): Promise<void> {
    if (this.writer == null) await this.acquire();
    await this.writer?.setAuthority(
      Object.fromEntries(channels.map((k) => [k, value])),
    );
  }

  async releaseAuthority(keys: channel.Keys): Promise<void> {
    if (this.writer == null) await this.acquire();
    await this.writer?.setAuthority(
      Object.fromEntries(keys.map((k) => [k, this.state.authority])),
    );
  }

  create(key: string, spec: telem.Spec): telem.Telem | null {
    switch (spec.type) {
      case NumericSink.TYPE: {
        const sink = new NumericSink(key, this);
        this.registry.set(sink, null);
        return sink;
      }
      case AuthoritySource.TYPE: {
        const source = new AuthoritySource(key, this);
        this.registry.set(source, null);
        return source;
      }
      case AuthoritySink.TYPE: {
        const sink = new AuthoritySink(key, this);
        this.registry.set(sink, null);
        return sink;
      }
      default:
        return null;
    }
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

export const authoritySourceProps = z.object({
  channel: z.number(),
});

export type AuthoritySourceProps = z.infer<typeof authoritySourceProps>;

export class AuthoritySource
  extends TelemMeta<typeof authoritySourceProps>
  implements telem.StatusSource, AetherControllerTelem
{
  static readonly TYPE = "controlled-status-source";
  private readonly controller: Controller;
  private valid = false;
  schema = authoritySourceProps;
  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  async channelKeys(client: Synnax): Promise<channel.Keys> {
    return [];
  }

  async value(): Promise<status.Spec> {
    const c = this.controller.controlState;
    if (c == null)
      return {
        key: this.key,
        variant: "disabled",
        message: "No control information available",
        time: TimeStamp.now(),
      };
    const state = c.states.get(this.props.channel);
    if (!this.valid) {
      this.controller.controlState?.onChange((t) => {
        if (
          t.some(
            (t) =>
              t.from?.resource === this.props.channel ||
              t.to?.resource === this.props.channel,
          )
        )
          this.notify?.();
      });
      this.valid = true;
    }
    if (state == null)
      return {
        key: this.key,
        variant: "disabled",
        message: "Uncontrolled",
        time: TimeStamp.now(),
      };
    return {
      key: this.key,
      variant: state.subject === this.controller.controlKey ? "success" : "error",
      message:
        state.subject === this.controller.controlKey
          ? "In Control"
          : `Controlled by ${state.subject}`,
      time: TimeStamp.now(),
    };
  }

  invalidate(): void {
    this.valid = false;
    this.notify?.();
  }
}

export const authoritySinkProps = z.object({
  authority: z.number().default(Authority.ABSOLUTE.valueOf()),
  channel: z.number(),
});

export type AuthoritySinkProps = z.infer<typeof authoritySinkProps>;

export class AuthoritySink
  extends TelemMeta<typeof authoritySinkProps>
  implements telem.BooleanSink, AetherControllerTelem
{
  static readonly TYPE = "controlled-authority-sink";
  private readonly controller: Controller;
  schema = authoritySinkProps;
  keys: channel.Keys = [];

  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  async channelKeys(client: Synnax): Promise<channel.Keys> {
    const chan = await client.channels.retrieve(this.props.channel);
    this.keys = [chan.key, chan.index];
    return [];
  }

  async set(value: boolean): Promise<void> {
    if (!value) await this.controller.releaseAuthority(this.keys);
    else await this.controller.setAuthority(this.keys, this.props.authority);
  }

  invalidate(): void {}
}

export const REGISTRY: aether.ComponentRegistry = {
  [Controller.TYPE]: Controller,
};

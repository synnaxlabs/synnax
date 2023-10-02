// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
// Copyright 2023 Synnax Labs, Inc.

import {
  type channel,
  Series,
  type Synnax,
  TimeStamp,
  framer,
  Authority,
} from "@synnaxlabs/client";
import { type Destructor } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { type theming } from "@/aetherIndex";
import { type color } from "@/color/core";
import { StateProvider } from "@/control/aether/state";
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
  parentTelemProv: telem.Provider;
  stateProv: StateProvider;
  addStatus: status.Aggregate;
  theme: theming.Theme;
  prevTrigger: number;
}

interface AetherControllerTelem extends telem.Telem {
  needsControlOf: (client: Synnax) => Promise<channel.Keys>;
}

export class Controller
  extends aether.Composite<typeof controllerStateZ, InternalState>
  implements telem.Provider, telem.Factory
{
  private readonly registry = new Map<AetherControllerTelem, null>();
  private writer?: framer.Writer;

  static readonly TYPE = "Controller";
  schema = controllerStateZ;

  afterUpdate(): void {
    if (
      this.internal.prevTrigger == null ||
      Math.abs(this.state.acquireTrigger - this.internal.prevTrigger) > 1
    )
      this.internal.prevTrigger = this.state.acquireTrigger;
    const nextClient = synnax.use(this.ctx);
    const nextStateProv = StateProvider.use(this.ctx);

    // If the identify of our client or control state provider has changed, invalidate
    // all of the telemetry we've created.
    if (
      nextClient !== this.internal.client ||
      nextStateProv !== this.internal.stateProv
    )
      void this.handleClientChange();
    this.internal.client = nextClient;
    this.internal.stateProv = nextStateProv;

    // Hijack the telemetry provider so that we can create telemetry that is
    // bound to this controller.
    const parentProv = telem.hijackProvider(this.ctx, this);
    if (parentProv != null) this.internal.parentTelemProv = parentProv;

    this.internal.addStatus = status.useAggregate(this.ctx);

    // Acquire or release control if necessary.
    if (this.state.acquireTrigger > this.internal.prevTrigger) {
      void this.acquire();
      this.internal.prevTrigger = this.state.acquireTrigger;
    } else if (this.state.acquireTrigger < this.internal.prevTrigger) {
      void this.release();
      this.internal.prevTrigger = this.state.acquireTrigger;
    }
  }

  afterDelete(): void {
    void this.release();
  }

  async handleClientChange(): Promise<void> {
    await this.release();
    this.registry.forEach((_, telem) => telem.invalidate());
  }

  private async channelKeys(): Promise<channel.Keys> {
    const keys = new Set<channel.Key>([]);
    for (const telem of this.registry.keys()) {
      const telemKeys = await telem.needsControlOf(this.internal.client as Synnax);
      for (const key of telemKeys) if (key !== 0) keys.add(key);
    }
    return Array.from(keys);
  }

  private async acquire(): Promise<void> {
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

      this.writer = await client.telem.newWriter({
        start: TimeStamp.now(),
        channels: keys,
        controlSubject: { key: this.key, name: this.state.name },
        authorities: this.state.authority,
      });
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

  private async release(): Promise<void> {
    await this.writer?.close();
    if (this.deleted) return;
    this.setState((p) => ({ ...p, status: "released" }));
    if (this.writer != null)
      this.internal.addStatus({
        message: `Released control on ${this.state.name}.`,
        variant: "success",
      });
    this.writer = undefined;
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

  /** @implements telem.Factory to create telemetry that is bound to this controller. */
  create(key: string, spec: telem.Spec): telem.Telem | null {
    switch (spec.type) {
      case NumericSink.TYPE: {
        const sink = new NumericSink(key, this);
        sink.setProps(spec.props);
        this.registry.set(sink, null);
        return sink;
      }
      case AuthoritySource.TYPE: {
        const source = new AuthoritySource(key, this.key, this.internal.stateProv);
        source.setProps(spec.props);
        this.registry.set(source, null);
        return source;
      }
      case AcquireSink.TYPE: {
        const sink = new AcquireSink(key, this);
        sink.setProps(spec.props);
        this.registry.set(sink, null);
        return sink;
      }
      default:
        return null;
    }
  }

  use<T>(key: string, spec: telem.Spec): telem.UseResult<T> {
    return this.internal.parentTelemProv.use<T>(key, spec, this);
  }
}

export const numericSinkProps = z.object({
  channel: z.number(),
});

export type NumericSinkProps = z.infer<typeof numericSinkProps>;

export class NumericSink
  extends TelemMeta<typeof numericSinkProps>
  implements telem.NumericSink, AetherControllerTelem
{
  static readonly TYPE = "controlled-numeric-telem-sink";

  private readonly controller: Controller;
  schema = numericSinkProps;

  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  async needsControlOf(client: Synnax): Promise<channel.Keys> {
    const chan = await client.channels.retrieve(this.props.channel);
    const keys = [chan.key];
    if (chan.index !== 0) keys.push(chan.index);
    return keys;
  }

  invalidate(): void {}

  async setNumber(value: number): Promise<void> {
    const { client } = this.controller.internal;
    if (client == null) return;
    const ch = await client.channels.retrieve(this.props.channel);
    const index = await client.channels.retrieve(ch.index);
    const frame = new framer.Frame(
      [ch.key, index.key],
      [
        // @ts-expect-error - issues with BigInt vsumber.
        new Series(new ch.dataType.Array([value])),
        // @ts-expect-error - issues with BigInt vs number.
        new Series(new index.dataType.Array([BigInt(TimeStamp.now())])),
      ],
    );
    await this.controller.set(frame);
  }
}

export const acquireSinkPropsZ = z.object({
  authority: z.number().default(Authority.ABSOLUTE.valueOf()),
  channel: z.number(),
});

export type AcquireSinkProps = z.infer<typeof acquireSinkPropsZ>;

export class AcquireSink
  extends TelemMeta<typeof acquireSinkPropsZ>
  implements telem.BooleanSink, AetherControllerTelem
{
  static readonly TYPE = "controlled-authority-sink";
  private readonly controller: Controller;
  schema = acquireSinkPropsZ;

  constructor(key: string, controller: Controller) {
    super(key);
    this.controller = controller;
  }

  async needsControlOf(client: Synnax): Promise<channel.Keys> {
    const chan = await client.channels.retrieve(this.props.channel);
    const keys = [chan.key];
    if (chan.index !== 0) keys.push(chan.index);
    return keys;
  }

  async setBoolean(acquire: boolean): Promise<void> {
    const { controller } = this;
    const { client } = controller.internal;
    if (client == null) return;
    const ch = await client.channels.retrieve(this.props.channel);
    const keys = [ch.key];
    if (ch.index !== 0) keys.push(ch.index);
    if (!acquire) await this.controller.releaseAuthority(keys);
    else await this.controller.setAuthority(keys, this.props.authority);
  }

  invalidate(): void {}
}

export const authoritySourceProps = z.object({
  channel: z.number(),
});

export type AuthoritySourceProps = z.infer<typeof authoritySourceProps>;

export class AuthoritySource
  extends TelemMeta<typeof authoritySourceProps>
  implements telem.StatusSource, telem.ColorSource, AetherControllerTelem
{
  static readonly TYPE = "controlled-status-source";
  private readonly prov: StateProvider;
  private valid = false;
  private stopListening?: Destructor;
  private readonly controlKey: string;
  schema = authoritySourceProps;

  constructor(key: string, controlKey: string, prov: StateProvider) {
    super(key);
    this.prov = prov;
    this.controlKey = controlKey;
  }

  async needsControlOf(): Promise<channel.Keys> {
    return [];
  }

  private maybeRevalidate(): void {
    if (this.valid) return;
    const { channel: ch } = this.props;
    this.stopListening = this.prov.onChange((t) => {
      if (t.some(({ from, to }) => from?.resource === ch || to?.resource === ch))
        this.notify?.();
    });
    this.valid = true;
  }

  async color(): Promise<color.Color> {
    this.maybeRevalidate();
    return this.prov.getColor(this.props.channel);
  }

  async status(): Promise<status.Spec> {
    this.maybeRevalidate();
    const c = this.prov.controlState;
    const state = c.get(this.props.channel);
    const time = TimeStamp.now();

    if (state == null)
      return {
        key: this.key,
        variant: "disabled",
        message: "Uncontrolled",
        time,
      };

    return {
      key: state.subject.key,
      variant: state.subject.key === this.controlKey ? "success" : "error",
      message: `Controlled by ${state.subject.name}`,
      time,
    };
  }

  cleanup(): void {
    this.stopListening?.();
    this.valid = false;
  }

  invalidate(): void {
    this.valid = false;
    this.stopListening?.();
    this.notify?.();
  }
}

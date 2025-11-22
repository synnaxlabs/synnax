// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import {
  channel,
  control,
  DisconnectedError,
  type framer,
  type Synnax,
  TimeStamp,
  ValidationError,
} from "@synnaxlabs/client";
import {
  color,
  compare,
  control as xcontrol,
  type CrudeSeries,
  type destructor,
  type status as xstatus,
  TimeSpan,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { type theming } from "@/ether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";
import { AbstractSink } from "@/telem/aether/telem";
import { StateProvider } from "@/telem/control/aether/state";

export const statusZ = z.enum(["acquired", "released", "overridden", "failed"]);
export type Status = z.infer<typeof statusZ>;

export const controllerStateZ = z.object({
  name: z.string(),
  authority: z.number().default(0),
  acquireTrigger: z.number(),
  status: statusZ.optional(),
  needsControlOf: channel.keyZ.array().default([]),
});

interface InternalState {
  client: Synnax | null;
  instrumentation: Instrumentation;
  stateProv: StateProvider;
  addStatus: status.Adder;
  runAsync: status.ErrorHandler;
  theme: theming.Theme;
  prevTrigger: number;
  telemCtx: telem.Context;
}

interface AetherControllerTelem extends telem.Telem {
  needsControlOf: (client: Synnax) => Promise<channel.Keys>;
}

/**
 * @summary Acquires control over a set of channels by opening a writer to a Synnax
 * cluster, and then acts as a factory for telemetry that can be used to send commands
 * to that writer.
 */
export class Controller
  extends aether.Composite<typeof controllerStateZ, InternalState>
  implements telem.Factory
{
  static readonly TYPE = "Controller";
  schema = controllerStateZ;

  private readonly registry = new Map<AetherControllerTelem, null>();
  private writer?: framer.Writer;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.instrumentation = alamos.useInstrumentation(ctx);
    i.addStatus = status.useAdder(ctx);
    i.runAsync = status.useErrorHandler(ctx);
    if (
      i.prevTrigger == null ||
      Math.abs(this.state.acquireTrigger - i.prevTrigger) > 1
    )
      i.prevTrigger = this.state.acquireTrigger;
    const nextClient = synnax.use(ctx);
    const nextStateProv = StateProvider.use(ctx);
    i.stateProv = nextStateProv;
    i.telemCtx = telem.useChildContext(ctx, this, i.telemCtx);
    i.client = nextClient;
    i.runAsync(async () => {
      if (i.client == null) await this.release();
      if (this.state.acquireTrigger > i.prevTrigger) await this.acquire();
      else if (this.state.acquireTrigger < i.prevTrigger) await this.release();
    }, "failed to acquire control");
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.runAsync(async () => await this.release(), "failed to release control");
  }

  private async updateNeedsControlOf(): Promise<void> {
    const { client } = this.internal;
    if (client == null) return;

    const keys = new Set<channel.Key>([]);
    for (const telem of this.registry.keys()) {
      const telemKeys = await telem.needsControlOf(client);
      telemKeys.forEach((k) => k !== 0 && keys.add(k));
    }
    const nextKeys = Array.from(keys);
    if (
      compare.unorderedPrimitiveArrays(this.state.needsControlOf, nextKeys) ===
      compare.EQUAL
    )
      return;

    this.setState((p) => ({ ...p, needsControlOf: nextKeys }));
  }

  private async acquire(): Promise<void> {
    this.internal.prevTrigger = this.state.acquireTrigger;
    const { client, addStatus } = this.internal;
    if (client == null)
      return addStatus({
        message: `Cannot acquire control on ${this.state.name} because no Core has been connected.`,
        variant: "warning",
      });

    try {
      await this.updateNeedsControlOf();
      const needsControlOf = this.state.needsControlOf;
      if (needsControlOf.length === 0)
        return addStatus({
          message: `Cannot acquire control on ${this.state.name} because there are no channels to control.`,
          variant: "warning",
        });

      // Subtracting 1 millisecond makes sure that we avoid accidentally
      // setting the start timestamp over the writer earlier than the first
      // sample we write, preventing a validation error when releasing control. We
      // choose 1 ms because it is the resolution of a JS timestamp.
      const start = TimeStamp.now().sub(TimeSpan.milliseconds(1));
      this.writer = await client.openWriter({
        start,
        channels: needsControlOf,
        controlSubject: { key: this.key, name: this.state.name },
        authorities: this.state.authority,
      });
      this.setState((p) => ({ ...p, status: "acquired" }));
    } catch (e) {
      this.setState((p) => ({ ...p, status: "failed" }));
      if (!(e instanceof Error)) throw e;
      addStatus({
        variant: "error",
        message: `${this.state.name} failed to acquire control`,
        description: e.message,
      });
    }
  }

  private async release(): Promise<void> {
    this.internal.prevTrigger = this.state.acquireTrigger;
    try {
      await this.writer?.close();
    } catch (e) {
      this.internal.addStatus({
        message: `${this.state.name} failed to release control: ${
          (e as Error).message
        }`,
        variant: "error",
      });
    } finally {
      this.setState((p) => ({ ...p, status: "released" }));
      this.writer = undefined;
    }
  }

  async set(
    frame: framer.CrudeFrame | Record<channel.KeyOrName, CrudeSeries>,
  ): Promise<void> {
    if (this.writer == null) await this.acquire();
    await this.writer?.write(frame);
  }

  async setAuthority(channels: channel.Keys, value: control.Authority): Promise<void> {
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

  deleteTelem(t: AetherControllerTelem): void {
    this.registry.delete(t);
    void this.updateNeedsControlOf();
  }

  /** @implements telem.Factory to create telemetry that is bound to this controller. */
  create<T>(spec: telem.Spec): T | null {
    const { internal: i } = this;
    const f = (): T | null => {
      switch (spec.type) {
        case SetChannelValue.TYPE: {
          const sink = new SetChannelValue(this, i.runAsync, spec.props);
          this.registry.set(sink, null);
          return sink as T;
        }
        case AuthoritySource.TYPE: {
          const source = new AuthoritySource(this, this.internal.stateProv, spec.props);
          this.registry.set(source, null);
          return source as T;
        }
        case AcquireChannelControl.TYPE: {
          const sink = new AcquireChannelControl(this, i.runAsync, spec.props);
          return sink as T;
        }
        default:
          return null;
      }
    };
    const t = f();
    if (t != null) void this.updateNeedsControlOf();
    return t;
  }
}

export const setChannelValuePropsZ = z.object({
  channel: z.number(),
});

export type SetChannelValueProps = z.infer<typeof setChannelValuePropsZ>;

export class SetChannelValue
  extends AbstractSink<typeof setChannelValuePropsZ>
  implements telem.NumberSink, AetherControllerTelem
{
  static readonly TYPE = "controlled-numeric-telem-sink";

  private readonly controller: Controller;
  private readonly runAsync: status.ErrorHandler;
  schema = setChannelValuePropsZ;

  constructor(controller: Controller, runAsync: status.ErrorHandler, props: unknown) {
    super(props);
    this.controller = controller;
    this.runAsync = runAsync;
  }

  invalidate(): void {}

  cleanup(): void {
    this.controller.deleteTelem(this);
  }

  async needsControlOf(client: Synnax): Promise<channel.Keys> {
    if (this.props.channel === 0) return [];
    const chan = await client.channels.retrieve(this.props.channel);
    const keys = [chan.key];
    if (chan.index !== 0) keys.push(chan.index);
    return keys;
  }

  set(...values: number[]): void {
    this.runAsync(async () => {
      const { client } = this.controller.internal;
      if (client == null) throw new DisconnectedError("No Core connected");
      if (this.props.channel === 0)
        throw new ValidationError("No command channel specified for actuator");
      const ch = await client.channels.retrieve(this.props.channel);
      const fr: Record<channel.KeyOrName, CrudeSeries> = { [ch.key]: values };
      if (ch.index !== 0) {
        const index = await client.channels.retrieve(ch.index);
        const now = TimeStamp.now();
        fr[index.key] = Array.from({ length: values.length }, (_, i) =>
          now.add(TimeSpan.nanoseconds(i)),
        );
      }
      await this.controller.set(fr);
    }, "Failed to command channel");
  }
}

export const setChannelValue = (props: SetChannelValueProps): telem.NumberSinkSpec => ({
  type: SetChannelValue.TYPE,
  props,
  variant: "sink",
  valueType: "number",
});

export const acquireChannelControlPropsZ = z.object({
  authority: z.number().default(control.ABSOLUTE_AUTHORITY),
  channel: z.number(),
});

export type AcquireChannelControlProps = z.infer<typeof acquireChannelControlPropsZ>;

export class AcquireChannelControl
  extends AbstractSink<typeof acquireChannelControlPropsZ>
  implements telem.BooleanSink, AetherControllerTelem
{
  static readonly TYPE = "acquire-channel-control";
  private readonly controller: Controller;
  private readonly runAsync: status.ErrorHandler;
  schema = acquireChannelControlPropsZ;

  constructor(controller: Controller, runAsync: status.ErrorHandler, props: unknown) {
    super(props);
    this.controller = controller;
    this.runAsync = runAsync;
  }

  cleanup(): void {
    this.controller.deleteTelem(this);
  }

  async needsControlOf(client: Synnax): Promise<channel.Keys> {
    const chan = await client.channels.retrieve(this.props.channel);
    const keys = [chan.key];
    if (chan.index !== 0) keys.push(chan.index);
    return keys;
  }

  set(acquire: boolean): void {
    this.runAsync(async () => {
      const { controller } = this;
      const { client } = controller.internal;
      if (client == null) return;
      const ch = await client.channels.retrieve(this.props.channel);
      const keys = [ch.key];
      if (ch.index !== 0) keys.push(ch.index);
      if (!acquire) await this.controller.releaseAuthority(keys);
      else await this.controller.setAuthority(keys, this.props.authority);
    }, "failed to set channel authority");
  }
}

export const acquireChannelControl = (
  props: AcquireChannelControlProps,
): telem.BooleanSinkSpec => ({
  type: AcquireChannelControl.TYPE,
  props,
  variant: "sink",
  valueType: "boolean",
});

export const authoritySourceProps = z.object({
  channel: z.number(),
});

export type AuthoritySourceProps = z.infer<typeof authoritySourceProps>;

export const authoritySourceDetailsZ = z.object({
  valid: z.boolean(),
  color: color.colorZ.optional(),
  authority: z.number(),
});

export type AuthoritySourceDetails = z.infer<typeof authoritySourceDetailsZ>;

export class AuthoritySource
  extends telem.AbstractSource<typeof authoritySourceProps>
  implements telem.StatusSource<AuthoritySourceDetails>, AetherControllerTelem
{
  static readonly TYPE = "controlled-status-source";
  private readonly prov: StateProvider;
  private valid = false;
  private stopListening?: destructor.Destructor;
  private readonly controller: Controller;
  schema = authoritySourceProps;

  constructor(controller: Controller, prov: StateProvider, props: unknown) {
    super(props);
    this.prov = prov;
    this.controller = controller;
  }

  async needsControlOf(): Promise<channel.Keys> {
    return [];
  }

  private maybeRevalidate(): void {
    if (this.valid) return;
    const { channel: ch } = this.props;
    this.stopListening?.();
    const filter = xcontrol.filterTransfersByChannelKey(ch);
    this.stopListening = this.prov.onChange((t) => {
      if (filter(t).length === 0) return;
      this.notify?.();
    });
    this.valid = true;
  }

  value(): xstatus.Status<AuthoritySourceDetails> {
    this.maybeRevalidate();

    const time = TimeStamp.now();
    if (this.props.channel === 0)
      return {
        name: this.controller.key,
        key: this.controller.key,
        variant: "disabled",
        message: "No Channel",
        time,
        details: { valid: false, authority: 0 },
      };

    const state = this.prov.get(this.props.channel);

    if (state == null)
      return {
        name: this.controller.key,
        key: this.controller.key,
        variant: "disabled",
        message: "Uncontrolled",
        time,
        details: { valid: true, color: undefined, authority: 0 },
      };

    return {
      name: this.controller.key,
      key: state.subject.key,
      variant: state.subject.key === this.controller.key ? "success" : "error",
      message: `Controlled by ${state.subject.name}`,
      time,
      details: { valid: true, color: state.subjectColor, authority: state.authority },
    };
  }

  cleanup(): void {
    this.controller.deleteTelem(this);
    this.stopListening?.();
  }
}

export const authoritySource = (
  props: AuthoritySourceProps,
): telem.StatusSourceSpec => ({
  type: AuthoritySource.TYPE,
  props,
  variant: "source",
  valueType: "status",
});

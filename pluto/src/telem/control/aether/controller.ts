// Copyright 2026 Synnax Labs, Inc.
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
  NotFoundError,
  SessionEndedError,
  status as cstatus,
  type Synnax,
  TimeStamp,
  ValidationError,
} from "@synnaxlabs/client";
import { StreamClosed, Unreachable } from "@synnaxlabs/freighter";
import { type theme } from "@synnaxlabs/lyra/theme";
import {
  color,
  compare,
  type CrudeSeries,
  type destructor,
  errors,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { flux } from "@/flux/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";
import { AbstractSink } from "@/telem/aether/telem";
import { Colors } from "@/telem/control/aether/colors";
import { retrieveDefinition, type RetrieveQuery } from "@/telem/control/aether/queries";

export const statusZ = z.enum(["acquired", "released", "overridden", "failed"]);
export type Status = z.infer<typeof statusZ>;

export const controllerStateZ = z.object({
  name: z.string(),
  authority: z.number().default(0),
  status: statusZ.optional(),
  needsControlOf: channel.keyZ.array().default([]),
  // Bars the controller from writing or taking control on its own. It gives up any
  // control it holds. An explicit acquire is still honored, so a caller that sets this
  // owns what taking control means for its own state.
  disabled: z.boolean().default(false),
});

export const controllerMethodsZ = {
  acquire: z.function({ input: z.tuple([]), output: z.void() }),
  release: z.function({ input: z.tuple([]), output: z.void() }),
};

/**
 * Opens the writer a {@link Controller} commands through. The client is an argument
 * rather than a binding because a Controller resolves its own only after construction.
 */
export type OpenWriter = (
  client: Synnax,
  config: framer.WriterConfig,
) => Promise<framer.Writer>;

const defaultOpenWriter: OpenWriter = async (client, config) =>
  await client.openWriter(config);

interface InternalState {
  client: Synnax | null;
  instrumentation: Instrumentation;
  colors: Colors;
  addStatus: status.Adder;
  runAsync: status.ErrorHandler;
  theme: theme.Theme;
  telemCtx: telem.Context;
}

interface AetherControllerTelem extends telem.Telem {
  needsControlOf: (client: Synnax) => Promise<channel.Key[]>;
}

/**
 * @summary Acquires control over a set of channels by opening a writer to a Synnax
 * cluster, and then acts as a factory for telemetry that can be used to send commands
 * to that writer.
 */
export class Controller
  extends aether.Composite<
    typeof controllerStateZ,
    InternalState,
    aether.Component,
    typeof controllerMethodsZ
  >
  implements telem.Factory, aether.HandlersFromSchema<typeof controllerMethodsZ>
{
  static readonly TYPE = "Controller";
  static readonly METHODS = controllerMethodsZ;

  schema = controllerStateZ;
  methods = controllerMethodsZ;

  private readonly registry = new Map<AetherControllerTelem, null>();
  private readonly openWriter: OpenWriter;
  private writer?: framer.Writer;
  private acquirePromise?: Promise<void>;
  /** Set while an acquisition the user asked for is in flight. The disabled bar does
   * not apply to it: taking control is what clears the bar. */
  private acquireExplicit = false;

  constructor(
    props: aether.ComponentConstructorProps,
    openWriter: OpenWriter = defaultOpenWriter,
  ) {
    super(props);
    this.openWriter = openWriter;
  }

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    i.instrumentation = alamos.useInstrumentation(ctx);
    i.addStatus = status.useAdder(ctx);
    i.runAsync = status.useErrorHandler(ctx);
    i.colors = Colors.use(ctx);
    i.telemCtx = telem.useChildContext(ctx, this, i.telemCtx);
    i.client = synnax.use(ctx);
    if (this.state.disabled && this.state.status === "acquired") this.release();
  }

  /** The cluster this controller is bound to, or null while disconnected. */
  get client(): Synnax | null {
    return this.internal.client;
  }

  afterDelete(): void {
    this.release();
  }

  /**
   * Collects the channels the registered telemetry writes. A channel deleted from the
   * Core is left out, so one stale symbol does not block control of the rest.
   * @returns The errors for the channels that no longer exist.
   */
  private async updateNeedsControlOf(): Promise<Error[]> {
    const { client } = this.internal;
    if (client == null) return [];

    const keys = new Set<channel.Key>([]);
    const missing: Error[] = [];
    for (const telem of this.registry.keys())
      try {
        const telemKeys = await telem.needsControlOf(client);
        telemKeys.forEach((k) => k !== 0 && keys.add(k));
      } catch (e) {
        if (!NotFoundError.matches(e)) throw errors.fromUnknown(e);
        missing.push(errors.fromUnknown(e));
      }
    const nextKeys = Array.from(keys);
    if (
      compare.unorderedPrimitiveArrays(this.state.needsControlOf, nextKeys) !==
      compare.EQUAL
    )
      this.setState((p) => ({ ...p, needsControlOf: nextKeys }));
    return missing;
  }

  acquire(): void {
    this.acquireExplicit = true;
    this.internal.runAsync(() => this.doAcquire(), {
      message: "Failed to acquire control",
      name: this.state.name,
    });
  }

  release(): void {
    this.internal.runAsync(() => this.doRelease(), {
      message: "Failed to release control",
      name: this.state.name,
    });
  }

  private async doAcquire(): Promise<void> {
    if (this.acquirePromise != null) return await this.acquirePromise;
    this.acquirePromise = this.doAcquireImpl();
    try {
      await this.acquirePromise;
    } finally {
      this.acquirePromise = undefined;
      this.acquireExplicit = false;
    }
  }

  private async doAcquireImpl(): Promise<void> {
    const { client, addStatus } = this.internal;
    const { name } = this.state;
    if (client == null)
      return addStatus({
        variant: "warning",
        name,
        message: "Failed to acquire control",
        description: "No Core is connected.",
      });

    try {
      const missing = await this.updateNeedsControlOf();
      if (missing.length > 0)
        addStatus({
          variant: "warning",
          name,
          message: "Skipped deleted channels",
          description: [
            "Some symbols use channels that no longer exist.",
            ...missing.map((e) => e.message),
          ].join("\n"),
        });
      const needsControlOf = this.state.needsControlOf;
      if (needsControlOf.length === 0)
        return addStatus({
          variant: "warning",
          name,
          message: "Failed to acquire control",
          description: "No symbols control any channels.",
        });

      this.writer = await this.openWriter(client, {
        channels: needsControlOf,
        controlSubject: { key: this.key, name: this.state.name },
        authorities: this.state.authority,
        autoIndex: true,
      });
      // disabled can turn on while the writer opens. afterUpdate cannot catch that,
      // because the status is not acquired yet.
      if (!this.acquireExplicit && this.state.disabled)
        return await this.closeAndRelease();
      this.setState((p) => ({ ...p, status: "acquired" }));
    } catch (err) {
      this.setState((p) => ({ ...p, status: "failed" }));
      addStatus(
        cstatus.fromException(err, { message: "Failed to acquire control", name }),
      );
    }
  }

  private async doRelease(): Promise<void> {
    // A writer still opening would otherwise outlive the release.
    if (this.acquirePromise != null) await this.acquirePromise;
    await this.closeAndRelease();
  }

  private async closeAndRelease(): Promise<void> {
    try {
      await this.writer?.close();
    } catch (err) {
      // The Core frees a writer's control when its session ends.
      if (SessionEndedError.matches(err)) return;
      this.internal.addStatus(
        cstatus.fromException(err, {
          message: "Failed to release control",
          name: this.state.name,
        }),
      );
    } finally {
      this.setState((p) => ({ ...p, status: "released" }));
      this.writer = undefined;
    }
  }

  private static isRetryable(e: unknown): boolean {
    return StreamClosed.matches(e) || Unreachable.matches(e);
  }

  private async closeWriter(): Promise<void> {
    try {
      await this.writer?.close();
    } catch (e) {
      if (!Controller.isRetryable(e)) throw errors.fromUnknown(e);
    } finally {
      this.writer = undefined;
    }
  }

  private async withRetry(fn: () => Promise<void>): Promise<void> {
    if (this.state.disabled) return;
    if (this.writer == null) await this.doAcquire();
    try {
      await fn();
    } catch (e) {
      if (!Controller.isRetryable(e)) throw errors.fromUnknown(e);
      await this.closeWriter();
      await this.doAcquire();
      await fn();
    }
  }

  async set(
    frame: framer.CrudeFrame | Record<channel.Key | channel.Name, CrudeSeries>,
  ): Promise<void> {
    await this.withRetry(async () => await this.writer?.write(frame));
  }

  async setAuthority(channels: channel.Key[], value: control.Authority): Promise<void> {
    await this.withRetry(
      async () =>
        await this.writer?.setAuthority(
          Object.fromEntries(channels.map((k) => [k, value])),
        ),
    );
  }

  async releaseAuthority(keys: channel.Key[]): Promise<void> {
    await this.withRetry(
      async () =>
        await this.writer?.setAuthority(
          Object.fromEntries(keys.map((k) => [k, this.state.authority])),
        ),
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
          const source = new AuthoritySource(this, i.colors, i.runAsync, spec.props);
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

  async needsControlOf(client: Synnax): Promise<channel.Key[]> {
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
      await this.controller.set({ [this.props.channel]: values });
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

  async needsControlOf(client: Synnax): Promise<channel.Key[]> {
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
    }, "Failed to set channel authority");
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
  implements telem.StatusSource<typeof authoritySourceDetailsZ>, AetherControllerTelem
{
  static readonly TYPE = "controlled-status-source";
  private readonly colors: Colors;
  private readonly controller: Controller;
  private readonly retrieve: flux.Retrieve<RetrieveQuery, control.KeyedState>;
  private readonly stopListening: destructor.Destructor;
  schema = authoritySourceProps;

  constructor(
    controller: Controller,
    colors: Colors,
    runAsync: status.ErrorHandler,
    props: unknown,
  ) {
    super(props);
    this.colors = colors;
    this.controller = controller;
    this.retrieve = new flux.Retrieve({
      definition: retrieveDefinition,
      onChange: () => this.notify?.(),
      onError: (error) => runAsync(error, "Failed to retrieve control state"),
    });
    this.stopListening = colors.onChange(() => this.notify?.());
  }

  async needsControlOf(): Promise<channel.Key[]> {
    return [];
  }

  value(): telem.Result<cstatus.Status<typeof authoritySourceDetailsZ>> {
    const time = TimeStamp.now();
    const { channel: key } = this.props;
    if (key === 0)
      return telem.success(
        cstatus.create<typeof authoritySourceDetailsZ>({
          name: this.controller.key,
          key: this.controller.key,
          variant: "disabled",
          message: "No channel",
          time,
          details: { valid: false, authority: 0 },
        }),
      );

    this.retrieve.update(this.controller.client, { key });
    const state = this.retrieve.value;

    if (state == null)
      return telem.success(
        cstatus.create<typeof authoritySourceDetailsZ>({
          name: this.controller.key,
          key: this.controller.key,
          variant: "disabled",
          message: "Uncontrolled",
          time,
          details: { valid: true, color: undefined, authority: 0 },
        }),
      );

    return telem.success(
      cstatus.create<typeof authoritySourceDetailsZ>({
        name: this.controller.key,
        key: state.subject.key,
        variant: state.subject.key === this.controller.key ? "success" : "error",
        message: `Controlled by ${state.subject.name}`,
        time,
        details: {
          valid: true,
          color: this.colors.get(state.subject.key),
          authority: state.authority,
        },
      }),
    );
  }

  cleanup(): void {
    this.controller.deleteTelem(this);
    this.stopListening();
    this.retrieve.close();
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

// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { QueryError, Synnax, synnaxPropsZ, UnexpectedError } from "@synnaxlabs/client";
import { z } from "zod";

import { AetherComposite, AetherContext, Update } from "@/core/aether/worker";
import { TelemContext, TelemProvider } from "@/core/vis/telem/TelemContext";
import { TelemSourceMeta } from "@/core/vis/telem/TelemSource";
import { BaseClient, ClientProxy } from "@/telem/client";
import { CompoundTelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";
import { RangeTelemFactory } from "@/telem/range/aether";
import { StaticTelemFactory } from "@/telem/static/aether";

export const removeMessage = z.object({
  variant: z.literal("remove"),
  key: z.string(),
});

export const setMessage = z.object({
  variant: z.literal("set"),
  key: z.string(),
  type: z.string(),
  props: z.any(),
});

export const connectMessage = z.object({
  variant: z.literal("connect"),
  props: synnaxPropsZ,
});

const message = z.union([setMessage, removeMessage, connectMessage]);

export const telemState = message.optional();

export type TelemState = z.input<typeof telemState>;

class TelemProviderImpl implements TelemProvider {
  readonly telem: Map<string, ModifiableTelemSourceMeta> = new Map();

  constructor() {
    this.telem = new Map();
  }

  get<T extends TelemSourceMeta>(key: string): T {
    const v = this.telem.get(key);
    if (v == null)
      throw new QueryError(`Telemetry service could not find source with key ${key}`);
    return v as unknown as T;
  }
}

export class Telem extends AetherComposite<typeof telemState> {
  factory: CompoundTelemFactory;
  client: ClientProxy;
  prov: TelemProviderImpl;

  static readonly TYPE = "telem";

  constructor(update: Update) {
    super(update, telemState);
    this.prov = new TelemProviderImpl();
    TelemContext.set(update.ctx, this.prov);
    this.client = new ClientProxy();
    this.factory = new CompoundTelemFactory([
      new StaticTelemFactory(),
      new RangeTelemFactory(this.client),
    ]);
  }

  handleUpdate(ctx: AetherContext): void {
    const msg = this.state;
    if (msg == null) return;

    if (msg.variant === "connect") {
      this.client.swap(new BaseClient(new Synnax(msg.props)));
      this.prov.telem.forEach((t) => t.invalidate());
      return TelemContext.set(ctx, this.prov);
    }

    const source = this.prov.telem.get(msg.key);
    if (msg.variant === "remove") {
      if (source == null)
        return console.warn(
          `Telemetry service could not find source with key ${msg.key} to remove`
        );
      this.prov.telem.delete(msg.key);
      return source.cleanup();
    }

    console.log(msg.key, msg.props);
    if (source == null) this.newSource(msg.key, msg.type, msg.props);
    else source.setProps(msg.props);
  }

  newSource(key: string, type: string, props: any): void {
    const source = this.factory.create(key, type, props);
    if (source == null) {
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${type}`
      );
    }
    this.prov.telem.set(key, source);
  }
}

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

import { Client } from "./client/client";
import { RangeTelemFactory } from "./range/worker";

import { AetherComposite, Update } from "@/core/aether/worker";
import { TelemContext, TelemProvider } from "@/core/vis/telem/TelemService";
import { TelemSourceMeta } from "@/core/vis/telem/TelemSource";
import { CompoundTelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";
import { StaticTelemFactory } from "@/telem/static/worker";

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
  client: Client | null = null;
  prov: TelemProviderImpl;

  static readonly TYPE = "telem";

  constructor(update: Update) {
    super(update, telemState);
    this.factory = new CompoundTelemFactory([new StaticTelemFactory()]);
    this.prov = new TelemProviderImpl();
    TelemContext.set(update.ctx, this.prov);
  }

  handleUpdate(): void {
    const msg = this.state;
    if (msg == null) return;

    if (msg.variant === "connect") {
      if (this.client != null) this.client.close();
      this.client = new Client(new Synnax(msg.props));
      this.factory.change(new RangeTelemFactory(this.client));
      return this.prov.telem.forEach((t) => t.invalidate());
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

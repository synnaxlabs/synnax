// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax, synnaxPropsZ, UnexpectedError } from "@synnaxlabs/client";
import { z } from "zod";

import { AetherComposite, AetherUpdate } from "@/core/aether/worker";
import { TelemContext, UseTelemResult } from "@/core/vis/telem/TelemContext";
import { TelemSourceProps } from "@/core/vis/telem/TelemSource";
import { BaseClient, ClientProxy } from "@/telem/client";
import { CompoundTelemFactory } from "@/telem/factory";
import { ModifiableTelemSourceMeta } from "@/telem/meta";
import { RangeTelemFactory } from "@/telem/range/aether";
import { StaticTelemFactory } from "@/telem/static/aether";

export const telemState = z.object({
  props: synnaxPropsZ.optional(),
});

export type TelemState = z.input<typeof telemState>;

export class Telem extends AetherComposite<typeof telemState> {
  factory: CompoundTelemFactory;
  readonly telem: Map<string, ModifiableTelemSourceMeta> = new Map();
  client: ClientProxy;

  static readonly TYPE = "telem";
  static readonly z = telemState;
  schema = Telem.z;

  constructor(update: AetherUpdate) {
    super(update);
    TelemContext.set(update.ctx, this);
    this.client = new ClientProxy();
    this.factory = new CompoundTelemFactory([
      new StaticTelemFactory(),
      new RangeTelemFactory(this.client),
    ]);
  }

  get<T>(key: string, props: TelemSourceProps): UseTelemResult<T> {
    // try to get the source
    let source = this.telem.get(key);
    if (source != null) source?.setProps(props.props);
    else source = this.newSource(key, props.type, props.props);
    return { telem: source as T, cleanupTelem: () => this.remove(key) };
  }

  private remove(key: string): void {
    const source = this.telem.get(key);
    source?.cleanup();
  }

  afterUpdate(): void {
    if (this.state.props == null) this.client.swap(null);
    else this.client.swap(new BaseClient(new Synnax(this.state.props)));
    this.telem.forEach((t) => t.invalidate());
    return TelemContext.set(this.ctx, this);
  }

  newSource<T>(key: string, type: string, props: any): T {
    const source = this.factory.create(key, type, props);
    if (source == null) {
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${type}`
      );
    }
    this.telem.set(key, source);
    return source as T;
  }
}

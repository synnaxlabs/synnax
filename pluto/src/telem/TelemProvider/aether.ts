// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import { z } from "zod";

import { AetherClient } from "@/client/aether";
import { AetherComposite } from "@/core/aether/worker";
import {
  Telem,
  TelemSpec,
  TelemContext,
  UseTelemResult,
  TelemProvider,
} from "@/core/vis/telem";
import { AetherBooleanTelem } from "@/telem/bool/aether";
import { BaseClient, ClientProxy } from "@/telem/client";
import { CompoundTelemFactory } from "@/telem/factory";
import { AetherRemoteTelem } from "@/telem/remote/aether";
import { AetherStaticTelem } from "@/telem/static/aether";

export const aetherTelemProviderState = z.object({});

export type AetherTelemProviderState = z.input<typeof aetherTelemProviderState>;

export class AetherTelemProvider
  extends AetherComposite<typeof aetherTelemProviderState>
  implements TelemProvider
{
  readonly telem: Map<string, Telem> = new Map();
  client: ClientProxy = new ClientProxy();
  factory: CompoundTelemFactory = new CompoundTelemFactory([
    new AetherBooleanTelem.Factory(),
    new AetherStaticTelem.Factory(),
    new AetherRemoteTelem.Factory(this.client),
  ]);

  static readonly TYPE = "TelemProvider";
  static readonly z = aetherTelemProviderState;
  schema = AetherTelemProvider.z;

  use<T>(key: string, spec: TelemSpec): UseTelemResult<T> {
    // try to get the source
    let telem = this.telem.get(key);
    if (telem != null) telem.setProps(spec.props);
    else telem = this.create(key, spec);
    return [telem as T, () => this.remove(key)];
  }

  private remove(key: string): void {
    const source = this.telem.get(key);
    source?.cleanup();
  }

  afterUpdate(): void {
    const client = AetherClient.Context.use(this.ctx);
    this.client.swap(new BaseClient(client));
    this.telem.forEach((t) => t.invalidate());
    return TelemContext.set(this.ctx, this);
  }

  create<T>(key: string, spec: TelemSpec): T {
    const telem = this.factory.create(key, spec, this.factory);
    if (telem == null)
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${spec.type}`
      );
    this.telem.set(key, telem);
    return telem as T;
  }
}

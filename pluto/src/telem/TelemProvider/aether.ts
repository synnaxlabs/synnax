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
import { CompoundTelemFactory, TelemFactory } from "@/telem/factory";
import { AetherNoopTelem } from "@/telem/noop/aether";
import { AetherRemoteTelem } from "@/telem/remote/aether";
import { AetherStaticTelem } from "@/telem/static/aether";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});
export class AetherTelemProvider
  extends AetherComposite<typeof providerStateZ>
  implements TelemProvider
{
  readonly telem: Map<string, Telem> = new Map();
  client: ClientProxy = new ClientProxy();
  factory: CompoundTelemFactory = new CompoundTelemFactory([
    new AetherBooleanTelem.Factory(),
    new AetherStaticTelem.Factory(),
    new AetherRemoteTelem.Factory(this.client),
    new AetherNoopTelem.Factory(),
  ]);

  static readonly TYPE = "TelemProvider";
  static readonly stateZ = providerStateZ;
  schema = AetherTelemProvider.stateZ;

  use<T>(key: string, spec: TelemSpec, extension?: TelemFactory): UseTelemResult<T> {
    let telem = this.telem.get(key);
    if (telem != null) telem.setProps(spec.props);
    else telem = this.create(key, spec, extension);
    return [telem as T, () => this.remove(key)];
  }

  private remove(key: string): void {
    const source = this.telem.get(key);
    source?.cleanup();
  }

  afterUpdate(): void {
    const client = AetherClient.use(this.ctx);
    if (client != null) this.client.swap(new BaseClient(client));
    this.telem.forEach((t) => t.invalidate());
    return TelemContext.set(this.ctx, this);
  }

  create<T>(key: string, spec: TelemSpec, extension?: TelemFactory): T {
    if (extension != null) this.factory.factories.push(extension);
    let telem = this.factory.create(key, spec, this.factory);
    if (telem == null) {
      telem = this.factory.create(key, spec, this.factory);
      if (telem == null)
        throw new UnexpectedError(
          `Telemetry service could not find a source for type ${spec.type}`
        );
    }
    this.telem.set(key, telem);
    if (extension != null) this.factory.factories = this.factory.factories.slice(0, -1);
    return telem as T;
  }
}

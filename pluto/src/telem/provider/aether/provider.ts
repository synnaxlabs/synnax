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

import { aether } from "@/aether/aether";
import { synnax } from "@/synnax/aether";
import { bool } from "@/telem/bool/aether";
import { client } from "@/telem/client";
import { telem } from "@/telem/core";
import { CompoundTelemFactory, type Factory } from "@/telem/core/factory";
import { noop } from "@/telem/noop";
import { remote } from "@/telem/remote/aether";
import { staticTelem } from "@/telem/static/aether";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

export class Provider
  extends aether.Composite<typeof providerStateZ>
  implements telem.Provider
{
  readonly telem = new Map<string, telem.Telem>();
  client: client.Proxy = new client.Proxy();
  factory: CompoundTelemFactory = new CompoundTelemFactory([
    new bool.Factory(),
    new staticTelem.Factory(),
    new remote.Factory(this.client),
    new noop.Factory(),
  ]);

  static readonly TYPE = "TelemProvider";
  static readonly stateZ = providerStateZ;
  schema = Provider.stateZ;

  use<T>(key: string, spec: telem.Spec, extension?: Factory): telem.UseResult<T> {
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
    const client_ = synnax.use(this.ctx);
    if (client_ != null) this.client.swap(new client.Core(client_));
    this.telem.forEach((t) => t.invalidate());
    return telem.setProvider(this.ctx, this);
  }

  create<T>(key: string, spec: telem.Spec, extension?: Factory): T {
    if (extension != null) this.factory.factories.push(extension);
    let telem = this.factory.create(key, spec, this.factory);
    if (telem == null) {
      telem = this.factory.create(key, spec, this.factory);
      if (telem == null)
        throw new UnexpectedError(
          `Telemetry service could not find a source for type ${spec.type}`,
        );
    }
    this.telem.set(key, telem);
    if (extension != null) this.factory.factories = this.factory.factories.slice(0, -1);
    return telem as T;
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [Provider.TYPE]: Provider,
};

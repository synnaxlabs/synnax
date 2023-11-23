// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { UnexpectedError } from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { synnax } from "@/synnax/aether";
import { telem } from "@/telem/aether";
import { client } from "@/telem/client";

import { type CompoundTelemFactory } from "./factory";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

interface InternalState {
  instrumentation: Instrumentation;
}

export class BaseProvider
  extends aether.Composite<typeof providerStateZ, InternalState>
  implements telem.Provider
{
  client: client.Proxy = new client.Proxy();
  factory: CompoundTelemFactory = telem.factory(this.client);

  static readonly TYPE = "TelemProvider";
  static readonly stateZ = providerStateZ;
  schema = BaseProvider.stateZ;

  create<T>(spec: telem.Spec): T {
    const { instrumentation: I } = this.internal;
    I.L.debug("creating telem", { spec });
    const telem = this.factory.create(spec);
    if (telem == null)
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${spec.type}`,
      );
    return telem as T;
  }

  registerFactory(f: telem.Factory): void {
    this.factory.add(f);
  }

  equals(other: telem.Provider): boolean {
    if (!(other instanceof BaseProvider)) return false;
    return this.client._client === other.client._client;
  }

  afterUpdate(): void {
    const client_ = synnax.use(this.ctx);
    const I = alamos.useInstrumentation(this.ctx, "telem");
    this.internal.instrumentation = I.child("provider");
    if (client_ != null) {
      I.L.info("swapping client", { client: client_ });
      this.client.swap(new client.Core(client_, this.internal.instrumentation));
    }
    return telem.setProvider(this.ctx, this);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [BaseProvider.TYPE]: BaseProvider,
};

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
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
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

interface InternalState {
  instrumentation: Instrumentation;
}

export class Provider
  extends aether.Composite<typeof providerStateZ, InternalState>
  implements telem.Provider
{
  equals: (other: telem.Provider) => boolean;
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

  create<T>(spec: telem.Spec): T {
    const { instrumentation: I } = this.internal;
    I.L.debug("creating telem", { spec });
    const telem = this.factory.create(spec);
    if (telem == null)
      throw new UnexpectedError(
        `Telemetry service could not find a source for type ${spec.type}`,
      );
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
  [Provider.TYPE]: Provider,
};

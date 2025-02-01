// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { type Synnax } from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { synnax } from "@/synnax/aether";
import { Context, setContext } from "@/telem/aether/context";
import { newFactory } from "@/telem/aether/factory";
import { client } from "@/telem/client";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

interface InternalState {
  instrumentation: Instrumentation;
}

export class BaseProvider extends aether.Composite<
  typeof providerStateZ,
  InternalState
> {
  static readonly TYPE = "telem.Provider";
  static readonly stateZ = providerStateZ;
  schema = BaseProvider.stateZ;
  prevClient: Synnax | null = null;

  async afterUpdate(ctx: aether.Context): Promise<void> {
    const core = synnax.use(ctx);
    const I = alamos.useInstrumentation(ctx, "telem");
    this.internal.instrumentation = I.child("provider");
    const shouldSwap = core !== this.prevClient;
    if (!shouldSwap) return;
    this.prevClient = core;
    const c =
      core == null
        ? new client.NoopClient()
        : new client.Core({ core, instrumentation: I });
    const f = newFactory(c);
    const value = new Context(f);
    setContext(ctx, value);
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [BaseProvider.TYPE]: BaseProvider,
};

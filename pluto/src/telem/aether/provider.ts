// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { type framer, type Synnax } from "@synnaxlabs/client";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { synnax } from "@/synnax/aether";
import { Context, CONTEXT_KEY, setContext } from "@/telem/aether/context";
import { GL_TRANSFORM } from "@/telem/aether/convertSeries";
import {
  type CompoundFactory,
  createFactory,
  type Factory,
} from "@/telem/aether/factory";
import { type Client } from "@/telem/aether/remote";

export type ProviderState = z.input<typeof providerStateZ>;
export const providerStateZ = z.object({});

export const PROVIDER_TYPE = "telem.Provider";

interface InternalState {
  instrumentation: Instrumentation;
  /** Error handler bound from the current afterUpdate's context, reused by afterDelete
   * (which has no update-triggering context of its own to derive one from). */
  handleError: status.ErrorHandler;
}

export const createProvider = (
  createFactory: (client: Client | null) => CompoundFactory,
): aether.ComponentConstructor => {
  class BaseProvider extends aether.Composite<typeof providerStateZ, InternalState> {
    static readonly TYPE = PROVIDER_TYPE;
    static readonly stateZ = providerStateZ;
    schema = BaseProvider.stateZ;
    prevCore: Synnax | null = null;
    private feed: framer.Feed | null = null;

    afterUpdate(ctx: aether.Context): void {
      const { internal: i } = this;
      const core = synnax.use(ctx);
      i.instrumentation = alamos.useInstrumentation(ctx, "telem").child("provider");
      i.handleError = status.useErrorHandler(ctx);
      const shouldSwap = core !== this.prevCore || !ctx.wasSetPreviously(CONTEXT_KEY);
      if (!shouldSwap) return;
      this.prevCore = core;
      this.closeFeed();
      let client: Client | null = null;
      if (core != null) {
        // The feed caches series in the representation this thread renders.
        this.feed = core.openFeed({
          transform: GL_TRANSFORM,
          instrumentation: i.instrumentation,
        });
        client = { feed: this.feed, channels: core.channels };
      }
      setContext(ctx, new Context(createFactory(client)));
    }

    afterDelete(): void {
      this.closeFeed();
    }

    private closeFeed(): void {
      const { feed } = this;
      if (feed == null) return;
      // Cleared before the close is dispatched so a re-entrant swap cannot close the
      // same feed twice.
      this.feed = null;
      this.internal.handleError(
        async () => await feed.close(),
        "failed to close telemetry feed",
      );
    }
  }
  return BaseProvider;
};

export const Provider = createProvider(createFactory);

export const REGISTRY: aether.ComponentRegistry = {
  [PROVIDER_TYPE]: Provider,
};

export type FactoryConstructor = (client: Client | null) => Factory;

export const createRegistry = (
  ...factoryConstructors: FactoryConstructor[]
): aether.ComponentRegistry => {
  const create = (client: Client | null): CompoundFactory =>
    createFactory(
      client,
      factoryConstructors.map((c) => c(client)),
    );
  return { [PROVIDER_TYPE]: createProvider(create) };
};

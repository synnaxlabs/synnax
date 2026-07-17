// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import z from "zod";

import { aether } from "@/aether/aether";
import { useAsyncErrorHandler, useErrorHandler } from "@/status/aether/aggregator";
import { synnax } from "@/synnax/aether";

/** State schema for the flux provider (currently empty) */
export const providerStateZ = z.object({});

/** Type representing the provider's state */
export type ProviderState = z.input<typeof providerStateZ>;

/** Type identifier for the flux provider component */
export const PROVIDER_TYPE = "flux.Provider";

/**
 * Routes the worker client's cache errors to the status aggregator and starts
 * change streaming.
 */
class Provider extends aether.Composite<
  typeof providerStateZ,
  { clientKey: string | undefined }
> {
  static readonly TYPE = PROVIDER_TYPE;
  static readonly stateZ = providerStateZ;
  schema = Provider.stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const client = synnax.use(ctx);
    if (client == null || client.key === i.clientKey) return;
    i.clientKey = client.key;
    const handleError = useErrorHandler(ctx);
    client.cache.setErrorHandlers(handleError, useAsyncErrorHandler(ctx));
    handleError(
      async () => await client.cache.ensureStreaming(),
      "failed to start flux change streaming",
    );
  }
}

/**
 * Creates an Aether component registry with the flux provider.
 *
 * @returns An Aether component registry containing the provider
 */
export const createRegistry = (): aether.ComponentRegistry => ({
  [PROVIDER_TYPE]: Provider,
});

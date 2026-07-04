// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { Pluto } from "@/pluto";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import {
  MOCK_RENDER_CONTEXT_REGISTRY,
  type MockRenderContext,
  MockRenderContextProvider,
  mockRenderContextProviderStateZ,
  registerMockRenderContext,
} from "@/testutil/render";

interface RenderContextSeedProps extends PropsWithChildren {
  contextKey: string;
}

const RenderContextSeed = ({
  contextKey,
  children,
}: RenderContextSeedProps): ReactElement => {
  const { path } = Aether.useLifecycle({
    type: MockRenderContextProvider.TYPE,
    schema: mockRenderContextProviderStateZ,
    initialState: { contextKey },
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};

const newWrapper = (
  client: Client | null,
  fluxClient: Flux.Client,
  additionalRegistry?: aether.ComponentRegistry,
  renderContext?: MockRenderContext,
) => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...flux.createRegistry({ storeConfig: {} }),
    ...(renderContext != null ? MOCK_RENDER_CONTEXT_REGISTRY : {}),
    ...additionalRegistry,
  });
  let contextKey: string | null = null;
  if (renderContext != null) {
    contextKey = id.create();
    registerMockRenderContext(contextKey, renderContext);
  }
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client}>
          <Flux.Provider client={fluxClient}>
            {contextKey == null ? (
              children
            ) : (
              <RenderContextSeed contextKey={contextKey}>{children}</RenderContextSeed>
            )}
          </Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return Wrapper;
};

export interface CreateSynnaxWrapperArgs {
  client: Client | null;
  excludeFluxStores?: string[];
  /** Overrides the flux error handler. Defaults to logging via console.error. */
  handleError?: status.ErrorHandler;
  /** Overrides the flux async error handler. Defaults to logging via console.error. */
  handleAsyncError?: status.AsyncErrorHandler;
  /** Extra aether components merged into the test render registry. */
  additionalRegistry?: aether.ComponentRegistry;
  /**
   * Seeds the given fake render context into the aether tree, so canvas-rendered
   * components can mount and record draw calls without a real canvas. Construct one
   * with {@link mockRenderContext} and keep the reference for assertions.
   */
  renderContext?: MockRenderContext;
}

const createFluxClient = (args: CreateSynnaxWrapperArgs): Flux.Client => {
  const { client, excludeFluxStores, handleError, handleAsyncError } = args;
  const storeConfig = { ...Pluto.FLUX_STORE_CONFIG };
  if (excludeFluxStores)
    excludeFluxStores.forEach((store) => delete storeConfig[store]);
  return new Flux.Client({
    client,
    storeConfig,
    handleError: handleError ?? status.createErrorHandler(console.error),
    handleAsyncError: handleAsyncError ?? status.createAsyncErrorHandler(console.error),
  });
};

export const createSynnaxWrapper = (
  args: CreateSynnaxWrapperArgs,
): FC<PropsWithChildren> =>
  newWrapper(
    args.client,
    createFluxClient(args),
    args.additionalRegistry,
    args.renderContext,
  );

export const createAsyncSynnaxWrapper = async (
  args: CreateSynnaxWrapperArgs,
): Promise<FC<PropsWithChildren>> => {
  const fluxClient = createFluxClient(args);
  await fluxClient.awaitInitialized();
  return newWrapper(
    args.client,
    fluxClient,
    args.additionalRegistry,
    args.renderContext,
  );
};

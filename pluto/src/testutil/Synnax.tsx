// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
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
import { canvasTest } from "@/vis/render/test";

interface RenderContextSeedProps extends PropsWithChildren {
  context: canvasTest.Recorder;
}

const RenderContextSeed = ({
  context,
  children,
}: RenderContextSeedProps): ReactElement => {
  const { path } = Aether.useLifecycle({
    type: canvasTest.RenderProvider.TYPE,
    schema: canvasTest.RenderProvider.stateZ,
    initialState: { context },
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};

const newWrapper = (
  client: Client | null,
  fluxClient: Flux.Client,
  additionalRegistry?: aether.ComponentRegistry,
  renderContext?: canvasTest.Recorder,
) => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...flux.createRegistry({ storeConfig: {} }),
    ...(renderContext != null
      ? { [canvasTest.RenderProvider.TYPE]: canvasTest.RenderProvider }
      : {}),
    ...additionalRegistry,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client}>
          <Flux.Provider client={fluxClient}>
            {renderContext == null ? (
              children
            ) : (
              <RenderContextSeed context={renderContext}>{children}</RenderContextSeed>
            )}
          </Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return Wrapper;
};

export interface CreateSynnaxWrapperParams {
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
   * with {@link canvasTest.record} and keep the reference for assertions.
   */
  renderContext?: canvasTest.Recorder;
}

const createFluxClient = (params: CreateSynnaxWrapperParams): Flux.Client => {
  const { client, excludeFluxStores, handleError, handleAsyncError } = params;
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
  params: CreateSynnaxWrapperParams,
): FC<PropsWithChildren> =>
  newWrapper(
    params.client,
    createFluxClient(params),
    params.additionalRegistry,
    params.renderContext,
  );

export const createAsyncSynnaxWrapper = async (
  params: CreateSynnaxWrapperParams,
): Promise<FC<PropsWithChildren>> => {
  const fluxClient = createFluxClient(params);
  await fluxClient.awaitInitialized();
  return newWrapper(
    params.client,
    fluxClient,
    params.additionalRegistry,
    params.renderContext,
  );
};

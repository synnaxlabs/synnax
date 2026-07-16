// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { cache, type Synnax as Client } from "@synnaxlabs/client";
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
  handlers: Pick<Flux.ProviderProps, "handleError" | "handleAsyncError">,
  additionalRegistry?: aether.ComponentRegistry,
  renderContext?: canvasTest.Recorder,
) => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...flux.createRegistry(),
    ...(renderContext != null
      ? { [canvasTest.RenderProvider.TYPE]: canvasTest.RenderProvider }
      : {}),
    ...additionalRegistry,
  });
  // One detached engine per wrapper, not per mount: sequential renders under the
  // same wrapper must share store state, as production providers outlive renders.
  // Controllers are prebuilt for the same reason: binding is once-per-engine.
  const detachedEngine =
    client == null || !client.cache.enabled
      ? new cache.Engine({ openStreamer: null })
      : undefined;
  let composers = Pluto.STORE_COMPOSERS;
  if (detachedEngine != null)
    composers = Object.fromEntries(
      Object.entries(Pluto.STORE_COMPOSERS).map(([key, compose]) => {
        const controller = compose({ client: null, engine: detachedEngine });
        return [key, () => controller];
      }),
    );
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client}>
          <Flux.Provider composers={composers} engine={detachedEngine} {...handlers}>
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
  /** Overrides the flux error handler. Defaults to the status aggregator. */
  handleError?: status.ErrorHandler;
  /** Overrides the flux async error handler. Defaults to the status aggregator. */
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

export const createSynnaxWrapper = ({
  client,
  handleError,
  handleAsyncError,
  additionalRegistry,
  renderContext,
}: CreateSynnaxWrapperParams): FC<PropsWithChildren> =>
  newWrapper(
    client,
    { handleError, handleAsyncError },
    additionalRegistry,
    renderContext,
  );

export const createAsyncSynnaxWrapper = async (
  params: CreateSynnaxWrapperParams,
): Promise<FC<PropsWithChildren>> => {
  const { client } = params;
  if (client != null && client.cache.enabled)
    await client.cache.engine.ensureStreaming();
  return createSynnaxWrapper(params);
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
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
  additionalRegistry?: aether.ComponentRegistry,
  renderContext?: canvasTest.Recorder,
  connectionStatus?: connection.Status,
) => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...(renderContext != null
      ? { [canvasTest.RenderProvider.TYPE]: canvasTest.RenderProvider }
      : {}),
    ...additionalRegistry,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={client} status={connectionStatus}>
          {renderContext == null ? (
            children
          ) : (
            <RenderContextSeed context={renderContext}>{children}</RenderContextSeed>
          )}
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return Wrapper;
};

export interface CreateSynnaxWrapperParams {
  client: Client | null;
  /** Extra aether components merged into the test render registry. */
  additionalRegistry?: aether.ComponentRegistry;
  /**
   * Seeds the given fake render context into the aether tree, so canvas-rendered
   * components can mount and record draw calls without a real canvas. Construct one
   * with {@link canvasTest.record} and keep the reference for assertions.
   */
  renderContext?: canvasTest.Recorder;
  /** Connection status the Synnax context reports; defaults to disconnected. */
  connectionStatus?: connection.Status;
}

export const createSynnaxWrapper = ({
  client,
  additionalRegistry,
  renderContext,
  connectionStatus,
}: CreateSynnaxWrapperParams): FC<PropsWithChildren> =>
  newWrapper(client, additionalRegistry, renderContext, connectionStatus);

export const createAsyncSynnaxWrapper = async (
  params: CreateSynnaxWrapperParams,
): Promise<FC<PropsWithChildren>> => {
  const { client } = params;
  if (client != null) await client.connect();
  return createSynnaxWrapper(params);
};

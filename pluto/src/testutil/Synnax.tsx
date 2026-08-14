// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection, type Synnax as Client } from "@synnaxlabs/client";
import { type FC, type PropsWithChildren, type ReactElement, Suspense } from "react";

import { Aether } from "@/aether";
import { type aether } from "@/aether/aether";
import { aetherTest } from "@/aether/test";
import { Alamos } from "@/alamos";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { Telem } from "@/telem";
import { telem } from "@/telem/aether";
import { theming } from "@/theming/aether";
import { canvasTest } from "@/vis/render/test";
import { Staleness } from "@/vis/staleness";
import { staleness } from "@/vis/staleness/aether";

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

const TEST_THEME = theming.themeZ.parse(theming.SYNNAX_THEMES.synnaxLight);

// Mounts the production aether theming provider without the React provider's
// font loading, which jsdom cannot perform (no FontFace).
const ThemingSeed = ({ children }: PropsWithChildren): ReactElement => {
  const { path } = Aether.useLifecycle({
    type: theming.Provider.TYPE,
    schema: theming.Provider.z,
    initialState: { theme: TEST_THEME, fontURLs: [] },
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
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
  /**
   * When defined, mounts the telemetry provider stack so components that resolve
   * telem sources (e.g. value cells) can mount. The factories are added to the
   * production factory set; pass `[new telemTest.TestFactory()]` to resolve
   * telemTest source and sink specs.
   */
  telemFactories?: telem.Factory[];
  /** Connection status the Synnax context reports; defaults to disconnected. */
  connectionStatus?: connection.Status;
}

export const createSynnaxWrapper = ({
  client,
  additionalRegistry,
  renderContext,
  telemFactories,
  connectionStatus,
}: CreateSynnaxWrapperParams): FC<PropsWithChildren> => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
    ...alamos.REGISTRY,
    ...theming.REGISTRY,
    ...staleness.REGISTRY,
    ...(renderContext != null
      ? { [canvasTest.RenderProvider.TYPE]: canvasTest.RenderProvider }
      : {}),
    ...(telemFactories != null
      ? {
          [telem.PROVIDER_TYPE]: telem.createProvider((c) =>
            telem.createFactory(c, telemFactories),
          ),
        }
      : {}),
    ...additionalRegistry,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => {
    let inner =
      renderContext == null ? (
        children
      ) : (
        <RenderContextSeed context={renderContext}>{children}</RenderContextSeed>
      );
    inner = <Staleness.Provider>{inner}</Staleness.Provider>;
    if (telemFactories != null) inner = <Telem.Provider>{inner}</Telem.Provider>;
    return (
      <AetherProvider>
        <Status.Aggregator>
          <Alamos.Provider>
            <Synnax.TestProvider client={client} status={connectionStatus}>
              <Suspense fallback={null}>
                <ThemingSeed>{inner}</ThemingSeed>
              </Suspense>
            </Synnax.TestProvider>
          </Alamos.Provider>
        </Status.Aggregator>
      </AetherProvider>
    );
  };
  return Wrapper;
};

export const createAsyncSynnaxWrapper = async (
  params: CreateSynnaxWrapperParams,
): Promise<FC<PropsWithChildren>> => {
  const { client } = params;
  if (client != null) await client.connect();
  return createSynnaxWrapper(params);
};

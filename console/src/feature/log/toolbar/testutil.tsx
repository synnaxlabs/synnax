// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log as clientLog, type log, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Log, Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, render, within } from "@testing-library/react";
import {
  type ComponentType,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  Suspense,
} from "react";

import { createResourceTab, primePanel } from "@/platform/panel/testutil";
import { type ConsolePreloadedState, createConsoleWrapper } from "@/testutil";

export const client = createTestClient();

let projectKey: string | undefined;
const project = async (): Promise<string> =>
  (projectKey ??= (await client.projects.create({ name: id.create(), layout: {} }))
    .key);

// loadLog primes key's flux cache through the production retrieve path. The single-hook
// bootstrap keeps the suspending useEnsure from being followed by other hooks,
// a shape that trips a React 19 concurrent-replay error.
const loadLog = async (Wrapper: FC<PropsWithChildren>, key: string): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    Log.useEnsure({ key });
    return <div data-testid="loaded" />;
  };
  let utils!: ReturnType<typeof render>;
  await act(async () => {
    utils = render(
      <Suspense fallback={null}>
        <Bootstrap />
      </Suspense>,
      { wrapper: Wrapper },
    );
  });
  await within(utils.container).findByTestId("loaded");
};

export interface RenderLogOptions {
  log?: Partial<log.New>;
  preloadedState?: (key: string) => ConsolePreloadedState;
  /** The client the component renders against; defaults to the root client. */
  as?: Client;
}

// renderLog creates a log on the server, mounts Component inside the panel and tab
// scopes of a seeded resource tab (the way the mosaic renders a tab) with the log
// loaded into the flux cache, and returns the render result plus the Redux store and
// log key.
export const renderLog = async (
  Component: ComponentType,
  { log: logOverrides, preloadedState, as = client }: RenderLogOptions = {},
) => {
  const created = await client.logs.create(await project(), {
    name: "Test Log",
    ...logOverrides,
  });
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client: as,
    preloadedState: preloadedState?.(created.key),
  });
  await loadLog(Wrapper, created.key);
  const { panelKey, tabKey } = await createResourceTab(
    client,
    clientLog.ontologyID(created.key),
  );
  await primePanel(Wrapper, panelKey);
  // The toolbar suspends on channel fetches; a tree that suspends inside a sync
  // act never commits, so the mount needs an async act (see renderHookSuspended).
  let result!: ReturnType<typeof render>;
  await act(async () => {
    result = render(
      <PlutoPanel.Scope.Provider value={panelKey}>
        <PlutoPanel.TabScope.Provider value={tabKey}>
          <Log.Scope.Provider value={created.key}>
            <Component />
          </Log.Scope.Provider>
        </PlutoPanel.TabScope.Provider>
      </PlutoPanel.Scope.Provider>,
      { wrapper: Wrapper },
    );
  });
  return { key: created.key, result, store };
};

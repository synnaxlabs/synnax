// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { LinePlot as PLinePlot, Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, render, within } from "@testing-library/react";
import {
  type ComponentType,
  type FC,
  type PropsWithChildren,
  type ReactElement,
  Suspense,
} from "react";

import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  type ConsolePreloadedState,
  createConsoleWrapper,
  uniqueName,
} from "@/testutil";

export const client = createTestClient();

let projectKey: string | undefined;
export const project = async (): Promise<string> =>
  (projectKey ??= (await client.projects.create({ name: id.create(), layout: {} }))
    .key);

// loadLinePlot primes key's flux cache through the production retrieve path. The
// single-hook bootstrap keeps the suspending useEnsureRetrieved from being followed by
// other hooks, a shape that trips a React 19 concurrent-replay error.
const loadLinePlot = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    PLinePlot.useEnsureRetrieved({ key });
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

export const createPreloadedState = (
  key: string,
  plotState: Partial<Session.LinePlot.State> = {},
): ConsolePreloadedState => ({
  [Session.LinePlot.SLICE_NAME]: {
    ...Session.LinePlot.ZERO_SLICE_STATE,
    plots: { [key]: { ...Session.LinePlot.ZERO_STATE, ...plotState } },
  },
});

// createResourceTab creates a single-leaf panel holding one resource tab that backs
// the given plot on the cluster, so the panel scope hooks a mounted tab content reads
// (useSelectTabResource) resolve to the plot's ontology ID.
const createResourceTab = async (
  key: string,
): Promise<{ panelKey: string; tabKey: string }> => {
  const tabKey = uuid.create();
  const doc = panel.panelZ.parse({
    name: uniqueName("panel"),
    root: {
      variant: "leaf",
      tabs: [{ variant: "resource", key: tabKey, resource: lineplot.ontologyID(key) }],
    },
  });
  await client.panels.create(doc);
  // Prime the query cache the way the mosaic's retrieve does.
  await client.panels.retrieve({ key: doc.key });
  return { panelKey: doc.key, tabKey };
};

export interface RenderLinePlotOptions {
  linePlot?: Partial<lineplot.New>;
  preloadedState?: (key: string) => ConsolePreloadedState;
}

// renderLinePlot creates a line plot on the server, mounts Component inside the panel
// and tab scopes of a seeded resource tab (the way the mosaic renders a tab) with the
// plot loaded into the flux cache and a live Modals.Stack, and returns the render
// result plus the Redux store and plot key.
export const renderLinePlot = async (
  Component: ComponentType,
  { linePlot: overrides, preloadedState }: RenderLinePlotOptions = {},
) => {
  const created = await client.lineplots.create(await project(), {
    name: "Test Plot",
    ...overrides,
  });
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: preloadedState?.(created.key),
  });
  await loadLinePlot(Wrapper, created.key);
  const { panelKey, tabKey } = await createResourceTab(created.key);
  const result = render(
    <PlutoPanel.Scope.Provider value={panelKey}>
      <PlutoPanel.TabScope.Provider value={tabKey}>
        <Component />
        <Modals.Stack />
      </PlutoPanel.TabScope.Provider>
    </PlutoPanel.Scope.Provider>,
    { wrapper: Wrapper },
  );
  return { key: created.key, result, store };
};

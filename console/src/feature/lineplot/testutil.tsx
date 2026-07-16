// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
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
import { type ConsolePreloadedState, createConsoleWrapper } from "@/testutil";

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

export const createLayoutState = (key: string, name: string): Session.Layout.State => ({
  key,
  windowKey: MAIN_WINDOW,
  type: "lineplot",
  name,
  location: "mosaic",
});

export const createPreloadedState = (
  key: string,
  name: string,
  plotState: Partial<Session.LinePlot.State> = {},
): ConsolePreloadedState => ({
  [Session.Layout.SLICE_NAME]: {
    ...Session.Layout.ZERO_SLICE_STATE,
    layouts: {
      ...Session.Layout.ZERO_SLICE_STATE.layouts,
      [key]: createLayoutState(key, name),
    },
    mosaics: {
      ...Session.Layout.ZERO_SLICE_STATE.mosaics,
      [MAIN_WINDOW]: {
        activeTab: key,
        focused: null,
        root: { key: 1, tabs: [{ tabKey: key, name }] },
      },
    },
  },
  [Session.LinePlot.SLICE_NAME]: {
    ...Session.LinePlot.ZERO_SLICE_STATE,
    plots: { [key]: { ...Session.LinePlot.ZERO_STATE, ...plotState } },
  },
});

export interface RenderLinePlotOptions {
  linePlot?: Partial<lineplot.New>;
  preloadedState?: (key: string) => ConsolePreloadedState;
}

// renderLinePlot creates a line plot on the server, mounts Component with the plot
// loaded into the flux cache and a live Modals.Stack, and returns the render result
// plus the Redux store and plot key.
export const renderLinePlot = async (
  Component: ComponentType<{ layoutKey: string }>,
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
  const result = render(
    <>
      <Component layoutKey={created.key} />
      <Modals.Stack />
    </>,
    { wrapper: Wrapper },
  );
  return { key: created.key, result, store };
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type table } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Table as PTable } from "@synnaxlabs/pluto";
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

// createCellGrid builds a two-cell text table body: cells "a" and "b" in one row.
export const createCellGrid = (): Pick<table.New, "rows" | "columns" | "cells"> => ({
  rows: [{ size: 36, cells: ["a", "b"] }],
  columns: [{ size: 72 }, { size: 72 }],
  cells: {
    a: {
      key: "a",
      variant: "text",
      props: { value: "Cell A", level: "h5", backgroundColor: "#ff0000ff" },
    },
    b: {
      key: "b",
      variant: "text",
      props: { value: "Cell B", level: "h5", backgroundColor: "#00ff00ff" },
    },
  },
});

// loadTable primes key's flux cache through the production retrieve path. The
// single-hook bootstrap keeps the suspending useEnsureRetrieved from being followed by
// other hooks, a shape that trips a React 19 concurrent-replay error.
const loadTable = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    PTable.useEnsureRetrieved({ key });
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
  type: "table",
  name,
  location: "mosaic",
});

export const createPreloadedState = (
  key: string,
  name: string,
  tableState: Partial<Session.Table.State> = {},
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
  [Session.Table.SLICE_NAME]: {
    ...Session.Table.ZERO_SLICE_STATE,
    tables: { [key]: { ...Session.Table.ZERO_STATE, ...tableState } },
  },
});

export interface RenderTableOptions {
  table?: Partial<table.New>;
  preloadedState?: (key: string) => ConsolePreloadedState;
}

// renderTable creates a table on the server, mounts Component with the table loaded
// into the flux cache and a live Modals.Stack, and returns the render result plus the
// Redux store and table key.
export const renderTable = async (
  Component: ComponentType<{ layoutKey: string }>,
  { table: overrides, preloadedState }: RenderTableOptions = {},
) => {
  const created = await client.tables.create(await project(), {
    name: "Test Table",
    ...overrides,
  });
  const { wrapper: Wrapper, store } = await createConsoleWrapper({
    client,
    preloadedState: preloadedState?.(created.key),
  });
  await loadTable(Wrapper, created.key);
  const result = render(
    <>
      <Component layoutKey={created.key} />
      <Modals.Stack />
    </>,
    { wrapper: Wrapper },
  );
  return { key: created.key, result, store };
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  combineReducers,
  configureStore,
  type Reducer,
  type UnknownAction,
} from "@reduxjs/toolkit";
import {
  createTestClient,
  type project,
  type Synnax,
  type table,
} from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { Table } from "@/table";
import { LAYOUT_TYPE } from "@/table/layout";
import { type PendingUpload } from "@/table/types";
import { useAutoUpload } from "@/table/useUpload";

const client: Synnax = createTestClient();

interface RootState {
  [Drift.SLICE_NAME]: Drift.SliceState;
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Table.SLICE_NAME]: Table.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
}

const rootReducer = combineReducers({
  [Drift.SLICE_NAME]: Drift.reducer,
  [Layout.SLICE_NAME]: Layout.reducer,
  [Table.SLICE_NAME]: Table.reducer,
  [Project.SLICE_NAME]: Project.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface SeedArgs {
  tableKey: string;
  layoutName: string;
  pendingUpload?: PendingUpload;
  activeProject?: project.Project;
}

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

const buildHarness = async ({
  tableKey,
  layoutName,
  pendingUpload,
  activeProject,
}: SeedArgs): Promise<Harness> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: () => {},
    handleAsyncError: async () => {},
  });
  await fluxClient.awaitInitialized();
  const preloadedState: RootState = {
    [Drift.SLICE_NAME]: Drift.ZERO_SLICE_STATE,
    [Layout.SLICE_NAME]: {
      ...Layout.ZERO_SLICE_STATE,
      layouts: {
        ...Layout.ZERO_SLICE_STATE.layouts,
        [tableKey]: {
          windowKey: "main",
          key: tableKey,
          type: LAYOUT_TYPE,
          name: layoutName,
          location: "mosaic",
        },
      },
    },
    [Table.SLICE_NAME]: {
      ...Table.ZERO_SLICE_STATE,
      tables: {
        [tableKey]: {
          ...Table.ZERO_STATE,
          key: tableKey,
          pendingUpload,
        },
      },
    },
    [Project.SLICE_NAME]: {
      ...Project.ZERO_SLICE_STATE,
      active: activeProject ?? null,
    },
  };
  const store = configureStore({ reducer: rootReducer, preloadedState });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <Status.Aggregator>
        <PSynnax.TestProvider client={client}>
          <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
        </PSynnax.TestProvider>
      </Status.Aggregator>
    </Provider>
  );
  return { wrapper: Wrapper, store };
};

const buildPendingUpload = (override: Partial<PendingUpload> = {}): PendingUpload => {
  const key = override.key ?? uuid.create();
  const a = id.create();
  const b = id.create();
  return {
    key,
    rows: [{ size: 36, cells: [a, b] }],
    columns: [{ size: 72 }, { size: 72 }],
    cells: {
      [a]: { key: a, variant: "text", props: { value: "A" } },
      [b]: { key: b, variant: "text", props: { value: "B" } },
    },
    ...override,
  };
};

describe("useAutoUpload", () => {
  let projectA: project.Project;

  beforeEach(async () => {
    projectA = await client.projects.create({ name: `proj-${id.create()}` });
  });

  it("returns true when there is no pending upload and does not call the server", async () => {
    const tableKey = uuid.create();
    const { wrapper } = await buildHarness({
      tableKey,
      layoutName: "Already Uploaded",
      activeProject: projectA,
    });
    const { result } = renderHook(() => useAutoUpload(tableKey), { wrapper });
    expect(result.current).toBe(true);
    await expect(client.tables.retrieve({ key: tableKey })).rejects.toThrow();
  });

  it("uploads the pending payload using the layout name, then returns true", async () => {
    const tableKey = uuid.create();
    const pendingUpload = buildPendingUpload({ key: tableKey });
    const { wrapper, store } = await buildHarness({
      tableKey,
      layoutName: "Live Layout Name",
      pendingUpload,
      activeProject: projectA,
    });
    const { result } = renderHook(() => useAutoUpload(tableKey), { wrapper });
    expect(result.current).toBe(false);
    await waitFor(() => expect(result.current).toBe(true));
    const uploaded: table.Table = await client.tables.retrieve({ key: tableKey });
    expect(uploaded.name).toEqual("Live Layout Name");
    expect(uploaded.rows).toEqual(pendingUpload.rows);
    expect(uploaded.columns).toEqual(pendingUpload.columns);
    expect(uploaded.cells).toEqual(pendingUpload.cells);
    expect(
      store.getState()[Table.SLICE_NAME].tables[tableKey].pendingUpload,
    ).toBeUndefined();
  });

  it("uploads without a project when none is active", async () => {
    const tableKey = uuid.create();
    const pendingUpload = buildPendingUpload({ key: tableKey });
    const { wrapper, store } = await buildHarness({
      tableKey,
      layoutName: "Projectless",
      pendingUpload,
    });
    const { result } = renderHook(() => useAutoUpload(tableKey), { wrapper });
    await waitFor(() => expect(result.current).toBe(true));
    const uploaded = await client.tables.retrieve({ key: tableKey });
    expect(uploaded.name).toEqual("Projectless");
    expect(uploaded.rows).toEqual(pendingUpload.rows);
    expect(
      store.getState()[Table.SLICE_NAME].tables[tableKey].pendingUpload,
    ).toBeUndefined();
  });

  it("does not re-upload on subsequent renders once pendingUpload is cleared", async () => {
    const tableKey = uuid.create();
    const pendingUpload = buildPendingUpload({ key: tableKey });
    const { wrapper, store } = await buildHarness({
      tableKey,
      layoutName: "Single Upload",
      pendingUpload,
      activeProject: projectA,
    });
    const { result, rerender } = renderHook(() => useAutoUpload(tableKey), {
      wrapper,
    });
    await waitFor(() => expect(result.current).toBe(true));
    const first = await client.tables.retrieve({ key: tableKey });
    rerender();
    rerender();
    expect(
      store.getState()[Table.SLICE_NAME].tables[tableKey].pendingUpload,
    ).toBeUndefined();
    const second = await client.tables.retrieve({ key: tableKey });
    expect(second).toEqual(first);
  });
});

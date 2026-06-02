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
  type Synnax,
  table,
  type workspace,
} from "@synnaxlabs/client";
import { Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Workspace } from "@/workspace";
import { useAdoptIntoActiveWorkspace } from "@/workspace/useAdoptIntoActiveWorkspace";

const client: Synnax = createTestClient();

interface RootState {
  [Workspace.SLICE_NAME]: Workspace.SliceState;
}

const rootReducer = combineReducers({
  [Workspace.SLICE_NAME]: Workspace.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

const stripLayout = (ws: workspace.Workspace): Omit<workspace.Workspace, "layout"> => {
  const { layout: _, ...rest } = ws;
  return rest;
};

const buildHarness = async (
  activeWorkspace?: workspace.Workspace,
): Promise<Harness> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: () => {},
    handleAsyncError: async () => {},
  });
  await fluxClient.awaitInitialized();
  const preloadedState: RootState = {
    [Workspace.SLICE_NAME]: {
      ...Workspace.ZERO_SLICE_STATE,
      active: activeWorkspace != null ? stripLayout(activeWorkspace) : null,
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

const parentKeys = async (tableKey: string): Promise<string[]> =>
  (await client.ontology.retrieveParents(table.ontologyID(tableKey))).map(
    (resource) => resource.id.key,
  );

const createOrphanTable = async (): Promise<string> => {
  const key = uuid.create();
  await client.tables.create(uuid.ZERO, { key, name: "orphan" });
  return key;
};

describe("useAdoptIntoActiveWorkspace", () => {
  let workspaceA: workspace.Workspace;
  let workspaceB: workspace.Workspace;

  beforeEach(async () => {
    workspaceA = await client.workspaces.create({
      name: `ws-a-${id.create()}`,
      layout: {},
    });
    workspaceB = await client.workspaces.create({
      name: `ws-b-${id.create()}`,
      layout: {},
    });
  });

  it("adopts an orphaned resource into the workspace active on mount", async () => {
    const tableKey = await createOrphanTable();
    expect(await parentKeys(tableKey)).toHaveLength(0);
    const { wrapper } = await buildHarness(workspaceA);
    renderHook(() => useAdoptIntoActiveWorkspace(table.ontologyID(tableKey)), {
      wrapper,
    });
    await waitFor(async () =>
      expect(await parentKeys(tableKey)).toEqual([workspaceA.key]),
    );
  });

  it("adopts an orphaned resource when a workspace becomes active after mount", async () => {
    const tableKey = await createOrphanTable();
    const { wrapper, store } = await buildHarness();
    renderHook(() => useAdoptIntoActiveWorkspace(table.ontologyID(tableKey)), {
      wrapper,
    });
    act(() => {
      store.dispatch(Workspace.setActive(stripLayout(workspaceA)));
    });
    // The resource ends up parented to exactly the workspace that became active,
    // which would not hold if the hook had acted while no workspace was active.
    await waitFor(async () =>
      expect(await parentKeys(tableKey)).toEqual([workspaceA.key]),
    );
  });

  it("does not transfer a resource that already belongs to another workspace", async () => {
    const ownedKey = uuid.create();
    await client.tables.create(workspaceA.key, { key: ownedKey, name: "in-a" });
    const orphanKey = await createOrphanTable();
    const { wrapper } = await buildHarness(workspaceB);
    renderHook(
      () => {
        useAdoptIntoActiveWorkspace(table.ontologyID(ownedKey));
        useAdoptIntoActiveWorkspace(table.ontologyID(orphanKey));
      },
      { wrapper },
    );
    // Adopting the orphan into B is the barrier: it takes strictly more server round
    // trips than the owned table's decision, so once it lands both effects have run.
    await waitFor(async () =>
      expect(await parentKeys(orphanKey)).toEqual([workspaceB.key]),
    );
    expect(await parentKeys(ownedKey)).toEqual([workspaceA.key]);
  });
});

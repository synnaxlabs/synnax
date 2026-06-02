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
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { Workspace } from "@/workspace";
import { useAdoptIntoActiveWorkspace } from "@/workspace/useAdoptIntoActiveWorkspace";

const client: Synnax = createTestClient();

interface RootState {
  [Workspace.SLICE_NAME]: Workspace.SliceState;
}

const rootReducer = combineReducers({
  [Workspace.SLICE_NAME]: Workspace.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

const stripLayout = (ws: workspace.Workspace): Omit<workspace.Workspace, "layout"> => {
  const { layout: _, ...rest } = ws;
  return rest;
};

const buildWrapper = async (
  activeWorkspace?: workspace.Workspace,
): Promise<FC<PropsWithChildren>> => {
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
  return Wrapper;
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

  it("adopts an orphaned resource into the active workspace", async () => {
    const tableKey = await createOrphanTable();
    expect(await parentKeys(tableKey)).toHaveLength(0);
    const wrapper = await buildWrapper(workspaceA);
    renderHook(() => useAdoptIntoActiveWorkspace(table.ontologyID(tableKey)), {
      wrapper,
    });
    await waitFor(async () =>
      expect(await parentKeys(tableKey)).toContain(workspaceA.key),
    );
  });

  it("does nothing when there is no active workspace", async () => {
    const tableKey = await createOrphanTable();
    const addChildren = vi.spyOn(client.ontology, "addChildren");
    const wrapper = await buildWrapper();
    renderHook(() => useAdoptIntoActiveWorkspace(table.ontologyID(tableKey)), {
      wrapper,
    });
    // Give the async effect time to flush its task queue and run to completion.
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(addChildren).not.toHaveBeenCalled();
    expect(await parentKeys(tableKey)).toHaveLength(0);
  });

  it("does not transfer a resource that already belongs to another workspace", async () => {
    const tableKey = uuid.create();
    await client.tables.create(workspaceA.key, { key: tableKey, name: "in-a" });
    expect(await parentKeys(tableKey)).toContain(workspaceA.key);
    const addChildren = vi.spyOn(client.ontology, "addChildren");
    const retrieveParents = vi.spyOn(client.ontology, "retrieveParents");
    const wrapper = await buildWrapper(workspaceB);
    renderHook(() => useAdoptIntoActiveWorkspace(table.ontologyID(tableKey)), {
      wrapper,
    });
    // The store cache is cold, so the hook must confirm the existing parent against
    // the server before deciding not to adopt.
    await waitFor(() => expect(retrieveParents).toHaveBeenCalled());
    expect(addChildren).not.toHaveBeenCalled();
    const parents = await parentKeys(tableKey);
    expect(parents).toContain(workspaceA.key);
    expect(parents).not.toContain(workspaceB.key);
  });
});

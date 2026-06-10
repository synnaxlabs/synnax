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
import { createTestClient, type project,type Synnax, table } from "@synnaxlabs/client";
import { Flux, Pluto, Status, Synnax as PSynnax } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Project } from "@/project";
import { useAdoptIntoActiveProject } from "@/project/useAdoptIntoActiveProject";

const client: Synnax = createTestClient();

interface RootState {
  [Project.SLICE_NAME]: Project.SliceState;
}

const rootReducer = combineReducers({
  [Project.SLICE_NAME]: Project.reducer,
}) as unknown as Reducer<RootState, UnknownAction>;

type RootStore = ReturnType<typeof configureStore<RootState>>;

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: RootStore;
}

const stripLayout = (ws: project.Project): Omit<project.Project, "layout"> => {
  const { layout: _, ...rest } = ws;
  return rest;
};

const buildHarness = async (activeProject?: project.Project): Promise<Harness> => {
  const fluxClient = new Flux.Client({
    client,
    storeConfig: Pluto.FLUX_STORE_CONFIG,
    handleError: () => {},
    handleAsyncError: async () => {},
  });
  await fluxClient.awaitInitialized();
  const preloadedState: RootState = {
    [Project.SLICE_NAME]: {
      ...Project.ZERO_SLICE_STATE,
      active: activeProject != null ? stripLayout(activeProject) : null,
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

describe("useAdoptIntoActiveProject", () => {
  let projectA: project.Project;
  let projectB: project.Project;

  beforeEach(async () => {
    projectA = await client.projects.create({
      name: `ws-a-${id.create()}`,
      layout: {},
    });
    projectB = await client.projects.create({
      name: `ws-b-${id.create()}`,
      layout: {},
    });
  });

  it("adopts an orphaned resource into the project active on mount", async () => {
    const tableKey = await createOrphanTable();
    expect(await parentKeys(tableKey)).toHaveLength(0);
    const { wrapper } = await buildHarness(projectA);
    renderHook(() => useAdoptIntoActiveProject(table.ontologyID(tableKey)), {
      wrapper,
    });
    await waitFor(async () =>
      expect(await parentKeys(tableKey)).toEqual([projectA.key]),
    );
  });

  it("adopts an orphaned resource when a project becomes active after mount", async () => {
    const tableKey = await createOrphanTable();
    const { wrapper, store } = await buildHarness();
    renderHook(() => useAdoptIntoActiveProject(table.ontologyID(tableKey)), {
      wrapper,
    });
    act(() => {
      store.dispatch(Project.setActive(stripLayout(projectA)));
    });
    // The resource ends up parented to exactly the project that became active,
    // which would not hold if the hook had acted while no project was active.
    await waitFor(async () =>
      expect(await parentKeys(tableKey)).toEqual([projectA.key]),
    );
  });

  it("does not transfer a resource that already belongs to another project", async () => {
    const ownedKey = uuid.create();
    await client.tables.create(projectA.key, { key: ownedKey, name: "in-a" });
    const orphanKey = await createOrphanTable();
    const { wrapper } = await buildHarness(projectB);
    renderHook(
      () => {
        useAdoptIntoActiveProject(table.ontologyID(ownedKey));
        useAdoptIntoActiveProject(table.ontologyID(orphanKey));
      },
      { wrapper },
    );
    // Adopting the orphan into B is the barrier: it takes strictly more server round
    // trips than the owned table's decision, so once it lands both effects have run.
    await waitFor(async () =>
      expect(await parentKeys(orphanKey)).toEqual([projectB.key]),
    );
    expect(await parentKeys(ownedKey)).toEqual([projectA.key]);
  });
});

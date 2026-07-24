// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { type Haul } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { assertDefined } from "@/testutil";

const HAULED: Haul.DraggingState = {
  source: { key: "file", type: "file" },
  items: [{ key: "a", type: "b" }],
};

const withoutDrift = (state: Session.State): Partial<Session.State> => {
  const copy: Partial<Session.State> = { ...state };
  delete copy[Drift.SLICE_NAME];
  return copy;
};

/**
 * Reads back a partition the persist middleware wrote to localStorage, following the
 * same version-indexed key scheme the production engine uses on reload.
 */
const readPartition = (base: string): Partial<Session.State> | null => {
  const kvBase = Session.Persist.STORE_PATH;
  const versionRaw = localStorage.getItem(
    `${kvBase}:${Session.Persist.partitionVersionKey(base)}`,
  );
  if (versionRaw == null) return null;
  const { version } = JSON.parse(versionRaw) as { version: number };
  const stateRaw = localStorage.getItem(
    `${kvBase}:${Session.Persist.partitionStateKey(base, version)}`,
  );
  return stateRaw == null ? null : (JSON.parse(stateRaw) as Partial<Session.State>);
};

describe("createStore", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("initializes every slice to its zero state", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
    });
    expect(Object.keys(store.getState()).sort()).toEqual(
      Object.keys(Session.ZERO_STATE).sort(),
    );
    expect(withoutDrift(store.getState())).toStrictEqual(
      withoutDrift(Session.ZERO_STATE),
    );
  });

  it("routes dispatched actions to their owning slice", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
    });
    store.dispatch(Session.Cluster.select("DEMO"));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    expect(Session.Cluster.selectSelectedKey(store.getState())).toBe("DEMO");
    expect(Session.Nav.selectWindowState(store.getState()).bottom.visible).toBe(true);
  });

  it("honors an explicit preloadedState", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
      preloadedState: deep.copy({
        ...Session.ZERO_STATE,
        [Session.Cluster.SLICE_NAME]: {
          ...Session.Cluster.ZERO_SLICE_STATE,
          selected: "DEMO",
        },
      }),
    });
    expect(Session.Cluster.selectSelectedKey(store.getState())).toBe("DEMO");
  });

  it("persists state and reloads it into a fresh store", async () => {
    const store = await Session.createStore({ enablePrerender: false });
    store.dispatch(Session.Cluster.select("DEMO"));
    await waitFor(() => {
      const l0 = readPartition(Session.Persist.l0PartitionBase());
      if (l0?.cluster?.selected !== "DEMO")
        throw new Error("cluster selection not persisted yet");
    });
    const reloaded = await Session.createStore({ enablePrerender: false });
    expect(Session.Cluster.selectSelectedKey(reloaded.getState())).toBe("DEMO");
  });

  it("excludes transient haul state from persistence", async () => {
    const projectKey = "6f7cd5f4-4b93-4a35-a55c-72ba9dae2c9d";
    const l2Base = Session.Persist.l2PartitionBase("DEMO", projectKey);
    const store = await Session.createStore({ enablePrerender: false });
    store.dispatch(Session.Cluster.select("DEMO"));
    store.dispatch(Session.Project.select(projectKey));
    await waitFor(() => {
      if (readPartition(l2Base) == null)
        throw new Error("project partition not persisted yet");
    });
    store.dispatch(Session.Haul.setHauled(HAULED));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    await waitFor(() => {
      const l2 = readPartition(l2Base);
      if (l2?.nav == null || !Object.values(l2.nav.windows)[0].bottom.visible)
        throw new Error("nav change not persisted yet");
    });
    const persisted = readPartition(l2Base);
    assertDefined(persisted);
    expect(store.getState().haul.state).toStrictEqual(HAULED);
    expect(persisted.haul).toStrictEqual(Session.Haul.ZERO_SLICE_STATE);
  });
});

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
 * Reads back the state the persist middleware wrote to localStorage, following the same
 * version-indexed key scheme the production engine uses on reload.
 */
const readPersisted = (): Session.State | null => {
  const base = Session.Persist.V2_STORE_PATH;
  const versionRaw = localStorage.getItem(`${base}:${Session.Persist.DB_VERSION_KEY}`);
  if (versionRaw == null) return null;
  const { version } = JSON.parse(versionRaw) as { version: number };
  const stateRaw = localStorage.getItem(
    `${base}:${Session.Persist.persistedStateKey(version)}`,
  );
  return stateRaw == null ? null : (JSON.parse(stateRaw) as Session.State);
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
    store.dispatch(Session.Node.select("DEMO"));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    expect(Session.Node.selectSelectedKey(store.getState())).toBe("DEMO");
    expect(Session.Nav.selectBottomVisible(store.getState())).toBe(true);
  });

  it("honors an explicit preloadedState", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
      preloadedState: deep.copy({
        ...Session.ZERO_STATE,
        [Session.Node.SLICE_NAME]: {
          ...Session.Node.ZERO_SLICE_STATE,
          selected: "DEMO",
        },
      }),
    });
    expect(Session.Node.selectSelectedKey(store.getState())).toBe("DEMO");
  });

  it("persists state and reloads it into a fresh store", async () => {
    const store = await Session.createStore({ enablePrerender: false });
    store.dispatch(Session.Node.select("DEMO"));
    await waitFor(() => {
      if (readPersisted()?.node.selected !== "DEMO")
        throw new Error("node selection not persisted yet");
    });
    const reloaded = await Session.createStore({ enablePrerender: false });
    expect(Session.Node.selectSelectedKey(reloaded.getState())).toBe("DEMO");
  });

  it("excludes transient haul state from persistence", async () => {
    const store = await Session.createStore({ enablePrerender: false });
    store.dispatch(Session.Node.select("DEMO"));
    store.dispatch(Session.Haul.setHauled(HAULED));
    await waitFor(() => {
      if (readPersisted()?.node.selected !== "DEMO")
        throw new Error("state not persisted yet");
    });
    const persisted = readPersisted();
    assertDefined(persisted);
    expect(store.getState().haul.state).toStrictEqual(HAULED);
    expect(persisted.haul).toStrictEqual(Session.Haul.ZERO_SLICE_STATE);
    expect(persisted.node.selected).toBe("DEMO");
  });
});

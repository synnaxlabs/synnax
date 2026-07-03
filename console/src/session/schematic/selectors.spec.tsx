// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import {
  access,
  channel,
  createTestClient,
  createTestClientWithPolicy,
  framer,
  schematic,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import { Schematic as PlutoSchematic } from "@synnaxlabs/pluto";
import { color, id, uuid } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor, within } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, Suspense } from "react";
import { Provider } from "react-redux";
import { beforeAll, describe, expect, it } from "vitest";

import { Schematic } from "@/session/schematic";
import { createConsoleWrapper } from "@/testutil";

const KEY = "schematic-1";

const customState = Schematic.stateZ.parse({
  control: { authority: 250, status: "acquired" },
  selected: ["a", "b"],
  legend: { visible: false, colors: { a: color.ZERO } },
  toolbar: { selectedTab: "properties", selectedSymbolGroup: "valves" },
  editable: true,
  fitViewOnResize: true,
  viewport: { position: { x: 7, y: 8 }, zoom: 2, mode: "pan" },
});

const storeState: Schematic.StoreState = {
  [Schematic.SLICE_NAME]: { schematics: { [KEY]: customState } },
};

const params = { state: storeState, key: KEY };

describe("schematic selectors", () => {
  describe("selectSliceState", () => {
    it("should return the slice state", () => {
      expect(Schematic.selectSliceState(storeState)).toBe(
        storeState[Schematic.SLICE_NAME],
      );
    });
  });

  describe("selectState", () => {
    it("should return the state for the given key", () => {
      expect(Schematic.selectState(params)).toEqual(customState);
    });

    it("should fall back to ZERO_STATE for an unknown key", () => {
      expect(Schematic.selectState({ state: storeState, key: "absent" })).toEqual(
        Schematic.ZERO_STATE,
      );
    });
  });

  describe("selectSelected", () => {
    it("should return the selected elements", () => {
      expect(Schematic.selectSelected(params)).toEqual(["a", "b"]);
    });
  });

  describe("selectControlStatus", () => {
    it("should return the control status", () => {
      expect(Schematic.selectControlStatus(params)).toBe("acquired");
    });
  });

  describe("selectControlIsAcquired", () => {
    it("should be true when the status is acquired", () => {
      expect(Schematic.selectControlIsAcquired(params)).toBe(true);
    });

    it("should be false for an unknown key defaulting to ZERO_STATE", () => {
      expect(
        Schematic.selectControlIsAcquired({ state: storeState, key: "absent" }),
      ).toBe(false);
    });
  });

  describe("selectAuthority", () => {
    it("should return the control authority", () => {
      expect(Schematic.selectAuthority(params)).toBe(250);
    });
  });

  describe("selectToolbar", () => {
    it("should return the toolbar state", () => {
      expect(Schematic.selectToolbar(params)).toEqual(customState.toolbar);
    });
  });

  describe("selectActiveToolbarTab", () => {
    it("should return the active toolbar tab", () => {
      expect(Schematic.selectActiveToolbarTab(params)).toBe("properties");
    });
  });

  describe("selectSelectedSymbolGroup", () => {
    it("should return the selected symbol group", () => {
      expect(Schematic.selectSelectedSymbolGroup(params)).toBe("valves");
    });
  });

  describe("selectLegend", () => {
    it("should return the legend state", () => {
      expect(Schematic.selectLegend(params)).toEqual(customState.legend);
    });
  });

  describe("selectLegendVisible", () => {
    it("should return the legend visibility", () => {
      expect(Schematic.selectLegendVisible(params)).toBe(false);
    });
  });

  describe("selectEditable", () => {
    it("should return the editable flag", () => {
      expect(Schematic.selectEditable(params)).toBe(true);
    });
  });

  describe("selectFitViewOnResize", () => {
    it("should return the fit view on resize flag", () => {
      expect(Schematic.selectFitViewOnResize(params)).toBe(true);
    });
  });

  describe("selectViewport", () => {
    it("should return the viewport", () => {
      expect(Schematic.selectViewport(params)).toEqual(customState.viewport);
    });
  });
});

const storeWith = (slice: Schematic.SliceState) =>
  configureStore({
    reducer: { [Schematic.SLICE_NAME]: Schematic.reducer },
    preloadedState: { [Schematic.SLICE_NAME]: slice },
  });

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PlutoSchematic.Scope.Provider value={key}>
        {children}
      </PlutoSchematic.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("schematic selector hooks", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ schematics: { [KEY]: customState } });

  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Schematic.useSelect(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Schematic.useSelect({ key: "absent" }), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(Schematic.ZERO_STATE);
  });

  it("should return the selected elements", () => {
    const { result } = renderHook(() => Schematic.useSelectSelected(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(["a", "b"]);
  });

  it("should return the control status", () => {
    const { result } = renderHook(() => Schematic.useSelectControlStatus(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("acquired");
  });

  it("should return whether control is acquired", () => {
    const { result } = renderHook(() => Schematic.useSelectControlIsAcquired(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should return the control authority", () => {
    const { result } = renderHook(() => Schematic.useSelectAuthority(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(250);
  });

  it("should return the toolbar state", () => {
    const { result } = renderHook(() => Schematic.useSelectToolbar(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState.toolbar);
  });

  it("should return the active toolbar tab", () => {
    const { result } = renderHook(() => Schematic.useSelectActiveToolbarTab(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("properties");
  });

  it("should return the selected symbol group", () => {
    const { result } = renderHook(() => Schematic.useSelectSelectedSymbolGroup(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe("valves");
  });

  it("should return the legend state", () => {
    const { result } = renderHook(() => Schematic.useSelectLegend(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState.legend);
  });

  it("should return the legend visibility", () => {
    const { result } = renderHook(() => Schematic.useSelectLegendVisible(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(false);
  });

  it("should return the fit view on resize flag", () => {
    const { result } = renderHook(() => Schematic.useSelectFitViewOnResize(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toBe(true);
  });

  it("should return the viewport", () => {
    const { result } = renderHook(() => Schematic.useSelectViewport(), {
      wrapper: wrapperFor(store(), KEY),
    });
    expect(result.current).toEqual(customState.viewport);
  });
});

describe("schematic selector stability under dispatch", () => {
  const store = (): ReturnType<typeof storeWith> =>
    storeWith({ schematics: { [KEY]: customState } });

  it("should keep a stable reference when an unrelated field changes", () => {
    const s = store();
    const { result } = renderHook(() => Schematic.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Schematic.setControlAuthority({ key: KEY, authority: 1 }));
    });
    expect(result.current).toBe(first);
  });

  it("should return a new reference when the tracked field changes", () => {
    const s = store();
    const { result } = renderHook(() => Schematic.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Schematic.selectToolbarTab({ key: KEY, tab: "symbols" }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current.selectedTab).toBe("symbols");
  });

  it("should ignore changes to other schematics", () => {
    const s = store();
    const { result } = renderHook(() => Schematic.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Schematic.internalCreate({ key: "schematic-2" }));
      s.dispatch(Schematic.selectToolbarTab({ key: "schematic-2", tab: "symbols" }));
    });
    expect(result.current).toBe(first);
  });

  it("should re-point the selector when its key dependency changes", () => {
    const s = storeWith({
      schematics: {
        [KEY]: customState,
        "schematic-2": Schematic.stateZ.parse({
          toolbar: { selectedSymbolGroup: "pumps" },
        }),
      },
    });
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) => Schematic.useSelectSelectedSymbolGroup({ key }),
      { wrapper: wrapperFor(s, KEY), initialProps: { key: KEY } },
    );
    expect(result.current).toBe("valves");
    rerender({ key: "schematic-2" });
    expect(result.current).toBe("pumps");
  });
});

const client = createTestClient();

const baseObjects = [
  channel.TYPE_ONTOLOGY_ID,
  framer.TYPE_ONTOLOGY_ID,
  user.TYPE_ONTOLOGY_ID,
  access.role.TYPE_ONTOLOGY_ID,
  access.policy.TYPE_ONTOLOGY_ID,
];

const loadSchematic = async (
  Wrapper: FC<PropsWithChildren>,
  key: string,
): Promise<void> => {
  const Bootstrap = (): ReactElement => {
    PlutoSchematic.useEnsureRetrieved({ key });
    return <span>schematic loaded</span>;
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
  await within(utils.container).findByText("schematic loaded");
};

interface SetupArgs {
  editable: boolean;
  snapshot?: boolean;
  userClient?: Synnax;
  scoped?: boolean;
}

const setup = async ({
  editable,
  snapshot = false,
  userClient = client,
  scoped = true,
}: SetupArgs) => {
  const created = await client.schematics.create(projectKey, {
    key: uuid.create(),
    name: id.create(),
    snapshot,
  });
  const { wrapper: Wrapper } = await createConsoleWrapper({
    client: userClient,
    preloadedState: {
      [Schematic.SLICE_NAME]: {
        schematics: { [created.key]: Schematic.stateZ.parse({ editable }) },
      },
    },
  });
  await loadSchematic(Wrapper, created.key);
  const ScopedWrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Wrapper>
      <PlutoSchematic.Scope.Provider value={created.key}>
        {children}
      </PlutoSchematic.Scope.Provider>
    </Wrapper>
  );
  const { result } = renderHook(
    () => Schematic.useSelectEditable(scoped ? undefined : { key: created.key }),
    { wrapper: scoped ? ScopedWrapper : Wrapper },
  );
  return result;
};

let projectKey: string;

describe("useSelectEditable", () => {
  beforeAll(async () => {
    projectKey = (await client.projects.create({ name: id.create(), layout: {} })).key;
  });

  it("permits editing when the user can update and edit mode is on", async () => {
    const result = await setup({ editable: true });
    await waitFor(() => {
      expect(result.current.canEdit).toBe(true);
      expect(result.current.isCurrentlyEditable).toBe(true);
    });
  });

  it("keeps canEdit but clears isCurrentlyEditable when edit mode is off", async () => {
    const result = await setup({ editable: false });
    await waitFor(() => expect(result.current.canEdit).toBe(true));
    expect(result.current.isCurrentlyEditable).toBe(false);
  });

  it("blocks editing of a snapshot even with permission and edit mode on", async () => {
    const result = await setup({ editable: true, snapshot: true });
    await waitFor(() => {
      expect(result.current.canEdit).toBe(false);
      expect(result.current.isCurrentlyEditable).toBe(false);
    });
  });

  it("blocks editing when the user lacks update permission", async () => {
    const userClient = await createTestClientWithPolicy(client, {
      name: id.create(),
      objects: [schematic.TYPE_ONTOLOGY_ID, ...baseObjects],
      actions: ["retrieve"],
    });
    const result = await setup({ editable: true, userClient });
    await waitFor(() => {
      expect(result.current.canEdit).toBe(false);
      expect(result.current.isCurrentlyEditable).toBe(false);
    });
  });

  it("honors an explicit key override for the snapshot check without a scope", async () => {
    const result = await setup({ editable: true, snapshot: true, scoped: false });
    await waitFor(() => {
      expect(result.current.canEdit).toBe(false);
      expect(result.current.isCurrentlyEditable).toBe(false);
    });
  });
});

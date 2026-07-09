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
  framer,
  schematic,
  type Synnax,
  user,
} from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { Access, Schematic as PSchematic } from "@synnaxlabs/pluto";
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
      <PSchematic.Scope.Provider value={key}>{children}</PSchematic.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const seeded = (): ReturnType<typeof storeWith> =>
  storeWith({ schematics: { [KEY]: customState } });

describe("schematic getters", () => {
  it("should read a schematic's state on demand across dispatches", () => {
    const store = storeWith(Schematic.ZERO_SLICE_STATE);
    const { result } = renderHook(() => Schematic.useGet(), {
      wrapper: wrapperFor(store, KEY),
    });
    const get = result.current;
    expect(get()).toEqual(Schematic.ZERO_STATE);
    act(() => {
      store.dispatch(Schematic.create({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
    });
    expect(get().selected).toEqual(["a"]);
    expect(get().toolbar.selectedTab).toBe("properties");
  });

  it("should resolve the key from scope and allow an explicit override", () => {
    const { result } = renderHook(() => Schematic.useGet(), {
      wrapper: wrapperFor(seeded(), "absent"),
    });
    expect(result.current()).toEqual(Schematic.ZERO_STATE);
    expect(result.current({ key: KEY })).toEqual(customState);
  });

  it("should read the selected elements", () => {
    const { result } = renderHook(() => Schematic.useGetSelected(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toEqual(["a", "b"]);
  });

  it("should read the control status", () => {
    const { result } = renderHook(() => Schematic.useGetControlStatus(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe("acquired");
  });

  it("should report whether control is acquired", () => {
    const { result } = renderHook(() => Schematic.useGetControlIsAcquired(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe(true);
    expect(result.current({ key: "absent" })).toBe(false);
  });

  it("should read the control authority", () => {
    const { result } = renderHook(() => Schematic.useGetAuthority(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe(250);
  });

  it("should read the toolbar state", () => {
    const { result } = renderHook(() => Schematic.useGetToolbar(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toEqual(customState.toolbar);
  });

  it("should read the active toolbar tab", () => {
    const { result } = renderHook(() => Schematic.useGetActiveToolbarTab(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe("properties");
  });

  it("should read the selected symbol group", () => {
    const { result } = renderHook(() => Schematic.useGetSelectedSymbolGroup(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe("valves");
  });

  it("should read the legend state", () => {
    const { result } = renderHook(() => Schematic.useGetLegend(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toEqual(customState.legend);
  });

  it("should read the legend visibility", () => {
    const { result } = renderHook(() => Schematic.useGetLegendVisible(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe(false);
  });

  it("should read the editable flag", () => {
    const { result } = renderHook(() => Schematic.useGetEditable(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe(true);
  });

  it("should read the fit view on resize flag", () => {
    const { result } = renderHook(() => Schematic.useGetFitViewOnResize(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toBe(true);
  });

  it("should read the viewport", () => {
    const { result } = renderHook(() => Schematic.useGetViewport(), {
      wrapper: wrapperFor(seeded(), KEY),
    });
    expect(result.current()).toEqual(customState.viewport);
  });
});

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
      s.dispatch(Schematic.create({ key: "schematic-2" }));
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
    PSchematic.useEnsureRetrieved({ key });
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
      <PSchematic.Scope.Provider value={created.key}>
        {children}
      </PSchematic.Scope.Provider>
    </Wrapper>
  );
  const { result } = renderHook(
    () => ({
      editable: Schematic.useSelectEditable(scoped ? undefined : { key: created.key }),
      granted: Access.useUpdateGranted(schematic.ontologyID(created.key)),
    }),
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
      expect(result.current.editable.canEdit).toBe(true);
      expect(result.current.editable.isCurrentlyEditable).toBe(true);
    });
  });

  it("keeps canEdit but clears isCurrentlyEditable when edit mode is off", async () => {
    const result = await setup({ editable: false });
    await waitFor(() => expect(result.current.editable.canEdit).toBe(true));
    expect(result.current.editable.isCurrentlyEditable).toBe(false);
  });

  it("blocks editing of a snapshot even with permission and edit mode on", async () => {
    const result = await setup({ editable: true, snapshot: true });
    await waitFor(() => expect(result.current.granted).toBe(true));
    expect(result.current.editable.canEdit).toBe(false);
    expect(result.current.editable.isCurrentlyEditable).toBe(false);
  });

  it("blocks editing when the user lacks update permission", async () => {
    const userClient = await createTestClientWithPolicy(client, {
      name: id.create(),
      objects: [schematic.TYPE_ONTOLOGY_ID, ...baseObjects],
      actions: ["retrieve"],
    });
    const result = await setup({ editable: true, userClient });
    await waitFor(() => {
      expect(result.current.editable.canEdit).toBe(false);
      expect(result.current.editable.isCurrentlyEditable).toBe(false);
    });
  });

  it("honors an explicit key override for the snapshot check without a scope", async () => {
    const result = await setup({ editable: true, snapshot: true, scoped: false });
    await waitFor(() => expect(result.current.granted).toBe(true));
    expect(result.current.editable.canEdit).toBe(false);
    expect(result.current.editable.isCurrentlyEditable).toBe(false);
  });
});

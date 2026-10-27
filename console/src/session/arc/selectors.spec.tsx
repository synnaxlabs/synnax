// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, arc, channel, framer, type Synnax, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { Arc as PArc } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Arc } from "@/session/arc";
import { createSliceStore, inWindow } from "@/session/window/testutil";
import { createConsoleWrapper } from "@/testutil";

const KEY = "arc-1";

const customState: Arc.State = {
  graph: {
    editable: false,
    fitViewOnResize: true,
    viewport: { position: { x: 7, y: 8 }, zoom: 2, mode: "pan" },
    selected: ["a", "b"],
  },
  toolbar: { selectedTab: "properties" },
};

const storeWith = (slice: Arc.SliceState) =>
  createSliceStore({
    name: Arc.SLICE_NAME,
    reducer: Arc.reducer,
    preloadedState: slice,
    middleware: Arc.MIDDLEWARE,
  });

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PArc.Scope.Provider value={key}>{children}</PArc.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const createCustomStore = () =>
  storeWith({ version: 0, windows: inWindow({ [KEY]: customState }) });

describe("arc selector hooks", () => {
  it("should resolve the key from the surrounding scope", () => {
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(customState.toolbar);
  });

  it("should let an explicit key override the scope", () => {
    const { result } = renderHook(() => Arc.useSelectToolbar({ key: "absent" }), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(Arc.ZERO_STATE.toolbar);
  });

  it("should return the selected elements", () => {
    const { result } = renderHook(() => Arc.useSelectSelected(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(["a", "b"]);
  });

  it("should return the viewport", () => {
    const { result } = renderHook(() => Arc.useSelectViewport(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(customState.graph.viewport);
  });

  it("should return the viewport mode", () => {
    const { result } = renderHook(() => Arc.useSelectViewportMode(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe("pan");
  });

  it("should return the toolbar state", () => {
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toEqual(customState.toolbar);
  });

  it("should return the fit-view-on-resize flag", () => {
    const { result } = renderHook(() => Arc.useSelectFitViewOnResize(), {
      wrapper: wrapperFor(createCustomStore(), KEY),
    });
    expect(result.current).toBe(true);
  });
});

describe("arc selector stability under dispatch", () => {
  it("should keep a stable reference when an unrelated field changes", () => {
    const s = createCustomStore();
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Arc.setViewportMode({ key: KEY, mode: "select" }));
    });
    expect(result.current).toBe(first);
  });

  it("should return a new reference when the tracked field changes", () => {
    const s = createCustomStore();
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Arc.selectToolbarTab({ key: KEY, tab: "stages" }));
    });
    expect(result.current).not.toBe(first);
    expect(result.current.selectedTab).toBe("stages");
  });

  it("should ignore changes to other arcs", () => {
    const s = createCustomStore();
    const { result } = renderHook(() => Arc.useSelectToolbar(), {
      wrapper: wrapperFor(s, KEY),
    });
    const first = result.current;
    act(() => {
      s.dispatch(Arc.selectToolbarTab({ key: "arc-2", tab: "stages" }));
    });
    expect(result.current).toBe(first);
  });

  it("should re-point the selector when its key dependency changes", () => {
    const s = storeWith({
      version: 0,
      windows: inWindow({
        [KEY]: customState,
        "arc-2": { ...Arc.ZERO_STATE, toolbar: { selectedTab: "stages" } },
      }),
    });
    const { result, rerender } = renderHook(
      ({ key }: { key: string }) => Arc.useSelectToolbar({ key }),
      { wrapper: wrapperFor(s, KEY), initialProps: { key: KEY } },
    );
    expect(result.current.selectedTab).toBe("properties");
    rerender({ key: "arc-2" });
    expect(result.current.selectedTab).toBe("stages");
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

interface SetupParams {
  editable: boolean;
  userClient?: Synnax;
}

const setup = async ({ editable, userClient = client }: SetupParams) => {
  const { wrapper: Wrapper } = await createConsoleWrapper({
    client: userClient,
    preloadedState: {
      [Arc.SLICE_NAME]: {
        version: 0,
        windows: inWindow({
          [KEY]: { ...Arc.ZERO_STATE, graph: { ...Arc.ZERO_STATE.graph, editable } },
        }),
      },
    },
  });
  const ScopedWrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Wrapper>
      <PArc.Scope.Provider value={KEY}>{children}</PArc.Scope.Provider>
    </Wrapper>
  );
  const { result } = renderHook(() => Arc.useSelectEditable(), {
    wrapper: ScopedWrapper,
  });
  return result;
};

describe("useSelectEditable", () => {
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

  it("blocks editing when the user lacks update permission", async () => {
    const userClient = await createTestClientWithPolicy(client, {
      name: id.create(),
      objects: [arc.TYPE_ONTOLOGY_ID, ...baseObjects],
      actions: ["retrieve"],
    });
    const result = await setup({ editable: true, userClient });
    await waitFor(() => {
      expect(result.current.canEdit).toBe(false);
      expect(result.current.isCurrentlyEditable).toBe(false);
    });
  });
});

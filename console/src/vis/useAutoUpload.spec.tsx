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
  createSlice,
  type PayloadAction,
} from "@reduxjs/toolkit";
import { type Flux } from "@synnaxlabs/pluto";
import { renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider, useSelector } from "react-redux";
import { describe, expect, it } from "vitest";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { createUseAutoUpload } from "@/vis/useAutoUpload";

type FakeInput = {
  key?: string;
  name?: string;
  project?: string;
  payload?: number;
};
type FakeOutput = { key: string };
type FakePending = Partial<FakeInput>;

const FAKE_SLICE = "fakeVis";

interface FakeSliceState {
  entries: Record<string, { pendingUpload?: FakePending }>;
}

const ZERO_FAKE_SLICE_STATE: FakeSliceState = { entries: {} };

const fakeSlice = createSlice({
  name: FAKE_SLICE,
  initialState: ZERO_FAKE_SLICE_STATE,
  reducers: {
    clearPendingUpload: (state, { payload }: PayloadAction<{ key: string }>) => {
      const entry = state.entries[payload.key];
      if (entry != null) entry.pendingUpload = undefined;
    },
  },
});

interface TestState {
  [Layout.SLICE_NAME]: Layout.SliceState;
  [Project.SLICE_NAME]: Project.SliceState;
  [FAKE_SLICE]: FakeSliceState;
}

const useSelectPendingUpload = (key: string): FakePending | undefined =>
  useSelector((state: TestState) => state[FAKE_SLICE].entries[key]?.pendingUpload);

/** Records every create call and lets a test decide whether the upload "lands". */
interface CreateController {
  succeed: boolean;
  calls: FakeInput[];
}

interface FakeCreateArgs {
  afterSuccess?: (params: { data: FakeOutput }) => void;
}

const makeUseCreate = (ctrl: CreateController): Flux.UseUpdate<FakeInput, FakeOutput> =>
  ((args?: FakeCreateArgs) => ({
    update: (data: FakeInput) => {
      ctrl.calls.push(data);
      if (ctrl.succeed && data.key != null)
        args?.afterSuccess?.({ data: { key: data.key } });
    },
  })) as unknown as Flux.UseUpdate<FakeInput, FakeOutput>;

const buildHook = (ctrl: CreateController): ((key: string) => boolean) =>
  createUseAutoUpload<FakePending, FakeInput, FakeOutput>({
    useSelectPendingUpload,
    toCreateParams: (pending, fields) => ({ ...pending, ...fields }),
    useCreate: makeUseCreate(ctrl),
    clearPendingUpload: fakeSlice.actions.clearPendingUpload,
  });

interface HarnessArgs {
  key: string;
  name: string;
  projectKey?: string;
  pendingUpload?: FakePending;
}

interface Harness {
  wrapper: FC<PropsWithChildren>;
  store: ReturnType<typeof configureStore<TestState>>;
}

const buildHarness = ({
  key,
  name,
  projectKey,
  pendingUpload,
}: HarnessArgs): Harness => {
  const reducer = combineReducers({
    [Layout.SLICE_NAME]: Layout.reducer,
    [Project.SLICE_NAME]: Project.reducer,
    [FAKE_SLICE]: fakeSlice.reducer,
  });
  const preloadedState: TestState = {
    [Layout.SLICE_NAME]: {
      ...Layout.ZERO_SLICE_STATE,
      layouts: {
        ...Layout.ZERO_SLICE_STATE.layouts,
        [key]: { windowKey: "main", key, type: FAKE_SLICE, name, location: "mosaic" },
      },
    },
    [Project.SLICE_NAME]: {
      ...Project.ZERO_SLICE_STATE,
      active: projectKey != null ? { key: projectKey, name: "ws" } : null,
    },
    [FAKE_SLICE]: {
      entries: pendingUpload != null ? { [key]: { pendingUpload } } : {},
    },
  };
  const store = configureStore({ reducer, preloadedState });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  return { wrapper, store };
};

const selectPending = (store: Harness["store"], key: string): FakePending | undefined =>
  store.getState()[FAKE_SLICE].entries[key]?.pendingUpload;

describe("createUseAutoUpload", () => {
  it("returns true and never calls create when there is no pending upload", () => {
    const ctrl: CreateController = { succeed: true, calls: [] };
    const useAutoUpload = buildHook(ctrl);
    const { wrapper } = buildHarness({ key: "k", name: "Settled", projectKey: "ws" });
    const { result } = renderHook(() => useAutoUpload("k"), { wrapper });
    expect(result.current).toBe(true);
    expect(ctrl.calls).toHaveLength(0);
  });

  it("uploads the pending body with the layout name and active project, then clears it", async () => {
    const ctrl: CreateController = { succeed: true, calls: [] };
    const useAutoUpload = buildHook(ctrl);
    const { wrapper, store } = buildHarness({
      key: "k",
      name: "Live Layout Name",
      projectKey: "ws",
      pendingUpload: { key: "k", payload: 7 },
    });
    const { result } = renderHook(() => useAutoUpload("k"), { wrapper });
    await waitFor(() => expect(result.current).toBe(true));
    expect(ctrl.calls).toHaveLength(1);
    expect(ctrl.calls[0]).toMatchObject({
      key: "k",
      name: "Live Layout Name",
      project: "ws",
      payload: 7,
    });
    expect(selectPending(store, "k")).toBeUndefined();
  });

  it("uploads without a project when none is active", async () => {
    const ctrl: CreateController = { succeed: true, calls: [] };
    const useAutoUpload = buildHook(ctrl);
    const { wrapper, store } = buildHarness({
      key: "k",
      name: "Projectless",
      pendingUpload: { key: "k" },
    });
    const { result } = renderHook(() => useAutoUpload("k"), { wrapper });
    await waitFor(() => expect(result.current).toBe(true));
    expect(ctrl.calls[0].project).toBeUndefined();
    expect(ctrl.calls[0].name).toEqual("Projectless");
    expect(selectPending(store, "k")).toBeUndefined();
  });

  it("creates exactly once and does not re-upload on subsequent renders", async () => {
    const ctrl: CreateController = { succeed: true, calls: [] };
    const useAutoUpload = buildHook(ctrl);
    const { wrapper } = buildHarness({
      key: "k",
      name: "Single Upload",
      projectKey: "ws",
      pendingUpload: { key: "k" },
    });
    const { result, rerender } = renderHook(() => useAutoUpload("k"), { wrapper });
    await waitFor(() => expect(result.current).toBe(true));
    rerender();
    rerender();
    expect(ctrl.calls).toHaveLength(1);
  });

  it("does not clear on failure and retries on a fresh mount", () => {
    const ctrl: CreateController = { succeed: false, calls: [] };
    const useAutoUpload = buildHook(ctrl);
    const { wrapper, store } = buildHarness({
      key: "k",
      name: "Retry",
      projectKey: "ws",
      pendingUpload: { key: "k" },
    });
    const first = renderHook(() => useAutoUpload("k"), { wrapper });
    expect(first.result.current).toBe(false);
    expect(ctrl.calls).toHaveLength(1);
    expect(selectPending(store, "k")).toBeDefined();
    first.unmount();
    const second = renderHook(() => useAutoUpload("k"), { wrapper });
    expect(second.result.current).toBe(false);
    expect(ctrl.calls).toHaveLength(2);
    second.unmount();
  });
});

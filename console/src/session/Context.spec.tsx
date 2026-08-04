// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { useSelector } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createSessionConsoleWrapper, renderHookWithConsole } from "@/testutil";

const useProbe = () => ({
  dispatch: Session.useDispatch(),
  selected: useSelector(Session.Cluster.selectSelectedKey),
  modals: Session.Modals.useStore("context-spec"),
});

describe("Session.Context", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("provides a live redux store and modals context to children", async () => {
    const { result } = renderHook(useProbe, { wrapper: Session.Context });

    await waitFor(() => {
      if (result.current == null) throw new Error("store has not resolved yet");
    });
    expect(result.current.modals).toBeDefined();
    expect(result.current.selected).toBeUndefined();

    act(() => {
      result.current.dispatch(Session.Cluster.select("DEMO"));
    });
    expect(result.current.selected).toBe("DEMO");
  });
});

const CLUSTER_KEY = "local";

const createClusterState = (): Session.Cluster.SliceState => ({
  ...Session.Cluster.ZERO_SLICE_STATE,
  clusters: {
    [CLUSTER_KEY]: {
      key: CLUSTER_KEY,
      name: "Local",
      host: "localhost",
      port: 9090,
      username: "synnax",
      password: "seldon",
      secure: false,
    },
  },
  selected: CLUSTER_KEY,
});

/**
 * Renders useSettled against a live cluster under the real synchronizers and
 * waits for the workspace to settle. The settled result is the positive
 * control that unsettling assertions measure against.
 */
const renderSettled = async () => {
  const { wrapper: Console, store } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: { [Session.Cluster.SLICE_NAME]: createClusterState() },
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Session.SettledProvider>{children}</Session.SettledProvider>
    </Console>
  );
  Wrapper.displayName = "SettledWrapper";
  const { result } = renderHook(() => Session.useSettled(), { wrapper: Wrapper });
  await waitFor(() => {
    if (!result.current) throw new Error("workspace never settled");
  });
  return { result, store };
};

describe("Session.useSettled", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("should report settled when there is no client to verify against", async () => {
    const { result } = await renderHookWithConsole(() => Session.useSettled());
    expect(result.current).toBe(true);
  });

  it("should unsettle while a partition swap is in flight", async () => {
    const { result, store } = await renderSettled();
    act(() => {
      store.dispatch(Session.Persist.beginSwap());
    });
    expect(result.current).toBe(false);
    act(() => {
      store.dispatch(Session.Persist.endSwap());
    });
    expect(result.current).toBe(true);
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax as Client } from "@synnaxlabs/client";
import { TEST_CORE_ADDRESS } from "@synnaxlabs/client/testutil";
import { Synnax } from "@synnaxlabs/pluto";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { useSelector } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createSessionConsoleWrapper, renderHookWithConsole } from "@/testutil";

const useProbe = () => ({
  dispatch: Session.useDispatch(),
  selected: useSelector(Session.Core.selectSelectedKey),
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
      result.current.dispatch(Session.Core.select("DEMO"));
    });
    expect(result.current.selected).toBe("DEMO");
  });
});

const CORE_KEY = "local";

const createCoreState = (): Session.Core.SliceState => ({
  ...Session.Core.ZERO_SLICE_STATE,
  cores: {
    [CORE_KEY]: {
      key: CORE_KEY,
      name: "Local",
      ...TEST_CORE_ADDRESS,
      username: "synnax",
      password: "seldon",
      secure: false,
    },
  },
  selected: CORE_KEY,
});

/**
 * Renders useSettled against a live Core under the real synchronizers and
 * waits for the workspace to settle. The settled result is the positive control that
 * unsettling assertions measure against. clients holds every client the hook saw.
 */
const renderSettled = async () => {
  const { wrapper: Console, store } = await createSessionConsoleWrapper({
    client: null,
    preloadedState: { [Session.Core.SLICE_NAME]: createCoreState() },
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Session.SettledProvider>{children}</Session.SettledProvider>
    </Console>
  );
  Wrapper.displayName = "SettledWrapper";
  const clients = new Set<Client>();
  const { result } = renderHook(
    () => {
      const client = Synnax.use();
      if (client != null) clients.add(client);
      return Session.useSettled();
    },
    { wrapper: Wrapper },
  );
  await waitFor(() => {
    if (!result.current) throw new Error("workspace never settled");
  });
  return { result, store, clients };
};

describe("Session.useSettled", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("should report settled when there is no client to verify against", async () => {
    const { result } = await renderHookWithConsole(() => Session.useSettled());
    expect(result.current).toBe(true);
  });

  it("should keep its client when it caches the cluster key", async () => {
    const { store, clients } = await renderSettled();
    expect(Session.Core.selectClusterKey(store.getState())).toBeDefined();
    expect(clients.size).toBe(1);
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

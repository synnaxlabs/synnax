// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connection, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Synnax } from "@synnaxlabs/pluto";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { SYNCHRONIZERS } from "@/session/core/synchronizer";
import { createCore, createCoreState } from "@/session/core/testutil";
import { Synchronizer } from "@/session/synchronizer";
import { pickSynchronizer } from "@/session/synchronizer/testutil";
import {
  createConnectedConsoleWrapper,
  createConsoleWrapper,
  renderHookWithConsole,
} from "@/testutil";

describe("useCloseOnCoreChange", () => {
  const AUX_LABEL = "aux";
  const AUX_KEY = "aux-key";

  const auxWindow: Drift.WindowState = {
    key: AUX_KEY,
    stage: "created",
    processCount: 0,
    reserved: true,
    focusCount: 0,
    centerCount: 0,
    ordinal: 2,
  };

  const auxDriftState = {
    ...Session.ZERO_STATE.drift,
    windows: { ...Session.ZERO_STATE.drift.windows, [AUX_LABEL]: auxWindow },
    labelKeys: { ...Session.ZERO_STATE.drift.labelKeys, [AUX_LABEL]: AUX_KEY },
    keyLabels: { ...Session.ZERO_STATE.drift.keyLabels, [AUX_KEY]: AUX_LABEL },
    nextOrdinal: 3,
  };

  const useSyncWithModals = () => {
    const modals = Session.Modals.useStore("useCloseOnCoreChange spec");
    Synchronizer.use(SYNCHRONIZERS);
    return { modals, status: Synnax.useConnectionStatus() };
  };

  // A single core reports a single Core key, so the swap is staged with a
  // client carrying the connection the synchronizer reads and nothing else.
  const createClient = (clusterKey: string): Client =>
    ({
      connection: {
        status: {
          ...connection.DEFAULT_STATUS,
          variant: "success",
          message: "Connected",
          details: {
            ...connection.DEFAULT_STATUS.details,
            authenticated: true,
            clusterKey,
            epoch: 1,
          },
        },
        onChange: () => () => {},
      },
    }) as unknown as Client;

  const useCloseSync = () => {
    const modals = Session.Modals.useStore("useCloseOnCoreChange spec");
    const verified = Synchronizer.use(
      pickSynchronizer(SYNCHRONIZERS, "close windows on Core change"),
    );
    return { modals, verified };
  };

  it("should leave open windows and modals alone on first connect", async () => {
    const active = createCore("Local");
    const { wrapper, store } = await createConnectedConsoleWrapper({
      client: null,
      connParams: active,
      preloadedState: {
        ...createCoreState([active], active.key),
        drift: auxDriftState,
      },
    });
    const { result } = renderHook(useSyncWithModals, { wrapper });
    act(() =>
      result.current.modals.push(
        () => null,
        {},
        () => {},
      ),
    );

    await waitFor(() => expect(result.current.status.variant).toBe("success"));
    await waitFor(() =>
      expect(result.current.status.details.epoch).toBeGreaterThanOrEqual(1),
    );

    expect(Drift.selectWindow(store.getState(), AUX_KEY)).not.toBeNull();
    expect(result.current.modals.isAnyOpen()).toBe(true);
  });

  it("should close torn-off windows and clear modals when the Core changes", async () => {
    const held = { client: createClient("Core-a") };
    const { wrapper: Console, store } = await createConsoleWrapper({
      client: null,
      preloadedState: {
        ...createCoreState([createCore("Core-a")], "Core-a"),
        drift: auxDriftState,
      },
    });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Console>
        <Synnax.TestProvider client={held.client}>{children}</Synnax.TestProvider>
      </Console>
    );
    Wrapper.displayName = "CoreSwapWrapper";
    const { result, rerender } = renderHook(useCloseSync, { wrapper: Wrapper });
    act(() =>
      result.current.modals.push(
        () => null,
        {},
        () => {},
      ),
    );
    await waitFor(() => expect(result.current.verified).toBe(true));
    expect(Drift.selectWindow(store.getState(), AUX_KEY)).not.toBeNull();
    expect(result.current.modals.isAnyOpen()).toBe(true);

    held.client = createClient("Core-b");
    rerender();

    await waitFor(() =>
      expect(Drift.selectWindow(store.getState(), AUX_KEY)).toBeNull(),
    );
    expect(result.current.modals.isAnyOpen()).toBe(false);
  });
});

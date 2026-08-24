// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { act, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Persist } from "@/feature/persist";
import { Session } from "@/session";
import { renderHookWithConsole } from "@/testutil";

const render = async () =>
  await renderHookWithConsole(() => {
    Persist.useStoreStatus();
    return Status.useNotifications();
  });

const warnings = <S extends { message: string }>(statuses: S[]): S[] =>
  statuses.filter(({ message }) => message === Persist.UNAVAILABLE_MESSAGE);

describe("Persist.useStoreStatus", () => {
  it("should stay quiet while the store is usable", async () => {
    const { result } = await render();
    expect(warnings(result.current.statuses)).toHaveLength(0);
  });

  it("should warn the user once the store turns out to be unusable", async () => {
    const { result, store } = await render();
    act(() => {
      store.dispatch(Session.Persist.storeUnavailable());
    });
    await waitFor(() => expect(warnings(result.current.statuses)).toHaveLength(1));
    expect(warnings(result.current.statuses)[0].variant).toBe("warning");
  });

  it("should warn only once however many times it is announced", async () => {
    const { result, store } = await render();
    act(() => {
      store.dispatch(Session.Persist.storeUnavailable());
      store.dispatch(Session.Persist.storeUnavailable());
    });
    await waitFor(() => expect(warnings(result.current.statuses)).toHaveLength(1));
  });
});

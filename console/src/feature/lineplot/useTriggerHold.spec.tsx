// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { LinePlot } from "@/feature/lineplot";
import { createPreloadedState } from "@/feature/lineplot/testutil";
import { Session } from "@/session";
import { createConsoleWrapper } from "@/testutil";

const renderTriggerHold = async (key: string) => {
  const { wrapper: Console, store } = await createConsoleWrapper({
    client: null,
    preloadedState: createPreloadedState(key, "Test Plot"),
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>{children}</Triggers.Provider>
    </Console>
  );
  renderHook(() => LinePlot.useTriggerHold(), { wrapper: Wrapper });
  return store;
};

describe("lineplot useTriggerHold", () => {
  it("toggles the active plot's hold when H is pressed", async () => {
    const key = uuid.create();
    const store = await renderTriggerHold(key);
    expect(
      Session.LinePlot.selectControlState({ state: store.getState(), key }).hold,
    ).toBe(false);
    fireEvent.keyDown(window, { code: "KeyH", key: "h" });
    await waitFor(() =>
      expect(
        Session.LinePlot.selectControlState({ state: store.getState(), key }).hold,
      ).toBe(true),
    );
    fireEvent.keyUp(window, { code: "KeyH", key: "h" });
  });
});

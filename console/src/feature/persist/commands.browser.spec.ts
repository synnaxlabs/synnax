// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Persist } from "@/feature/persist";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";

const CONFIRM_MESSAGE = "Are you sure you want to clear the Console's local storage?";

const selectClearCommand = async () => {
  const { store, openCommandPalette, selectCommand } = await renderPalette({
    commands: Persist.COMMANDS,
  });
  // clearState is handled by the persistence middleware, which the test store omits,
  // so the dispatch itself is the only observable outcome. Call-through spy only.
  const dispatch = vi.spyOn(store, "dispatch");
  await openCommandPalette();
  await selectCommand("Clear local storage");
  await screen.findByText(CONFIRM_MESSAGE);
  return dispatch;
};

describe("Persist Commands", () => {
  it("should clear local storage after the user confirms", async () => {
    const dispatch = await selectClearCommand();
    fireEvent.click(findButton("Confirm"));
    await waitFor(() =>
      expect(dispatch).toHaveBeenCalledWith(Session.Persist.clearState()),
    );
  });

  it("should not clear local storage when the user cancels", async () => {
    const dispatch = await selectClearCommand();
    fireEvent.click(findButton("Cancel"));
    await waitFor(() => expect(screen.queryByText(CONFIRM_MESSAGE)).toBeNull());
    await act(async () => {});
    expect(dispatch).not.toHaveBeenCalledWith(Session.Persist.clearState());
  });
});

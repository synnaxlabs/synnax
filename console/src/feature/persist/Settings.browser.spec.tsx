// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Persist } from "@/feature/persist";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("Persist.Settings", () => {
  it("should clear local storage after the user confirms", async () => {
    const { store } = await renderWithConsole(
      <Modals.Stack>
        <Persist.Settings />
      </Modals.Stack>,
    );
    // clearState is handled by the persistence middleware, which the test store omits,
    // so the dispatch itself is the only observable outcome. Call-through spy only.
    const dispatch = vi.spyOn(store, "dispatch");
    fireEvent.click(screen.getByRole("button", { name: "Clear" }));
    fireEvent.click(await waitFor(() => findButton("Confirm")));
    await waitFor(() =>
      expect(dispatch).toHaveBeenCalledWith(Session.Persist.clearState()),
    );
  });
});

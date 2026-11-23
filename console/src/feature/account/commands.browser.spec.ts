// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Account } from "@/feature/account";
import { renderPalette } from "@/feature/command/testutil";
import { getSwitch } from "@/platform/modals/testutil";
import { Session } from "@/session";

describe("Account commands", () => {
  it("should turn usage data off and back on", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Account.COMMANDS,
      preloadedState: {
        [Session.Account.SLICE_NAME]: {
          ...Session.Account.ZERO_SLICE_STATE,
          analyticsAllowed: true,
        },
      },
    });
    const allowed = (): boolean =>
      Session.Account.selectSliceState(store.getState()).analyticsAllowed;
    await openCommandPalette("usage");
    await selectCommand("Change usage data settings");
    await screen.findByText("Share usage data");
    expect(getSwitch("Share usage data").checked).toBe(true);
    fireEvent.click(getSwitch("Share usage data"));
    await waitFor(() => expect(allowed()).toBe(false));
    fireEvent.click(getSwitch("Share usage data"));
    await waitFor(() => expect(allowed()).toBe(true));
  });
});

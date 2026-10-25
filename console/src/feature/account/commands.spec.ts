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
import { type TestStore } from "@/testutil";

const openUsageData = async (analyticsAllowed: boolean): Promise<TestStore> => {
  const palette = await renderPalette({
    commands: Account.COMMANDS,
    preloadedState: {
      [Session.Account.SLICE_NAME]: {
        ...Session.Account.ZERO_SLICE_STATE,
        analyticsAllowed,
      },
    },
  });
  await palette.openCommandPalette("usage");
  await palette.selectCommand("Change usage data settings");
  await screen.findByText("Share usage data");
  return palette.store;
};

const allowed = (store: TestStore): boolean =>
  Session.Account.selectSliceState(store.getState()).analyticsAllowed;

describe("Account commands", () => {
  it("should turn usage data off and back on", async () => {
    const store = await openUsageData(true);
    expect(getSwitch("Share usage data").checked).toBe(true);
    fireEvent.click(getSwitch("Share usage data"));
    await waitFor(() => expect(allowed(store)).toBe(false));
    fireEvent.click(getSwitch("Share usage data"));
    await waitFor(() => expect(allowed(store)).toBe(true));
  });

  it("should link to the page that lists the usage data", async () => {
    await openUsageData(false);
    const link = screen.getByRole("link", { name: /Learn what Synnax Desktop sends/ });
    expect(link.getAttribute("href")).toBe(
      "https://docs.synnaxlabs.com/reference/desktop/usage-data",
    );
  });
});

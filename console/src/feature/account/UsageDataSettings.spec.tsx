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
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("Account.UsageDataSettings", () => {
  it("should share usage data when the user turns the switch on", async () => {
    const { store } = await renderWithConsole(<Account.UsageDataSettings />);
    const allowed = () => store.getState()[Session.Account.SLICE_NAME].analyticsAllowed;
    expect(allowed()).toBe(false);
    fireEvent.click(screen.getByRole("checkbox", { name: "Share usage data" }));
    await waitFor(() => expect(allowed()).toBe(true));
  });
});

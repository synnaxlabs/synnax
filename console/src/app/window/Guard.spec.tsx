// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Guard } from "@/app/window/Guard";
import { renderWithConsole } from "@/testutil";

const renderGuard = async (): Promise<void> => {
  await renderWithConsole(
    <Guard>
      <span>workspace</span>
    </Guard>,
  );
};

describe("app/window/Guard", () => {
  it("should ask for a login when no Core is selected", async () => {
    await renderGuard();
    expect(screen.getAllByText("Log in").length).toBeGreaterThan(0);
    expect(screen.queryByText("Starting Synnax...")).toBeNull();
    expect(screen.queryByText("workspace")).toBeNull();
  });
});

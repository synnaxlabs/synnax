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

describe("app/window/Guard in Synnax Desktop", () => {
  it("should wait for the embedded Core and never ask for a login", async () => {
    await renderGuard();
    expect(await screen.findByText("Starting Synnax...")).toBeTruthy();
    expect(screen.queryByText("Log in")).toBeNull();
    expect(screen.queryByText("workspace")).toBeNull();
  });
});

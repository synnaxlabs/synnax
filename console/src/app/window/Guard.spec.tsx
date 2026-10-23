// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Guard } from "@/app/window/Guard";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  createStatusConsoleWrapper,
  renderWithConsole,
  UNLICENSED_STATUS,
} from "@/testutil";

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

  it("should show the activation screen when the Core is unlicensed", async () => {
    const core = createCore("Local");
    const { wrapper } = await createStatusConsoleWrapper({
      client: null,
      status: UNLICENSED_STATUS,
      preloadedState: createCoreState([core], core.key),
    });
    render(
      <Guard>
        <span>workspace</span>
      </Guard>,
      { wrapper },
    );
    expect(screen.getByRole("textbox", { name: "License key" })).toBeTruthy();
    expect(screen.queryByText("workspace")).toBeNull();
  });
});

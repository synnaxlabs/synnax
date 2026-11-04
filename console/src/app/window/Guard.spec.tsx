// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type connection } from "@synnaxlabs/client";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Guard } from "@/app/window/Guard";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  createStatusConsoleWrapper,
  INCOMPATIBLE_STATUS,
  INCOMPATIBLE_UNLICENSED_STATUS,
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

const MISMATCH_TITLE = "Console 0.61.0 cannot work with Core 0.60.1";

const renderCoreGuard = async (status: connection.Status): Promise<void> => {
  const core = createCore("Local");
  const { wrapper } = await createStatusConsoleWrapper({
    client: null,
    status,
    preloadedState: createCoreState([core], core.key),
  });
  render(
    <Guard>
      <span>workspace</span>
    </Guard>,
    { wrapper },
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
    await renderCoreGuard(UNLICENSED_STATUS);
    expect(screen.getByRole("textbox", { name: "License key" })).toBeTruthy();
    expect(screen.queryByText("workspace")).toBeNull();
  });

  it("should show the mismatch screen when the Core runs another minor", async () => {
    await renderCoreGuard(INCOMPATIBLE_STATUS);
    expect(screen.getByText(MISMATCH_TITLE)).toBeTruthy();
    expect(screen.queryByText("workspace")).toBeNull();
  });

  it("should show the mismatch screen before the activation screen", async () => {
    await renderCoreGuard(INCOMPATIBLE_UNLICENSED_STATUS);
    expect(screen.getByText(MISMATCH_TITLE)).toBeTruthy();
    expect(screen.queryByRole("textbox", { name: "License key" })).toBeNull();
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Core } from "@/feature/core";
import { clickConnectionBadge, hoverConnectionBadge } from "@/feature/core/testutil";
import { CONNECTION_PARAMS } from "@/session/core/testutil";
import { createConnectedConsoleWrapper, renderWithConsole } from "@/testutil";

describe("ConnectionBadge", () => {
  it("should report disconnected when no Core connection exists", async () => {
    const { container } = await renderWithConsole(<Core.ConnectionBadge />);
    hoverConnectionBadge(container);
    expect((await screen.findAllByText("Disconnected")).length).toBeGreaterThan(0);
  });

  it("should report connected once the provider reaches the Core", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    const { container } = render(<Core.ConnectionBadge />, { wrapper });
    hoverConnectionBadge(container);
    expect(await screen.findByText("Connected")).toBeTruthy();
  });

  it("should open the diagnostics dialog on click", async () => {
    const { container } = await renderWithConsole(<Core.ConnectionBadge />);
    clickConnectionBadge(container);
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getAllByText("Disconnected").length).toBeGreaterThan(0);
  });

  it("should show server facts in the dialog once connected", async () => {
    const { wrapper } = await createConnectedConsoleWrapper({
      client: null,
      connParams: CONNECTION_PARAMS,
    });
    const { container } = render(<Core.ConnectionBadge />, { wrapper });
    clickConnectionBadge(container);
    const dialog = await screen.findByRole("dialog");
    await within(dialog).findByText("Connected", {}, { timeout: 10000 });
    expect(within(dialog).getByText(/^Core v/)).toBeTruthy();
  });
});

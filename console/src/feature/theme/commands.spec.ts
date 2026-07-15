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

import { renderPalette } from "@/feature/command/testutil";
import { Theme } from "@/feature/theme";
import { stubGeometry } from "@/testutil";

stubGeometry();

describe("Theme Commands", () => {
  it("exposes a single command to change the color theme", async () => {
    const { openCommandPalette } = await renderPalette({ commands: Theme.COMMANDS });
    await openCommandPalette("theme");
    expect(await screen.findByText("Change color theme")).toBeTruthy();
  });

  it("opens a picker with light, dark, and system options", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Theme.COMMANDS,
    });
    await openCommandPalette("theme");
    await selectCommand("Change color theme");
    expect(await screen.findByText("Light")).toBeTruthy();
    expect(screen.getByText("Dark")).toBeTruthy();
    expect(screen.getByText("System")).toBeTruthy();
  });

  it("applies the chosen mode and dismisses the picker", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Theme.COMMANDS,
    });
    await openCommandPalette("theme");
    await selectCommand("Change color theme");
    fireEvent.click(await screen.findByText("Dark"));
    await waitFor(() => expect(store.getState().theme.mode).toBe("dark"));
    await waitFor(() => expect(screen.queryByText("Color theme")).toBeNull());
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Feed } from "@/app/notifications/Feed";
import { renderPalette } from "@/feature/command/testutil";
import { Theme } from "@/feature/theme";
import { Theme as SessionTheme } from "@/session/theme";
import { renderWithConsole, stubGeometry } from "@/testutil";

stubGeometry();

describe("Theme Commands", () => {
  it("should toggle the color theme when the command is selected", () => {
    const toggleTheme = vi.fn();
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Theming.Provider toggleTheme={toggleTheme}>{children}</Theming.Provider>
    );
    const [Toggle] = Theme.COMMANDS;
    render(<Toggle key={Toggle.key} itemKey={Toggle.key} index={0} />, {
      wrapper: Wrapper,
    });
    fireEvent.click(screen.getByText("Toggle color theme"));
    expect(toggleTheme).toHaveBeenCalledTimes(1);
  });

  it("ranks 'Toggle color theme' above the sync command when searching 'theme'", async () => {
    const { openCommandPalette } = await renderPalette({ commands: Theme.COMMANDS });
    await openCommandPalette("theme");
    const toggle = await screen.findByText("Toggle color theme");
    const sync = screen.getByText("Disable theme sync with system");
    expect(
      toggle.compareDocumentPosition(sync) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });

  describe("ToggleSyncWithSystemCommand", () => {
    const [, SyncCommand] = Theme.COMMANDS;

    beforeEach(() => {
      const root = document.createElement("div");
      root.id = "root";
      document.body.appendChild(root);
    });
    afterEach(() => document.getElementById("root")?.remove());

    it("disables sync, relabels, and notifies when currently enabled", async () => {
      const { store } = await renderWithConsole(
        <>
          <SyncCommand key={SyncCommand.key} itemKey={SyncCommand.key} index={0} />
          <Feed />
        </>,
        { preloadedState: { theme: { ...SessionTheme.ZERO_SLICE_STATE } } },
      );
      fireEvent.click(screen.getByText("Disable theme sync with system"));
      expect(store.getState().theme.syncWithSystem).toBe(false);
      expect(screen.getByText("Disabled theme sync with system")).toBeTruthy();
    });

    it("enables sync, relabels, and notifies when currently disabled", async () => {
      const { store } = await renderWithConsole(
        <>
          <SyncCommand key={SyncCommand.key} itemKey={SyncCommand.key} index={0} />
          <Feed />
        </>,
        {
          preloadedState: {
            theme: { ...SessionTheme.ZERO_SLICE_STATE, syncWithSystem: false },
          },
        },
      );
      fireEvent.click(screen.getByText("Enable theme sync with system"));
      expect(store.getState().theme.syncWithSystem).toBe(true);
      expect(screen.getByText("Enabled theme sync with system")).toBeTruthy();
    });
  });
});

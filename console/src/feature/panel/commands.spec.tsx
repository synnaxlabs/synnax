// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Icon } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri" } => ({ engine: "web" }));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

import { renderPalette } from "@/feature/command/testutil";
import { Panel } from "@/feature/panel";
import { Command } from "@/platform/command";
import { Session } from "@/session";

// Proves the list rendered before the window command is missed.
const SentinelCommand = Command.create({
  key: "sentinel",
  name: "Sentinel command",
  icon: <Icon.Info />,
  useOnSelect: () => () => {},
});

const NAME = "Open new window";

describe("Panel Commands", () => {
  beforeEach(() => {
    mocks.engine = "web";
  });

  it("should open a window on the selected panel", async () => {
    mocks.engine = "tauri";
    const panelKey = uuid.create();
    const { openCommandPalette, selectCommand, store } = await renderPalette({
      commands: Panel.COMMANDS,
    });
    act(() => {
      store.dispatch(
        Session.Panel.select({ key: panelKey, windowKey: Drift.MAIN_WINDOW }),
      );
    });
    await openCommandPalette();
    await selectCommand(NAME);

    await waitFor(() => {
      const state = store.getState();
      const win = Drift.selectWindows(state).find(
        ({ key, reserved }) => key !== Drift.MAIN_WINDOW && reserved,
      );
      expect(win).toBeDefined();
      expect(state.panels.windows[win!.key]?.selected).toEqual(panelKey);
    });
  });

  it("should hide the command in the browser, which cannot open a window", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: [...Panel.COMMANDS, SentinelCommand],
    });
    await openCommandPalette();
    await screen.findByText("Sentinel command");
    expect(screen.queryByText(NAME)).toBeNull();
  });
});

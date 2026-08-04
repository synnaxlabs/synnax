// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(
  (): {
    engine: "web" | "tauri";
    version: string;
    update: unknown;
    relaunch: ReturnType<typeof vi.fn>;
  } => ({
    engine: "web",
    version: "1.5.0",
    update: null,
    relaunch: vi.fn(async () => {}),
  }),
);

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/app", () => ({
  getVersion: vi.fn(async () => mocks.version),
}));

vi.mock("@tauri-apps/plugin-updater", () => ({
  check: vi.fn(async () => mocks.update),
}));

vi.mock("@tauri-apps/plugin-process", () => ({ relaunch: mocks.relaunch }));

import { renderPalette } from "@/feature/command/testutil";
import { Version } from "@/feature/version";
import { Command } from "@/platform/command";

// Always-visible sentinel proving the command list rendered before asserting the
// version commands are absent.
const SentinelCommand = Command.create({
  key: "sentinel",
  name: "Sentinel command",
  icon: <Icon.Info />,
  useOnSelect: () => () => {},
});

describe("Version Commands", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.version = "1.5.0";
    mocks.update = null;
    mocks.relaunch.mockClear();
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("should open the version info dialog", async () => {
    mocks.engine = "tauri";
    mocks.version = "2.4.6";
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Version.COMMANDS,
    });
    await openCommandPalette();
    await selectCommand("Show version info");
    expect(await screen.findByText("Console v2.4.6")).toBeTruthy();
    expect(await screen.findByText("Up to date")).toBeTruthy();
  });

  it("should hide the version info command in the browser", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: [...Version.COMMANDS, SentinelCommand],
    });
    await openCommandPalette();
    await screen.findByText("Sentinel command");
    expect(screen.queryByText("Show version info")).toBeNull();
  });

  it("should not offer a command that updates without confirmation", async () => {
    mocks.engine = "tauri";
    mocks.update = { version: "9.9.9" };
    const { openCommandPalette } = await renderPalette({
      commands: [...Version.COMMANDS, SentinelCommand],
    });
    await openCommandPalette();
    await screen.findByText("Sentinel command");
    expect(screen.queryByText(/Update/)).toBeNull();
    expect(mocks.relaunch).not.toHaveBeenCalled();
  });
});

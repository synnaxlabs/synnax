// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Project } from "@/feature/project";
import { Session } from "@/session";
import {
  installPickedDirectory,
  interceptFilePicker,
  removeFilePickers,
  stubGeometry,
  uniqueName,
} from "@/testutil";

stubGeometry();

const client = createTestClient();

describe("Project Commands", () => {
  afterEach(() => {
    removeFilePickers();
    vi.restoreAllMocks();
  });

  it("should open the project creation modal", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Project.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Create a project");
    expect(await screen.findByRole("dialog")).toBeTruthy();
  });

  it("should open a directory picker when importing a project", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: Project.COMMANDS,
      client,
    });
    const picker = interceptFilePicker();
    await openCommandPalette();
    const item = await screen.findByText("Import a project");
    // The picker interceptor swallows the select frame's synthetic click, so fire the
    // detail-0 click that invokes onSelect directly.
    await act(async () => {
      fireEvent.click(item, { detail: 0 });
    });
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    expect(picker.lastInput().webkitdirectory).toBe(true);
    picker.cancel();
  });

  it("should export the current project to the picked directory", async () => {
    const p = await client.projects.create({ name: uniqueName("proj"), layout: {} });
    const writes = installPickedDirectory({ exists: false });
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Project.COMMANDS,
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: p.key },
      },
    });
    await openCommandPalette();
    await selectCommand("Export current project");
    await waitFor(() => expect(writes.has(Project.LAYOUT_FILE_NAME)).toBe(true));
  });
});

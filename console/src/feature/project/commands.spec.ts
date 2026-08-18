// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, project } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Project } from "@/feature/project";
import { Session } from "@/session";
import { captureBrowserDownloads, removeSaveFilePicker, uniqueName } from "@/testutil";

const client = createTestClient();

const createProjectWithPanel = async (): Promise<{
  proj: project.Project;
  logName: string;
}> => {
  const proj = await client.projects.create({ name: uniqueName("proj"), layout: {} });
  const logName = uniqueName("log");
  const createdLog = await client.logs.create(proj.key, { name: logName });
  await client.panels.create({
    name: "Main",
    root: {
      variant: "leaf",
      tabs: [
        {
          variant: "resource",
          key: uuid.create(),
          resource: log.ontologyID(createdLog.key),
        },
      ],
    },
    parent: project.ontologyID(proj.key),
  });
  return { proj, logName };
};

describe("Project Commands", () => {
  afterEach(() => {
    removeSaveFilePicker();
    vi.restoreAllMocks();
  });

  it("should open the project creation modal", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Project.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Create project");
    expect(await screen.findByRole("dialog")).toBeTruthy();
  });

  it("should open the import modal when importing a project", async () => {
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Project.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Import project");
    expect(await screen.findByText("Drop a .zip or folder here")).toBeTruthy();
  });

  it("should export the current project as a zip download", async () => {
    const { proj, logName } = await createProjectWithPanel();
    const downloads = captureBrowserDownloads();
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Project.COMMANDS,
      client,
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: proj.key },
      },
    });
    await openCommandPalette();
    await selectCommand("Export current project");
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${proj.name}.zip`);
    // Zip entry names are stored uncompressed, so the archive names its own files.
    const archive = new TextDecoder().decode(await downloads.blobs[0].arrayBuffer());
    expect(archive.startsWith("PK")).toBe(true);
    expect(archive).toContain("manifest.json");
    expect(archive).toContain(`${logName}.json`);
    expect(archive).toContain("Main.json");
  });
});

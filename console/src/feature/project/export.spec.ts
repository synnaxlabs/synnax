// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, project, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Project } from "@/feature/project";
import {
  captureBrowserDownloads,
  removeFilePickers,
  renderHookWithConsole,
  uniqueName,
} from "@/testutil";

const client: Synnax = createTestClient();

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

describe("Project.useExport", () => {
  afterEach(() => {
    removeFilePickers();
    vi.restoreAllMocks();
  });

  it("downloads the project as a zip named after it", async () => {
    const downloads = captureBrowserDownloads();
    const { proj, logName } = await createProjectWithPanel();
    const { result } = await renderHookWithConsole(() => Project.useExport(), {
      client,
    });
    act(() => result.current(proj.key));
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

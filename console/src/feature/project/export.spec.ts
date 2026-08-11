// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, project, type status, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Project } from "@/feature/project";
import { createExecutingHandleError } from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  captureBrowserDownloads,
  createTestStore,
  removeFilePickers,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client: Synnax = createTestClient();

const createExportContext = (
  store: TestStore,
): { ctx: Project.ExportContext; statuses: status.Crude[] } => {
  const statuses: status.Crude[] = [];
  const ctx: Project.ExportContext = {
    client,
    store,
    handleError: createExecutingHandleError((message, exc) => {
      console.error(message, exc);
    }),
    addStatus: (s) => void statuses.push(s),
  };
  return { ctx, statuses };
};

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

describe("project export", () => {
  afterEach(() => {
    removeFilePickers();
    vi.restoreAllMocks();
  });

  it("downloads the active project as a zip named after it", async () => {
    const downloads = captureBrowserDownloads();
    const { proj, logName } = await createProjectWithPanel();
    const store = await createTestStore({
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: proj.key },
      },
    });
    const { ctx, statuses } = createExportContext(store);
    Project.export_(null, ctx);
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${proj.name}.zip`);
    // Zip entry names are stored uncompressed, so the archive names its own files.
    const archive = new TextDecoder().decode(await downloads.blobs[0].arrayBuffer());
    expect(archive.startsWith("PK")).toBe(true);
    expect(archive).toContain("manifest.json");
    expect(archive).toContain(`${logName}.json`);
    expect(archive).toContain("Main.json");
    await waitFor(() =>
      expect(statuses.some((s) => s.variant === "success")).toBe(true),
    );
  });

  it("downloads a non-active project by key", async () => {
    const downloads = captureBrowserDownloads();
    const { proj } = await createProjectWithPanel();
    const store = await createTestStore({
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: "other" },
      },
    });
    const { ctx } = createExportContext(store);
    Project.export_(proj.key, ctx);
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${proj.name}.zip`);
  });
});

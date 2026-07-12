// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type panel, project, type status, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Project } from "@/feature/project";
import { createExecutingHandleError } from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  createTestStore,
  installPickedDirectory,
  removeFilePickers,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client: Synnax = createTestClient();

interface CreateExportContextArgs {
  store: TestStore;
  confirmResult?: boolean;
  extractors?: Record<
    string,
    (key: string, ctx: unknown) => Promise<{ data: string; name: string }>
  >;
}

const createExportContext = ({
  store,
  confirmResult = true,
  extractors = {},
}: CreateExportContextArgs): {
  ctx: Project.ExportContext;
  statuses: status.Crude[];
  confirm: ReturnType<typeof vi.fn>;
} => {
  const statuses: status.Crude[] = [];
  const confirm = vi.fn(async () => confirmResult);
  const ctx: Project.ExportContext = {
    client,
    store,
    confirm,
    handleError: createExecutingHandleError((message, exc) => {
      console.error(message, exc);
    }),
    extractors,
    addStatus: (s) => void statuses.push(s),
  };
  return { ctx, statuses, confirm };
};

const createPanelWithLog = async (
  projectKey: project.Key,
  logKey: string,
): Promise<panel.Panel> =>
  await client.panels.create({
    name: "Main",
    root: {
      variant: "leaf",
      tabs: [
        { variant: "resource", key: uuid.create(), resource: log.ontologyID(logKey) },
      ],
    },
    parent: project.ontologyID(projectKey),
  });

describe("project export", () => {
  afterEach(() => {
    removeFilePickers();
    vi.restoreAllMocks();
  });

  it("should export the active project's panels and component files", async () => {
    const p = await client.projects.create({ name: uniqueName("proj"), layout: {} });
    const logKey = uuid.create();
    const exported = await createPanelWithLog(p.key, logKey);
    const store = await createTestStore({
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: p.key },
      },
    });
    const writes = installPickedDirectory();
    const extractor = vi
      .fn()
      .mockResolvedValue({ data: '{"type":"log"}', name: "My Log" });
    const { ctx, statuses } = createExportContext({
      store,
      extractors: { log: extractor },
    });
    Project.export_(null, ctx);
    await waitFor(() => expect(writes.has(Project.PANELS_FILE_NAME)).toBe(true));
    await waitFor(() => expect(writes.has("My Log.json")).toBe(true));
    const panels = JSON.parse(
      writes.get(Project.PANELS_FILE_NAME) ?? "[]",
    ) as panel.Panel[];
    expect(panels.map(({ key }) => key)).toContain(exported.key);
    expect(extractor).toHaveBeenCalledWith(logKey, { store, client });
    await waitFor(() =>
      expect(
        statuses.some((s) => s.variant === "success" && s.message?.includes(p.name)),
      ).toBe(true),
    );
  });

  it("should abort the export when the user declines to replace", async () => {
    const p = await client.projects.create({ name: uniqueName("proj"), layout: {} });
    const store = await createTestStore({
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: p.key },
      },
    });
    const writes = installPickedDirectory();
    const { ctx, statuses, confirm } = createExportContext({
      store,
      confirmResult: false,
    });
    Project.export_(null, ctx);
    await waitFor(() => expect(confirm).toHaveBeenCalledTimes(1));
    await act(async () => {});
    expect(writes.size).toBe(0);
    expect(statuses).toHaveLength(0);
  });

  it("should export a non-active project's panels from the server", async () => {
    const target = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    await createPanelWithLog(target.key, uuid.create());
    const store = await createTestStore({
      preloadedState: {
        [Session.Project.SLICE_NAME]: { version: 0, selected: "other" },
      },
    });
    const writes = installPickedDirectory();
    const { ctx, statuses } = createExportContext({ store });
    Project.export_(target.key, ctx);
    await waitFor(() => expect(writes.has(Project.PANELS_FILE_NAME)).toBe(true));
    const panels = JSON.parse(
      writes.get(Project.PANELS_FILE_NAME) ?? "[]",
    ) as panel.Panel[];
    expect(panels).toHaveLength(1);
    await waitFor(() =>
      expect(statuses.some((s) => s.variant === "success")).toBe(true),
    );
  });
});

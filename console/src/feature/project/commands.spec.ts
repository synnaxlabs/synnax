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
import { type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Project } from "@/feature/project";
import { Session } from "@/session";
import {
  captureBrowserDownloads,
  fakePickedFile,
  interceptFilePicker,
  removeSaveFilePicker,
  uniqueName,
} from "@/testutil";

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

const ZONE_TEXT = "Drop a .zip or folder here";

/** Opens the import modal through the palette, capturing raised statuses. */
const openImportModal = async () => {
  const statuses: Status.NotificationSpec[] = [];
  const { openCommandPalette, selectCommand, store } = await renderPalette({
    commands: Project.COMMANDS,
    client,
    onStatuses: (next) => statuses.splice(0, statuses.length, ...next),
  });
  await openCommandPalette();
  await selectCommand("Import project");
  await screen.findByText(ZONE_TEXT);
  return { statuses, store };
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
    expect(await screen.findByText(ZONE_TEXT)).toBeTruthy();
  });

  it("should import a picked folder with nested group directories", async () => {
    const { statuses, store } = await openImportModal();
    // Installed after the palette click: the interceptor spies every element's click
    // method, which would swallow the palette's synthetic command-select click.
    const picker = interceptFilePicker();
    const name = uniqueName("modal_proj");
    fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([
      fakePickedFile(
        "manifest.json",
        JSON.stringify({ version: 1, type: "project", name }),
        `${name}/manifest.json`,
      ),
      fakePickedFile(
        "Pressure.json",
        JSON.stringify({ version: 2, type: "log", name: "Pressure", channels: [] }),
        `${name}/Propulsion/Pressure.json`,
      ),
    ]);
    await waitFor(() =>
      expect(
        statuses.some(
          (st) =>
            st.variant === "success" && st.message === `Imported project "${name}"`,
        ),
      ).toBe(true),
    );
    const imported = Session.Project.selectSelected(store.getState());
    const proj = await client.projects.retrieve(imported);
    expect(proj.name).toEqual(name);
    const children = await client.ontology.retrieveChildren(
      project.ontologyID(imported),
    );
    expect(children).toHaveLength(1);
    expect(children[0].name).toEqual("Propulsion");
    expect(children[0].id.type).toEqual("group");
    const grandChildren = await client.ontology.retrieveChildren(children[0].id);
    expect(grandChildren.map(({ name }) => name)).toEqual(["Pressure"]);
  });

  it("should raise an error when the picked folder is not a bundle", async () => {
    const { statuses } = await openImportModal();
    const picker = interceptFilePicker();
    fireEvent.click(screen.getByRole("button", { name: "Select folder" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("stray.json", "{}", "no_manifest/stray.json")]);
    await waitFor(() =>
      expect(
        statuses.some(
          (st) => st.variant === "error" && st.message === "Failed to import project",
        ),
      ).toBe(true),
    );
    expect(screen.queryByText(ZONE_TEXT)).not.toBeNull();
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

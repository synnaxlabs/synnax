// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { project, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul, type Status } from "@synnaxlabs/pluto";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Project } from "@/feature/project";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import {
  CaptureStatuses,
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client: Synnax = createTestClient();

afterEach(() => {
  vi.restoreAllMocks();
});

/** Exports the project as a bundle and returns its zip bytes. */
const exportBundle = async (key: project.Key): Promise<Uint8Array<ArrayBuffer>> => {
  const stream = await client.projects.export(key, { encoding: "JSON" });
  return new Uint8Array(await new Response(stream).arrayBuffer());
};

describe("Project.ingest", () => {
  it("imports a bundle and selects the created project", async () => {
    const src = await client.projects.create({
      name: uniqueName("ingest"),
      layout: {},
    });
    await client.logs.create(src.key, { name: "Metrics" });
    const bundle = await exportBundle(src.key);
    const { store } = await createConsoleWrapper({ client });
    await Project.ingest(`${src.name}.zip`, bundle, { client, store });
    const imported = Session.Project.selectSelected(store.getState());
    expect(imported).not.toEqual(src.key);
    const proj = await client.projects.retrieve(imported);
    expect(proj.name).toEqual(src.name);
    const children = await client.ontology.retrieveChildren(
      project.ontologyID(imported),
    );
    expect(children.map(({ name }) => name)).toEqual(["Metrics"]);
  });
});

const ZONE_TEXT = "Drop a .zip or folder here";

const renderModal = async () => {
  const statuses: Status.NotificationSpec[] = [];
  const { wrapper, store } = await createConsoleWrapper({ client });
  const Opener = (): ReactElement => {
    const open = Project.useImportModal();
    return <button onClick={() => open()}>open import</button>;
  };
  await renderSuspended(
    <Haul.Provider>
      <Opener />
      <Modals.Stack />
      <CaptureStatuses
        onStatuses={(next) => statuses.splice(0, statuses.length, ...next)}
      />
    </Haul.Provider>,
    { wrapper },
  );
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "open import" }));
  });
  await screen.findByText(ZONE_TEXT);
  return { statuses, store };
};

describe("Project.useImportModal", () => {
  it("imports a picked folder with nested group directories", async () => {
    const picker = interceptFilePicker();
    const { statuses, store } = await renderModal();
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

  it("raises an error when the picked folder is not a bundle", async () => {
    const picker = interceptFilePicker();
    const { statuses } = await renderModal();
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
});

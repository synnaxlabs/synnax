// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, type Synnax } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  commitTextEdit,
  createConsoleWrapper,
  getBySelector,
  renderWithConsole,
  uniqueName,
} from "@/testutil";

const client: Synnax = createTestClient();

const roles = new RoleClients(client);

const TRIGGER = ".console-project-selector__trigger";

/** Opens the selector dialog with a freshly created project matched by `term`. */
const openSelectorAt = async (term: string, as: Synnax = client) => {
  const active = await client.projects.create({
    name: `proj-active-${id.create()}`,
    layout: {},
  });
  const target = await client.projects.create({ name: term, layout: {} });
  const { wrapper } = await createConsoleWrapper({
    client: as,
    preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(active) },
  });
  const { container } = render(<Project.Selector />, { wrapper });
  fireEvent.click(await waitFor(() => getBySelector(container, TRIGGER)));
  const search = await screen.findByPlaceholderText("Search projects...");
  fireEvent.change(search, { target: { value: term } });
  await screen.findByText(term);
  return { active, target };
};

describe("Project.Selector", () => {
  it("renders nothing when the user lacks retrieve permission", async () => {
    const { container } = await renderWithConsole(<Project.Selector />, {
      preloadedState: {
        [Session.Project.SLICE_NAME]: {
          ...Session.Project.ZERO_SLICE_STATE,
          selected: id.create(),
        },
      },
    });
    expect(container.querySelector(TRIGGER)).toBeNull();
  });

  it("switches the active project on selection", async () => {
    const active: project.Project = await client.projects.create({
      name: `proj-active-${id.create()}`,
      layout: {},
    });
    const target: project.Project = await client.projects.create({
      name: `proj-target-${id.create()}`,
      layout: {},
    });
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(active) },
    });
    const { container } = render(<Project.Selector />, { wrapper });

    const trigger = await waitFor(() => getBySelector(container, TRIGGER));
    fireEvent.click(trigger);
    const search = await screen.findByPlaceholderText("Search projects...");
    fireEvent.change(search, { target: { value: target.name } });
    fireEvent.click(await screen.findByText(target.name));

    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(target.key),
    );
  });

  it("gives numbered siblings different avatar initials", async () => {
    const active: project.Project = await client.projects.create({
      name: `proj-active-${id.create()}`,
      layout: {},
    });
    const prefix = uniqueName("stand");
    await client.projects.create({ name: `${prefix}_1`, layout: {} });
    await client.projects.create({ name: `${prefix}_2`, layout: {} });
    const { wrapper } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(active) },
    });
    const { container } = render(<Project.Selector />, { wrapper });

    const trigger = await waitFor(() => getBySelector(container, TRIGGER));
    fireEvent.click(trigger);
    const search = await screen.findByPlaceholderText("Search projects...");
    fireEvent.change(search, { target: { value: prefix } });
    await screen.findByText(`${prefix}_1`);

    expect(await screen.findByText("S1")).toBeTruthy();
    expect(await screen.findByText("S2")).toBeTruthy();
  });

  it("renames a project in place from the context menu", async () => {
    const { target } = await openSelectorAt(uniqueName("proj"));
    fireEvent.contextMenu(await screen.findByText(target.name));
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditingElement();
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.projects.retrieve(target.key)).name).toBe(renamed),
    );
  });

  it("does not start an edit when the project name is double-clicked", async () => {
    const { target } = await openSelectorAt(uniqueName("proj"));
    const name = await screen.findByText(target.name);
    fireEvent.doubleClick(name);
    await act(async () => {});
    expect(name.getAttribute("contenteditable")).not.toBe("true");
  });

  it("offers a menu when the create row is right-clicked", async () => {
    await openSelectorAt(uniqueName("proj"));
    fireEvent.contextMenu(await screen.findByText("New project"));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("renders the project name as plain text for a viewer", async () => {
    const { target } = await openSelectorAt(
      uniqueName("proj"),
      await roles.get("Viewer"),
    );
    const name = await screen.findByText(target.name);
    await act(async () => {});
    expect(name.className).not.toContain("pluto-text--editable");
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type Synnax, user } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithPolicy,
} from "@synnaxlabs/client/testutil";
import { Triggers } from "@synnaxlabs/pluto";
import { id, uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole, uniqueName } from "@/testutil";

const client: Synnax = createTestClient();

describe("project/Splash", () => {
  describe("without permissions", () => {
    it("should hide the project list and create action when there is no client", async () => {
      await renderWithConsole(<Project.Splash />);
      expect(screen.getByText("Projects")).toBeDefined();
      expect(
        screen.getByText("You do not have permission to create a project."),
      ).toBeDefined();
      expect(screen.queryByText("New project")).toBeNull();
    });

    it("should report why the list is empty when it never loaded", async () => {
      await renderWithConsole(<Project.Splash />);
      expect(await screen.findByText("Failed to retrieve projects")).toBeDefined();
      expect(screen.queryByText("No projects")).toBeNull();
    });

    it("should name the denial when the list read is refused", async () => {
      await client.projects.create({ name: uniqueName("argon"), layout: {} });
      const denied = await createTestClientWithPolicy(client, {
        name: uuid.create(),
        objects: [
          user.TYPE_ONTOLOGY_ID,
          access.role.TYPE_ONTOLOGY_ID,
          access.policy.TYPE_ONTOLOGY_ID,
        ],
        actions: ["retrieve"],
      });
      const { wrapper } = await createConsoleWrapper({ client: denied });
      render(<Project.Splash />, { wrapper });
      expect(
        await screen.findByText(
          "Failed to retrieve projects: You do not have permission to do that",
        ),
      ).toBeDefined();
      expect(screen.queryByText("No projects created.")).toBeNull();
    });
  });

  describe("selecting an existing project", () => {
    it("should activate the project", async () => {
      const name = `proj-${id.create()}`;
      const proj = await client.projects.create({ name, layout: {} });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Project.Splash />, { wrapper });

      const search = await screen.findByPlaceholderText("Search projects...");
      fireEvent.change(search, { target: { value: name } });
      fireEvent.click(await screen.findByText(name));

      await waitFor(() => {
        const active = Session.Project.selectOptionalSelected(store.getState());
        expect(active).toEqual(proj.key);
      });
    });
  });

  describe("searching", () => {
    it("should filter the list to matching projects and allow selecting one", async () => {
      const first = uniqueName("hydrogen");
      const second = uniqueName("xenon");
      await client.projects.create({ name: first, layout: {} });
      const created = await client.projects.create({ name: second, layout: {} });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Project.Splash />, { wrapper });

      const input = await screen.findByPlaceholderText("Search projects...");
      fireEvent.change(input, { target: { value: first } });
      await screen.findByText(first);

      fireEvent.change(input, { target: { value: second } });
      await screen.findByText(second);
      await waitFor(() => expect(screen.queryByText(first)).toBeNull());

      fireEvent.click(screen.getByText(second));
      await waitFor(() => {
        const active = Session.Project.selectOptionalSelected(store.getState());
        expect(active).toEqual(created.key);
      });
    });

    it("should show a no-match message instead of the created-none empty state", async () => {
      await client.projects.create({ name: uniqueName("krypton"), layout: {} });
      const { wrapper } = await createConsoleWrapper({ client });
      render(<Project.Splash />, { wrapper });

      const input = await screen.findByPlaceholderText("Search projects...");
      fireEvent.change(input, { target: { value: uniqueName("nomatchterm") } });

      await screen.findByText("No matching projects");
      expect(screen.queryByText("No projects")).toBeNull();
    });
  });

  describe("keyboard navigation", () => {
    it("should select the hovered project when Enter is pressed", async () => {
      const name = uniqueName("kb_enter");
      const proj = await client.projects.create({ name, layout: {} });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(
        <Triggers.Provider>
          <Project.Splash />
        </Triggers.Provider>,
        { wrapper },
      );

      const search = await screen.findByPlaceholderText("Search projects...");
      fireEvent.change(search, { target: { value: name } });
      await screen.findByText(name);

      fireEvent.keyDown(search, { code: "ArrowDown" });
      fireEvent.keyUp(search, { code: "ArrowDown" });
      fireEvent.keyDown(search, { code: "Enter" });
      await waitFor(() => {
        const active = Session.Project.selectOptionalSelected(store.getState());
        expect(active).toEqual(proj.key);
      });
    });

    it("should move the hover with the arrow keys before selecting", async () => {
      const prefix = uniqueName("kb_arrows");
      const a = await client.projects.create({ name: `${prefix}_a`, layout: {} });
      const b = await client.projects.create({ name: `${prefix}_b`, layout: {} });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(
        <Triggers.Provider>
          <Project.Splash />
        </Triggers.Provider>,
        { wrapper },
      );

      const search = await screen.findByPlaceholderText("Search projects...");
      fireEvent.change(search, { target: { value: prefix } });
      const elA = await screen.findByText(a.name);
      const elB = await screen.findByText(b.name);
      const aIsFirst =
        (elA.compareDocumentPosition(elB) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;
      const second = aIsFirst ? b : a;

      fireEvent.keyDown(search, { code: "ArrowDown" });
      fireEvent.keyUp(search, { code: "ArrowDown" });
      fireEvent.keyDown(search, { code: "ArrowDown" });
      fireEvent.keyUp(search, { code: "ArrowDown" });
      fireEvent.keyDown(search, { code: "Enter" });
      await waitFor(() => {
        const active = Session.Project.selectOptionalSelected(store.getState());
        expect(active).toEqual(second.key);
      });
    });

    it("should not select a project while a modal is open", async () => {
      const name = uniqueName("kb_modal");
      await client.projects.create({ name, layout: {} });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(
        <Triggers.Provider>
          <Project.Splash />
          <Modals.Stack />
        </Triggers.Provider>,
        { wrapper },
      );

      const search = await screen.findByPlaceholderText("Search projects...");
      fireEvent.change(search, { target: { value: name } });
      await screen.findByText(name);
      fireEvent.keyDown(search, { code: "ArrowDown" });
      fireEvent.keyUp(search, { code: "ArrowDown" });

      fireEvent.click(screen.getByText("New project"));
      await screen.findByPlaceholderText("Name");
      fireEvent.keyDown(document.body, { code: "Enter" });
      await act(async () => {});
      expect(Session.Project.selectOptionalSelected(store.getState())).toBeUndefined();
    });
  });

  describe("creating a new project", () => {
    it("should open the create modal and activate the created project", async () => {
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(
        <>
          <Project.Splash />
          <Modals.Stack />
        </>,
        { wrapper },
      );
      const name = `proj-${id.create()}`;

      fireEvent.click(await screen.findByText("New project"));
      fireEvent.change(await screen.findByPlaceholderText("Name"), {
        target: { value: name },
      });
      fireEvent.click(screen.getByRole("button", { name: "Create" }));

      const active = await waitFor(() => {
        const key = Session.Project.selectOptionalSelected(store.getState());
        if (key == null) throw new Error("no active project selected");
        return key;
      });
      const created = await client.projects.retrieve(active);
      expect(created.name).toEqual(name);
    });
  });
});

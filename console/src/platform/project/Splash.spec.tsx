// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/platform/project";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client: Synnax = createTestClient();

describe("project/Splash", () => {
  describe("without permissions", () => {
    it("should hide the project list and create form when there is no client", async () => {
      await renderWithConsole(<Project.Splash />);
      expect(screen.getByText("New Project")).toBeDefined();
      expect(
        screen.getByText("You do not have permission to create a project."),
      ).toBeDefined();
      expect(screen.queryByText("Open a Project")).toBeNull();
    });
  });

  describe("selecting an existing project", () => {
    it("should activate the project", async () => {
      const name = `proj-${id.create()}`;
      const proj = await client.projects.create({ name, layout: {} });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Project.Splash />, { wrapper });

      fireEvent.click(await screen.findByText(name));

      await waitFor(() => {
        const active = Session.Project.selectOptionalSelected(store.getState());
        expect(active).toEqual(proj.key);
      });
    });
  });

  describe("creating a new project", () => {
    it("should create the project on the server and activate it", async () => {
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Project.Splash />, { wrapper });
      const name = `proj-${id.create()}`;

      fireEvent.change(await screen.findByPlaceholderText("Project name"), {
        target: { value: name },
      });
      fireEvent.click(screen.getByText("Create Project"));

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

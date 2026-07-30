// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";
import { createConsoleWrapper, renderWithConsole } from "@/testutil";

const client: Synnax = createTestClient();

describe("project/Splash", () => {
  describe("without permissions", () => {
    it("should hide the project list and create action when there is no client", async () => {
      await renderWithConsole(<Project.Splash />);
      expect(screen.getByText("Projects")).toBeDefined();
      expect(
        screen.getByText("You do not have permission to create a project."),
      ).toBeDefined();
      expect(screen.queryByText("New Project")).toBeNull();
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

      fireEvent.click(await screen.findByText("New Project"));
      fireEvent.change(await screen.findByPlaceholderText("Project Name"), {
        target: { value: name },
      });
      fireEvent.click(screen.getByRole("button", { name: "Create" }));

      const active = await waitFor(() => {
        const key = Session.Project.selectOptionalSelected(store.getState());
        if (key == null) throw new Error("no active project selected");
        return key;
      });
      const created = await client.projects.retrieve({ key: active });
      expect(created.name).toEqual(name);
    });
  });
});

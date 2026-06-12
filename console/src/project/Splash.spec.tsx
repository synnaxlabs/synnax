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
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/layout";
import { Project } from "@/project";
import { Splash } from "@/project/Splash";
import { createConsoleWrapper, renderWithConsole } from "@/testUtils";

const client: Synnax = createTestClient();

// A layout slice with a single placed mosaic tab, distinguishable from the zero
// slice so the select flow can prove the project's saved layout was loaded.
const savedLayout = (layoutKey: string): Layout.SliceState => {
  let s = Layout.reducer(undefined, { type: "@@INIT" });
  s = Layout.reducer(
    s,
    Layout.place({
      windowKey: "main",
      key: layoutKey,
      type: "schematic",
      name: "Operator",
      location: "mosaic",
    }),
  );
  return s;
};

describe("project/Splash", () => {
  describe("without permissions", () => {
    it("should hide the project list and create form when there is no client", () => {
      renderWithConsole(<Splash />);
      expect(screen.getByText("New Project")).toBeDefined();
      expect(
        screen.getByText("You do not have permission to create a project."),
      ).toBeDefined();
      expect(screen.queryByText("Open a Project")).toBeNull();
    });
  });

  describe("without any projects", () => {
    it("should hide the project list when no projects exist", async () => {
      const emptyClient = createTestClient();
      const retrieveSpy = vi
        .spyOn(emptyClient.projects, "retrieve")
        .mockResolvedValue([]);
      const { wrapper } = await createConsoleWrapper({ client: emptyClient });
      render(<Splash />, { wrapper });

      await screen.findByText("Create Project");
      await waitFor(() => expect(retrieveSpy).toHaveBeenCalled());
      expect(screen.queryByText("Open a Project")).toBeNull();
    });
  });

  describe("selecting an existing project", () => {
    it("should activate the project and load its saved layout", async () => {
      const name = `proj-${id.create()}`;
      const layoutKey = id.create();
      const proj = await client.projects.create({
        name,
        layout: savedLayout(layoutKey),
      });
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Splash />, { wrapper });

      fireEvent.click(await screen.findByText(name));

      await waitFor(() => {
        const active = Project.selectOptionalActive(store.getState());
        expect(active?.key).toEqual(proj.key);
        expect(active?.name).toEqual(name);
      });
      const placed = Layout.selectByFilter(
        store.getState(),
        (l) => l.key === layoutKey,
      );
      expect(placed).toBeDefined();
      expect(placed?.name).toEqual("Operator");
    });
  });

  describe("creating a new project", () => {
    it("should create the project on the server and activate it", async () => {
      const { wrapper, store } = await createConsoleWrapper({ client });
      render(<Splash />, { wrapper });
      const name = `proj-${id.create()}`;

      fireEvent.change(await screen.findByPlaceholderText("Project name"), {
        target: { value: name },
      });
      fireEvent.click(screen.getByText("Create Project"));

      await waitFor(() => {
        const active = Project.selectOptionalActive(store.getState());
        expect(active?.name).toEqual(name);
      });
      const active = Project.selectOptionalActive(store.getState());
      const created = await client.projects.retrieve(active!.key);
      expect(created.name).toEqual(name);
    });
  });
});

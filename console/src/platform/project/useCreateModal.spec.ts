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
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { openModal } from "@/platform/modals/testutil";
import { Project } from "@/platform/project";
import { Session } from "@/session";

const client: Synnax = createTestClient();

const submitName = async (name: string): Promise<void> => {
  await waitFor(() => expect(screen.getByPlaceholderText("Name")).toBeTruthy());
  fireEvent.change(screen.getByPlaceholderText("Name"), { target: { value: name } });
  fireEvent.click(screen.getByRole("button", { name: "Create" }));
};

describe("Project.useCreateModal", () => {
  it("renders the create form and disables Create without a client", async () => {
    await openModal(Project.useCreateModal);
    await waitFor(() => expect(screen.getByPlaceholderText("Name")).toBeTruthy());
    const createBtn = await waitFor(() =>
      screen.getByRole("button", { name: "Create" }),
    );
    expect(createBtn.className).toContain("pluto--disabled");
  });

  it("creates the project on the Core, activates it, and closes", async () => {
    const { store } = await openModal(Project.useCreateModal, { client });
    const name = `proj-${id.create()}`;
    await submitName(name);

    await waitFor(() => {
      const active = Session.Project.selectOptionalSelected(store.getState());
      expect(active).not.toBeUndefined();
    });
    await waitFor(() => expect(screen.queryByPlaceholderText("Name")).toBeNull());

    const active = Session.Project.selectSelected(store.getState());
    const created = await client.projects.retrieve(active);
    expect(created.name).toBe(name);
  });

  it("seeds the project with a panel holding a component selector tab", async () => {
    const { store } = await openModal(Project.useCreateModal, { client });
    await submitName(`proj-${id.create()}`);

    await waitFor(() => {
      const active = Session.Project.selectOptionalSelected(store.getState());
      expect(active).not.toBeUndefined();
    });
    const active = Session.Project.selectSelected(store.getState());

    await waitFor(async () => {
      const panels = await client.panels.retrieve({
        parent: project.ontologyID(active),
      });
      expect(panels).toHaveLength(1);
    });
    const [pan] = await client.panels.retrieve({
      parent: project.ontologyID(active),
    });
    expect(pan.name).toBe("New panel");
    if (pan.root.variant !== "leaf") throw new Error("expected a leaf root");
    expect(pan.root.tabs).toHaveLength(1);
    const [tab] = pan.root.tabs;
    if (tab.variant !== "view") throw new Error("expected a view tab");
    expect(tab.type).toBe("selector");

    await waitFor(() =>
      expect(Session.Panel.selectSelected(store.getState())).toBe(pan.key),
    );
  });
});

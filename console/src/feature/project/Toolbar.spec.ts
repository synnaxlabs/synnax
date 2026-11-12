// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, onTestFinished } from "vitest";

import { Project } from "@/feature/project";
import { findTreeRow, renderToolbar } from "@/platform/tree/testutil";
import { getIconButton, getIconButtons, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

describe("project toolbar", () => {
  it("should list the top-level project groups and the projects in no group", async () => {
    const p = await client.projects.create({ name: uniqueName("project") });
    const g = await client.groups.create({
      scope: "project",
      name: uniqueName("group"),
    });
    onTestFinished(async () => {
      await client.projects.delete(p.key);
      await client.groups.delete(g.key);
    });
    await renderToolbar(Project.TOOLBAR.content, { client, items: Project.TREE_ITEMS });
    expect(await findTreeRow(p.name)).toBeTruthy();
    expect(await findTreeRow(g.name)).toBeTruthy();
  });

  it("should open the create modal from the create action", async () => {
    await renderToolbar(Project.TOOLBAR.content, { client, items: Project.TREE_ITEMS });
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    expect(await screen.findByRole("dialog")).toBeTruthy();
  });
});

describe("project toolbar permissions", () => {
  it("should withhold the create action from a viewer", async () => {
    await renderToolbar(Project.TOOLBAR.content, {
      client: await roles.get("Viewer"),
      items: Project.TREE_ITEMS,
    });
    expect(await screen.findByText("Projects")).toBeTruthy();
    expect(getIconButtons(document.body, "add")).toHaveLength(0);
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Project } from "@/feature/project";
import { renderToolbar } from "@/platform/tree/menuTestutil";
import { getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

describe("project toolbar", () => {
  it("should list projects in the tree", async () => {
    await client.projects.create({ name: uniqueName("project"), layout: {} });
    const roots = await client.ontology.children.retrieve({ ids: ontology.ROOT_ID });
    const projectsGroup = roots.find((r) => r.name === "Projects");
    if (projectsGroup == null) throw new Error("Projects group not found");
    const [firstChild] = await client.ontology.children.retrieve({ ids: projectsGroup.id });
    await renderToolbar(Project.TOOLBAR.content, { client });
    expect(screen.getByText("Projects")).toBeTruthy();
    expect(await screen.findByText(firstChild.name)).toBeTruthy();
  });

  it("should open the create modal from the create action", async () => {
    await renderToolbar(Project.TOOLBAR.content, { client });
    await waitFor(() => getIconButton(document.body, "add"));
    fireEvent.click(getIconButton(document.body, "add"));
    expect(await screen.findByRole("dialog")).toBeTruthy();
  });
});

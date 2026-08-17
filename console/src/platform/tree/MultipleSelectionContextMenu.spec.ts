// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  group,
  label,
  ontology,
  ranger,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { createTestRange } from "@/platform/range/testutil";
import { getTreeRow } from "@/platform/tree/menuTestutil";
import { renderOntologyTree } from "@/platform/tree/treeTestutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  createTestClientWithGrants,
  uniqueName,
  withControlHeld,
} from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

interface MixedSelection {
  parentID: ontology.ID;
  child: group.Group;
  range: ranger.Range;
}

const setupMixedSelection = async (as: Client = client): Promise<MixedSelection> => {
  const parent = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("parent"),
  });
  const parentID = group.ontologyID(parent.key);
  const child = await client.groups.create({
    parent: parentID,
    name: uniqueName("child"),
  });
  const range = await createTestRange(client);
  await client.ontology.addChildren(parentID, ranger.ontologyID(range.key));
  await renderOntologyTree({ client: as, projectClient: client, root: parentID });
  await screen.findByText(child.name);
  await screen.findByText(range.name);
  fireEvent.click(getTreeRow(child.name));
  withControlHeld(() => fireEvent.click(getTreeRow(range.name)));
  fireEvent.contextMenu(getTreeRow(range.name));
  return { parentID, child, range };
};

describe("Tree.MultipleSelectionContextMenu", () => {
  it("should offer grouping and reload for a mixed-type selection", async () => {
    await setupMixedSelection();
    await screen.findByText("Group selection");
    await screen.findByText("Reload Console");
  });

  it("should group the mixed selection under a new group when the group item is clicked", async () => {
    const { parentID, child, range } = await setupMixedSelection();
    fireEvent.click(await screen.findByText("Group selection"));
    const editable = await awaitTextEditingElement();
    const name = uniqueName("grp");
    await act(async () => {
      commitTextEdit(editable, name);
    });
    let newKey = "";
    await waitFor(async () => {
      const siblings = await client.ontology.children.retrieve({ ids: parentID });
      const created = siblings.find((c) => c.name === name);
      expect(created).toBeDefined();
      newKey = created?.id.key ?? "";
    });
    await waitFor(async () => {
      const grouped = await client.ontology.children.retrieve({
        ids: group.ontologyID(newKey),
      });
      const keys = grouped.map((c) => c.id.key);
      expect(keys).toContain(child.key);
      expect(keys).toContain(range.key);
    });
  });
});

describe("Tree.MultipleSelectionContextMenu permissions", () => {
  it("should withhold grouping from a viewer", async () => {
    await setupMixedSelection(await roles.get("Viewer"));
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Group selection")).toBeNull();
  });

  // Grouping mints a group, so a subject who may rewrite the selected resources still
  // needs its own grant to create one.
  it("should withhold grouping from a subject who cannot create groups", async () => {
    const writer = await createTestClientWithGrants(client, {
      retrieve: [group.TYPE_ONTOLOGY_ID, ranger.TYPE_ONTOLOGY_ID, label.TYPE_ONTOLOGY_ID],
      update: [group.TYPE_ONTOLOGY_ID, ranger.TYPE_ONTOLOGY_ID],
    });
    await setupMixedSelection(writer);
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Group selection")).toBeNull();
  });

  it("should offer grouping to an engineer", async () => {
    await setupMixedSelection(await roles.get("Engineer"));
    expect(await screen.findByText("Group selection")).toBeTruthy();
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology, ranger } from "@synnaxlabs/client";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  renderOntologyTree,
  treeRow,
  withControlHeld,
} from "@/platform/group/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

interface MixedSelection {
  parentID: ontology.ID;
  child: group.Group;
  range: ranger.Range;
}

const setupMixedSelection = async (): Promise<MixedSelection> => {
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
  await renderOntologyTree({ client, root: parentID });
  await screen.findByText(child.name);
  await screen.findByText(range.name);
  fireEvent.click(treeRow(child.name));
  withControlHeld(() => fireEvent.click(treeRow(range.name)));
  fireEvent.contextMenu(treeRow(range.name));
  return { parentID, child, range };
};

describe("Ontology.MultipleSelectionContextMenu", () => {
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
      const siblings = await client.ontology.retrieveChildren(parentID);
      const created = siblings.find((c) => c.name === name);
      expect(created).toBeDefined();
      newKey = created?.id.key ?? "";
    });
    await waitFor(async () => {
      const grouped = await client.ontology.retrieveChildren(group.ontologyID(newKey));
      const keys = grouped.map((c) => c.id.key);
      expect(keys).toContain(child.key);
      expect(keys).toContain(range.key);
    });
  });
});

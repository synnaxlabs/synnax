// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology } from "@synnaxlabs/client";
import { Tree } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Group } from "@/feature/group";
import { renderTreeContextMenu } from "@/platform/ontology/menuTestutil";
import { createResource } from "@/platform/ontology/testutil";
import { assertDefined, uniqueName } from "@/testutil";

const client = createTestClient();

const createGroup = async (parent: ontology.ID) =>
  await client.groups.create({ parent, name: uniqueName("group") });

const groupResource = (key: string, name: string) =>
  createResource(group.ontologyID(key), name);

describe("group ontology service", () => {
  it("should always accept drops and haul the group's own id", () => {
    expect(
      Group.ONTOLOGY_SERVICE.canDrop({ source: { key: "s", type: "t" }, items: [] }),
    ).toBe(true);
    const res = groupResource("g1", "g");
    expect(Group.ONTOLOGY_SERVICE.haulItems(res)).toEqual([res.id]);
  });

  it("should hide rename and ungroup for a zero-depth selection", async () => {
    const g = await createGroup(ontology.ROOT_ID);
    assertDefined(Group.ONTOLOGY_SERVICE.TreeContextMenu);
    await renderTreeContextMenu(Group.ONTOLOGY_SERVICE.TreeContextMenu, {
      client,
      resources: [groupResource(g.key, g.name)],
    });
    expect(await screen.findByText("New group")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Ungroup")).toBeNull();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should ungroup a nested group, moving its children to the parent", async () => {
    const parent = await createGroup(ontology.ROOT_ID);
    const child = await createGroup(group.ontologyID(parent.key));
    const grandchild = await createGroup(group.ontologyID(child.key));
    const parentID = group.ontologyID(parent.key);
    const childID = group.ontologyID(child.key);
    const grandchildID = group.ontologyID(grandchild.key);
    const [parentKey, childKey, grandchildKey] = [parentID, childID, grandchildID].map(
      (id) => ontology.idToString(id),
    );
    const nodes: Tree.Node<string>[] = [
      {
        key: parentKey,
        children: [{ key: childKey, children: [{ key: grandchildKey }] }],
      },
    ];
    const shape = Tree.flatten({ nodes, expanded: [parentKey, childKey] });
    const resources = [
      groupResource(parent.key, parent.name),
      groupResource(child.key, child.name),
      groupResource(grandchild.key, grandchild.name),
    ];
    assertDefined(Group.ONTOLOGY_SERVICE.TreeContextMenu);
    await renderTreeContextMenu(Group.ONTOLOGY_SERVICE.TreeContextMenu, {
      client,
      resources,
      selected: [childID],
      parentID,
      stateOverrides: { nodes, shape },
    });
    fireEvent.click(await screen.findByText("Ungroup"));
    await waitFor(async () => {
      const children = await client.ontology.retrieveChildren(parentID);
      expect(children.some((r) => r.name === grandchild.name)).toBe(true);
      expect(children.some((r) => r.name === child.name)).toBe(false);
    });
  });

  it("should label ungroup as delete for a childless nested group", async () => {
    const parent = await createGroup(ontology.ROOT_ID);
    const child = await createGroup(group.ontologyID(parent.key));
    const parentID = group.ontologyID(parent.key);
    const childID = group.ontologyID(child.key);
    const [parentKey, childKey] = [parentID, childID].map((id) =>
      ontology.idToString(id),
    );
    const nodes: Tree.Node<string>[] = [
      { key: parentKey, children: [{ key: childKey }] },
    ];
    const shape = Tree.flatten({ nodes, expanded: [parentKey] });
    assertDefined(Group.ONTOLOGY_SERVICE.TreeContextMenu);
    await renderTreeContextMenu(Group.ONTOLOGY_SERVICE.TreeContextMenu, {
      client,
      resources: [
        groupResource(parent.key, parent.name),
        groupResource(child.key, child.name),
      ],
      selected: [childID],
      stateOverrides: { nodes, shape },
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Ungroup")).toBeNull();
  });
});

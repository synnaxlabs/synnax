// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  access,
  channel,
  group,
  resource,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { Tree } from "@synnaxlabs/lyra/tree";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Group } from "@/platform/group";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const parentID = group.resourceID(id.create());
const childID = channel.resourceID(1);
const parentKey = resource.idToString(parentID);
const childKey = resource.idToString(childID);

const shape: Tree.Shape = Tree.flatten({
  nodes: [{ key: parentKey, children: [{ key: childKey }] }],
  expanded: [parentKey],
});

const rootID = channel.TYPE_RESOURCE_ID;

const renderItems = async (ui: ReactElement) => {
  const { wrapper } = await createConsoleWrapper({ client });
  return render(ui, { wrapper });
};

describe("Group.ContextMenuItem", () => {
  it("should dispatch the group item key through the menu when clicked", async () => {
    const onChange = vi.fn();
    await renderItems(
      <PMenu.Menu onChange={onChange}>
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={rootID}
          parentID={parentID}
          shape={shape}
        />
      </PMenu.Menu>,
    );
    const item = await waitFor(() => screen.getByText("Group selection"));
    fireEvent.click(item);
    await waitFor(() => expect(onChange).toHaveBeenCalledWith("group"));
  });

  it("should show the group item for a top-level selection", async () => {
    await renderItems(
      <Group.ContextMenuItem
        ids={[parentID]}
        rootID={rootID}
        parentID={rootID}
        shape={shape}
      />,
    );
    await waitFor(() => expect(screen.getByText("Group selection")).toBeTruthy());
  });

  it("should hide the group item in a tree that holds no groups", async () => {
    await renderItems(
      <>
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={rootID}
          parentID={parentID}
          shape={shape}
        />
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={access.role.TYPE_RESOURCE_ID}
          parentID={parentID}
          shape={shape}
        />
      </>,
    );
    await waitFor(() => expect(screen.getAllByText("Group selection").length).toBe(1));
  });

  it("should hide the group item when the selection is absent from the tree", async () => {
    const absentID = channel.resourceID(3);
    await renderItems(
      <>
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={rootID}
          parentID={parentID}
          shape={shape}
        />
        <Group.ContextMenuItem
          ids={[absentID]}
          rootID={rootID}
          parentID={rootID}
          shape={shape}
        />
      </>,
    );
    await waitFor(() => expect(screen.getAllByText("Group selection").length).toBe(1));
  });

  it("should hide the group item when the selection has more than one parent", async () => {
    await renderItems(
      <>
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={rootID}
          parentID={parentID}
          shape={shape}
        />
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={rootID}
          parentID={null}
          shape={shape}
        />
      </>,
    );
    await waitFor(() => expect(screen.getAllByText("Group selection").length).toBe(1));
  });

  it("should show the group item for a multi-item selection sharing the same minimum depth", async () => {
    const siblingID = channel.resourceID(2);
    const siblingKey = resource.idToString(siblingID);
    const multiShape: Tree.Shape = Tree.flatten({
      nodes: [{ key: parentKey, children: [{ key: childKey }, { key: siblingKey }] }],
      expanded: [parentKey],
    });
    await renderItems(
      <Group.ContextMenuItem
        ids={[childID, siblingID]}
        rootID={rootID}
        parentID={parentID}
        shape={multiShape}
      />,
    );
    await waitFor(() => expect(screen.getByText("Group selection")).toBeTruthy());
  });
});

describe("Group.ContextMenuItem permissions", () => {
  const renderAs = async (as: Client) => {
    const { wrapper } = await createConsoleWrapper({ client: as });
    return render(
      <Group.ContextMenuItem
        ids={[childID]}
        rootID={rootID}
        parentID={parentID}
        shape={shape}
      />,
      { wrapper },
    );
  };

  it("should withhold the group item from a viewer", async () => {
    const { container } = await renderAs(await roles.get("Viewer"));
    await waitFor(() => expect(container.textContent).toBe(""));
  });

  it("should offer the group item to an engineer", async () => {
    const { container } = await renderAs(await roles.get("Engineer"));
    await waitFor(() => expect(container.textContent).toContain("Group selection"));
  });
});

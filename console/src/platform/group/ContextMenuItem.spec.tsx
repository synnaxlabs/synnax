// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { group, ontology, ranger } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Menu as PMenu, Tree } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Group } from "@/platform/group";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const parentID = group.ontologyID(id.create());
const childID = ranger.ontologyID(id.create());
const parentKey = ontology.idToString(parentID);
const childKey = ontology.idToString(childID);

const shape: Tree.Shape = Tree.flatten({
  nodes: [{ key: parentKey, children: [{ key: childKey }] }],
  expanded: [parentKey],
});

const nestedRootID = group.ontologyID(id.create());

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
          rootID={ontology.ROOT_ID}
          shape={shape}
        />
      </PMenu.Menu>,
    );
    const item = await waitFor(() => screen.getByText("Group selection"));
    fireEvent.click(item);
    await waitFor(() => expect(onChange).toHaveBeenCalledWith("group"));
  });

  it("should show the group item for a top-level selection when the root is not the ontology root", async () => {
    await renderItems(
      <Group.ContextMenuItem ids={[parentID]} rootID={nestedRootID} shape={shape} />,
    );
    await waitFor(() => expect(screen.getByText("Group selection")).toBeTruthy());
  });

  it("should hide the group item for a top-level selection at the ontology root", async () => {
    await renderItems(
      <>
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={ontology.ROOT_ID}
          shape={shape}
        />
        <Group.ContextMenuItem
          ids={[parentID]}
          rootID={ontology.ROOT_ID}
          shape={shape}
        />
      </>,
    );
    await waitFor(() => expect(screen.getAllByText("Group selection").length).toBe(1));
  });

  it("should hide the group item when the selection is absent from the tree", async () => {
    const absentID = ranger.ontologyID(id.create());
    await renderItems(
      <>
        <Group.ContextMenuItem
          ids={[childID]}
          rootID={ontology.ROOT_ID}
          shape={shape}
        />
        <Group.ContextMenuItem
          ids={[absentID]}
          rootID={ontology.ROOT_ID}
          shape={shape}
        />
      </>,
    );
    await waitFor(() => expect(screen.getAllByText("Group selection").length).toBe(1));
  });

  it("should show the group item for a multi-item selection sharing the same minimum depth", async () => {
    const siblingID = ranger.ontologyID(id.create());
    const siblingKey = ontology.idToString(siblingID);
    const multiShape: Tree.Shape = Tree.flatten({
      nodes: [{ key: parentKey, children: [{ key: childKey }, { key: siblingKey }] }],
      expanded: [parentKey],
    });
    await renderItems(
      <Group.ContextMenuItem
        ids={[childID, siblingID]}
        rootID={ontology.ROOT_ID}
        shape={multiShape}
      />,
    );
    await waitFor(() => expect(screen.getByText("Group selection")).toBeTruthy());
  });

  it("should render a bottom divider only when showBottomDivider is set", async () => {
    const { container, unmount } = await renderItems(
      <Group.ContextMenuItem
        ids={[childID]}
        rootID={ontology.ROOT_ID}
        shape={shape}
        showBottomDivider
      />,
    );
    await waitFor(() => expect(screen.getByText("Group selection")).toBeTruthy());
    expect(container.querySelector(".pluto-menu__divider")).not.toBeNull();
    unmount();
    const { container: bare } = await renderItems(
      <Group.ContextMenuItem ids={[childID]} rootID={ontology.ROOT_ID} shape={shape} />,
    );
    await waitFor(() => expect(screen.getByText("Group selection")).toBeTruthy());
    expect(bare.querySelector(".pluto-menu__divider")).toBeNull();
  });
});

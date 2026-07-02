// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, group, ontology, ranger } from "@synnaxlabs/client";
import { Tree } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Group } from "@/platform/group";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const TIMEOUT = { timeout: 5000 };

const parentID = group.ontologyID(id.create());
const childID = ranger.ontologyID(id.create());
const parentKey = ontology.idToString(parentID);
const childKey = ontology.idToString(childID);

const shape: Tree.Shape = Tree.flatten({
  nodes: [{ key: parentKey, children: [{ key: childKey }] }],
  expanded: [parentKey],
});

const nestedRootID = group.ontologyID(id.create());

describe("Group.ContextMenuItem", () => {
  it("should render the group item for a nested selection under the root", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <Group.ContextMenuItem ids={[childID]} rootID={ontology.ROOT_ID} shape={shape} />,
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getByText("Group selection")).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("should render the group item for a top-level selection when the root is not the ontology root", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <Group.ContextMenuItem ids={[parentID]} rootID={nestedRootID} shape={shape} />,
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getByText("Group selection")).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("should hide the group item for a top-level selection at the ontology root", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    render(
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
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getAllByText("Group selection").length).toBe(1),
      TIMEOUT,
    );
  });

  it("should hide the group item when the selection is absent from the tree", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    const absentID = ranger.ontologyID(id.create());
    render(
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
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getAllByText("Group selection").length).toBe(1),
      TIMEOUT,
    );
  });

  it("should group a multi-item selection sharing the same minimum depth", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    const siblingID = ranger.ontologyID(id.create());
    const siblingKey = ontology.idToString(siblingID);
    const multiShape: Tree.Shape = Tree.flatten({
      nodes: [{ key: parentKey, children: [{ key: childKey }, { key: siblingKey }] }],
      expanded: [parentKey],
    });
    render(
      <Group.ContextMenuItem
        ids={[childID, siblingID]}
        rootID={ontology.ROOT_ID}
        shape={multiShape}
      />,
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getByText("Group selection")).toBeTruthy(),
      TIMEOUT,
    );
  });

  it("should render a bottom divider when showBottomDivider is set", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    const { container } = render(
      <Group.ContextMenuItem
        ids={[childID]}
        rootID={ontology.ROOT_ID}
        shape={shape}
        showBottomDivider
      />,
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getByText("Group selection")).toBeTruthy(),
      TIMEOUT,
    );
    expect(container.querySelector(".pluto-menu__divider")).not.toBeNull();
  });

  it("should not render a bottom divider by default", async () => {
    const { wrapper } = await createConsoleWrapper({ client });
    const { container } = render(
      <Group.ContextMenuItem ids={[childID]} rootID={ontology.ROOT_ID} shape={shape} />,
      { wrapper },
    );
    await waitFor(
      () => expect(screen.getByText("Group selection")).toBeTruthy(),
      TIMEOUT,
    );
    expect(container.querySelector(".pluto-menu__divider")).toBeNull();
  });
});

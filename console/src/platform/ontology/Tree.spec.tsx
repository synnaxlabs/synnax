// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DataType, group, ontology } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Ontology } from "@/platform/ontology";
import { buildServices } from "@/platform/ontology/testutil";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const TIMEOUT = { timeout: 5000 };

const renderTree = async (
  root: ontology.ID | null,
  services: Ontology.Services = buildServices(),
) => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Ontology.ServicesProvider services={services}>
        {children}
      </Ontology.ServicesProvider>
    </Console>
  );
  return render(<Ontology.Tree root={root} />, { wrapper: Wrapper });
};

describe("Ontology.Tree", () => {
  it("should render nothing when the root is null", async () => {
    const { container } = await renderTree(null);
    expect(container.innerHTML).toBe("");
  });

  it("should render the children of the root resource", async () => {
    const name = id.create();
    await client.groups.create({ parent: ontology.ROOT_ID, name });
    await renderTree(ontology.ROOT_ID);
    await waitFor(() => expect(screen.getByText(name)).toBeTruthy(), TIMEOUT);
  });

  it("should load a group's children when it is expanded", async () => {
    const groupName = id.create();
    const created = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: groupName,
    });
    const childName = id.create();
    const child = await client.groups.create({
      parent: group.ontologyID(created.key),
      name: childName,
    });
    await renderTree(ontology.ROOT_ID);
    const parentItem = await screen.findByText(groupName, undefined, TIMEOUT);
    const row = parentItem.closest(".pluto-tree__item");
    const caret = row?.querySelector(".pluto-tree__expansion-indicator");
    expect(caret).not.toBeNull();
    fireEvent.click(caret as Element);
    await waitFor(() => expect(screen.getByText(childName)).toBeTruthy(), TIMEOUT);
    await client.groups.delete(child.key);
  });

  describe("context menu", () => {
    it("should render the default context menu when right-clicking empty space", async () => {
      const name = id.create();
      await client.groups.create({ parent: ontology.ROOT_ID, name });
      const { container } = await renderTree(ontology.ROOT_ID);
      await screen.findByText(name, undefined, TIMEOUT);
      const tree = container.querySelector(".pluto-tree");
      expect(tree).not.toBeNull();
      fireEvent.contextMenu(tree as Element);
      await waitFor(() => expect(screen.getByText("Reload Console")).toBeTruthy());
      await waitFor(() => expect(screen.getByText("New group")).toBeTruthy(), TIMEOUT);
    });
  });

  it("should render resources under a non-root group", async () => {
    const idxName = id.create();
    await client.channels.create({
      name: idxName,
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const rootChildren = await client.ontology.retrieveChildren(ontology.ROOT_ID);
    const channelsGroup = rootChildren.find((r) => r.name === "Channels");
    expect(channelsGroup).toBeDefined();
    await renderTree(channelsGroup!.id);
    await waitFor(() => expect(screen.getByText(idxName)).toBeTruthy(), TIMEOUT);
  });

  it("should invoke the service's onSelect handler when an item is double-clicked", async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: id.create(),
    });
    const childName = id.create();
    await client.groups.create({
      parent: group.ontologyID(parent.key),
      name: childName,
    });
    const onSelect = vi.fn();
    const services = buildServices({
      group: { ...Ontology.NOOP_SERVICE, type: "group", onSelect },
    });
    await renderTree(group.ontologyID(parent.key), services);
    const item = await screen.findByText(childName, undefined, TIMEOUT);
    fireEvent.doubleClick(item);
    await waitFor(() => expect(onSelect).toHaveBeenCalled(), TIMEOUT);
    const [args] = onSelect.mock.calls[0];
    expect(args.selection).toHaveLength(1);
    expect(args.selection[0].name).toBe(childName);
  });

  it("should add a node when a child is created under the root after render", async () => {
    const parent = await client.groups.create({
      parent: ontology.ROOT_ID,
      name: id.create(),
    });
    const parentID = group.ontologyID(parent.key);
    await renderTree(parentID);
    const childName = id.create();
    await client.groups.create({ parent: parentID, name: childName });
    await waitFor(() => expect(screen.getByText(childName)).toBeTruthy(), TIMEOUT);
  });

  it("should remove a node when its resource is deleted after render", async () => {
    const name = id.create();
    const created = await client.groups.create({
      parent: ontology.ROOT_ID,
      name,
    });
    await renderTree(ontology.ROOT_ID);
    await screen.findByText(name, undefined, TIMEOUT);
    await client.groups.delete(created.key);
    await waitFor(() => expect(screen.queryByText(name)).toBeNull(), TIMEOUT);
  });
});

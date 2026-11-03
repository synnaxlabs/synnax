// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc as clientArc, type ontology } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Explorer } from "@/feature/arc/explorer/Explorer";
import { retrieveRootDirectoryID } from "@/platform/arc/testutil";
import {
  expandTreeRow,
  findTreeRow,
  findTreeRowIcon,
  openTreeRootContextMenu,
  openTreeRowContextMenu,
  renderToolbar,
} from "@/platform/tree/testutil";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

const createArc = async () =>
  await client.arcs.create({
    name: uniqueName("arc"),
    mode: "graph",
    type: "automation",
    graph: { nodes: [], edges: [] },
  });

const createModule = async () =>
  await client.arcs.create({ name: uniqueName("mod"), mode: "text", type: "module" });

const createDirectory = async (parent: ontology.ID) =>
  await client.arcs.directories.create({ parent: parent.key, name: uniqueName("dir") });

const renderExplorer = async (): Promise<void> => {
  await renderToolbar(<Explorer />, { client, items: Arc.TREE_ITEMS });
};

describe("arc/Explorer", () => {
  it("shows an Arc the Core placed in the Arcs directory", async () => {
    const arc = await createArc();
    await renderExplorer();
    expect(await findTreeRow(arc.name)).toBeTruthy();
  });

  it("creates a directory in the Arcs directory from the root context menu", async () => {
    const arc = await createArc();
    const arcsID = await retrieveRootDirectoryID(client);
    await renderExplorer();
    await findTreeRow(arc.name);
    await openTreeRootContextMenu();
    fireEvent.click(await screen.findByText("New directory"));
    const name = uniqueName("dir");
    commitTextEdit(await awaitTextEditingElement(), name);
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: arcsID });
      expect(children.some((r) => r.name === name)).toBe(true);
    });
  });

  it("draws a module with the module icon", async () => {
    const module = await createModule();
    const automation = await createArc();
    await renderExplorer();
    expect(await findTreeRowIcon(module.name, "module")).not.toBeNull();
    expect(await findTreeRowIcon(automation.name, "arc")).not.toBeNull();
  });

  it("shows an Arc inside nested directories once each parent is expanded", async () => {
    const arcsID = await retrieveRootDirectoryID(client);
    const dir = await createDirectory(arcsID);
    const sub = await createDirectory(clientArc.directory.ontologyID(dir.key));
    const module = await createModule();
    await client.arcs.directories.move(
      arcsID,
      clientArc.directory.ontologyID(sub.key),
      clientArc.ontologyID(module.key),
    );
    await renderExplorer();
    await findTreeRow(dir.name);
    expandTreeRow(dir.name);
    await findTreeRow(sub.name);
    expandTreeRow(sub.name);
    expect(await findTreeRow(module.name)).toBeTruthy();
  });

  it("creates a directory inside a directory from its context menu", async () => {
    const dir = await createDirectory(await retrieveRootDirectoryID(client));
    await renderExplorer();
    await openTreeRowContextMenu(dir.name);
    fireEvent.click(await screen.findByText("New directory"));
    const name = uniqueName("sub");
    commitTextEdit(await awaitTextEditingElement(), name);
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({
        ids: clientArc.directory.ontologyID(dir.key),
      });
      expect(children.some((r) => r.name === name)).toBe(true);
    });
  });

  it("opens the create modal from Create arc, listed above New directory", async () => {
    const arc = await createArc();
    await renderExplorer();
    await findTreeRow(arc.name);
    await openTreeRootContextMenu();
    const createArcItem = await screen.findByText("Create arc");
    const newDirectoryItem = await screen.findByText("New directory");
    expect(
      createArcItem.compareDocumentPosition(newDirectoryItem) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    await act(async () => {
      fireEvent.click(createArcItem);
    });
    expect(await screen.findByPlaceholderText("Arc name")).toBeTruthy();
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, log, project } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Log } from "@/feature/log";
import { findModalButton, renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import { assertDefined, renderHookWithConsole, uniqueName } from "@/testutil";

const client = createTestClient();

const createLog = async () => {
  const proj = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  return await client.logs.create(proj.key, { name: uniqueName("log") });
};

const logResource = (key: string, name: string) =>
  createResource(log.ontologyID(key), name);

describe("log ontology service", () => {
  it("should expose rename, group, delete, export, and link actions", async () => {
    const l = await createLog();
    assertDefined(Log.TREE_ITEM.ContextMenu);
    await renderTreeContextMenu(Log.TREE_ITEM.ContextMenu, {
      client,
      resources: [logResource(l.key, l.name)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Export")).toBeTruthy();
    expect(screen.getByText("Copy link")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should delete the log, its layout, and its session state", async () => {
    const l = await createLog();
    const removeLayout = vi.fn();
    assertDefined(Log.TREE_ITEM.ContextMenu);
    await renderTreeContextMenu(Log.TREE_ITEM.ContextMenu, {
      client,
      resources: [logResource(l.key, l.name)],
      baseOverrides: { removeLayout },
    });
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${l.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(() => expect(removeLayout).toHaveBeenCalledWith(l.key));
    const logExists = async (): Promise<boolean> => {
      try {
        await client.logs.retrieve({ key: l.key });
        return true;
      } catch {
        return false;
      }
    };
    await waitFor(async () => expect(await logExists()).toBe(false));
  });

  it("should place a log layout when the resource is double-clicked", async () => {
    const proj = await client.projects.create({
      name: uniqueName("project"),
      layout: {},
    });
    const l = await client.logs.create(proj.key, { name: uniqueName("log") });
    const { store } = await renderOntologyTree({
      client,
      root: project.ontologyID(proj.key),
      items: { log: Log.TREE_ITEM },
    });
    fireEvent.doubleClick(await findTreeRow(l.name));
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), l.key)?.name).toBe(l.name),
    );
  });

  it("should place a log layout in the target mosaic node on drop", async () => {
    const l = await createLog();
    const { result, store } = await renderHookWithConsole(() => Log.useMosaicDrop(), {
      client,
    });
    result.current({ id: log.ontologyID(l.key), nodeKey: 3, location: "top" });
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), l.key);
      expect(placed?.name).toBe(l.name);
      expect(placed?.type).toBe("log");
      expect(placed?.tab).toMatchObject({ mosaicKey: 3, location: "top" });
    });
  });

  it("should haul a mosaic tab creation item", () => {
    const res = logResource("abc", "l");
    const items = Log.TREE_ITEM.haulItems(res);
    expect(items).toHaveLength(1);
  });
});

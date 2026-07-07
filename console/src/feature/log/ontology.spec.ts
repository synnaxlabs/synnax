// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, log } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Log } from "@/feature/log";
import {
  findModalButton,
  renderTreeContextMenu,
} from "@/platform/ontology/menuTestutil";
import {
  createBaseProps,
  createExecutingHandleError,
  createResource,
} from "@/platform/ontology/testutil";
import { assertDefined, createTestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createLog = async () => {
  const project = await client.projects.create({
    name: uniqueName("project"),
    layout: {},
  });
  return await client.logs.create(project.key, { name: uniqueName("log") });
};

const logResource = (key: string, name: string) =>
  createResource(log.ontologyID(key), name);

describe("log ontology service", () => {
  it("should expose rename, group, delete, export, and link actions", async () => {
    const l = await createLog();
    assertDefined(Log.ONTOLOGY_SERVICE.TreeContextMenu);
    await renderTreeContextMenu(Log.ONTOLOGY_SERVICE.TreeContextMenu, {
      client,
      resources: [logResource(l.key, l.name)],
    });
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
    expect(screen.getByText("Export")).toBeTruthy();
    expect(screen.getByText("Copy link")).toBeTruthy();
    expect(screen.getByText("Copy properties")).toBeTruthy();
  });

  it("should delete the log from the cluster after confirmation", async () => {
    const l = await createLog();
    assertDefined(Log.ONTOLOGY_SERVICE.TreeContextMenu);
    await renderTreeContextMenu(Log.ONTOLOGY_SERVICE.TreeContextMenu, {
      client,
      resources: [logResource(l.key, l.name)],
    });
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${l.name}?`);
    fireEvent.click(findModalButton("Delete"));
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

  it("should open the log as a tab when the resource is selected", async () => {
    const l = await createLog();
    const store = await createTestStore();
    const openTab = vi.fn();
    const id = log.ontologyID(l.key);
    assertDefined(Log.ONTOLOGY_SERVICE.onSelect);
    Log.ONTOLOGY_SERVICE.onSelect({
      ...createBaseProps({
        client,
        store,
        overrides: { openTab, handleError: createExecutingHandleError() },
      }),
      selection: [logResource(l.key, l.name)],
    });
    await waitFor(() => expect(openTab).toHaveBeenCalledTimes(1));
    expect(openTab).toHaveBeenCalledWith({ resource: id });
  });

  it("should haul a mosaic tab creation item", () => {
    const res = logResource("abc", "l");
    const items = Log.ONTOLOGY_SERVICE.haulItems(res);
    expect(items).toHaveLength(1);
  });
});

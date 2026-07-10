// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
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
import { type Session } from "@/session";
import { assertDefined, createTestStore, type TestStore, uniqueName } from "@/testutil";

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

// Placers accept either a layout object or a creator function; resolve the creator
// the same way usePlacer does.
const resolvePlaced = (
  placeLayout: ReturnType<typeof vi.fn>,
  store: TestStore,
): Session.Layout.BaseState => {
  const arg = placeLayout.mock.calls[0][0];
  if (typeof arg !== "function") return arg as Session.Layout.BaseState;
  return arg({ dispatch: store.dispatch, store, windowKey: "main" });
};

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

  it("should delete the log, its layout, and its session state", async () => {
    const l = await createLog();
    const removeLayout = vi.fn();
    assertDefined(Log.ONTOLOGY_SERVICE.TreeContextMenu);
    await renderTreeContextMenu(Log.ONTOLOGY_SERVICE.TreeContextMenu, {
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

  it("should place a log layout when the resource is selected", async () => {
    const l = await createLog();
    const store = await createTestStore();
    const placeLayout = vi.fn();
    assertDefined(Log.ONTOLOGY_SERVICE.onSelect);
    Log.ONTOLOGY_SERVICE.onSelect({
      ...createBaseProps({
        client,
        store,
        overrides: { placeLayout, handleError: createExecutingHandleError() },
      }),
      selection: [logResource(l.key, l.name)],
    });
    await waitFor(() => expect(placeLayout).toHaveBeenCalledTimes(1));
    const placed = resolvePlaced(placeLayout, store);
    expect(placed).toMatchObject({ key: l.key, name: l.name, type: "log" });
  });

  it("should place a log layout in the target mosaic node on drop", async () => {
    const l = await createLog();
    const store = await createTestStore();
    const placeLayout = vi.fn();
    assertDefined(Log.ONTOLOGY_SERVICE.onMosaicDrop);
    Log.ONTOLOGY_SERVICE.onMosaicDrop({
      ...createBaseProps({
        client,
        store,
        overrides: { placeLayout, handleError: createExecutingHandleError() },
      }),
      id: log.ontologyID(l.key),
      nodeKey: 3,
      location: "top",
    });
    await waitFor(() => expect(placeLayout).toHaveBeenCalledTimes(1));
    const placed = resolvePlaced(placeLayout, store);
    expect(placed).toMatchObject({
      key: l.key,
      name: l.name,
      type: "log",
      location: "mosaic",
      tab: { mosaicKey: 3, location: "top" },
    });
  });

  it("should haul a mosaic tab creation item", () => {
    const res = logResource("abc", "l");
    const items = Log.ONTOLOGY_SERVICE.haulItems(res);
    expect(items).toHaveLength(1);
  });
});

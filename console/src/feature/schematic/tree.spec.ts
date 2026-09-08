// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, project, ranger, schematic } from "@synnaxlabs/client";
import { RoleClients } from "@synnaxlabs/client/testutil";
import { Status } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  client,
  createSchematic,
  renderSchematicTree,
  testProjectKey,
} from "@/feature/schematic/testutil";
import { renderTreeContextMenu } from "@/platform/tree/menuTestutil";

const roles = new RoleClients(client);
import { findButton } from "@/platform/modals/testutil";
import { createTestRange } from "@/platform/range/testutil";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  assertDefined,
  awaitTextEditingElement,
  captureBrowserDownloads,
  commitTextEdit,
  createTestStore,
  renderHookWithConsole,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

const Item = Schematic.TREE_ITEMS.schematic;

describe("Schematic.TREE_ITEMS", () => {
  it("hauls a mosaic tab create item keyed by the ontology id", async () => {
    const s = await createSchematic();
    const id = schematic.ontologyID(s.key);
    const items = Item.haulItems(createResource(id, s.name));
    expect(items).toHaveLength(1);
    expect(items[0].key).toContain(s.key);
  });

  describe("onSelect", () => {
    it("retrieves the schematic and opens it as a tab when double-clicked", async () => {
      const s = await createSchematic();
      const { store } = await renderOntologyTree({
        client,
        root: project.ontologyID(await testProjectKey()),
        items: Schematic.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(s.name));
      const tab = await resolveFocusedTab(store, client);
      if (tab.variant !== "resource") throw new Error("expected a resource tab");
      expect(tab.resource.key).toBe(s.key);
    });
  });
});

describe("Schematic.useRangeSnapshot", () => {
  it("raises an error status when there is no active range", async () => {
    const s = await createSchematic();
    const store = await createTestStore();
    const { result } = await renderHookWithConsole(
      () => ({
        snapshot: Schematic.useRangeSnapshot(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    const id = schematic.ontologyID(s.key);
    result.current.snapshot({
      ...createBaseProps({ client, store }),
      selection: createSelection({ ids: [id] }),
      state: createState([createResource(id, s.name)]),
    });
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "error" && st.message === "Failed to snapshot schematics",
        ),
      ).toBe(true),
    );
  });

  it("snapshots the selected schematics under the active range", async () => {
    const s = await createSchematic();
    const rng = await createTestRange(client);
    const store = await createTestStore();
    const { result } = await renderHookWithConsole(
      () => ({
        snapshot: Schematic.useRangeSnapshot(),
        notifications: Status.useNotifications(),
      }),
      {
        client,
        preloadedState: {
          [Session.Range.SLICE_NAME]: {
            ...Session.Range.ZERO_SLICE_STATE,
            selected: rng.key,
            ranges: [
              ...Session.Range.ZERO_SLICE_STATE.ranges,
              {
                variant: "persisted" as const,
                key: rng.key,
              },
            ],
          },
        },
      },
    );
    const id = schematic.ontologyID(s.key);
    result.current.snapshot({
      ...createBaseProps({ client, store }),
      selection: createSelection({ ids: [id] }),
      state: createState([createResource(id, s.name)]),
    });
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({
        ids: ranger.ontologyID(rng.key),
      });
      expect(children.map((c) => c.name)).toContain(`${s.name} (Snapshot)`);
    });
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) => st.variant === "success" && st.message.includes("Snapshotted"),
        ),
      ).toBe(true),
    );
  });
});

describe("Schematic TreeContextMenu", () => {
  it("deletes the schematic from the Core and session state after confirmation", async () => {
    const { schematic: s, result } = await renderSchematicTree();
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${s.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => {
      await expect(client.schematics.retrieve(s.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
    result.unmount();
  });

  it("renames the schematic on the Core through the inline editor", async () => {
    const { schematic: s, result } = await renderSchematicTree();
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    const next = uniqueName("renamed");
    commitTextEdit(editable, next);
    await waitFor(async () => {
      const retrieved = await client.schematics.retrieve(s.key);
      expect(retrieved.name).toBe(next);
    });
    result.unmount();
  });

  it("copies the schematic with a (copy) suffix and begins an inline rename", async () => {
    const { schematic: s, rootID, result } = await renderSchematicTree();
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Copy"));
    let copyKey = "";
    await waitFor(async () => {
      const children = await client.ontology.children.retrieve({ ids: rootID });
      const copy = children.find((c) => c.name === `${s.name} (copy)`);
      if (copy == null) throw new Error("copy not created yet");
      copyKey = copy.id.key;
    });
    const editable = await awaitTextEditingElement();
    const renamed = uniqueName("copy_renamed");
    commitTextEdit(editable, renamed);
    await waitFor(async () => {
      const retrieved = await client.schematics.retrieve(copyKey);
      expect(retrieved.name).toBe(renamed);
    });
    result.unmount();
  });

  it("exports the schematic from the context menu", async () => {
    const downloads = captureBrowserDownloads();
    const { schematic: s, result } = await renderSchematicTree();
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Export"));
    await waitFor(() => expect(downloads.anchors).toHaveLength(1));
    expect(downloads.anchors[0].download).toBe(`${s.name}.json`);
    result.unmount();
  });
});

describe("permission to write the schematic", () => {
  it("should withhold rename, grouping, copying, and delete from a viewer", async () => {
    const s = await createSchematic();
    assertDefined(Item.ContextMenu);
    await renderTreeContextMenu(Item.ContextMenu, {
      client: await roles.get("Viewer"),
      resources: [
        createResource(schematic.ontologyID(s.key), s.name, { snapshot: false }),
      ],
    });
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Group selection")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });
});

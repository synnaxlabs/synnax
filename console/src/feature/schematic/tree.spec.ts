// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, project, ranger, schematic } from "@synnaxlabs/client";
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
import { findButton } from "@/platform/modals/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { createEntry } from "@/platform/tree/testutil";
import { findTreeRow, renderOntologyTree } from "@/platform/tree/treeTestutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  captureBrowserDownloads,
  commitTextEdit,
  createTestFluxStore,
  renderHookWithConsole,
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
    const items = Item.haulItems(createEntry(id, s.name), createTestFluxStore());
    expect(items).toHaveLength(1);
    expect(items[0].key).toContain(s.key);
  });

  describe("onSelect", () => {
    it("places a schematic layout when a tree row is double-clicked", async () => {
      const s = await createSchematic();
      const { store } = await renderOntologyTree({
        client,
        root: project.ontologyID(await testProjectKey()),
        items: Schematic.TREE_ITEMS,
      });
      fireEvent.doubleClick(await findTreeRow(s.name));
      await waitFor(() =>
        expect(Session.Layout.select(store.getState(), s.key)?.name).toBe(s.name),
      );
    });
  });
});

describe("Schematic.useRangeSnapshot", () => {
  it("raises an error status when there is no active range", async () => {
    const s = await createSchematic();
    const { result } = await renderHookWithConsole(
      () => ({
        snapshot: Schematic.useRangeSnapshot(),
        notifications: Status.useNotifications(),
      }),
      { client },
    );
    result.current.snapshot([{ key: s.key, name: s.name }]);
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "error" &&
            st.message === "Cannot snapshot schematics without an active range",
        ),
      ).toBe(true),
    );
  });

  it("snapshots the selected schematics under the active range", async () => {
    const s = await createSchematic();
    const rng = await createTestRange(client);
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
                key: rng.key,
                name: rng.name,
                variant: "static",
                persisted: true,
                timeRange: {
                  start: Number(rng.timeRange.start),
                  end: Number(rng.timeRange.end),
                },
              },
            ],
          },
        },
      },
    );
    result.current.snapshot([{ key: s.key, name: s.name }]);
    await waitFor(async () => {
      const children = await client.ontology.retrieveChildren(
        ranger.ontologyID(rng.key),
      );
      const schematics = await client.schematics.retrieve({
        keys: children.filter((c) => c.id.type === "schematic").map((c) => c.id.key),
      });
      expect(schematics.map((sc) => sc.name)).toContain(`${s.name} (Snapshot)`);
    });
    await waitFor(() =>
      expect(
        result.current.notifications.statuses.some(
          (st) =>
            st.variant === "success" && st.message.includes("Successfully snapshotted"),
        ),
      ).toBe(true),
    );
  });
});

describe("Schematic TreeContextMenu", () => {
  it("deletes the schematic from the cluster and session state after confirmation", async () => {
    const { schematic: s, result } = await renderSchematicTree();
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${s.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => {
      await expect(client.schematics.retrieve({ key: s.key })).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
    result.unmount();
  });

  it("renames the schematic on the cluster through the inline editor", async () => {
    const { schematic: s, result } = await renderSchematicTree();
    fireEvent.contextMenu(await screen.findByText(s.name));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    const next = uniqueName("renamed");
    commitTextEdit(editable, next);
    await waitFor(async () => {
      const retrieved = await client.schematics.retrieve({ key: s.key });
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
      const children = await client.ontology.retrieveChildren(rootID);
      const schematics = await client.schematics.retrieve({
        keys: children.filter((c) => c.id.type === "schematic").map((c) => c.id.key),
      });
      const copy = schematics.find((sc) => sc.name === `${s.name} (copy)`);
      if (copy == null) throw new Error("copy not created yet");
      copyKey = copy.key;
    });
    const editable = await awaitTextEditingElement();
    const renamed = uniqueName("copy_renamed");
    commitTextEdit(editable, renamed);
    await waitFor(async () => {
      const retrieved = await client.schematics.retrieve({ key: copyKey });
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

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NotFoundError, ranger, schematic } from "@synnaxlabs/client";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Status } from "@synnaxlabs/pluto";
import { errors, uuid } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Schematic } from "@/feature/schematic";
import {
  client,
  createSchematic,
  renderSchematicTree,
} from "@/feature/schematic/testutil";
import { type Layout } from "@/platform/layout";
import { findButton } from "@/platform/modals/testutil";
import {
  createBaseProps,
  createResource,
  createSelection,
  createState,
} from "@/platform/ontology/testutil";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import {
  awaitTextEditingElement,
  captureBrowserDownloads,
  commitTextEdit,
  createTestStore,
  renderHookWithConsole,
  type TestStore,
  uniqueName,
} from "@/testutil";

afterEach(() => {
  vi.restoreAllMocks();
});

interface RecordingPlacer {
  placeLayout: Layout.Placer;
  placed: Session.Layout.BaseState[];
}

const createRecordingPlacer = (store: TestStore): RecordingPlacer => {
  const placed: Session.Layout.BaseState[] = [];
  const placeLayout: Layout.Placer = (base) => {
    const layout =
      typeof base === "function"
        ? base({ dispatch: store.dispatch, store, windowKey: MAIN_WINDOW })
        : base;
    placed.push(layout);
    return { windowKey: MAIN_WINDOW, key: layout.key ?? "" };
  };
  return { placeLayout, placed };
};

const schematicExists = async (key: schematic.Key): Promise<boolean> => {
  try {
    await client.schematics.retrieve({ key });
    return true;
  } catch (e) {
    if (NotFoundError.matches(e)) return false;
    throw errors.fromUnknown(e);
  }
};

describe("Schematic.ONTOLOGY_SERVICE", () => {
  it("hauls a mosaic tab create item keyed by the ontology id", async () => {
    const s = await createSchematic();
    const id = schematic.ontologyID(s.key);
    const items = Schematic.ONTOLOGY_SERVICE.haulItems(createResource(id, s.name));
    expect(items).toHaveLength(1);
    expect(items[0].key).toContain(s.key);
  });

  describe("onSelect", () => {
    it("places a schematic layout for the selected resource", async () => {
      const s = await createSchematic();
      const store = await createTestStore();
      const { placeLayout, placed } = createRecordingPlacer(store);
      Schematic.ONTOLOGY_SERVICE.onSelect?.({
        ...createBaseProps({ client, store, overrides: { placeLayout } }),
        selection: [createResource(schematic.ontologyID(s.key), s.name)],
      });
      await waitFor(() => expect(placed).toHaveLength(1));
      expect(placed[0]).toMatchObject({
        key: s.key,
        name: s.name,
        type: "schematic",
      });
      expect(
        Session.Schematic.selectSliceState(store.getState()).schematics[s.key],
      ).toBeDefined();
    });

    it("routes retrieval failures to the error handler with the schematic name", async () => {
      const store = await createTestStore();
      const handleError = vi.fn();
      const placeLayout = vi.fn(() => ({ windowKey: "", key: "" }));
      Schematic.ONTOLOGY_SERVICE.onSelect?.({
        ...createBaseProps({
          client,
          store,
          overrides: { placeLayout, handleError },
        }),
        selection: [
          createResource(schematic.ontologyID(uuid.create()), "Ghost Schematic"),
        ],
      });
      await waitFor(() => expect(handleError).toHaveBeenCalledTimes(1));
      expect(handleError.mock.calls[0][1]).toBe("Failed to select Ghost Schematic");
      expect(placeLayout).not.toHaveBeenCalled();
    });
  });

  describe("onMosaicDrop", () => {
    it("places the schematic into the target mosaic node", async () => {
      const s = await createSchematic();
      const store = await createTestStore();
      const { placeLayout, placed } = createRecordingPlacer(store);
      const handleError: Status.ErrorHandler = (excOrFunc) => {
        if (typeof excOrFunc === "function") void excOrFunc();
      };
      Schematic.ONTOLOGY_SERVICE.onMosaicDrop?.({
        ...createBaseProps({
          client,
          store,
          overrides: { placeLayout, handleError },
        }),
        id: schematic.ontologyID(s.key),
        nodeKey: 3,
        location: "center",
      });
      await waitFor(() => expect(placed).toHaveLength(1));
      expect(placed[0]).toMatchObject({
        key: s.key,
        name: s.name,
        tab: { mosaicKey: 3, location: "center" },
      });
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
            st.variant === "error" &&
            st.message === "Cannot snapshot schematics without an active range",
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
    const id = schematic.ontologyID(s.key);
    result.current.snapshot({
      ...createBaseProps({ client, store }),
      selection: createSelection({ ids: [id] }),
      state: createState([createResource(id, s.name)]),
    });
    await waitFor(async () => {
      const children = await client.ontology.retrieveChildren(
        ranger.ontologyID(rng.key),
      );
      expect(children.map((c) => c.name)).toContain(`${s.name} (Snapshot)`);
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
    await waitFor(async () => expect(await schematicExists(s.key)).toBe(false));
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
      const copy = children.find((c) => c.name === `${s.name} (copy)`);
      if (copy == null) throw new Error("copy not created yet");
      copyKey = copy.id.key;
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

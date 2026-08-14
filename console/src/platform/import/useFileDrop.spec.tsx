// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log as clientLog, panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Import } from "@/platform/import";
import {
  createJSONFile,
  fakeDirectoryEntry,
  fakeFileDropEvent,
  fakeFileEntry,
} from "@/platform/import/testutil";
import {
  createPanelWrapper,
  createServerPanel,
  primePanel,
} from "@/platform/panel/testutil";
import { Session } from "@/session";
import { CaptureStatuses } from "@/testutil";

const client = createTestClient();

const leaf = (node: panel.Node): panel.NodeLeaf => {
  if (node.variant !== "leaf") throw new Error("expected a leaf");
  return node;
};

const resourceKeys = ({ tabs }: panel.NodeLeaf): string[] =>
  tabs.map((tab) => {
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    return tab.resource.key;
  });

interface RenderParams {
  fileIngesters?: Import.FileIngesters;
  ingestDirectory?: Import.DirectoryIngester;
  panelKey?: panel.Key;
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

const renderFileDrop = async ({
  fileIngesters = {},
  ingestDirectory = vi.fn(),
  panelKey,
  onStatuses,
}: RenderParams = {}) => {
  const { wrapper: Console, store } = await createPanelWrapper({ client, panelKey });
  if (panelKey != null) await primePanel(Console, panelKey);
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Import.FileIngestersProvider fileIngesters={fileIngesters}>
        {children}
        {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
      </Import.FileIngestersProvider>
    </Console>
  );
  return {
    ...renderHook(() => Import.useFileDrop({ ingestDirectory }), { wrapper }),
    store,
  };
};

describe("Import.useFileDrop", () => {
  it("hands a dropped JSON file to the ingester registered for its type", async () => {
    const log = vi.fn();
    const { result } = await renderFileDrop({ fileIngesters: { log } });
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeFileEntry(createJSONFile("widget.json", { type: "log", key: "abc" })),
        ]),
      }),
    );
    await waitFor(() => expect(log).toHaveBeenCalledTimes(1));
    expect(log.mock.calls[0][0]).toEqual({ type: "log", key: "abc" });
  });

  it("hands a dropped directory's parsed files to the directory ingester", async () => {
    const ingestDirectory = vi.fn();
    const { result } = await renderFileDrop({ ingestDirectory });
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeDirectoryEntry("my-directory", [
            createJSONFile("a.json", { type: "log" }),
          ]),
        ]),
      }),
    );
    await waitFor(() => expect(ingestDirectory).toHaveBeenCalledTimes(1));
    expect(ingestDirectory.mock.calls[0][0]).toBe("my-directory");
    expect(ingestDirectory.mock.calls[0][1]).toEqual([
      { name: "a.json", data: { type: "log" } },
    ]);
  });

  it("reports a non-JSON file and still imports the rest of the drop", async () => {
    const log = vi.fn();
    let statuses: Status.NotificationSpec[] = [];
    const { result } = await renderFileDrop({
      fileIngesters: { log },
      onStatuses: (s) => (statuses = s),
    });
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeFileEntry(new File(["hello"], "notes.txt", { type: "text/plain" })),
          fakeFileEntry(createJSONFile("widget.json", { type: "log" })),
        ]),
      }),
    );
    await waitFor(() => expect(log).toHaveBeenCalledTimes(1));
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to import notes.txt"),
    );
  });

  // A dropped project selects itself once imported, and the drop's other files import
  // alongside it. A file that finishes reading after the switch still belongs to the
  // project that was open when the drop landed.
  it("imports every file into the project open when the drop landed", async () => {
    let releaseFile = (): void => {};
    const held = new Promise<void>((resolve) => {
      releaseFile = resolve;
    });
    const imported = uuid.create();
    const log = vi.fn();
    const ingestDirectory = vi.fn<Import.DirectoryIngester>(
      async (_name, _files, { store }) => {
        store.dispatch(Session.Project.select(imported));
      },
    );
    const { result, store } = await renderFileDrop({
      fileIngesters: { log },
      ingestDirectory,
    });
    const open = Session.Project.selectSelected(store.getState());
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeDirectoryEntry("my-project", [createJSONFile("panels.json", [])]),
          fakeFileEntry(createJSONFile("widget.json", { type: "log" }), held),
        ]),
      }),
    );
    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(imported),
    );
    releaseFile();
    await waitFor(() => expect(log).toHaveBeenCalledTimes(1));
    expect(log.mock.calls[0][1]).toMatchObject({ projectKey: open });
  });

  it("ignores items the drag carries that are not files", async () => {
    const log = vi.fn();
    let statuses: Status.NotificationSpec[] = [];
    const { result } = await renderFileDrop({
      fileIngesters: { log },
      onStatuses: (s) => (statuses = s),
    });
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([null]),
      }),
    );
    await act(async () => {});
    expect(log).not.toHaveBeenCalled();
    expect(statuses).toHaveLength(0);
  });

  // The no-panel state reports a drop with no leaf. Placing it would resolve
  // against a mosaic that does not exist, so the tabs open in a created panel.
  it("opens the tabs in a new panel when the drop carries no leaf", async () => {
    const key = uuid.create();
    const log = vi.fn(async () => clientLog.ontologyID(key));
    const { result, store } = await renderFileDrop({ fileIngesters: { log } });
    act(() =>
      result.current({
        event: fakeFileDropEvent([
          fakeFileEntry(createJSONFile("a.json", { type: "log" })),
        ]),
      }),
    );
    await waitFor(() => expect(log).toHaveBeenCalledTimes(1));
    const panelKey = await waitFor(() => {
      const selected = Session.Panel.selectSelected(store.getState());
      if (selected == null) throw new Error("no panel created for the drop");
      return selected;
    });
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(panelKey);
      expect(resourceKeys(leaf(root))).toEqual([key]);
    });
  });

  it("splits the leaf once and fills the new half with every dropped file", async () => {
    const seedTabKey = uuid.create();
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: seedTabKey, type: "seed", args: {} }],
    });
    const [first, second] = [uuid.create(), uuid.create()];
    const log = vi.fn(async (data: unknown) =>
      clientLog.ontologyID((data as { key: string }).key),
    );
    const { result } = await renderFileDrop({
      fileIngesters: { log },
      panelKey: existing.key,
    });
    // A right-edge drop puts the new half last, where an unplaced fallback would not
    // reach: it resolves to the first leaf, which still holds the seed.
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "right",
        event: fakeFileDropEvent([
          fakeFileEntry(createJSONFile("a.json", { type: "log", key: first })),
          fakeFileEntry(createJSONFile("b.json", { type: "log", key: second })),
        ]),
      }),
    );
    await waitFor(() => expect(log).toHaveBeenCalledTimes(2));
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(existing.key);
      if (root.variant !== "split") throw new Error("root did not split");
      expect(resourceKeys(leaf(root.last))).toEqual([first, second]);
      expect(leaf(root.first).tabs.map((t) => t.key)).toEqual([seedTabKey]);
    });
  });
});

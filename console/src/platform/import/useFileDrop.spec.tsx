// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, panel, project } from "@synnaxlabs/client";
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

// A legacy Console log state: version-stamped, no type, no name. The Core recognizes it
// by its frozen channels-array marker and names the log after the file it arrived in.
const LEGACY_LOG_STATE = {
  version: "0.0.0",
  channels: [1, 2, 3],
  remoteCreated: false,
};

const leaf = (node: panel.Node): panel.NodeLeaf => {
  if (node.variant !== "leaf") throw new Error("expected a leaf");
  return node;
};

const resourceIDs = ({ tabs }: panel.NodeLeaf): ontology.ID[] =>
  tabs.map((tab) => {
    if (tab.variant !== "resource") throw new Error("expected a resource tab");
    return tab.resource;
  });

const logNames = async (ids: ontology.ID[]): Promise<string[]> =>
  await Promise.all(
    ids.map(async (id) => (await client.logs.retrieve({ key: id.key })).name),
  );

interface RenderParams {
  ingestDirectory?: Import.DirectoryIngester;
  panelKey?: panel.Key;
  onStatuses?: (statuses: Status.NotificationSpec[]) => void;
}

const renderFileDrop = async ({
  ingestDirectory = vi.fn(),
  panelKey,
  onStatuses,
}: RenderParams = {}) => {
  const { wrapper: Console, store } = await createPanelWrapper({ client, panelKey });
  if (panelKey != null) await primePanel(Console, panelKey);
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      {children}
      {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
    </Console>
  );
  return {
    ...renderHook(() => Import.useFileDrop({ ingestDirectory }), { wrapper }),
    store,
  };
};

describe("Import.useFileDrop", () => {
  it("streams a dropped JSON file to the Core and opens the resource it created", async () => {
    const { result, store } = await renderFileDrop({ panelKey: undefined });
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeFileEntry(createJSONFile("Dropped Log.json", LEGACY_LOG_STATE)),
        ]),
      }),
    );
    const panelKey = await waitFor(() => {
      const selected = Session.Panel.selectSelected(store.getState());
      if (selected == null) throw new Error("no panel created for the drop");
      return selected;
    });
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(panelKey);
      expect(await logNames(resourceIDs(leaf(root)))).toEqual(["Dropped Log"]);
    });
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
    let statuses: Status.NotificationSpec[] = [];
    const { result, store } = await renderFileDrop({
      onStatuses: (s) => (statuses = s),
    });
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeFileEntry(new File(["hello"], "notes.txt", { type: "text/plain" })),
          fakeFileEntry(createJSONFile("Survivor.json", LEGACY_LOG_STATE)),
        ]),
      }),
    );
    await waitFor(() =>
      expect(statuses.map((s) => s.message)).toContain("Failed to import notes.txt"),
    );
    const panelKey = await waitFor(() => {
      const selected = Session.Panel.selectSelected(store.getState());
      if (selected == null) throw new Error("no panel created for the drop");
      return selected;
    });
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(panelKey);
      expect(await logNames(resourceIDs(leaf(root)))).toEqual(["Survivor"]);
    });
  });

  // A dropped project selects itself once imported, and the drop's other files import
  // alongside it. A file that finishes reading after the switch still belongs to the
  // project that was open when the drop landed.
  it("imports every file into the project open when the drop landed", async () => {
    let releaseFile = (): void => {};
    const held = new Promise<void>((resolve) => {
      releaseFile = resolve;
    });
    const switched = await client.projects.create({
      name: `switched-${uuid.create()}`,
      layout: {},
    });
    const ingestDirectory = vi.fn<Import.DirectoryIngester>(
      async (_name, _files, { store }) => {
        store.dispatch(Session.Project.select(switched.key));
      },
    );
    const { result, store } = await renderFileDrop({ ingestDirectory });
    const open = Session.Project.selectSelected(store.getState());
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "center",
        event: fakeFileDropEvent([
          fakeDirectoryEntry("my-project", [createJSONFile("panels.json", [])]),
          fakeFileEntry(createJSONFile("Pinned.json", LEGACY_LOG_STATE), held),
        ]),
      }),
    );
    await waitFor(() =>
      expect(Session.Project.selectSelected(store.getState())).toBe(switched.key),
    );
    releaseFile();
    const children = await waitFor(async () => {
      const resources = await client.ontology.children.retrieve({
        ids: project.ontologyID(open),
      });
      const pinned = resources.filter((r) => r.name === "Pinned");
      if (pinned.length === 0) throw new Error("the held file did not import");
      return pinned;
    });
    expect(children).toHaveLength(1);
  });

  it("ignores items the drag carries that are not files", async () => {
    let statuses: Status.NotificationSpec[] = [];
    const { result, store } = await renderFileDrop({
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
    expect(statuses).toHaveLength(0);
    expect(Session.Panel.selectSelected(store.getState())).toBeUndefined();
  });

  it("splits the leaf once and fills the new half with every dropped file", async () => {
    const seedTabKey = uuid.create();
    const existing = await createServerPanel(client, {
      variant: "leaf",
      tabs: [{ variant: "view", key: seedTabKey, type: "seed", args: {} }],
    });
    const { result } = await renderFileDrop({ panelKey: existing.key });
    // A right-edge drop puts the new half last, where an unplaced fallback would not
    // reach: it resolves to the first leaf, which still holds the seed.
    act(() =>
      result.current({
        nodeKey: panel.ROOT_NODE_KEY,
        location: "right",
        event: fakeFileDropEvent([
          fakeFileEntry(createJSONFile("First.json", LEGACY_LOG_STATE)),
          fakeFileEntry(createJSONFile("Second.json", LEGACY_LOG_STATE)),
        ]),
      }),
    );
    await waitFor(async () => {
      const { root } = await client.panels.retrieve(existing.key);
      if (root.variant !== "split") throw new Error("root did not split");
      const names = await logNames(resourceIDs(leaf(root.last)));
      expect(names.toSorted()).toEqual(["First", "Second"]);
      expect(leaf(root.first).tabs.map((t) => t.key)).toEqual([seedTabKey]);
    });
  });
});

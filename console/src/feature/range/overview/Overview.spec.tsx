// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type ontology,
  panel,
  type ranger,
  ranger as rangerClient,
  schematic,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon, Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { Range as PlatformRange } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import {
  createConsoleWrapper,
  getIconButton,
  stubGeometry,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

stubGeometry();

interface RenderOverviewResult {
  onSnapshotClick: ReturnType<typeof vi.fn>;
  onSnapshotDelete: ReturnType<typeof vi.fn>;
  setTabResource: (rangeKey: string) => Promise<void>;
}

const renderOverview = async (rangeKey: string): Promise<RenderOverviewResult> => {
  const onSnapshotClick = vi.fn(async () => {});
  const onSnapshotDelete = vi.fn(async () => {});
  const services: PlatformRange.SnapshotServices = {
    schematic: {
      icon: <Icon.Schematic />,
      onClick: onSnapshotClick,
      onDelete: onSnapshotDelete,
    },
  };
  const { wrapper } = await createConsoleWrapper({ client });
  const tabKey = uuid.create();
  const doc = panel.panelZ.parse({
    key: uuid.create(),
    name: uniqueName("panel"),
    root: {
      variant: "leaf",
      tabs: [
        {
          variant: "resource",
          key: tabKey,
          resource: rangerClient.ontologyID(rangeKey),
        },
      ],
    },
  });
  await client.panels.create(doc);
  render(
    <PlutoPanel.Scope.Provider value={doc.key}>
      <PlutoPanel.TabScope.Provider value={tabKey}>
        <PlatformRange.SnapshotServicesProvider services={services}>
          <Range.Overview.Overview />
          <Modals.Stack />
        </PlatformRange.SnapshotServicesProvider>
      </PlutoPanel.TabScope.Provider>
    </PlutoPanel.Scope.Provider>,
    { wrapper },
  );
  const setTabResource = async (nextKey: string) =>
    await act(async () => {
      await client.panels.dispatch(doc.key, [
        panel.setTabResource({
          key: tabKey,
          resource: rangerClient.ontologyID(nextKey),
        }),
      ]);
    });
  return { onSnapshotClick, onSnapshotDelete, setTabResource };
};

const createChildRange = async (parent: ranger.Range): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName("child"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(1))),
    parent: { key: parent.key },
  });
};

const createSnapshot = async (rng: ranger.Range): Promise<ontology.ID> => {
  const project = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  const sch = await client.schematics.create(project.key, {
    name: uniqueName("sch"),
  });
  const snap = await client.schematics.copy({
    key: sch.key,
    name: uniqueName("snap"),
    snapshot: true,
  });
  const id = schematic.ontologyID(snap.key);
  await client.ontology.addChildren(rangerClient.ontologyID(rng.key), id);
  return id;
};

describe("range/overview/Overview", () => {
  it("renders the range details alongside the child range and snapshot sections", async () => {
    const rng = await createTestRange(client);
    await renderOverview(rng.key);
    expect(await screen.findByDisplayValue(rng.name)).toBeTruthy();
    expect(await screen.findByText("Child Ranges")).toBeTruthy();
    expect(await screen.findByText("Snapshots")).toBeTruthy();
  });

  it("lists the range's children", async () => {
    const rng = await createTestRange(client);
    const child = await createChildRange(rng);
    await renderOverview(rng.key);
    expect(await screen.findByText(child.name)).toBeTruthy();
  });

  it("lists the new range's children after the tab swaps resource", async () => {
    const parent = await createTestRange(client);
    const child = await createChildRange(parent);
    const { setTabResource } = await renderOverview(child.key);
    expect(await screen.findByDisplayValue(child.name)).toBeTruthy();
    await setTabResource(parent.key);
    expect(await screen.findByDisplayValue(parent.name)).toBeTruthy();
    expect(await screen.findByText(child.name)).toBeTruthy();
  });

  it("opens the create modal for a new child range", async () => {
    const rng = await createTestRange(client);
    await renderOverview(rng.key);
    await screen.findByText("Child Ranges");
    fireEvent.click(await waitFor(() => getIconButton(document.body, "add")));
    expect(await screen.findByText("Save Locally")).toBeTruthy();
  });

  it("lists snapshots and routes selection to the snapshot service", async () => {
    const rng = await createTestRange(client);
    const snapID = await createSnapshot(rng);
    const { onSnapshotClick } = await renderOverview(rng.key);
    const item = await screen.findByText((await client.ontology.retrieve(snapID)).name);
    fireEvent.click(item);
    await waitFor(() => expect(onSnapshotClick).toHaveBeenCalledTimes(1));
    const [resource] = onSnapshotClick.mock.calls[0] as [ontology.Resource];
    expect(resource.id.key).toBe(snapID.key);
  });

  it("deletes a snapshot through its service after confirmation", async () => {
    const rng = await createTestRange(client);
    const snapID = await createSnapshot(rng);
    const name = (await client.ontology.retrieve(snapID)).name;
    const { onSnapshotDelete } = await renderOverview(rng.key);
    await screen.findByText(name);
    fireEvent.click(await waitFor(() => getIconButton(document.body, "delete")));
    await screen.findByText(`Are you sure you want to delete ${name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(() => expect(onSnapshotDelete).toHaveBeenCalledTimes(1));
  });
});

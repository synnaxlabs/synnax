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
  query,
  type ranger,
  ranger as rangerClient,
  schematic,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Errors, Flux, Icon, Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ComponentType } from "react";
import { assert, describe, expect, it, vi } from "vitest";

import { Range } from "@/feature/range";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import { createResourceTab } from "@/platform/panel/testutil";
import { Range as PlatformRange } from "@/platform/range";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import { createConsoleWrapper, getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

interface RenderOverviewResult {
  onSnapshotClick: ReturnType<typeof vi.fn>;
  onSnapshotDelete: ReturnType<typeof vi.fn>;
  setTabResource: (rangeKey: string) => Promise<void>;
}

const renderOverview = async (
  rangeKey: string,
  FallbackComponent?: ComponentType<Errors.FallbackProps>,
): Promise<RenderOverviewResult> => {
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
  const { panelKey, tabKey } = await createResourceTab(
    client,
    rangerClient.ontologyID(rangeKey),
  );
  render(
    <PlutoPanel.Scope.Provider value={panelKey}>
      <PlutoPanel.TabScope.Provider value={tabKey}>
        <PlatformRange.SnapshotServicesProvider services={services}>
          <Errors.SuspenseBoundary FallbackComponent={FallbackComponent}>
            <Range.Overview.Overview />
          </Errors.SuspenseBoundary>
          <Modals.Stack />
        </PlatformRange.SnapshotServicesProvider>
      </PlutoPanel.TabScope.Provider>
    </PlutoPanel.Scope.Provider>,
    { wrapper },
  );
  const setTabResource = async (nextKey: string) =>
    await act(async () => {
      await client.panels.dispatch(panelKey, [
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
    expect(await screen.findByText("Save locally")).toBeTruthy();
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

describe("range/overview tab", () => {
  it("throws the deleted range to the tab's boundary while it is open", async () => {
    const rng = await createTestRange(client);
    const DeletedProbe = ({ error }: Errors.FallbackProps) => (
      <div>{`deleted-${Flux.DeletedError.matches(error) ? error.corpseName : ""}`}</div>
    );
    DeletedProbe.displayName = "DeletedProbe";
    await renderOverview(rng.key, DeletedProbe);
    expect(await screen.findByDisplayValue(rng.name)).toBeTruthy();
    await act(async () => {
      await client.ranges.delete(rng.key);
    });
    expect(await screen.findByText(`deleted-${rng.name}`)).toBeTruthy();
    // The whole tab tombstones. A section left behind would render its own
    // inline "failed to retrieve" state next to the tombstone.
    expect(screen.queryByText("Child Ranges")).toBeNull();
    expect(screen.queryByText("Snapshots")).toBeNull();
  });

  it("shows the restored range's details after retry", async () => {
    const { restore } = Range.Overview.TABS[rangerClient.TYPE_ONTOLOGY_ID.type];
    assert(restore != null);
    const rng = await createTestRange(client);
    const Probe = ({ resetErrorBoundary }: Errors.FallbackProps) => (
      <button data-testid="restore" onClick={resetErrorBoundary}>
        restore
      </button>
    );
    Probe.displayName = "Probe";
    await renderOverview(rng.key, Probe);
    expect(await screen.findByDisplayValue(rng.name)).toBeTruthy();
    await act(async () => {
      await client.ranges.delete(rng.key);
    });
    await screen.findByTestId("restore");

    const project = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    await act(async () => {
      await restore({ client, project: project.key, resource: rng.ontologyID });
    });
    fireEvent.click(screen.getByTestId("restore"));
    expect(await screen.findByDisplayValue(rng.name)).toBeTruthy();
  });

  it("restores a deleted range under its original key", async () => {
    const { restore } = Range.Overview.TABS[rangerClient.TYPE_ONTOLOGY_ID.type];
    assert(restore != null);
    const rng = await createTestRange(client);
    // Restore rebuilds from the corpse the cache holds, which exists only once
    // the client has seen the range live.
    await client.ranges.retrieve(rng.key);
    await client.ranges.delete(rng.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.ranges.getCached(rng.key))).toBe(true),
    );

    const project = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    await restore({ client, project: project.key, resource: rng.ontologyID });

    // The tab heals off the cache going live again, not off the create call.
    await waitFor(() =>
      expect(query.isLive(client.ranges.getCached(rng.key))).toBe(true),
    );
    // Read back through a second client: the restoring client write-throughs its
    // own cache, so retrieving on it would pass even if the cluster never got it.
    const remote = createTestClient();
    const [restored] = await remote.ranges.retrieve([rng.key]);
    expect(restored.name).toEqual(rng.name);
    expect(restored.timeRange).toEqual(rng.timeRange);
    // The tab chip and every tree view read the range through the ontology, not
    // the range store.
    const resource = await remote.ontology.retrieve(rng.ontologyID);
    expect(resource.name).toEqual(rng.name);
  });
});

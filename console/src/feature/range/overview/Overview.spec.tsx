// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  label,
  panel,
  project,
  query,
  type ranger,
  ranger as rangerClient,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Errors, Flux, Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type ComponentType } from "react";
import { assert, describe, expect, it } from "vitest";

import { Range } from "@/feature/range";
import { Overview } from "@/feature/range/overview/Overview";
import { Modals } from "@/platform/modals";
import { createResourceTab, primePanel } from "@/platform/panel/testutil";
import { createTestRange, uniqueRangeName } from "@/platform/range/testutil";
import {
  createConsoleWrapper,
  createTestClientWithGrants,
  getHeaderIconButton,
  getIconButton,
  type Grants,
  queryIconButton,
  renderSuspended,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

interface RenderOverviewResult {
  setTabResource: (rangeKey: string) => Promise<void>;
}

const renderOverview = async (
  rangeKey: string,
  FallbackComponent?: ComponentType<Errors.FallbackProps>,
  as: Client = client,
): Promise<RenderOverviewResult> => {
  const { wrapper } = await createConsoleWrapper({ client: as });
  const { panelKey, tabKey } = await createResourceTab(
    client,
    rangerClient.resourceID(rangeKey),
  );
  await primePanel(wrapper, panelKey);
  await renderSuspended(
    <PlutoPanel.Scope.Provider value={panelKey}>
      <PlutoPanel.TabScope.Provider value={tabKey}>
        <Errors.SuspenseBoundary FallbackComponent={FallbackComponent}>
          <Overview />
        </Errors.SuspenseBoundary>
        <Modals.Stack />
      </PlutoPanel.TabScope.Provider>
    </PlutoPanel.Scope.Provider>,
    // A spec that passes a fallback expects the boundary to catch.
    { wrapper, onCaughtError: FallbackComponent == null ? undefined : () => {} },
  );
  const setTabResource = async (nextKey: string) =>
    await act(async () => {
      await client.panels.dispatch(panelKey, [
        panel.setTabResource({
          key: tabKey,
          resource: rangerClient.resourceID(nextKey),
        }),
      ]);
    });
  return { setTabResource };
};

const createChildRange = async (parent: ranger.Range): Promise<ranger.Range> => {
  const start = TimeStamp.now();
  return await client.ranges.create({
    name: uniqueRangeName("child"),
    timeRange: new TimeRange(start, start.add(TimeSpan.seconds(1))),
    parent: parent.key,
  });
};

describe("range/overview/Overview", () => {
  it("renders the range details alongside the child range section", async () => {
    const rng = await createTestRange(client);
    await renderOverview(rng.key);
    expect(await screen.findByDisplayValue(rng.name)).toBeTruthy();
    expect(await screen.findByText("Child ranges")).toBeTruthy();
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
    await screen.findByText("Child ranges");
    fireEvent.click(await waitFor(() => getIconButton(document.body, "add")));
    expect(await screen.findByText("Save locally")).toBeTruthy();
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
    expect(screen.queryByText("Child ranges")).toBeNull();
  });

  it("shows the restored range's details after retry", async () => {
    const { restore } = Range.Overview.TABS[rangerClient.TYPE_RESOURCE_ID.type];
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
    });
    await act(async () => {
      await restore({ client, project: project.key, resource: rng.resourceID });
    });
    fireEvent.click(screen.getByTestId("restore"));
    expect(await screen.findByDisplayValue(rng.name)).toBeTruthy();
  });

  it("restores a deleted range under its original key", async () => {
    const { restore } = Range.Overview.TABS[rangerClient.TYPE_RESOURCE_ID.type];
    assert(restore != null);
    const rng = await createTestRange(client);
    // Restore rebuilds from the corpse the cache holds, which exists only once
    // the client has seen the range live.
    await client.ranges.retrieve(rng.key);
    await client.ranges.delete(rng.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.ranges.watch(rng.key).getCached())).toBe(
        true,
      ),
    );

    const project = await client.projects.create({
      name: uniqueName("proj"),
    });
    await restore({ client, project: project.key, resource: rng.resourceID });

    // The tab heals off the cache going live again, not off the create call.
    await waitFor(() =>
      expect(query.isLive(client.ranges.watch(rng.key).getCached())).toBe(true),
    );
    // Read back through a second client: the restoring client write-throughs its own
    // cache, so retrieving on it would pass even if the Core never got it.
    const remote = createTestClient();
    const [restored] = await remote.ranges.retrieve([rng.key]);
    expect(restored.name).toEqual(rng.name);
    expect(restored.timeRange).toEqual(rng.timeRange);
    // Name lookups read the range through the resource service, not the range store.
    const named = await remote.resources.retrieve(rng.resourceID);
    expect(named.name).toEqual(rng.name);
  });
});

describe("range/overview/ChildRanges permissions", () => {
  const READS = [
    rangerClient.TYPE_RESOURCE_ID,
    label.TYPE_RESOURCE_ID,
    panel.TYPE_RESOURCE_ID,
    project.TYPE_RESOURCE_ID,
  ];

  const createSubject = async (grants: Grants) =>
    await createTestClientWithGrants(client, { ...grants, retrieve: READS });

  const renderAs = async (as: Client): Promise<void> => {
    const rng = await createTestRange(client);
    const child = await createChildRange(rng);
    await renderOverview(rng.key, undefined, as);
    // The listed child proves the section resolved its data before the button is
    // looked for.
    await screen.findByText(child.name);
  };

  it("should withhold the add button from a subject who cannot create ranges", async () => {
    await renderAs(await createSubject({ update: [rangerClient.TYPE_RESOURCE_ID] }));
    const header = screen.getByText("Child ranges").closest("header");
    if (header == null) throw new Error("no child ranges header");
    expect(queryIconButton(header, "add")).toBeNull();
  });

  it("should offer the add button to a subject who may create ranges", async () => {
    await renderAs(await createSubject({ create: [rangerClient.TYPE_RESOURCE_ID] }));
    await waitFor(() => expect(getHeaderIconButton("Child ranges")).toBeTruthy());
  });
});

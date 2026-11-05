// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type panel, type resource, schematic } from "@synnaxlabs/client";
import { createPanelProject, createTestClient } from "@synnaxlabs/client/testutil";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Panel as PPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  renderHookSuspended,
  type TestStore,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

const createTestPlot = async (): Promise<lineplot.LinePlot> => {
  const project = await client.projects.create({
    name: uniqueName("proj"),
  });
  return await client.lineplots.create(project.key, { name: uniqueName("plot") });
};

// createFocusedPanel persists a panel whose only tab holds the resource id, and marks
// the panel selected and the tab focused in the session store.
const createFocusedPanel = async (
  store: TestStore,
  id: resource.ID,
): Promise<panel.Panel> => {
  const tabKey = uuid.create();
  const created = await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    root: {
      variant: "leaf",
      tabs: [{ variant: "resource", key: tabKey, resource: id }],
    },
    project: await createPanelProject(client),
  });
  store.dispatch(Session.Panel.select({ key: created.key, windowKey: MAIN_WINDOW }));
  store.dispatch(
    Session.Panel.internalSelectTab({
      key: created.key,
      tabKey,
      otherTabKeys: [tabKey],
      windowKey: MAIN_WINDOW,
    }),
  );
  return created;
};

// renderAddToActivePlot mounts the hook alongside a panel retrieve that primes the
// flux cache with the focused panel's document.
const renderAddToActivePlot = async (wrapper: FC<PropsWithChildren>, key: string) => {
  const { result } = await renderHookSuspended(
    () => ({
      add: Range.useAddToActivePlot(),
      panel: PPanel.use({ key }),
    }),
    { wrapper },
  );
  await waitFor(() => expect(result.current.panel).not.toBeNull());
  return result;
};

describe("Range.useAddToActivePlot", () => {
  it("should add the range to the slice and to the active plot's x1 axis", async () => {
    const range = await createTestRange(client);
    const plot = await createTestPlot();
    const { wrapper, store } = await createConsoleWrapper({ client });
    const doc = await createFocusedPanel(store, lineplot.resourceID(plot.key));
    const result = await renderAddToActivePlot(wrapper, doc.key);
    act(() => result.current.add([range.key]));
    await waitFor(() =>
      expect(
        store.getState()[Session.Range.SLICE_NAME].ranges.map(({ key }) => key),
      ).toContain(range.key),
    );
    await waitFor(async () => {
      const { ranges } = await client.lineplots.retrieve(plot.key);
      expect(ranges).toContainEqual({ variant: "key", key: range.key });
    });
  });

  it("should do nothing when the focused tab is not a line plot", async () => {
    const skipped = await createTestRange(client);
    const inactive = await createConsoleWrapper({ client });
    const otherPlot = await createTestPlot();
    const inactiveDoc = await createFocusedPanel(
      inactive.store,
      schematic.resourceID(uuid.create()),
    );
    const skippedResult = await renderAddToActivePlot(
      inactive.wrapper,
      inactiveDoc.key,
    );
    act(() => skippedResult.current.add([skipped.key]));

    const added = await createTestRange(client);
    const plot = await createTestPlot();
    const active = await createConsoleWrapper({ client });
    const activeDoc = await createFocusedPanel(
      active.store,
      lineplot.resourceID(plot.key),
    );
    const addedResult = await renderAddToActivePlot(active.wrapper, activeDoc.key);
    act(() => addedResult.current.add([added.key]));
    await waitFor(() =>
      expect(
        active.store.getState()[Session.Range.SLICE_NAME].ranges.map(({ key }) => key),
      ).toContain(added.key),
    );
    await waitFor(async () => {
      const { ranges } = await client.lineplots.retrieve(plot.key);
      expect(ranges).toContainEqual({ variant: "key", key: added.key });
    });

    expect(
      inactive.store.getState()[Session.Range.SLICE_NAME].ranges.map(({ key }) => key),
    ).not.toContain(skipped.key);
    const { ranges } = await client.lineplots.retrieve(otherPlot.key);
    expect(ranges).not.toContainEqual({ variant: "key", key: skipped.key });
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type ontology, type panel, schematic } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { MAIN_WINDOW } from "@synnaxlabs/drift";
import { Panel as PPanel } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, type TestStore, uniqueName } from "@/testutil";

const client = createTestClient();

const createTestPlot = async (): Promise<lineplot.LinePlot> => {
  const project = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  return await client.lineplots.create(project.key, { name: uniqueName("plot") });
};

// createFocusedPanel persists a panel whose only tab holds resource, and marks the
// panel selected and the tab focused in the session store.
const createFocusedPanel = async (
  store: TestStore,
  resource: ontology.ID,
): Promise<panel.Panel> => {
  const tabKey = uuid.create();
  const created = await client.panels.create({
    key: uuid.create(),
    name: uniqueName("panel"),
    root: {
      variant: "leaf",
      tabs: [{ variant: "resource", key: tabKey, resource }],
    },
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
  const { result } = renderHook(
    () => ({
      add: Range.useAddToActivePlot(),
      panel: PPanel.useRetrieve({ key }),
    }),
    { wrapper },
  );
  await waitFor(() => expect(result.current.panel.variant).toBe("success"));
  return result;
};

describe("Range.useAddToActivePlot", () => {
  it("should add the range to the slice and to the active plot's x1 axis", async () => {
    const range = await createTestRange(client);
    const plot = await createTestPlot();
    const { wrapper, store } = await createConsoleWrapper({ client });
    const doc = await createFocusedPanel(store, lineplot.ontologyID(plot.key));
    const result = await renderAddToActivePlot(wrapper, doc.key);
    act(() => result.current.add([range.key]));
    await waitFor(() =>
      expect(Session.Range.selectKeys(store.getState())).toContain(range.key),
    );
    await waitFor(async () => {
      const { ranges } = await client.lineplots.retrieve({ key: plot.key });
      expect(ranges.x1).toContain(range.key);
    });
  });

  it("should do nothing when the focused tab is not a line plot", async () => {
    const skipped = await createTestRange(client);
    const inactive = await createConsoleWrapper({ client });
    const otherPlot = await createTestPlot();
    const inactiveDoc = await createFocusedPanel(
      inactive.store,
      schematic.ontologyID(uuid.create()),
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
      lineplot.ontologyID(plot.key),
    );
    const addedResult = await renderAddToActivePlot(active.wrapper, activeDoc.key);
    act(() => addedResult.current.add([added.key]));
    await waitFor(() =>
      expect(Session.Range.selectKeys(active.store.getState())).toContain(added.key),
    );
    await waitFor(async () => {
      const { ranges } = await client.lineplots.retrieve({ key: plot.key });
      expect(ranges.x1).toContain(added.key);
    });

    expect(Session.Range.selectKeys(inactive.store.getState())).not.toContain(
      skipped.key,
    );
    const { ranges } = await client.lineplots.retrieve({ key: otherPlot.key });
    expect(ranges.x1).not.toContain(skipped.key);
  });
});

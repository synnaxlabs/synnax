// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type lineplot } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { placeLayout } from "@/platform/layout/testutil";
import { LinePlot } from "@/platform/lineplot";
import { Range } from "@/platform/range";
import { createTestRange } from "@/platform/range/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createTestPlot = async (): Promise<lineplot.LinePlot> => {
  const project = await client.projects.create({
    name: uniqueName("proj"),
    layout: {},
  });
  return await client.lineplots.create(project.key, { name: uniqueName("plot") });
};

describe("Range.useAddToActivePlot", () => {
  it("should add the range to the slice and to the active plot's x1 axis", async () => {
    const range = await createTestRange(client);
    const plot = await createTestPlot();
    const { wrapper, store } = await createConsoleWrapper({ client });
    placeLayout(store, plot.key, { type: LinePlot.LAYOUT_TYPE });
    const { result } = renderHook(() => Range.useAddToActivePlot(), { wrapper });
    act(() => result.current([range.key]));
    await waitFor(() =>
      expect(Session.Range.selectKeys(store.getState())).toContain(range.key),
    );
    await waitFor(async () => {
      const { ranges } = await client.lineplots.retrieve({ key: plot.key });
      expect(ranges.x1).toContain(range.key);
    });
  });

  it("should do nothing when the active mosaic layout is not a line plot", async () => {
    const skipped = await createTestRange(client);
    const inactive = await createConsoleWrapper({ client });
    placeLayout(inactive.store, uniqueName("layout"));
    const { result: skippedResult } = renderHook(() => Range.useAddToActivePlot(), {
      wrapper: inactive.wrapper,
    });
    act(() => skippedResult.current([skipped.key]));

    const added = await createTestRange(client);
    const plot = await createTestPlot();
    const active = await createConsoleWrapper({ client });
    placeLayout(active.store, plot.key, { type: LinePlot.LAYOUT_TYPE });
    const { result: addedResult } = renderHook(() => Range.useAddToActivePlot(), {
      wrapper: active.wrapper,
    });
    act(() => addedResult.current([added.key]));
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
    const { ranges } = await client.lineplots.retrieve({ key: plot.key });
    expect(ranges.x1).not.toContain(skipped.key);
  });
});

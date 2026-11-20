// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type ranger } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";
import { color as xcolor, dynamicWindow, TimeSpan } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  client,
  createLinePlotWrapper,
  createPreloadedState,
} from "@/feature/lineplot/testutil";
import { useResetZoomOnRangeChange } from "@/feature/lineplot/useResetZoomOnRangeChange";
import { Session } from "@/session";

const rolling = (span: TimeSpan): ranger.Ref => ({
  variant: "window",
  window: dynamicWindow(span),
});

const ZOOMED: Session.LinePlot.ViewportState = {
  renderTrigger: 0,
  zoom: { width: 0.5, height: 0.5 },
  pan: { x: 0.25, y: 0.25 },
};

const renderResetZoom = async () => {
  const { key, wrapper } = await createLinePlotWrapper({
    linePlot: { ranges: [rolling(TimeSpan.minutes(5))] },
    preloadedState: (key) => createPreloadedState(key, { viewport: ZOOMED }),
  });
  const { result } = renderHook(
    () => {
      useResetZoomOnRangeChange({ key });
      return {
        viewport: Session.LinePlot.useSelect({ key }).viewport,
        name: LinePlot.useName({ key }),
        ranges: LinePlot.useRanges({ key }),
      };
    },
    { wrapper },
  );
  return { key, result };
};

describe("lineplot useResetZoomOnRangeChange", () => {
  it("should keep the zoom when the plot mounts", async () => {
    const { result } = await renderResetZoom();
    await act(async () => {});
    expect(result.current.viewport).toEqual(ZOOMED);
  });

  it("should reset the zoom when the plot's ranges change", async () => {
    const { key, result } = await renderResetZoom();
    await act(async () => {
      await client.lineplots.dispatch(
        key,
        lineplot.setRanges({ ranges: [rolling(TimeSpan.hours(1))] }),
      );
    });
    await expect
      .poll(() => result.current.viewport)
      .toMatchObject({
        ...Session.LinePlot.ZERO_STATE.viewport,
        renderTrigger: 1,
      });
  });

  it("should keep the zoom when the color of a range changes", async () => {
    const { key, result } = await renderResetZoom();
    const color = { type: "fixed" as const, value: xcolor.construct("#ff8000") };
    await act(async () => {
      await client.lineplots.dispatch(
        key,
        lineplot.setRangeColor({ range: rolling(TimeSpan.minutes(5)), color }),
      );
    });
    await expect.poll(() => result.current.ranges[0]?.color).toEqual(color);
    expect(result.current.viewport).toEqual(ZOOMED);
  });

  it("should keep the zoom when an edit leaves the ranges alone", async () => {
    const { key, result } = await renderResetZoom();
    await act(async () => {
      await client.lineplots.dispatch(
        key,
        lineplot.set({ path: ["name"], value: "Renamed" }),
      );
    });
    await expect.poll(() => result.current.name).toBe("Renamed");
    expect(result.current.viewport).toEqual(ZOOMED);
  });
});

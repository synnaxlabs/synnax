// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot, Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { act, fireEvent } from "@testing-library/react";
import type { ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Controls } from "@/feature/lineplot/Controls";
import { createPreloadedState, renderLinePlot } from "@/feature/lineplot/testutil";
import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { getIconButton, queryIconButton } from "@/testutil";

const renderControls = async (hasBreaks: boolean) => {
  const InPlot = (): ReactElement => {
    const { key } = PlutoPanel.useTabResource();
    return (
      <LinePlot.Suspended linePlotKey={key}>
        <Controls hasAnnotations={false} hasBreaks={hasBreaks} />
      </LinePlot.Suspended>
    );
  };
  const handle = await renderLinePlot(InPlot, {
    preloadedState: (key) => createPreloadedState(key),
  });
  const plot = () =>
    documentIn(handle.store.getState()[Session.LinePlot.SLICE_NAME], handle.key);
  return { ...handle, plot };
};

describe("lineplot/Controls", () => {
  describe("breaks", () => {
    it("should hide the breaks of the plot and reset its zoom", async () => {
      const { result, plot } = await renderControls(true);
      const trigger = plot()?.viewport.renderTrigger ?? 0;
      act(() => void fireEvent.click(getIconButton(result.container, "split-x")));
      expect(plot()?.breaks.hidden).toBe(true);
      expect(plot()?.viewport.renderTrigger).toBe(trigger + 1);
    });

    it("should show the breaks again on a second click", async () => {
      const { result, plot } = await renderControls(true);
      act(() => void fireEvent.click(getIconButton(result.container, "split-x")));
      act(() => void fireEvent.click(getIconButton(result.container, "split-x")));
      expect(plot()?.breaks.hidden).toBe(false);
    });

    it("should show no breaks toggle on a plot with no breaks", async () => {
      const { result } = await renderControls(false);
      expect(queryIconButton(result.container, "split-x")).toBeNull();
    });
  });
});

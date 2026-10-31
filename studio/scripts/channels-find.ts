// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, director, fixtures } from "@/index";

const QUERY = "tank";
const TARGET = "tank_pressure";
const ZOOM = 1.5;

/**
 * Docs `client/channels/find`: with live tank and ox_tank channels in the Core, open
 * the search palette, type "tank", and select tank_pressure from the matches. A line
 * plot of tank_pressure opens.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const tank = await fixtures.sineTelemetry({ prefix: "tank" });
  const oxTank = await fixtures.sineTelemetry({ prefix: "ox_tank", base: 40 });
  try {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    // Fills the plot's recent window so the plot that opens has a full trace.
    await session.settleWall(20000);
    // Clear of the empty panel's logo and its Create component link.
    await session.moveTo({ x: 1100, y: 650 });

    session.startRecording();
    await session.hold(1500);

    await session.click(page.locator(".console-palette button").first());
    const input = page.locator(".console-palette__input input[role='textbox']");
    await session.waitFor(input);
    const dialog = page
      .locator(".lyra-dialog__dialog:has(.console-palette__input)")
      .first();
    const box = await dialog.boundingBox();
    if (box == null) throw new Error("search palette is hidden");
    // The palette is wider than the zoomed frame, so zoom on its left side, where the
    // input and the result names are.
    const { width } = page.viewportSize() ?? { width: 1600 };
    await session.zoom(
      {
        x: box.x - director.ZOOM_RECT_MARGIN_PX + width / ZOOM / 2,
        y: box.y + box.height / 2,
      },
      ZOOM,
    );
    await session.hold(600);

    await session.type(QUERY);
    const result = dialog
      .locator(".lyra-list__item")
      .filter({ has: page.getByText(TARGET, { exact: true }) })
      .first();
    await session.waitFor(result);
    await session.hold(1500);
    await session.click(result.getByText(TARGET, { exact: true }).first(), {
      text: true,
    });
    session.endZoom();

    const plotTab = capture.tab(page, "Line plot");
    await session.waitFor(plotTab);
    await session.waitFor(page.locator(".pluto-line-plot").first());
    // The plot draws on wall time; let it fetch and render the trace.
    await session.settleWall(3000);
    const tabBox = await plotTab.boundingBox();
    if (tabBox == null) throw new Error("line plot tab is hidden");
    await session.moveTo({ x: tabBox.x + tabBox.width + 320, y: tabBox.y + 12 });
    await session.hold(2800);
  } finally {
    await Promise.all([tank.stop(), oxTank.stop()]);
  }
};

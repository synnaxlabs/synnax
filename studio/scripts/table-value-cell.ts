// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { drawer, resize } from "./common/table";

/**
 * Docs `console/tables/value-cell`: select a cell, press V, change the cell to a
 * value cell, and pick its channel on the Telemetry tab; the cell starts showing
 * the channel's live value.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Table");
    await session.waitFor(page.locator(".console-table").first());
    // The default cells are too small to read on camera.
    for (const column of ["A", "B"])
      await resize(session, `Resize column ${column}`, { x: 110, y: 0 });
    for (const row of ["1", "2"])
      await resize(session, `Resize row ${row}`, { x: 0, y: 14 });
    // The table opens its toolbar on its own; the shot opens it with V.
    await capture.hideBottomToolbar(session);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1800);

    // No zoom anywhere: the cell sits far from the toolbar that edits it.
    await session.click(page.locator(".pluto-table__cell").nth(1), { zoom: false });
    await session.hold(600);
    await session.press("v");
    const toolbar = drawer(page);
    const change = toolbar.locator("[aria-label='Change cell type']").first();
    await session.waitFor(change);
    await session.hold(900);

    await session.click(change, { text: true, zoom: false });
    const option = page
      .locator(".pluto-table-cell-change-variant__dialog [role='option']")
      .filter({ hasText: "Value" })
      .first();
    await session.waitFor(option);
    await session.hold(600);
    await session.click(option, { text: true, zoom: false });

    const telemetry = toolbar.getByText("Telemetry", { exact: true }).first();
    await session.waitFor(telemetry);
    await session.hold(600);
    await session.click(telemetry, { text: true, zoom: false });
    await session.hold(500);

    const channel = fixture.channels[0];
    await session.click(toolbar.getByText("Select channel", { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    const search = page.locator("input[placeholder*='Search']").first();
    await session.waitFor(search);
    await session.hold(500);
    session.setSpeed(1.5);
    await session.type(channel);
    session.setSpeed(1);
    const item = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: channel })
      .first();
    await session.waitFor(item);
    await session.hold(400);
    await session.click(item.getByText(channel, { exact: true }).first(), {
      text: true,
      zoom: false,
    });
    await session.hold(800);

    // Clear of the cell, so the live value reads.
    await session.moveTo({ x: 756, y: 500 });
    await session.settleWall(1500);
    await session.hold(2800);
  } finally {
    await fixture.stop();
  }
};

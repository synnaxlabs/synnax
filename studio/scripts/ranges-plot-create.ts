// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

/**
 * Docs `console/ranges/plot-create`: drag a selection over plotted data, create
 * a range from it through the selection menu, name it, and save; From and To
 * arrive pre-filled from the selection and the range lands in the toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.createComponent(session, "Line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    // A saved range carries no color, so it draws no annotation on the plot;
    // the toolbar list is where the viewer sees it arrive.
    await capture.openToolbar(session, "range");
    await capture.resizeToolbar(session, 380);
    // The selection pins to data on screen, so stop the rolling window first. A
    // hold snapshots the axis as it turns on, so wait for real samples before
    // pausing.
    await session.settleWall(8000);
    await session.settle(1000);
    await session.click(capture.control(page, "pause"), { zoom: false });
    await session.settle(500);
    await session.moveTo({ x: 950, y: 500 });

    session.startRecording();
    await session.hold(1400);

    // No zoom: the controls sit in a corner and the camera must hold the plot.
    await session.click(capture.control(page, "selection"), { zoom: false });
    await session.hold(800);
    await session.drag({ x: 620, y: 300 }, { x: 1080, y: 640 }, { zoom: false });
    await session.hold(1000);

    await capture.contextMenu(
      session,
      { x: 850, y: 470 },
      "Create range from selection",
    );
    const modal = page.locator(".console-modal").first();
    const name = modal.getByPlaceholder("Name").first();
    await session.waitFor(name);
    await session.hold(700);
    await session.zoom(modal);

    await session.click(name);
    await session.type("Burn Window 3");
    await session.hold(1400);

    await capture.clickButton(session, "Save to Core");
    await session.waitForHidden(modal);
    session.endZoom();
    const created = page
      .locator(".console-nav__drawer")
      .getByText("Burn Window 3", { exact: true })
      .first();
    await session.waitFor(created);
    await session.hold(2600);
  } finally {
    await fixture.stop();
  }
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const SAVED = capture.PROJECT;

/**
 * Docs `console/workspaces/load`: from an empty project, open the project selector
 * and select a project with a saved line plot; its layout loads.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await capture.login(session);

    await capture.commandPalette(session, "Create line plot");
    await session.waitFor(page.locator(".pluto-line-plot").first());
    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await capture.hideBottomToolbar(session);
    // The layout saves to the Core on a debounce: let it land before switching.
    await session.settleWall(3000);

    await capture.commandPalette(session, "Create project");
    const name = page.locator(capture.MODAL).getByPlaceholder("Name").first();
    await session.waitFor(name);
    await name.fill("Coldflow Campaign");
    await capture.clickButton(session, "Create");
    await session.waitForHidden(name);
    await session.waitForHidden(page.locator(".pluto-line-plot").first());

    await session.settleWall(1000);
    await session.moveTo({ x: 900, y: 500 });

    session.startRecording();
    await session.hold(1800);

    // The dialog opens beside the small avatar, so a punch-in would crop it.
    await session.click(page.locator(".console-project-selector__trigger").first(), {
      zoom: false,
    });
    const dialog = page.locator(".console-project-selector-dialog").first();
    const saved = dialog.getByText(SAVED, { exact: true }).first();
    await session.waitFor(saved);
    await session.zoom(dialog);
    await session.hold(900);
    // Loading swaps the whole layout, so the camera returns to wide.
    await session.click(saved, { text: true, zoom: false });
    session.endZoom();
    await session.waitFor(page.locator(".pluto-line-plot").first());
    // The reloaded plot has a gap where its stream restarted. It fills on wall time,
    // so wait out a full rolling window before the plot is on camera.
    await session.settleWall(32000);
    await session.hold(2500);
  } finally {
    await fixture.stop();
  }
};

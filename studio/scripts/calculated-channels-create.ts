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
 * Docs `console/calculated-channels/create`: click the calculated action in
 * the Channels toolbar, name the channel, write an Arc expression over a live
 * channel, and create it.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    const project = `Docs Videos ${Date.now().toString(36)}`;
    await capture.login(session, { username: "synnax", password: "seldon" }, project);

    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 400);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1000);

    // The calculated action is the first of the toolbar's two create actions.
    await session.click(
      page.locator(".console-nav__drawer .console-toolbar__action").first(),
    );
    const name = page.locator(".console-modal").getByPlaceholder("Name").first();
    await session.waitFor(name);
    await session.hold(600);
    await session.zoom(page.locator(".console-modal").first());

    await session.click(name);
    await session.type("pressure_doubled");
    await session.hold(400);

    await session.click(page.locator(".console-calculated-editor").first());
    await session.type(`return ${fixture.channels[0]} * 2`);
    await session.hold(800);

    await capture.clickButton(session, "Create");
    await session.waitForHidden(name);
    session.endZoom();
    const created = page
      .locator(".console-nav__drawer")
      .getByText("pressure_doubled", { exact: true })
      .first();
    await session.waitFor(created);
    await session.hold(2200);
  } finally {
    await fixture.stop();
  }
};

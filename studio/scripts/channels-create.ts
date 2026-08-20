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
 * Docs `console/channels/create`: click the "+" action in the Channels
 * toolbar, fill the dialog (name, data type, index), and submit; the new
 * channel appears in the toolbar.
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

    await capture.clickToolbarCreate(session);
    const name = page.getByPlaceholder("Name").first();
    await session.waitFor(name);
    await session.hold(600);
    await session.zoom(page.locator(".console-modal").first());

    await session.click(name);
    await session.type("pressure_pt_01");
    await session.hold(400);

    await session.click(page.getByText("Select channel", { exact: true }).first(), {
      text: true,
    });
    const index = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: "demo_time" })
      .first();
    await session.waitFor(index);
    await session.click(index.getByText("demo_time", { exact: true }).first(), {
      text: true,
    });
    await session.hold(600);

    await capture.clickButton(session, "Create");
    await session.waitForHidden(name);
    session.endZoom();
    const created = page
      .locator(".console-nav__drawer")
      .getByText("pressure_pt_01", { exact: true })
      .first();
    await session.waitFor(created);
    await session.hold(2200);
  } finally {
    await fixture.stop();
  }
};

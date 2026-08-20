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
 * Docs `console/calculated-channels/edit`: right-click a calculated channel in
 * the Channels toolbar, pick "Edit calculation", rewrite the expression, and
 * save.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    const { page } = session;
    await fixtures.seedCalculatedChannels([
      { name: "pressure_doubled", expression: `return ${fixture.channels[0]} * 2` },
    ]);
    await capture.login(session, { username: "synnax", password: "seldon" });

    await capture.openToolbar(session, "channel");
    await capture.resizeToolbar(session, 400);
    const item = capture.treeItem(page, "channel:", "pressure_doubled");
    await session.waitFor(item);
    await session.moveTo({ x: 756, y: 500 });

    session.startRecording();
    await session.hold(1000);

    await capture.contextMenu(session, item, "Edit calculation");
    const editor = page.locator(".console-calculated-editor").first();
    await session.waitFor(editor);
    await session.hold(600);
    await session.zoom(page.locator(".console-modal").first());

    await session.click(editor);
    await session.press("ControlOrMeta+a");
    await session.type(`return ${fixture.channels[0]} * 10`);
    await session.hold(800);

    await capture.clickButton(session, "Save");
    await session.waitForHidden(editor);
    session.endZoom();
    await session.hold(2200);
  } finally {
    await fixture.stop();
  }
};

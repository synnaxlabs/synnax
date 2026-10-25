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
 * Still for docs image `console/logs/thumbnail`: a log streaming lines from two
 * channels. The last captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry({ periodMs: 250 });
  try {
    const { page } = session;
    await capture.login(session);

    await capture.createComponent(session, "Log");
    await session.waitFor(page.locator(".pluto-log").first());
    const add = page.getByText("Add channel", { exact: true }).first();
    if (!(await add.isVisible().catch(() => false)))
      await session.click(page.locator(".console-main-nav__item").last(), {
        zoom: false,
      });
    await session.waitFor(add);
    for (const channel of fixture.channels) {
      await session.click(page.getByText("Add channel", { exact: true }).first(), {
        text: true,
        zoom: false,
      });
      await capture.pickChannel(session, channel);
      await session.settle(400);
    }
    await capture.hideBottomToolbar(session);

    // A log shows only values that arrive while it is open, on wall time.
    await session.settleWall(12000);
    await session.settle(500);
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(2500);
  } finally {
    await fixture.stop();
  }
};

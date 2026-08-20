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
 * Recreates the docs line-plots data-tab flow: create a plot, select live
 * channels onto Y1, and watch the data stream in.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry();
  try {
    await capture.login(session, { username: "synnax", password: "seldon" });
    await capture.clearPanel(session);

    session.startRecording();
    await session.hold(1200);

    await capture.clickPanelCreate(session);
    await session.hold(800);
    await capture.createComponent(session, "Line plot");
    await session.waitFor(session.page.locator(".pluto-line-plot").first());
    await session.hold(1000);

    await capture.addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await session.hold(4000);
  } finally {
    await fixture.stop();
  }
};

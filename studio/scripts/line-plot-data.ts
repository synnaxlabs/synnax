// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { commandPalette, login } from "@/capture/console";
import { addChannels } from "@/capture/plot";
import { type CaptureSession } from "@/capture/rig";
import { sineTelemetry } from "@/fixtures/telemetry";

/**
 * Recreates the docs line-plots data-tab flow: create a plot, select live
 * channels onto Y1, and watch the data stream in.
 */
export default async (session: CaptureSession): Promise<void> => {
  const fixture = await sineTelemetry();
  try {
    const project = `Docs Videos ${Date.now().toString(36)}`;
    await login(session, { username: "synnax", password: "seldon" }, project);

    session.startRecording();
    await session.hold(800);

    await commandPalette(session, "Create a Line Plot");
    await session.waitFor(session.page.locator(".pluto-line-plot").first());
    await session.hold(1000);

    await addChannels(session, "Y1", fixture.channels, { search: "demo" });
    await session.hold(5000);
  } finally {
    await fixture.stop();
  }
};

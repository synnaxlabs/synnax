// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { commandPalette, login } from "@/capture/console";
import { type CaptureSession } from "@/capture/rig";

/**
 * Demo flow for the prototype: log in, create a line plot from the command
 * palette, and open its data tab.
 */
export default async (session: CaptureSession): Promise<void> => {
  await login(session, { username: "synnax", password: "seldon" });

  session.startRecording();
  await session.hold(800);

  await commandPalette(session, "Create a Line Plot");
  await session.waitFor(session.page.locator(".pluto-line-plot"));
  await session.hold(1200);

  const dataTab = session.page.locator("#data");
  if (await dataTab.isVisible().catch(() => false)) {
    await session.click(dataTab);
    await session.hold(1500);
  }

  await session.hold(1000);
};

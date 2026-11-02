// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

export const viewport = { width: 1280, height: 800 };

/**
 * Still for the hero of docs page `reference/control/arc`: the TPC stand's Ox Coldflow
 * sequence alone in the Automations panel's text editor while it runs on the Core, with
 * the Arc toolbar open to show it running.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  try {
    const { page } = session;
    await capture.login(session, world.project);
    await session.click(page.getByText("Automations", { exact: true }).first());
    const editor = page.locator(".pluto-mosaic__leaf:visible .monaco-editor").first();
    await session.waitFor(editor);
    await capture.openToolbar(session, "arc");
    const toolbar = page.locator(".console-arc-toolbar").first();
    await session.waitFor(toolbar.getByText("Running", { exact: true }).first());
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await world.stop();
  }
};

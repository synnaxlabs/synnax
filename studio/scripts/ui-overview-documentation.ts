// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

/**
 * Docs `console/ui-overview/documentation`: from an empty panel, click the
 * question-mark icon in the top right; the Synnax documentation site opens in a
 * tab inside the Console.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  const button = page.locator(".console-docs__open-button").first();
  const loading = page.locator(".console-docs__loading").first();
  // The docs tab is an iframe onto the live site, which loads on wall time. Open
  // it once during setup so the recorded open reads from the browser cache.
  await button.click({ timeout: 5000 });
  await session.waitFor(page.locator(".console-docs iframe").first());
  await session.settleWall(8000);
  await capture.clearPanel(session);
  await session.moveTo({ x: 756, y: 520 });

  session.startRecording();
  await session.hold(1200);

  // No zoom: the click fills the whole panel with the documentation site.
  await session.click(button, { zoom: false });
  await session.waitFor(capture.tab(page, "Documentation"));
  await session.hold(300);
  // Off the button so nothing sits over the page as it paints.
  await session.moveTo({ x: 756, y: 520 });
  await session.waitForHidden(loading, 120).catch(() => {});
  await session.hold(2600);
};

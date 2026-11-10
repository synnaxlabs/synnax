// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { openProject } from "./common/desktop";

export const desktop = true;

/**
 * Docs `reference/desktop/troubleshooting` (Diagnostics): run "Open diagnostics", read
 * the health of Synnax Desktop and the end of its log, click Copy logs, then click
 * Export. The Desktop shell is simulated, so the log and the export are not real.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await openProject(session);
  await capture.clearPanel(session);
  await capture.hideBottomToolbar(session);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.commandPalette(session, "Open diagnostics");
  const modal = page.locator(capture.MODAL).first();
  const copy = modal.getByRole("button", { name: "Copy logs", exact: true });
  await session.waitFor(copy);
  await session.zoom(modal);
  await session.hold(2500);

  await session.click(copy);
  await session.hold(1500);
  await session.click(modal.getByRole("button", { name: "Export", exact: true }));
  await session.hold(2000);
  session.endZoom();
  await session.moveTo({ x: 1200, y: 840 });
  await session.hold(2500);
};

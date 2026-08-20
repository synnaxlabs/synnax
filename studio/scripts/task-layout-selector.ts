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
 * Docs `reference/driver/task-basics` (Layout Selector tab): from an empty
 * panel, click the mosaic "+" to open the component selector and pick a task
 * type; the task configuration form opens in its place as a tab.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });
  await capture.clearPanel(session);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1200);

  await capture.clickPanelCreate(session);
  await session.hold(800);
  await capture.createComponent(session, "NI analog read task");
  await session.waitFor(page.locator(".console-task-configure").first());
  await session.hold(3000);
};

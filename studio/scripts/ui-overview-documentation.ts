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
 * Docs `console/ui-overview/documentation`: point at the question-mark icon in the top
 * right until its "Open documentation" tooltip shows. The click opens the docs in the
 * system browser, which the capture cannot record.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });
  await session.moveTo({ x: 756, y: 520 });

  session.startRecording();
  await session.hold(1200);

  const button = page.locator(".console-docs__open-button").first();
  await session.zoom(button, 1.8);
  await session.moveTo(button);
  await session.waitFor(page.getByText("Open documentation").first());
  await session.hold(3000);
};

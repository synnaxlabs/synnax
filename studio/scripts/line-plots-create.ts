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
 * Docs `console/line-plots/create`: from an empty panel, open the component selector
 * with "+" and pick Line plot, landing on the log's empty state.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);
  await capture.clearPanel(session);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(500);

  await capture.clickPanelCreate(session);
  await session.hold(800);
  await capture.createComponent(session, "Line plot");
  await session.waitFor(page.locator(".pluto-line-plot").first());
  await session.hold(1500);
};

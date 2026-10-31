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
 * Docs `client/channels/rename`: right-click tank_pressure in the Channels toolbar,
 * select "Rename", and rename it to ox_tank_pressure.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const fixture = await fixtures.sineTelemetry({ prefix: "tank" });
  await fixture.stop();
  const { page } = session;
  await capture.login(session);

  await capture.clearPanel(session);
  await capture.openToolbar(session, "channel");
  await capture.resizeToolbar(session, 400);
  const item = capture.treeItem(page, "channel:", "tank_pressure");
  await session.waitFor(item);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.contextMenu(session, item, "Rename");
  // The name turns into a contenteditable with its text selected, so typing
  // replaces it.
  const editable = page.locator(".lyra-text--editable[contenteditable='true']").first();
  await session.waitFor(editable);
  await session.hold(400);
  await session.type("ox_tank_pressure");
  await session.hold(400);
  await session.press("Enter");
  session.endZoom();
  await session.waitFor(
    page.locator(".console-nav__drawer").getByText("ox_tank_pressure", { exact: true }),
  );
  await session.hold(2800);
};

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
 * Docs `console/workspaces/create`: open the Project Selector in the top-left
 * corner, click "New project", name it, and create it; the new project becomes
 * active in the selector.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1000);

  // The dialog opens beside the small avatar, so a punch-in would crop it.
  const trigger = page.locator(".console-project-selector__trigger").first();
  await session.click(trigger, { zoom: false });
  const create = page
    .locator(".console-project-selector-dialog")
    .getByText("New project", { exact: true })
    .first();
  await session.waitFor(create);
  await session.hold(900);
  await session.click(create, { text: true, zoom: false });

  const name = page.locator(".console-modal").getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(600);
  await session.zoom(page.locator(".console-modal").first());
  await session.type("Hotfire Campaign");
  await session.hold(500);
  await capture.clickButton(session, "Create");
  await session.waitForHidden(name);
  session.endZoom();
  await session.waitFor(page.getByText("New panel").first());
  await session.hold(1200);

  // The avatar is the only sign of the switch, so reopen the selector on the list.
  await session.click(trigger, { zoom: false });
  const dialog = page.locator(".console-project-selector-dialog").first();
  await session.waitFor(dialog.getByText("Hotfire Campaign").first());
  await session.zoom(dialog);
  await session.hold(2800);
};

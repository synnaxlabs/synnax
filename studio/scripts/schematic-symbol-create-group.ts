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
 * Docs `console/schematics/symbol-create-group`: in the symbols browser, click
 * the create group action, name the group, and save; the new group joins the
 * library and opens empty.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await capture.selectToolbarTab(session, "Symbols");
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1200);

  await session.click(capture.toolbarButton(page, "group", "add"));
  const name = page.locator(".console-modal").getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(600);
  await session.zoom(page.locator(".console-modal").first());

  await session.click(name);
  await session.type("Test stand");
  await session.hold(600);
  await capture.clickButton(session, "Save");
  await session.waitForHidden(name);
  session.endZoom();

  const group = page
    .locator(".console-schematic__symbols-group-list [role='tab']")
    .filter({ hasText: "Test stand" })
    .first();
  await session.waitFor(group);
  await session.hold(800);
  await session.click(group, { text: true, zoom: false });
  await session.hold(600);
  // Clear the cursor off the new group's tab so its name stays readable.
  await session.moveTo({ x: 756, y: 500 });
  await session.hold(2400);
};

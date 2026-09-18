// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import path from "node:path";

import { capture } from "@/index";

// Not scripts/assets/: the repo ignores every directory named "assets".
const SVG = path.join(import.meta.dirname, "files", "tank.svg");

/**
 * Docs `console/schematics/symbol-import-svg`: with a custom group selected,
 * open the create symbol dialog, import an SVG, and save; the editor lists the
 * color regions it found and the symbol joins the group.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await capture.selectToolbarTab(session, "Symbols");

  // Create and open the group first: the create symbol action stays disabled
  // while a built-in group is selected.
  await session.click(capture.toolbarButton(page, "group", "add"), { zoom: false });
  const groupName = page.locator(".console-modal").getByPlaceholder("Name").first();
  await session.waitFor(groupName);
  await groupName.fill("Test stand");
  await session.settle(300);
  await capture.clickButton(session, "Save");
  await session.waitForHidden(groupName);
  const group = page
    .locator(".console-schematic__symbols-group-list [role='tab']")
    .filter({ hasText: "Test stand" })
    .first();
  await session.waitFor(group);
  await session.click(group, { text: true, zoom: false });
  await session.settle(600);
  await session.moveTo({ x: 756, y: 480 });

  session.startRecording();
  await session.hold(1200);

  await session.click(capture.toolbarButton(page, "schematic", "add"));
  const modal = page.locator(".console-modal").first();
  const drop = page.locator(".console-schematic-file-drop").first();
  await session.waitFor(drop);
  await session.hold(800);
  await session.zoom(modal);

  // The Console opens a detached file input, so the picker never renders; the
  // chooser event is the only seam to hand it a file.
  const chooser = page.waitForEvent("filechooser");
  await session.click(drop, { zoom: false });
  await (await chooser).setFiles(SVG);
  const region = page.locator(".console-schematic-region-list-item").first();
  await session.waitFor(region);
  await session.hold(2400);

  await session.click(
    modal.getByRole("button", { name: "Create", exact: true }).first(),
    { zoom: false },
  );
  await session.waitForHidden(drop);
  session.endZoom();

  const symbol = page
    .locator(".console-schematic-symbols__button")
    .filter({ has: page.getByText("Tank", { exact: true }) })
    .first();
  await session.waitFor(symbol);
  await session.hold(600);
  // Park the cursor clear of the new symbol, then frame the group tabs and the
  // symbol together: at full width the drawer entry is too small to read.
  await session.moveTo({ x: 420, y: 900 });
  await session.zoom({ x: 546, y: 830 }, 1.45);
  await session.hold(2600);
};

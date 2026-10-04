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
 * Docs `console/schematics/symbol-import-svg`: with a custom group selected, open
 * the create symbol dialog, import an SVG, make it an actuator, recolor one region
 * in its Active state, and create it; the symbol joins the group.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);

  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await capture.selectToolbarTab(session, "Symbols");

  // Create and open the group first: the create symbol action stays disabled
  // while a built-in group is selected.
  await session.click(capture.toolbarButton(page, "group", "add"), { zoom: false });
  const groupName = page
    .locator(".pluto-dialog__dialog.pluto--modal")
    .getByPlaceholder("Name")
    .first();
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
  // The strip clips tabs past its end under the group actions.
  await group.scrollIntoViewIfNeeded();
  await session.click(group, { text: true, zoom: false });
  await session.settle(600);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1800);

  await session.click(capture.toolbarButton(page, "schematic", "add"), { zoom: false });
  const modal = page.locator(".pluto-dialog__dialog.pluto--modal").first();
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
  await session.hold(1800);

  await session.click(
    modal.locator(".pluto-select-btn").filter({ hasText: "Actuator" }).first(),
    { text: true, zoom: false },
  );
  const active = modal
    .locator(".console-schematic-state-list-item")
    .filter({ hasText: "Active" })
    .first();
  await session.waitFor(active);
  await session.hold(700);
  await session.click(active, { text: true, zoom: false });
  await session.hold(700);

  // The second swatch of a region is its fill. The 300px sidebar clips it, so scroll
  // the sidebar to it first: the rig clicks at screen coordinates.
  const fill = region.locator(".pluto-color-swatch--chip").nth(1);
  await fill.scrollIntoViewIfNeeded();
  await session.settle(300);
  await session.click(fill, { zoom: false });
  const picker = page.locator(".pluto-color-picker").first();
  await session.waitFor(picker);
  await session.hold(600);
  // The second preset is the theme's secondary color.
  await session.click(
    picker.locator(".pluto-color-picker__swatches .pluto-color-swatch").nth(1),
    { zoom: false },
  );
  await session.hold(500);
  // A click outside the floating picker closes it.
  await session.click(modal.getByText("Colors", { exact: true }).first(), {
    text: true,
    zoom: false,
  });
  await session.waitForHidden(picker);
  await session.hold(1400);

  await session.click(
    modal.getByRole("button", { name: "Create", exact: true }).first(),
    { zoom: false },
  );
  session.endZoom();
  await session.waitForHidden(drop);

  const symbol = page
    .locator(".console-schematic-symbols__button")
    .filter({ has: page.getByText("Tank", { exact: true }) })
    .first();
  await session.waitFor(symbol);
  await session.moveTo({ x: 800, y: 450 });
  await session.hold(2800);
};

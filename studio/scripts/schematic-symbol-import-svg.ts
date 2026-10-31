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

import { setup } from "./common/symbol-group";

// Not scripts/assets/: the repo ignores every directory named "assets".
const SVG = path.join(import.meta.dirname, "files", "tank.svg");

/**
 * Docs `console/schematics/symbol-import-svg`: with a custom group selected, open
 * the create symbol dialog, import an SVG, make it an actuator, recolor one region
 * in its Active state, and create it; the symbol joins the group.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await setup(session);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1800);

  await session.click(capture.toolbarButton(page, "schematic", "add"), { zoom: false });
  const modal = page.locator(capture.MODAL).first();
  const drop = page.locator(".console-schematic-file-drop").first();
  await session.waitFor(drop);
  await session.hold(800);
  // The Console opens a detached file input, so the picker never renders; the
  // chooser event is the only seam to hand it a file.
  const chooser = page.waitForEvent("filechooser");
  await session.click(drop, { zoom: false });
  await (await chooser).setFiles(SVG);
  const region = page.locator(".console-schematic-region-list-item").first();
  await session.waitFor(region);
  // The dialog nearly fills the window, so a zoom on it renders at 1x. The sidebar
  // shows only after the import: zoom on its edge, and let the camera pan to Create.
  const sidebar = await modal
    .locator(".console-schematic__sidebar")
    .first()
    .boundingBox();
  if (sidebar == null) throw new Error("symbol sidebar is hidden");
  await session.zoom(
    { x: sidebar.x + sidebar.width, y: sidebar.y + sidebar.height / 2 },
    1.5,
  );
  await session.hold(1800);

  await session.click(
    modal.locator(".lyra-select-btn").filter({ hasText: "Actuator" }).first(),
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
  const fill = region.locator(".lyra-color-swatch--chip").nth(1);
  await fill.scrollIntoViewIfNeeded();
  await session.settle(300);
  await session.click(fill, { zoom: false });
  const picker = page.locator(".lyra-color-picker").first();
  await session.waitFor(picker);
  await session.hold(600);
  // The second preset is the theme's secondary color.
  await session.click(
    picker.locator(".lyra-color-picker__swatches .lyra-color-swatch").nth(1),
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

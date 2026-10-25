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

const SVG = path.join(import.meta.dirname, "files", "tank.svg");

/**
 * Still for docs image `console/schematics/symbol-color-regions`: the create symbol
 * dialog with an SVG loaded, its Colors list showing the stroke and fill of each
 * region. The last captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await setup(session);

  await session.click(capture.toolbarButton(page, "schematic", "add"), { zoom: false });
  const modal = page.locator(capture.MODAL).first();
  const drop = page.locator(".console-schematic-file-drop").first();
  await session.waitFor(drop);
  // The Console opens a detached file input; the chooser event hands it the file.
  const chooser = page.waitForEvent("filechooser");
  await session.click(drop, { zoom: false });
  await (await chooser).setFiles(SVG);
  await session.waitFor(page.locator(".console-schematic-region-list-item").first());
  await session.settle(800);
  const box = await modal.boundingBox();
  const view = page.viewportSize();
  if (box == null || view == null)
    throw new Error("the create symbol dialog is not on screen");
  // Beside the dialog, clear of the Colors list.
  await session.moveTo({
    x: Math.min(box.x + box.width + 40, view.width - 16),
    y: box.y + box.height / 2,
  });

  session.startRecording();
  await session.zoom(modal);
  await session.hold(2500);
};

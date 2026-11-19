// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { ASSET, fillUpload, siftScript } from "./common/sift";

export const desktop = true;

const PLOT = "Coldflow 11 Pressures";
const CHANNELS = ["ox_pt_1", "ox_pt_2"];

/**
 * Docs `reference/ranges/sift` (Upload data): select a region of the coldflow pressure
 * plot, upload two of its channels to a Sift asset from the context menu of the
 * selection, and find the finished upload in the status list.
 */
export default siftScript(async (session) => {
  const { page } = session;
  await session.click(page.getByText("Review", { exact: true }).first());
  const plot = page.locator(".pluto-line-plot:visible").first();
  await session.waitFor(plot);
  await session.moveTo({ x: 800, y: 450 });
  await session.settleWall(1500);
  await session.settle(600);
  const box = await plot.boundingBox();
  if (box == null) throw new Error("the line plot is hidden");
  const at = (fx: number, fy: number): { x: number; y: number } => ({
    x: box.x + box.width * fx,
    y: box.y + box.height * fy,
  });

  session.startRecording();
  await session.hold(1000);
  await session.zoom(plot, 1.2);
  await session.hold(600);
  await page.keyboard.down("Alt");
  await session.drag(at(0.35, 0.3), at(0.65, 0.7));
  await page.keyboard.up("Alt");
  await session.hold(500);
  await session.rightClick(at(0.5, 0.5));
  const menu = page.locator(".lyra-menu-context").first();
  await session.waitFor(menu);
  await session.hold(400);
  const item = menu.getByText("Upload region to Sift", { exact: true }).first();
  await session.click(item, { text: true });
  session.endZoom();
  const modal = page.locator(capture.MODAL).last();
  await session.waitFor(modal.getByText(`Upload ${PLOT} to Sift`));
  await fillUpload(session, modal, "ox_pt", CHANNELS);
  await session.settleWall(1500);
  await session.hold(800);

  await capture.openToolbar(session, "status");
  const sent = page.getByText(new RegExp(`^Sent .+ to asset ${ASSET}\\.$`)).first();
  await session.waitFor(sent);
  await session.zoom(page.locator(".console-nav__drawer.lyra--location-left").first());
  await session.hold(3000);
  session.endZoom();
  await session.hold(1500);
});

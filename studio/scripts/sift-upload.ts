// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { ASSET, siftScript } from "./common/sift";

export const desktop = true;

const RUN = "Coldflow 11";
const CHANNELS = ["ox_pt_1", "ox_pt_2", "ox_tc_1", "ox_tc_2"];

/**
 * Docs `reference/ranges/sift` (Upload data) and the Sift section of the v0.62 release
 * notes: a test engineer right-clicks a finished coldflow run in the Ranges toolbar,
 * uploads four channels of it to a Sift asset, and watches the upload finish in the
 * range overview.
 */
export default siftScript(async (session) => {
  const { page } = session;
  await session.click(page.getByText("Review", { exact: true }).first());
  await session.waitFor(page.locator(".pluto-line-plot:visible").first());
  await capture.openToolbar(session, "range");
  await capture.resizeToolbar(session, 300);
  const toolbar = page.locator(".console-nav__drawer.lyra--location-left").first();
  const run = toolbar.getByText(RUN, { exact: true }).first();
  await session.waitFor(run);
  await session.moveTo({ x: 26, y: 600 });
  await session.settleWall(1500);
  await session.settle(600);

  session.startRecording();
  await session.zoom(toolbar);
  await session.hold(1200);
  await capture.contextMenu(session, run, "Upload to Sift");
  const modal = page.locator(capture.MODAL).last();
  await session.waitFor(modal.getByText(`Upload ${RUN} to Sift`));
  // The modal zooms in as it opens, and the camera frames its box at the start.
  await session.settle(300);
  await session.zoom(modal);
  const asset = modal.getByText("Select asset", { exact: true }).first();
  await session.waitFor(asset);
  await session.click(asset, { text: true });
  await session.settleWall(1200);
  const tpc = page.getByRole("option", { name: ASSET, exact: true }).first();
  await session.waitFor(tpc);
  await session.click(tpc, { text: true });
  const channels = modal.getByText("Select channels to upload").first();
  await session.click(channels, { text: true });
  await session.hold(500);
  await session.type("ox_");
  for (const name of CHANNELS) {
    const option = page.getByRole("option", { name, exact: true }).first();
    await session.waitFor(option);
    await session.click(option, { text: true });
    await session.hold(250);
  }
  await page.keyboard.press("Escape");
  // The camera followed the clicks in the channel list, so frame the modal again.
  await session.zoom(modal);
  await session.hold(900);
  await session.click(modal.getByRole("button", { name: "Upload", exact: true }));
  session.endZoom();
  await session.settleWall(1500);
  await session.hold(1200);
  await session.click(run, { text: true });
  const uploads = page.getByText("Sift uploads", { exact: true }).first();
  await session.waitFor(uploads);
  await session.waitFor(page.getByText(new RegExp(`^Sent .+ to asset ${ASSET}\\.$`)));
  await session.zoom(uploads.locator("xpath=../.."));
  await session.settleWall(3000);
  await session.hold(2500);
  session.endZoom();
  await session.hold(2000);
});

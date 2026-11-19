// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const SIFT_PORT = 50551;
const RUN = "Coldflow 11";
const CHANNELS = ["ox_pt_1", "ox_pt_2", "ox_tc_1", "ox_tc_2"];

/**
 * Video for the Sift section of the v0.62 release notes: a test engineer right-clicks a
 * finished coldflow run in the Ranges toolbar, uploads four channels of it to a Sift
 * asset, and watches the upload finish in the range overview.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const sift = await fixtures.simulator({ protocol: "sift", port: SIFT_PORT });
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    const [rack] = await client.racks.retrieve({ integrations: "sift" });
    const key = await client.secrets.set({
      source: "stored",
      name: "Sift API key",
      value: "studio",
    });
    await client.devices.create({
      key: "sift",
      name: "Sift",
      make: "sift",
      model: "sift",
      location: `localhost:${SIFT_PORT}`,
      rack: rack.key,
      configured: true,
      properties: { uri: `localhost:${SIFT_PORT}`, apiKey: key.key },
    });

    await capture.login(session, world.project);
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
    await session.zoom(modal);
    const asset = modal.getByText("Select asset", { exact: true }).first();
    await session.waitFor(asset);
    await session.click(asset, { text: true });
    await session.settleWall(1200);
    const tpc = page.getByRole("option", { name: "TPC", exact: true }).first();
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
    await session.hold(600);
    await session.click(modal.getByRole("button", { name: "Upload", exact: true }));
    session.endZoom();
    await session.settleWall(1500);
    await session.hold(1200);
    await session.click(run, { text: true });
    const uploads = page.getByText("Sift uploads", { exact: true }).first();
    await session.waitFor(uploads);
    await session.zoom(uploads.locator("xpath=../.."));
    await session.settleWall(3000);
    await session.hold(2500);
    session.endZoom();
    await session.hold(2000);
  } finally {
    await client.close();
    await world.stop();
    await sift.stop();
  }
};

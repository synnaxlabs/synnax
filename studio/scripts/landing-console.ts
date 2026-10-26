// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

import { startColdflow } from "./common/coldflow";

/**
 * The landing page hero still: the TPC stand's Operations panel mid-coldflow, with the
 * OX pressures regulating on an annotated plot and the Arc sequence open beside it.
 * The last captured frame is the still.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const world = await fixtures.testStand({ now: () => session.now() });
  const client = fixtures.connect();
  try {
    const { page } = session;
    await capture.login(session, world.project);
    await startColdflow(session, world, client);
    await session.settle(9000);

    const plot = page
      .locator(".pluto-mosaic__leaf:visible")
      .filter({ hasText: "OX Pressures" })
      .locator(".pluto-line-plot")
      .first();
    const box = await plot.boundingBox();
    if (box == null) throw new Error("the OX Pressures plot is not on screen");
    await session.moveTo({ x: box.x + box.width * 0.82, y: box.y + box.height * 0.6 });

    session.startRecording();
    await session.hold(300);
  } finally {
    await client.close();
    await world.stop();
  }
};

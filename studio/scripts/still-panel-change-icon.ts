// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type capture } from "@/index";

import { createStand, pill } from "./common/panels";

export const viewport = { width: 1280, height: 800 };

/**
 * Still for docs image `console/panels/change-icon`: the icon picker of a panel, open
 * from the context menu of the panel selector, with a color chosen.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const stand = await createStand(session);
  try {
    const { page } = session;
    await session.rightClick(pill(session, "Test stand"), { text: true, zoom: false });
    const menu = page.locator(".lyra-menu-context").first();
    await session.waitFor(menu);
    await session.click(menu.getByText("Change icon", { exact: true }).first(), {
      zoom: false,
    });
    const picker = page.locator(".console-appearance-dialog").first();
    await session.waitFor(picker);
    await session.click(picker.locator(".lyra-color-swatch").nth(4), { zoom: false });
    await session.settle(400);
    // On the empty part of the main nav, so nothing on screen shows its hover state.
    await session.moveTo({ x: 26, y: 600 });
    await session.settleWall(1500);
    await session.settle(1200);
    await session.still("window");
  } finally {
    await stand.stop();
  }
};

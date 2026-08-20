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
 * Docs `console/schematics/create`: from an empty panel, open the component
 * selector with "+" and pick Schematic; an empty canvas opens in edit mode
 * with the symbol library below it.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session, { username: "synnax", password: "seldon" });
  await capture.clearPanel(session);
  await session.moveTo({ x: 756, y: 500 });

  session.startRecording();
  await session.hold(1200);

  await capture.clickPanelCreate(session);
  await session.hold(800);
  await capture.createComponent(session, "Schematic");
  await session.waitFor(page.locator(".pluto-diagram").first());
  await session.hold(1000);

  const symbols = page.getByText("Symbols", { exact: true }).first();
  if (!(await symbols.isVisible().catch(() => false)))
    await session.click(page.locator(".console-main-nav__item").last(), {
      zoom: false,
    });
  await session.waitFor(symbols);
  await session.hold(3000);
};

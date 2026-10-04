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
 * createComponentScript returns a script that clicks the mosaic "+" on an empty panel
 * and picks a component from the selector, which opens in its place as a tab. `ready`
 * is a selector for the opened component.
 */
export const createComponentScript =
  (title: string, ready: string) =>
  async (session: capture.CaptureSession): Promise<void> => {
    const { page } = session;
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1800);

    await capture.clickPanelCreate(session);
    const selector = page.locator(".console-layout-selector__frame").first();
    await session.waitFor(selector);
    // The selector fills the panel, so zoom on a point over its buttons.
    const box = await selector.locator(".console-layout-selector__items").boundingBox();
    if (box == null) throw new Error("layout selector buttons are not on screen");
    await session.zoom({ x: box.x + box.width / 2, y: box.y + box.height / 2 }, 1.4);
    await session.hold(800);
    await capture.createComponent(session, title);
    session.endZoom();
    await session.waitFor(page.locator(ready).first());
    await session.hold(2800);
  };

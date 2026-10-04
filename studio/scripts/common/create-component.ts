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
    // The selector frame fills the panel, so a zoom on it renders at 1x. Zoom on the
    // center of its options instead.
    const options = page.locator(".console-layout-selector__frame").getByRole("button");
    await session.waitFor(options.first());
    const [first, last] = await Promise.all([
      options.first().boundingBox(),
      options.last().boundingBox(),
    ]);
    if (first == null || last == null) throw new Error("selector options are hidden");
    await session.zoom(
      {
        x: (first.x + last.x + last.width) / 2,
        y: (first.y + last.y + last.height) / 2,
      },
      1.5,
    );
    await session.hold(800);
    await capture.createComponent(session, title);
    session.endZoom();
    await session.waitFor(page.locator(ready).first());
    await session.hold(2800);
  };

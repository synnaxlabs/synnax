// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, director } from "@/index";

const ZOOM = 1.5;

/**
 * pickComponent zooms on the open "Create component" selector's options and picks
 * `title`, then waits for `ready`, a selector for the opened component. The zoom ends
 * at the pick.
 */
export const pickComponent = async (
  session: capture.CaptureSession,
  title: string,
  ready: string,
): Promise<void> => {
  const { page } = session;
  // The selector frame fills the panel, so a zoom on it renders at 1x. Zoom on the
  // center of its options instead.
  const options = page.locator(".console-layout-selector__frame").getByRole("button");
  await session.waitFor(options.first());
  const [first, last] = await Promise.all([
    options.first().boundingBox(),
    options.last().boundingBox(),
  ]);
  const target = await page
    .locator(".console-layout-selector__frame")
    .getByRole("button", { name: title })
    .first()
    .boundingBox();
  const viewport = page.viewportSize();
  if (first == null || last == null || target == null || viewport == null)
    throw new Error("selector options are hidden");
  // Shift the zoom off the options' center only as far as it takes to hold the picked
  // option inside the crop's margin, so the camera does not pan to reach it.
  const keep = (center: number, pos: number, size: number, span: number): number => {
    const half = span / ZOOM / 2;
    const lo = pos + size + director.ZOOM_RECT_MARGIN_PX - half;
    const hi = pos - director.ZOOM_RECT_MARGIN_PX + half;
    return Math.min(hi, Math.max(lo, center));
  };
  await session.zoom(
    {
      x: keep(
        (first.x + last.x + last.width) / 2,
        target.x,
        target.width,
        viewport.width,
      ),
      y: keep(
        (first.y + last.y + last.height) / 2,
        target.y,
        target.height,
        viewport.height,
      ),
    },
    ZOOM,
  );
  await session.hold(800);
  await capture.createComponent(session, title);
  session.endZoom();
  await session.waitFor(page.locator(ready).first());
};

/**
 * createComponentScript returns a script that clicks the mosaic "+" on an empty panel
 * and picks a component from the selector, which opens in its place as a tab. `ready`
 * is a selector for the opened component.
 */
export const createComponentScript =
  (title: string, ready: string) =>
  async (session: capture.CaptureSession): Promise<void> => {
    await capture.login(session);
    await capture.clearPanel(session);
    await session.moveTo({ x: 800, y: 450 });

    session.startRecording();
    await session.hold(1800);

    await capture.clickPanelCreate(session);
    await pickComponent(session, title, ready);
    await session.hold(2800);
  };

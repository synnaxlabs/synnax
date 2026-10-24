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
 * Docs `control/arc/get-started/create-automation`: from an empty panel, open
 * the component selector with "+", pick Arc automation, name it, choose the
 * Text editor mode, and create it; the Arc text editor opens.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
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
  await capture.createComponent(session, "Arc automation");

  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(600);
  await session.zoom(modal, 1.6);

  await session.click(name);
  await session.type("Tank Pressure Guard");
  await session.hold(600);

  await session.click(
    modal
      .locator(".console-arc-create-modal__mode-select-button")
      .filter({ hasText: "Text" })
      .first(),
  );
  await session.hold(800);

  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(modal);
  await session.waitFor(page.locator(".pluto-editor").first());
  await session.hold(3000);
};

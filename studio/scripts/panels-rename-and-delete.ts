// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import {
  createPanel,
  createStand,
  NEW_PANEL,
  pill,
  renamePanel,
  selectPanel,
  STAND,
  zoomSelector,
} from "./common/panels";

const DRAFT = "Draft";

/**
 * Docs `console/panels/rename-and-delete`: double-click New panel, rename it Test
 * stand, then right-click the Draft panel, select Delete, and confirm.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // The stand takes its New panel name after Draft exists, because Draft also starts
  // as New panel.
  const stand = await createStand(session);
  try {
    const { page } = session;
    await createPanel(session, DRAFT);
    await renamePanel(session, STAND, NEW_PANEL);
    await selectPanel(session, NEW_PANEL);
    await session.moveTo(capture.PARK);

    session.startRecording();
    await session.hold(1800);

    await zoomSelector(session);
    await session.doubleClick(pill(session, NEW_PANEL), { text: true, zoom: false });
    const editable = page
      .locator(".lyra-text--editable[contenteditable='true']")
      .first();
    await session.waitFor(editable);
    await session.hold(300);
    await session.press("ControlOrMeta+a");
    await session.type(STAND);
    await session.hold(300);
    await session.press("Enter");
    await session.waitFor(pill(session, STAND));
    await session.hold(1000);

    session.endZoom();
    await capture.contextMenu(session, pill(session, DRAFT), /^Delete/);
    const dialog = page.locator(capture.MODAL).first();
    await session.waitFor(dialog);
    await session.zoom(dialog);
    await session.hold(800);
    await session.click(dialog.getByRole("button", { name: "Delete", exact: true }));
    session.endZoom();
    await session.waitForHidden(pill(session, DRAFT));
    await session.moveTo(capture.PARK);
    await session.hold(2800);
  } finally {
    await stand.stop();
  }
};

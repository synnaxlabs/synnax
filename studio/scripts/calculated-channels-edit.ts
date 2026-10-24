// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { create, PARK, setup } from "./common/calculated-channel";

/**
 * Docs `client/calculated-channels/edit`: right-click a calculated channel in the
 * Channels toolbar, select Edit calculation, change the expression, select the Max
 * operation with a 15s Window, and click Save.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const m = await setup(session);
  // Off-camera: startRecording drops the input events of the create.
  const item = await create(session, m);
  await session.moveTo({ x: 800, y: 450 });
  await session.settle(1000);

  session.startRecording();
  await session.hold(1500);

  await capture.contextMenu(session, item, "Edit calculation");
  await session.waitFor(m.editor);
  await m.zoom();
  await session.hold(300);

  await m.focusEditor();
  await session.press("ControlOrMeta+a");
  await session.hold(300);
  await session.press("Delete");
  await session.hold(300);
  await m.typeExpression(" / 2");
  await m.blurEditor();
  await session.hold(300);

  await session.click(m.modal.getByText("Max", { exact: true }).first(), {
    text: true,
  });
  const windowInput = m.modal.locator(".console-operations__window input").first();
  await session.waitFor(windowInput);
  await session.hold(300);
  await session.click(windowInput);
  await session.press("ControlOrMeta+a");
  await session.hold(300);
  await session.press("Delete");
  await session.hold(300);
  await session.type("15", 40);
  await session.press("Enter");
  await session.hold(500);

  await capture.clickButton(session, "Save");
  session.endZoom();
  await session.waitForHidden(m.editor);
  await session.moveTo(PARK);
  await session.hold(3000);
};

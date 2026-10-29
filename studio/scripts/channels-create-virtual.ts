// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { kindButton, selectItem } from "./common/channel-dialog";

const NAME = "start_cmd";

/**
 * Docs `client/channels/create-virtual`: click the "+" action in the Channels toolbar,
 * name the channel start_cmd, select Virtual so the Index field hides, and select the
 * Boolean data type. Create puts start_cmd in the toolbar.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  const { page } = session;
  await capture.login(session);

  await capture.clearPanel(session);
  await capture.openToolbar(session, "channel");
  await capture.resizeToolbar(session, 400);
  await session.moveTo({ x: 800, y: 450 });

  session.startRecording();
  await session.hold(1500);

  await capture.clickToolbarCreate(session);
  const modal = page.locator(capture.MODAL).first();
  const name = modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(600);
  await session.zoom(modal);

  await session.click(name);
  await session.type(NAME);
  await session.hold(400);
  await session.click(kindButton(modal, "Virtual"));
  // Long enough for the reader to see the Index field hide.
  await session.hold(1200);
  await session.click(
    modal.locator(".lyra-dialog__trigger").filter({ hasText: "float32" }).first(),
  );
  // Boolean sits below the list's visible end, so filter to it.
  await session.hold(400);
  await session.type("Boolean");
  await selectItem(session, "Boolean");
  await session.hold(500);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(name);
  await session.waitFor(
    page.locator(".console-nav__drawer").getByText(NAME, { exact: true }),
  );
  await session.moveTo({ x: 1180, y: 680 });
  await session.hold(2800);
};

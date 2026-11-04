// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@/index";

import { kindButton, selectItem, switchFor } from "./common/channel-dialog";

/**
 * Docs `client/channels/create`: click the "+" action in the Channels toolbar, create
 * the index channel tank_time with "Create more" on, then create tank_pressure as a
 * float32 channel on that index. Both channels land in the toolbar.
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
  await session.type("tank_time");
  await session.hold(400);
  await session.click(kindButton(modal, "Index"));
  await session.hold(500);
  await session.click(switchFor(modal, "Create more"));
  await session.hold(500);
  await capture.clickButton(session, "Create");
  await session.waitFor(
    page.locator(".console-nav__drawer").getByText("tank_time", { exact: true }),
  );
  await session.hold(800);

  await session.click(name);
  await session.type("tank_pressure");
  await session.hold(400);
  await session.click(
    modal.locator(".lyra-dialog__trigger").filter({ hasText: "float32" }).first(),
  );
  await selectItem(session, "float32");
  await session.click(modal.getByText("Select channel", { exact: true }).first(), {
    text: true,
  });
  await selectItem(session, "tank_time");
  await session.hold(600);
  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitFor(
    page.locator(".console-nav__drawer").getByText("tank_pressure", { exact: true }),
  );
  await session.hold(800);

  await session.click(modal.getByRole("button", { name: "Close" }).first());
  await session.waitForHidden(name);
  await session.hold(2800);
};

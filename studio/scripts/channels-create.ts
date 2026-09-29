// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture } from "@/index";

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
  await session.hold(1000);

  await capture.clickToolbarCreate(session);
  const modal = page.locator(".console-modal").first();
  const name = modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(600);
  await session.zoom(modal);

  await session.click(name);
  await session.type("tank_time");
  await session.hold(400);
  await session.click(switchFor(modal, "Is index"));
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
    modal.locator(".pluto-dialog__trigger").filter({ hasText: "float32" }).first(),
  );
  await selectItem(session, "float32");
  await session.click(modal.getByText("Select channel", { exact: true }).first(), {
    text: true,
  });
  await selectItem(session, "tank_time");
  await session.hold(600);
  await capture.clickButton(session, "Create");
  await session.waitFor(
    page.locator(".console-nav__drawer").getByText("tank_pressure", { exact: true }),
  );
  await session.hold(800);

  await session.click(modal.getByRole("button", { name: "Close" }).first());
  await session.waitForHidden(name);
  session.endZoom();
  await session.hold(2400);
};

/** switchFor returns the switch input beside a label in the create dialog. */
const switchFor = (modal: Locator, label: string): Locator =>
  modal.getByText(label, { exact: true }).locator("..").locator("input").first();

/** selectItem clicks an option in the open dropdown list. */
const selectItem = async (
  session: capture.CaptureSession,
  text: string,
): Promise<void> => {
  const item = session.page
    .locator(".pluto-list__item:not(.pluto-tree__item)")
    .filter({ has: session.page.getByText(text, { exact: true }) })
    .first();
  await session.waitFor(item);
  await session.hold(600);
  await session.click(item.getByText(text, { exact: true }).first(), { text: true });
};

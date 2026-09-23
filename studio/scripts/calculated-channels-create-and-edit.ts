// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture, fixtures } from "@/index";

const NAME = "cpu_calc";
/**
 * Docs `client/calculated-channels/create-and-edit`: from the Channels toolbar, create a
 * calculated channel over a live metrics channel, then right-click it, edit the
 * expression, and set a Max operation over a 15s window.
 */
export default async (session: capture.CaptureSession): Promise<void> => {
  // Off-camera: the shot creates the channel itself, so drop any earlier copy.
  await fixtures.removeChannels([NAME, `${NAME}_time`]);
  const { page } = session;
  await capture.login(session);

  const placeholder = capture.tab(page, "Create component");
  if (await placeholder.isVisible().catch(() => false)) {
    await placeholder.hover();
    await placeholder.locator(".pluto-tabs__close").first().click();
  }
  await capture.openToolbar(session, "channel");
  await capture.resizeToolbar(session, 500);
  await session.moveTo({ x: 756, y: 500 });

  const modal = page.locator(".console-modal");
  const editor = modal.locator(".console-calculated-editor").first();

  session.startRecording();
  await session.hold(500);

  // The calculated action is the first of the toolbar's two create actions.
  const action = page.locator(".console-nav__drawer .console-toolbar__action").first();
  await session.moveTo(action);
  await session.hold(300);
  await session.click(action);
  const name = modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await session.hold(300);

  await session.click(name);
  await session.type(NAME, 40);
  await session.hold(300);

  // Type the channel prefix and let the editor's suggestion complete the name.
  const typeExpression = async (tail: string, tailMs = 40): Promise<void> => {
    await session.type("return sy_node_1", 40);
    await session.hold(500);
    await session.press("Tab");
    await session.type(tail, tailMs);
  };

  // Slower here so the channel name has time to highlight before Create.
  await session.click(editor);
  await typeExpression(" * 2", 10);
  // Drop editor focus instantly so the caret does not blink through the hold.
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
  await session.hold(800);

  await capture.clickButton(session, "Create");
  await session.waitForHidden(name);
  const item = capture.treeItem(page, "channel:", NAME);
  await session.waitFor(item);
  await session.hold(600);

  await session.rightClick(item, { zoom: false });
  const menu = page.locator(".pluto-menu-context").first();
  await session.waitFor(menu);
  await session.hold(300);
  const edit = menu.getByText("Edit calculation", { exact: true }).first();
  await session.moveTo(edit);
  await session.hold(500);
  await session.click(edit, { zoom: false });
  await session.waitFor(editor);
  await session.hold(300);

  await session.click(editor);
  await session.press("ControlOrMeta+a");
  await session.hold(300);
  await session.press("Delete");
  await session.hold(300);
  await typeExpression(" / 2");
  await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
  await session.hold(300);

  await session.click(modal.getByText("Max", { exact: true }).first(), { text: true });
  const windowInput = modal.locator(".console-operations__window input").first();
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
  await session.waitForHidden(editor);
  await session.hold(3000);
};

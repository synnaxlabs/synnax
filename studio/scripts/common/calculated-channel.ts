// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { capture, fixtures } from "@/index";

/** NAME is the calculated channel the shots create and edit. */
const NAME = "cpu_calc";

/** A point in the empty panel with no control under it, for the end of a shot. */
export const PARK = { x: 1180, y: 680 };

/** Locators and actions for the calculated channel modal. */
export interface Modal {
  modal: Locator;
  editor: Locator;
  /** zoom frames the modal, which is too large for its fit to zoom. */
  zoom: () => Promise<void>;
  /**
   * focusEditor clicks the expression editor once it loads and waits for it to take
   * focus. Keys typed before then fire the Console's single-key shortcuts.
   */
  focusEditor: () => Promise<void>;
  /** blurEditor drops editor focus at once, so no caret blinks through a hold. */
  blurEditor: () => Promise<void>;
  /**
   * typeExpression types the channel prefix, accepts the editor's suggestion with Tab,
   * and types the tail.
   */
  typeExpression: (tail: string, tailMs?: number) => Promise<void>;
}

/**
 * setup logs in with the Channels toolbar open and no calculated channel in the Core,
 * and returns the modal helpers.
 */
export const setup = async (session: capture.CaptureSession): Promise<Modal> => {
  await fixtures.removeChannels([NAME, `${NAME}_time`]);
  const { page } = session;
  await capture.login(session);

  const placeholder = capture.tab(page, "Create component");
  if (await placeholder.isVisible().catch(() => false)) {
    await placeholder.hover();
    await placeholder.locator(".lyra-tabs__close").first().click();
  }
  await capture.openToolbar(session, "channel");
  await capture.resizeToolbar(session, 500);
  await session.moveTo({ x: 800, y: 450 });

  const modal = page.locator(capture.MODAL);
  const editor = modal.locator(".console-calculated-editor").first();
  return {
    modal,
    editor,
    focusEditor: async () => {
      const mounted = editor.locator(".monaco-editor");
      // The editor loads on wall time. Off camera, ticks pass too fast for it to load.
      for (let i = 0; i < 50 && !(await mounted.isVisible()); i++)
        await session.settleWall(200);
      await session.waitFor(mounted);
      await session.click(editor);
      await session.waitFor(editor.locator(".monaco-editor.focused"));
    },
    blurEditor: async () =>
      await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur()),
    zoom: async () => await capture.zoomCenter(session, modal.first(), 1.4),
    typeExpression: async (tail, tailMs = 40) => {
      await session.type("return sy_node_1", 40);
      await session.hold(500);
      await session.press("Tab");
      await session.type(tail, tailMs);
    },
  };
};

/**
 * create clicks the create calculated channel button in the Channels toolbar, enters
 * NAME and an expression, and clicks Create. It returns the new channel's toolbar item.
 */
export const create = async (
  session: capture.CaptureSession,
  m: Modal,
): Promise<Locator> => {
  const { page } = session;
  // The calculated action is the first of the toolbar's two create actions.
  const action = page.locator(".console-nav__drawer .console-toolbar__action").first();
  await session.moveTo(action);
  await session.hold(300);
  await session.click(action);
  const name = m.modal.getByPlaceholder("Name").first();
  await session.waitFor(name);
  await m.zoom();
  await session.hold(300);

  await session.click(name);
  await session.type(NAME, 40);
  await session.hold(300);

  // Slower here so the channel name has time to highlight before Create.
  await m.focusEditor();
  await m.typeExpression(" * 2", 10);
  await m.blurEditor();
  await session.hold(800);

  await capture.clickButton(session, "Create");
  session.endZoom();
  await session.waitForHidden(name);
  const item = capture.treeItem(page, "channel:", NAME);
  await session.waitFor(item);
  return item;
};

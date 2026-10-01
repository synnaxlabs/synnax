// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator, type Page } from "playwright";

import { type CaptureSession } from "@/capture/rig";

/**
 * control returns a visualization control button (zoom, pan, selection, rule,
 * ...) by the name of the icon it carries. The buttons have no accessible
 * name of their own.
 */
export const control = (page: Page, icon: string): Locator =>
  page.locator(`.console-controls button:has(.pluto-icon--${icon})`).first();

/**
 * showVisualizationToolbar opens the bottom visualization drawer if it is not
 * already visible. The toggle is the last main-nav item: it carries the
 * focused tab's icon, so it has no stable id of its own.
 */
export const showVisualizationToolbar = async (
  session: CaptureSession,
): Promise<void> => {
  const { page } = session;
  const y1 = page.locator("label").filter({ hasText: "Y1" }).first();
  if (await y1.isVisible().catch(() => false)) return;
  await session.click(page.locator(".console-main-nav__item").last(), {
    zoom: false,
  });
  await session.waitFor(y1);
};

export interface AddChannelsOptions {
  /** Query typed into the channel search before selecting. */
  search?: string;
}

/**
 * addChannels opens the plot's Data tab and selects channels onto an axis, as
 * recorded input. Mirrors integration/console/plot.py `add_channels`, adapted
 * to the redesigned toolbar.
 */
export const addChannels = async (
  session: CaptureSession,
  axis: "Y1" | "Y2" | "X1",
  channels: string[],
  { search }: AddChannelsOptions = {},
): Promise<void> => {
  const { page } = session;
  await showVisualizationToolbar(session);
  const trigger = page
    .locator("label")
    .filter({ hasText: axis })
    .locator("..")
    .locator(".pluto-dialog__trigger")
    .first();
  // Wide rows are mostly empty space: click their left-aligned text so the
  // cursor lands where a person aims and the camera gets a rect worth framing.
  const placeholder = trigger.getByText(/Select channel/).first();
  if (await placeholder.isVisible().catch(() => false))
    await session.click(placeholder);
  else await session.click(trigger);
  const input = page.locator("input[placeholder*='Search']");
  await session.waitFor(input);
  // Let the dropdown land before typing so the viewer can register it.
  await session.hold(600);
  if (search != null) await session.type(search);
  for (const channel of channels) {
    const item = page
      .locator(".pluto-list__item:not(.pluto-tree__item)")
      .filter({ hasText: channel })
      .first();
    await session.waitFor(item);
    await session.click(item.getByText(channel, { exact: true }).first(), {
      text: true,
    });
  }
  await session.press("Escape");
};

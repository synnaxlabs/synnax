// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Locator } from "playwright";

import { type capture } from "@/index";

/** CHANNEL is the selector of each channel in a Modbus task form. */
export const CHANNEL = ".console-channel-item";

/**
 * choose opens the select in a channel that shows `current`, searches, and picks
 * `next`. The selects of a Modbus channel have no label, so the value finds them.
 */
export const choose = async (
  session: capture.CaptureSession,
  channel: Locator,
  current: string,
  next: string,
): Promise<void> => {
  await session.click(
    channel.locator(".lyra-dialog__trigger").filter({ hasText: current }).first(),
    { text: true },
  );
  await session.hold(300);
  await session.type(next);
  const item = session.page
    .locator(".lyra-list__item:not(.lyra-tree__item)")
    .filter({ has: session.page.getByText(next, { exact: true }) })
    .first();
  await session.waitFor(item);
  await session.hold(500);
  await session.click(item.getByText(next, { exact: true }).first(), { text: true });
  await session.hold(400);
};

/** setAddress replaces the address of a channel. */
export const setAddress = async (
  session: capture.CaptureSession,
  channel: Locator,
  address: string,
): Promise<void> => {
  await session.click(channel.locator("input").first());
  await session.press("ControlOrMeta+a");
  await session.type(address);
  await session.press("Enter");
  await session.hold(300);
};

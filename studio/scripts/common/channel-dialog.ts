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

/** switchFor returns the switch input beside a label in the create channel dialog. */
export const switchFor = (modal: Locator, label: string): Locator =>
  modal.getByText(label, { exact: true }).locator("..").locator("input").first();

/** kindButton returns the button that picks a kind in the create channel dialog. */
export const kindButton = (
  modal: Locator,
  kind: "Data" | "Index" | "Virtual",
): Locator => modal.getByRole("button", { name: kind, exact: true });

/** selectItem clicks an option in the open dropdown list. */
export const selectItem = async (
  session: capture.CaptureSession,
  text: string,
): Promise<void> => {
  const item = session.page
    .locator(".lyra-list__item:not(.lyra-tree__item)")
    .filter({ has: session.page.getByText(text, { exact: true }) })
    .first();
  await session.waitFor(item);
  await session.hold(600);
  await session.click(item.getByText(text, { exact: true }).first(), { text: true });
};

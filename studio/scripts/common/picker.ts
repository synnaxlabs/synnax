// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Locator } from "playwright";

import { capture } from "@/index";

/** pickValues checks the values with the given names in the pane of the dialog. */
export const pickValues = async (
  session: capture.CaptureSession,
  dialog: Locator,
  names: string[],
): Promise<void> => {
  for (const name of names) {
    const check = dialog.getByRole("checkbox", { name: `Record ${name}`, exact: true });
    await session.waitFor(check);
    await session.hold(400);
    await session.click(check);
  }
  await session.hold(800);
};

/** submitPicks clicks the button that adds the picks, then frames the task form. */
export const submitPicks = async (
  session: capture.CaptureSession,
  dialog: Locator,
  form: Locator,
  label: string,
): Promise<void> => {
  await session.click(dialog.getByRole("button", { name: label, exact: true }));
  session.endZoom();
  await session.waitForHidden(dialog);
  await capture.zoomCenter(
    session,
    form.locator(".console-task-channel-form-container"),
    1.35,
  );
  await session.hold(800);
};

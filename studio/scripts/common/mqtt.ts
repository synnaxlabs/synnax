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

/** openDiscover clicks the button that opens the discovery dialog and zooms on it. */
export const openDiscover = async (
  session: capture.CaptureSession,
  form: Locator,
  label: string,
): Promise<Locator> => {
  const { page } = session;
  await session.click(form.getByRole("button", { name: label, exact: true }).first());
  const dialog = page
    .locator(capture.MODAL)
    .filter({ has: page.getByRole("button", { name: "JSON topics", exact: true }) })
    .first();
  await session.waitFor(dialog);
  await session.zoom(dialog);
  return dialog;
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor, within } from "@testing-library/react";

import { findDialogTriggerByText } from "@/testutil";

/**
 * Opens the message format field that shows current, and searches for an ICD name.
 * @returns the dialog.
 */
export const searchMessageFormats = async (current: string, icdName: string) => {
  fireEvent.click(await findDialogTriggerByText(current));
  const dialogs = await screen.findAllByRole("dialog");
  const dialog = within(dialogs[dialogs.length - 1]);
  fireEvent.change(await dialog.findByPlaceholderText("Search message formats..."), {
    target: { value: icdName },
  });
  return dialog;
};

/**
 * Picks the message next of the ICD named icdName in the message format field that
 * shows current.
 */
export const pickMessageFormat = async (
  current: string,
  next: string,
  icdName: string,
): Promise<void> => {
  const dialog = await searchMessageFormats(current, icdName);
  // waitFor resolves a timer tick after its match. A render in that tick can replace
  // the option. A click in the callback goes to the option on the page.
  await waitFor(() => {
    const [match] = dialog
      .getAllByText(next)
      .filter((n) => n.parentElement?.textContent?.includes(icdName));
    if (match == null) throw new Error(`no ${next} of ${icdName}`);
    fireEvent.click(match);
  });
};

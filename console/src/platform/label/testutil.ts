// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";

import { findDialogTrigger } from "@/testutil";

/**
 * Opens the mounted label select dialog, narrows the paginated list down to name via
 * the dialog's search input, and clicks the matching label option.
 */
export const searchAndClickLabel = async (name: string): Promise<void> => {
  fireEvent.click(await findDialogTrigger());
  await searchAndClickLabelOption(name);
};

/**
 * Like searchAndClickLabel, but opens the last mounted label select trigger, for label
 * selects nested inside another open dialog (e.g. a filter menu).
 */
export const searchAndClickNestedLabel = async (name: string): Promise<void> => {
  const trigger = await waitFor(() => {
    const triggers = document.querySelectorAll<HTMLElement>(".pluto-dialog__trigger");
    if (triggers.length === 0) throw new Error("no dialog trigger found");
    return triggers[triggers.length - 1];
  });
  fireEvent.click(trigger);
  await searchAndClickLabelOption(name);
};

const searchAndClickLabelOption = async (name: string): Promise<void> => {
  const search = await waitFor(() => screen.getByPlaceholderText("Search labels..."));
  fireEvent.change(search, { target: { value: name } });
  const option = await waitFor(() => screen.getByText(name));
  fireEvent.click(option);
};

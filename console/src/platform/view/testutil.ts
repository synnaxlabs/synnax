// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";

import { getBySelector, getIconButton } from "@/testutil";

/**
 * Waits for the mounted View.Frame's edit toggle (which only appears once the async
 * update-permission check resolves) and clicks it, flipping the view into its editable
 * state.
 */
export const enableEditing = async (): Promise<void> => {
  const toggle = await waitFor(() => getIconButton(document.body, "edit"));
  fireEvent.click(toggle);
};

/**
 * Waits for the icon button with the given pluto icon name inside the mounted
 * View.Toolbar (which only renders once editing is enabled) and returns it.
 */
export const findToolbarIconButton = async (icon: string): Promise<HTMLButtonElement> =>
  await waitFor(() =>
    getIconButton(getBySelector(document.body, ".console-view__toolbar"), icon),
  );

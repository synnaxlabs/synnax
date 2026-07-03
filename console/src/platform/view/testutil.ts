// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";

import { getIconButton } from "@/testutil";

/**
 * Waits for the mounted View.Frame's edit toggle (which only appears once the async
 * update-permission check resolves) and clicks it, flipping the view into its editable
 * state.
 */
export const enableEditing = async (): Promise<void> => {
  const toggle = await waitFor(() => getIconButton(document.body, "edit"));
  fireEvent.click(toggle);
};

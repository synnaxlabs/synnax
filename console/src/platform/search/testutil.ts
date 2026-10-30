// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";

import { getIconButton } from "@/testutil";

/** Opens the filter menu beside the search input and clicks the item with the text. */
export const clickFilterItem = async (text: string): Promise<void> => {
  fireEvent.click(await waitFor(() => getIconButton(document.body, "filter")));
  fireEvent.click(await screen.findByText(text));
};

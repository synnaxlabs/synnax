// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen, within } from "@testing-library/react";

/** @returns the entry in the open problems view that shows the text. */
export const findProblem = async (text: string): Promise<HTMLElement> => {
  const found = (
    await within(screen.getByRole("tabpanel")).findByText(text)
  ).closest<HTMLElement>(".console-status-problem");
  if (found == null) throw new Error(`no problem shows ${text}`);
  return found;
};

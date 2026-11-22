// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { vi } from "vitest";
import { page, userEvent } from "vitest/browser";

import { Enumeration } from "@/enumeration";

/** Gives the forms below it a create prompt that never opens. */
export const CreatePromptStub = ({ children }: PropsWithChildren): ReactElement => (
  <Enumeration.CreatePromptProvider value={vi.fn()}>
    {children}
  </Enumeration.CreatePromptProvider>
);

/**
 * Picks the enumeration with the name in the Enumeration select of the rendered form.
 * The click finds the option only when it clicks, since the list moves its rows when
 * other specs create enumerations.
 */
export const pickEnumeration = async (name: string): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Enumeration" }));
  fireEvent.change(screen.getByPlaceholderText("Search enumerations..."), {
    target: { value: name },
  });
  await userEvent.click(page.getByRole("option", { name, exact: true }));
};

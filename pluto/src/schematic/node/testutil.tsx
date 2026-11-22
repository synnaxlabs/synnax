// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import type { PropsWithChildren, ReactElement } from "react";
import { expect, vi } from "vitest";

import { Enumeration } from "@/enumeration";

/** Gives the forms below it a create prompt that never opens. */
export const CreatePromptStub = ({ children }: PropsWithChildren): ReactElement => (
  <Enumeration.CreatePromptProvider value={vi.fn()}>
    {children}
  </Enumeration.CreatePromptProvider>
);

/**
 * Picks the enumeration with the given name in the Enumeration select of a node form.
 * The name must be unique, so the search answer lists only that enumeration.
 */
export const pickEnumeration = async (name: string): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Enumeration" }));
  fireEvent.change(screen.getByPlaceholderText("Search enumerations..."), {
    target: { value: name },
  });
  // An option of the unsearched list can leave the virtual window and remount before
  // a click lands. The match heads the search answer, so it stays mounted.
  await waitFor(() =>
    expect(screen.getAllByRole("option").map((o) => o.textContent)).toEqual([
      name,
      "No enumeration",
    ]),
  );
  fireEvent.click(screen.getByRole("option", { name }));
};

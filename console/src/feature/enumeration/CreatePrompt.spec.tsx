// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { Enumeration as PEnumeration } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { act, fireEvent, renderHook, screen, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Enumeration } from "@/feature/enumeration";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

interface Opened {
  /** Settles when the modal closes, with the key of the new enumeration or null. */
  result: Promise<enumeration.Key | null>;
}

/** Opens the prompt that the provider gives. */
const openPrompt = async (params: PEnumeration.CreatePromptParams): Promise<Opened> => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Console>
      <Triggers.Provider>
        <Enumeration.CreatePromptProvider>{children}</Enumeration.CreatePromptProvider>
        <Modals.Stack />
      </Triggers.Provider>
    </Console>
  );
  const { result } = renderHook(() => PEnumeration.useCreatePrompt(), { wrapper });
  const opened: Opened = { result: Promise.resolve(null) };
  // Modal content that suspends needs an awaited act on its first open.
  await act(async () => {
    opened.result = result.current(params);
  });
  return opened;
};

const nameField = async (): Promise<HTMLInputElement> =>
  await screen.findByLabelText<HTMLInputElement>("Enumeration name");

describe("Enumeration.CreatePromptProvider", () => {
  it("should create the enumeration with the typed name and the entries", async () => {
    const entries = [{ key: id.create(), name: "Off", value: 0, abnormal: false }];
    const { result } = await openPrompt({ name: "Pump", entries });
    const field = await nameField();
    expect(field.value).toBe("Pump");
    const name = `Pump ${id.create()}`;
    fireEvent.change(field, { target: { value: name } });
    fireEvent.click(screen.getByRole("button", { name: /^Create/ }));
    const key = await result;
    if (key == null) throw new Error("expected the key of the new enumeration");
    const e = await client.enumerations.retrieve(key);
    expect(e.name).toBe(name);
    expect(e.entries.map((en) => [en.name, en.value])).toEqual([["Off", 0]]);
    await waitFor(() => expect(screen.queryByLabelText("Enumeration name")).toBeNull());
  });

  it("should show an error at the name field on a Create with no name", async () => {
    await openPrompt({ name: "", entries: [] });
    const field = await nameField();
    fireEvent.change(field, { target: { value: " " } });
    fireEvent.click(screen.getByRole("button", { name: /^Create/ }));
    expect(await screen.findByText("Name is required")).toBeTruthy();
    fireEvent.change(field, { target: { value: "Pump" } });
    expect(screen.queryByText("Name is required")).toBeNull();
  });
});

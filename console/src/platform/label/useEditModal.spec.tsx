// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Label } from "@/platform/label";
import { Modals } from "@/platform/modals";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const Harness = (): ReactElement => {
  const open = Label.useEditModal();
  return <button onClick={() => open()}>open</button>;
};
Harness.displayName = "Harness";

const getAddButton = (): HTMLButtonElement => {
  const btn = document.querySelector<HTMLButtonElement>(".console-label__add-btn");
  if (btn == null) throw new Error("add button not found");
  return btn;
};

const openModal = async () => {
  const client = createTestClient();
  const { wrapper } = await createConsoleWrapper({ client });
  const result = render(
    <>
      <Harness />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return { ...result, client };
};

describe("Label.useEditModal", () => {
  it("should render the label editor with a search input", async () => {
    await openModal();
    await waitFor(() => expect(screen.getByText("Search labels")).toBeTruthy());
    expect(screen.getByText("Edit")).toBeTruthy();
  });

  it("should reveal the create form when the add button is clicked", async () => {
    await openModal();
    await waitFor(() => expect(screen.getByText("Search labels")).toBeTruthy());
    fireEvent.click(getAddButton());
    const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Label Name");
    expect(nameInput).toBeTruthy();
  });

  it("should persist a new label to the cluster from the create form", async () => {
    const { client } = await openModal();
    await waitFor(() => expect(screen.getByText("Search labels")).toBeTruthy());
    fireEvent.click(getAddButton());
    const name = id.create();
    const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Label Name");
    fireEvent.change(nameInput, { target: { value: name } });
    const check = screen.getByLabelText("pluto-icon--check").closest("button");
    if (check == null) throw new Error("check button not found");
    fireEvent.click(check);
    await waitFor(async () => {
      const found = await client.labels.retrieve({ names: [name] });
      expect(found.length).toBe(1);
    }, TIMEOUT);
  });
});

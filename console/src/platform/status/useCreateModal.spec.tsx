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

import { Modals } from "@/platform/modals";
import { Status } from "@/platform/status";
import { createConsoleWrapper } from "@/testutil";

const TIMEOUT = { timeout: 5000 };

const Harness = ({ params }: { params?: Status.CreateModalParams }): ReactElement => {
  const open = Status.useCreateModal();
  return <button onClick={() => open(params)}>open</button>;
};
Harness.displayName = "Harness";

const openModal = async (params?: Status.CreateModalParams) => {
  const client = createTestClient();
  const { wrapper } = await createConsoleWrapper({ client });
  const result = render(
    <>
      <Harness params={params} />
      <Modals.Stack />
    </>,
    { wrapper },
  );
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  await screen.findByText("Variant");
  return { ...result, client };
};

const createButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByText("Create")
    .map((el) => el.closest<HTMLButtonElement>("button"))
    .find((b) => b != null);
  if (btn == null) throw new Error("Create button not found");
  return btn;
};

describe("Status.useCreateModal", () => {
  it("should render the create form fields", async () => {
    await openModal();
    expect(screen.getByPlaceholderText("Name")).toBeTruthy();
    expect(screen.getByText("Variant")).toBeTruthy();
    expect(screen.getByText("Message")).toBeTruthy();
  });

  it("should start with an empty name field", async () => {
    await openModal({ name: "ignored" });
    expect(screen.getByPlaceholderText<HTMLInputElement>("Name").value).toBe("");
  });

  it("should create the status and close the modal on save", async () => {
    await openModal();
    const nameInput = screen.getByPlaceholderText<HTMLInputElement>("Name");
    fireEvent.change(nameInput, { target: { value: id.create() } });
    fireEvent.click(createButton());
    await waitFor(() => expect(screen.queryByText("Variant")).toBeNull(), TIMEOUT);
  });
});

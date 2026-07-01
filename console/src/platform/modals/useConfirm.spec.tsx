// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, type RenderResult, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { renderWithModals } from "@/platform/modals/testutil";

let lastPrompt: Promise<boolean | null> | undefined;

const ConfirmHarness = ({ params }: { params: Modals.ConfirmParams }): ReactElement => {
  const confirm = Modals.useConfirm();
  return (
    <button
      onClick={() => {
        lastPrompt = confirm(params);
      }}
    >
      open
    </button>
  );
};
ConfirmHarness.displayName = "ConfirmHarness";

const openConfirm = (params: Modals.ConfirmParams): RenderResult => {
  const result = renderWithModals(<ConfirmHarness params={params} />);
  fireEvent.click(screen.getByRole("button", { name: "open" }));
  return result;
};

const findCloseButton = (): HTMLButtonElement => {
  const btn = screen
    .getAllByRole("button")
    .find((b) => (b.textContent ?? "").trim() === "");
  if (btn == null) throw new Error("close button not found");
  return btn as HTMLButtonElement;
};

const BASE: Modals.ConfirmParams = {
  message: "Are you sure?",
  description: "This cannot be undone.",
};

beforeEach(() => {
  lastPrompt = undefined;
});

describe("Confirm", () => {
  it("should render the title, message, and description", async () => {
    openConfirm({ ...BASE, title: "Delete Channel" });
    await waitFor(() => expect(screen.getByText("Delete Channel")).toBeTruthy());
    expect(screen.getByText("Are you sure?")).toBeTruthy();
    expect(screen.getByText("This cannot be undone.")).toBeTruthy();
  });

  it("should resolve true when the confirm button is clicked", async () => {
    openConfirm(BASE);
    fireEvent.click(await screen.findByRole("button", { name: "Confirm" }));
    await expect(lastPrompt).resolves.toBe(true);
  });

  it("should resolve false when the cancel button is clicked", async () => {
    openConfirm(BASE);
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await expect(lastPrompt).resolves.toBe(false);
  });

  it("should resolve null when dismissed via the close button", async () => {
    openConfirm(BASE);
    await screen.findByText("Are you sure?");
    fireEvent.click(findCloseButton());
    await expect(lastPrompt).resolves.toBeNull();
  });

  it("should render and honor custom confirm and cancel labels", async () => {
    openConfirm({ ...BASE, confirm: { label: "Yes" }, cancel: { label: "No" } });
    fireEvent.click(await screen.findByRole("button", { name: "Yes" }));
    await expect(lastPrompt).resolves.toBe(true);
  });

  it("should default the confirm label to Confirm", async () => {
    openConfirm(BASE);
    expect(await screen.findByRole("button", { name: "Confirm" })).toBeTruthy();
  });
});

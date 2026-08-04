// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import {
  findDismissButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";

const BASE: Modals.ConfirmParams = {
  message: "Are you sure?",
  description: "This cannot be undone.",
};

const openConfirm = async (
  params: Modals.ConfirmParams,
): Promise<ModalOpenerHandle<Promise<boolean | null>>> =>
  await renderModalOpener(Modals.useConfirm, [params]);

describe("Confirm", () => {
  it("should display the prompt and resolve true when confirmed", async () => {
    const handle = await openConfirm({ ...BASE, title: "Delete Channel" });
    await waitFor(() => expect(screen.getByText("Delete Channel")).toBeTruthy());
    expect(screen.getByText("Are you sure?")).toBeTruthy();
    expect(screen.getByText("This cannot be undone.")).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: "Confirm" }));
    await expect(handle.result()).resolves.toBe(true);
    await waitFor(() => expect(screen.queryByText("Are you sure?")).toBeNull());
  });

  it("should resolve false and close the modal when cancelled", async () => {
    const handle = await openConfirm(BASE);
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await expect(handle.result()).resolves.toBe(false);
    await waitFor(() => expect(screen.queryByText("Are you sure?")).toBeNull());
  });

  it("should resolve true when confirmed via the save trigger", async () => {
    const handle = await openConfirm(BASE);
    await screen.findByRole("button", { name: "Confirm" });
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.keyDown(document.body, { code: "Enter" });
    fireEvent.keyUp(document.body, { code: "Enter" });
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
    await expect(handle.result()).resolves.toBe(true);
  });

  it("should resolve null when dismissed without an answer", async () => {
    const handle = await openConfirm(BASE);
    await screen.findByText("Are you sure?");
    fireEvent.click(findDismissButton());
    await expect(handle.result()).resolves.toBeNull();
  });

  it("should honor custom confirm and cancel labels", async () => {
    const handle = await openConfirm({
      ...BASE,
      confirm: { label: "Yes" },
      cancel: { label: "No" },
    });
    expect(await screen.findByRole("button", { name: "No" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Yes" }));
    await expect(handle.result()).resolves.toBe(true);
  });

  it("should answer each invocation independently when reopened", async () => {
    const handle = await openConfirm(BASE);
    fireEvent.click(await screen.findByRole("button", { name: "Confirm" }));
    await expect(handle.result()).resolves.toBe(true);
    await waitFor(() => expect(screen.queryByText("Are you sure?")).toBeNull());
    handle.reopen();
    fireEvent.click(await screen.findByRole("button", { name: "Cancel" }));
    await expect(handle.result()).resolves.toBe(false);
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Modals } from "@/platform/modals";
import {
  findDismissButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";

const openRename = async (
  params: Modals.RenameParams = {},
): Promise<ModalOpenerHandle<Promise<string | null>>> =>
  await renderModalOpener(Modals.useRename, [params]);

const getInput = (): HTMLInputElement => screen.getByRole("textbox");

describe("Rename", () => {
  it("should populate the input with the initial value", async () => {
    await openRename({ initialValue: "original" });
    await waitFor(() => expect(getInput().value).toEqual("original"));
  });

  it("should resolve with the edited value and close the modal when saved", async () => {
    const handle = await openRename({ initialValue: "original" });
    const input = await screen.findByRole("textbox");
    fireEvent.change(input, { target: { value: "renamed" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await expect(handle.result()).resolves.toEqual("renamed");
    await waitFor(() => expect(screen.queryByRole("textbox")).toBeNull());
  });

  it("should refuse to save an empty name until one is entered", async () => {
    const handle = await openRename({ allowEmpty: false });
    const save = await screen.findByRole("button", { name: "Save" });
    expect(save.className).toContain("pluto--disabled");
    const settled = vi.fn();
    void handle.result()?.then(settled, settled);
    fireEvent.click(save);
    await act(async () => {});
    expect(settled).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox")).toBeTruthy();
    fireEvent.change(getInput(), { target: { value: "channel_1" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await expect(handle.result()).resolves.toEqual("channel_1");
  });

  it("should resolve null when empty values are allowed and saved empty", async () => {
    const handle = await openRename({ allowEmpty: true });
    fireEvent.click(await screen.findByRole("button", { name: "Save" }));
    await expect(handle.result()).resolves.toBeNull();
  });

  it("should resolve null when dismissed without saving", async () => {
    const handle = await openRename({ initialValue: "original" });
    await screen.findByRole("textbox");
    fireEvent.click(findDismissButton());
    await expect(handle.result()).resolves.toBeNull();
  });

  it("should render a custom title and label", async () => {
    await openRename({ title: "Rename Range", label: "Range Name" });
    await waitFor(() => expect(screen.getByText("Rename Range")).toBeTruthy());
    expect(screen.getByText("Range Name")).toBeTruthy();
  });
});

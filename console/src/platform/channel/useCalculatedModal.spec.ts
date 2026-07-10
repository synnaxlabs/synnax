// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/platform/channel";
import { uniqueChannelName } from "@/platform/channel/testutil";
import {
  findButton,
  type ModalOpenerHandle,
  renderModalOpener,
} from "@/platform/modals/testutil";

const client = createTestClient();

const openModal = async (
  params?: Channel.CalculatedModalParams,
): Promise<ModalOpenerHandle<void>> => {
  const handle = await renderModalOpener(Channel.useCalculatedModal, [params ?? {}], {
    client,
  });
  await screen.findByPlaceholderText("Name");
  return handle;
};

describe("useCalculatedModal", () => {
  describe("operation selection", () => {
    it("should hide the window and reset-channel fields for the derivative operation", async () => {
      await openModal();
      await waitFor(() => expect(screen.getByText("Window")).toBeTruthy());
      fireEvent.click(screen.getByText("Derivative"));
      await waitFor(() => expect(screen.queryByText("Window")).toBeNull());
      expect(screen.queryByText("Reset Channel")).toBeNull();
    });

    it("should restore the window and reset-channel fields when switching back to a windowed operation", async () => {
      await openModal();
      await waitFor(() => expect(screen.getByText("Derivative")).toBeTruthy());
      fireEvent.click(screen.getByText("Derivative"));
      await waitFor(() => expect(screen.queryByText("Window")).toBeNull());
      fireEvent.click(screen.getByText("Average"));
      await waitFor(() => expect(screen.getByText("Window")).toBeTruthy());
      expect(screen.getByText("Reset Channel")).toBeTruthy();
    });
  });

  describe("validation", () => {
    it("should keep the modal open and show errors when the name and expression are empty", async () => {
      await openModal();
      await expect
        .poll(() => findButton("Create").className.includes("disabled"))
        .toBe(false);
      fireEvent.click(findButton("Create"));
      // expect.poll instead of waitFor: the editor suspends forever without a real
      // monaco runtime, which starves waitFor's act-wrapped polling.
      await expect.poll(() => screen.queryByText("Name is required")).toBeTruthy();
      expect(
        screen.getByText("Expression must contain a return statement"),
      ).toBeTruthy();
      expect(screen.getByPlaceholderText("Name")).toBeTruthy();
    });
  });

  describe("edit mode", () => {
    it("should prefill the form from the existing channel and persist edits on save", async () => {
      const name = uniqueChannelName("calc");
      const ch = await client.channels.create({
        name,
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return 2",
      });
      await openModal({ channelKey: ch.key });
      await waitFor(() =>
        expect(screen.getByPlaceholderText<HTMLInputElement>("Name").value).toBe(name),
      );
      const next = uniqueChannelName("calc_renamed");
      fireEvent.change(screen.getByPlaceholderText("Name"), {
        target: { value: next },
      });
      await act(async () => {
        fireEvent.click(findButton("Save"));
      });
      await waitFor(async () => {
        const updated = await client.channels.retrieve(ch.key);
        expect(updated.name).toBe(next);
        expect(updated.expression).toBe("return 2");
      });
      await waitFor(() => expect(screen.queryByPlaceholderText("Name")).toBeNull());
    });
  });
});

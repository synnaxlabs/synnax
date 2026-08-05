// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { findButton, renderModalOpener } from "@/platform/modals/testutil";
import { Status } from "@/platform/status";
import { findDialogTrigger, uniqueName } from "@/testutil";

const client = createTestClient();

const openModal = async (params?: Status.CreateModalParams) => {
  const handle = await renderModalOpener(() => Status.useCreateModal(), [params], {
    client,
  });
  await screen.findByText("Variant");
  return handle;
};

describe("Status.useCreateModal", () => {
  it("should create a status with the entered name, message, and variant, then close", async () => {
    await openModal();
    const name = uniqueName("status");
    fireEvent.change(screen.getByPlaceholderText("Name"), {
      target: { value: name },
    });
    fireEvent.change(screen.getByPlaceholderText("Message"), {
      target: { value: "it broke" },
    });
    fireEvent.click(await findDialogTrigger());
    fireEvent.click(await screen.findByText("Error"));
    fireEvent.click(findButton("Create"));
    await waitFor(() => expect(screen.queryByText("Variant")).toBeNull());
    await waitFor(async () => {
      const matches = await client.statuses.retrieve({ searchTerm: name });
      expect(matches).toHaveLength(1);
      expect(matches[0].name).toBe(name);
      expect(matches[0].message).toBe("it broke");
      expect(matches[0].variant).toBe("error");
    });
  });

  it("should prefill from and update the existing status when statusKey is provided", async () => {
    const name = uniqueName("status");
    const existing = await client.statuses.set({
      name,
      message: "original message",
      variant: "info",
    });
    await openModal({ statusKey: existing.key });
    await waitFor(() => {
      expect(screen.getByPlaceholderText<HTMLInputElement>("Name").value).toBe(name);
      expect(screen.getByPlaceholderText<HTMLInputElement>("Message").value).toBe(
        "original message",
      );
    });
    fireEvent.change(screen.getByPlaceholderText("Message"), {
      target: { value: "updated message" },
    });
    fireEvent.click(findButton("Create"));
    await waitFor(() => expect(screen.queryByText("Variant")).toBeNull());
    await waitFor(async () => {
      const updated = await client.statuses.retrieve(existing.key);
      expect(updated.message).toBe("updated message");
      expect(updated.name).toBe(name);
    });
  });
});

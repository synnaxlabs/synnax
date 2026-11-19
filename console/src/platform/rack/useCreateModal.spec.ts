// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { findButton, findDismissButton, openModal } from "@/platform/modals/testutil";
import { Rack } from "@/platform/rack";

describe("useCreateModal", () => {
  it("should create a rack with a key and show the key once", async () => {
    const client = createTestClient();
    await openModal(Rack.useCreateModal, { client });
    const name = uniqueName("rack");
    fireEvent.change(screen.getByPlaceholderText("Rack name"), {
      target: { value: name },
    });
    fireEvent.click(findButton("Create"));
    const shown = await screen.findByText(/^syk_/);
    const created = await client.racks.retrieve({ name });
    const keys = await client.auth.retrieveKeys([rack.resourceID(created.key)]);
    expect(keys).toHaveLength(1);
    expect(shown.textContent?.startsWith("syk_")).toBe(true);
    expect(findButton("Create").getAttribute("aria-disabled")).toBe("true");
  });

  it("should block closing while the key is being created", async () => {
    const client = createTestClient();
    const create = client.auth.createKey.bind(client.auth);
    let release = (): void => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const createKey = vi
      .spyOn(client.auth, "createKey")
      .mockImplementation(async (params) => {
        await gate;
        return await create(params);
      });
    await openModal(Rack.useCreateModal, { client });
    fireEvent.change(screen.getByPlaceholderText("Rack name"), {
      target: { value: uniqueName("rack") },
    });
    fireEvent.click(findButton("Create"));
    await expect.poll(() => createKey.mock.calls.length).toBe(1);
    fireEvent.click(findDismissButton());
    fireEvent.keyDown(window, { code: "Escape" });
    fireEvent.keyUp(window, { code: "Escape" });
    expect(screen.queryByPlaceholderText("Rack name")).not.toBeNull();
    release();
    await screen.findByText(/^syk_/);
    fireEvent.click(findDismissButton());
    await waitFor(() => expect(screen.queryByPlaceholderText("Rack name")).toBeNull());
  });

  it("should delete the key when the modal unmounts before showing it", async () => {
    const client = createTestClient();
    const create = client.auth.createKey.bind(client.auth);
    let release = (): void => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const createKey = vi
      .spyOn(client.auth, "createKey")
      .mockImplementation(async (params) => {
        await gate;
        return await create(params);
      });
    const { unmount } = await openModal(Rack.useCreateModal, { client });
    const name = uniqueName("rack");
    fireEvent.change(screen.getByPlaceholderText("Rack name"), {
      target: { value: name },
    });
    fireEvent.click(findButton("Create"));
    await expect.poll(() => createKey.mock.calls.length).toBe(1);
    unmount();
    release();
    await createKey.mock.results[0].value;
    const created = await client.racks.retrieve({ name });
    const subject = rack.resourceID(created.key);
    await expect
      .poll(async () => await client.auth.retrieveKeys([subject]))
      .toHaveLength(0);
  });

  it("should not create a rack with a blank name", async () => {
    await openModal(Rack.useCreateModal, { client: createTestClient() });
    fireEvent.change(screen.getByPlaceholderText("Rack name"), {
      target: { value: "   " },
    });
    expect(findButton("Create").getAttribute("aria-disabled")).toBe("true");
  });
});

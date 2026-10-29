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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { findButton, findDismissButton, openModal } from "@/platform/modals/testutil";
import { Rack } from "@/platform/rack";
import { uniqueName } from "@/testutil";

describe("useCreateModal", () => {
  it("should create a rack with a key and show the key once", async () => {
    const client = createTestClient();
    await openModal(Rack.useCreateModal, { client });
    const name = uniqueName("rack");
    fireEvent.change(screen.getByPlaceholderText("Rack name"), {
      target: { value: name },
    });
    fireEvent.click(findButton("Create"));
    const shown = await screen.findByDisplayValue(/^syk_/);
    const [created] = (await client.racks.retrieve({ names: [name] })).filter(
      (r) => r.name === name,
    );
    const keys = await client.auth.retrieveAPIKeys([rack.ontologyID(created.key)]);
    expect(keys).toHaveLength(1);
    expect(shown.getAttribute("value")?.startsWith("syk_")).toBe(true);
    expect(findButton("Create").className).toContain("pluto--disabled");
  });

  it("should block closing while the key is being created", async () => {
    const client = createTestClient();
    const create = client.auth.createAPIKey.bind(client.auth);
    let release = (): void => {};
    const gate = new Promise<void>((resolve) => (release = resolve));
    const createKey = vi
      .spyOn(client.auth, "createAPIKey")
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
    await screen.findByDisplayValue(/^syk_/);
    fireEvent.click(findDismissButton());
    await waitFor(() => expect(screen.queryByPlaceholderText("Rack name")).toBeNull());
  });
});

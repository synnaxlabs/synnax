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
import { fireEvent, screen } from "@testing-library/react";
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
    const keys = await client.auth.retrieveKeys([rack.ontologyID(created.key)]);
    expect(keys).toHaveLength(1);
    expect(shown.getAttribute("value")?.startsWith("syk_")).toBe(true);
    expect(findButton("Create").className).toContain("pluto--disabled");
  });

  it("should delete the key when the modal closes before it arrives", async () => {
    const client = createTestClient();
    // API keys have no change stream, so the delete call is the only completion signal.
    const deleteKeys = vi.spyOn(client.auth, "deleteKeys");
    await openModal(Rack.useCreateModal, { client });
    const name = uniqueName("rack");
    fireEvent.change(screen.getByPlaceholderText("Rack name"), {
      target: { value: name },
    });
    fireEvent.click(findButton("Create"));
    fireEvent.click(findDismissButton());
    await expect.poll(() => deleteKeys.mock.results.length).toBe(1);
    await deleteKeys.mock.results[0].value;
    const [created] = await client.racks.retrieve({ names: [name] });
    expect(await client.auth.retrieveKeys([rack.ontologyID(created.key)])).toHaveLength(
      0,
    );
  });
});

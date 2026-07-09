// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, NotFoundError, rack } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Rack } from "@/feature/rack";
import { findModalButton, renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import { assertDefined, uniqueName } from "@/testutil";

const client = createTestClient();

const Item = Rack.TREE_ITEMS.rack;

const createRack = async () => await client.racks.create({ name: uniqueName("rack") });

const rackResource = (key: number, name: string) =>
  createResource(rack.ontologyID(key), name);

const renderMenu = async (racks: { key: number; name: string }[]) => {
  assertDefined(Item.ContextMenu);
  return await renderTreeContextMenu(Item.ContextMenu, {
    client,
    resources: racks.map((r) => rackResource(r.key, r.name)),
  });
};

describe("rack ontology service", () => {
  it("should expose rename, arc creation, and delete for a single rack", async () => {
    const r = await createRack();
    await renderMenu([r]);
    expect(await screen.findByText("Rename")).toBeTruthy();
    expect(screen.getByText("Create Arc automation")).toBeTruthy();
    expect(screen.getByText("Delete")).toBeTruthy();
  });

  it("should hide single-selection actions for a multi-rack selection", async () => {
    const a = await createRack();
    const b = await createRack();
    await renderMenu([a, b]);
    expect(await screen.findByText("Delete")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Create Arc automation")).toBeNull();
    expect(screen.queryByText("Copy properties")).toBeNull();
  });

  it("should delete the rack on the cluster after confirmation", async () => {
    const r = await createRack();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${r.name}?`);
    fireEvent.click(findModalButton("Delete"));
    await waitFor(async () => {
      await expect(client.racks.retrieve({ key: r.key })).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should open the arc creation modal from the context menu", async () => {
    const r = await createRack();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Create Arc automation"));
    expect(await screen.findByText("Create Automation")).toBeTruthy();
  });
});

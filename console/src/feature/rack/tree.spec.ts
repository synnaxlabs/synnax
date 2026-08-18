// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  ni,
  NotFoundError,
  type ontology,
  rack,
  type Synnax as Client,
  task,
} from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { Rack } from "@/feature/rack";
import { findModalButton, renderTreeContextMenu } from "@/platform/tree/menuTestutil";
import { createResource } from "@/platform/tree/testutil";
import { assertDefined, createTestClientWithGrants, uniqueName } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Item = Rack.TREE_ITEMS.rack;

const createRack = async () => await client.racks.create({ name: uniqueName("rack") });

const createNIRackWithScanner = async () => {
  const r = await client.racks.create({
    name: uniqueName("ni_rack"),
    integrations: ["ni"],
  });
  const scanTask = await r.createTask(
    {
      name: uniqueName("ni_scanner"),
      type: NI.Task.SCAN_TYPE,
      config: ni.scanConfigZ.parse({}),
    },
    NI.Task.SCAN_SCHEMAS,
  );
  return { rack: r, scanTask };
};

const rackResource = (key: number, name: string) =>
  createResource(rack.ontologyID(key), name);

const renderMenu = async (
  racks: { key: number; name: string }[],
  as: Client = client,
) => {
  assertDefined(Item.ContextMenu);
  return await renderTreeContextMenu(Item.ContextMenu, {
    client: as,
    resources: racks.map((r) => rackResource(r.key, r.name)),
  });
};

/**
 * A client that may rename racks and read everything the menu touches, differing only
 * in the type it may create.
 */
const createRackWriter = async (creatable: ontology.ID) =>
  await createTestClientWithGrants(client, {
    retrieve: [rack.TYPE_ONTOLOGY_ID, task.TYPE_ONTOLOGY_ID, arc.TYPE_ONTOLOGY_ID],
    update: [rack.TYPE_ONTOLOGY_ID],
    create: [creatable],
  });

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
      await expect(client.racks.retrieve(r.key)).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });

  it("should open the arc creation modal from the context menu", async () => {
    const r = await createRack();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Create Arc automation"));
    expect(await screen.findByText("Create Arc automation")).toBeTruthy();
  });

  it("should hide the toggle scanner item for a rack without the NI integration", async () => {
    const r = await createRack();
    await renderMenu([r]);
    await screen.findByText("Rename");
    expect(screen.queryByText("Toggle NI device scanner")).toBeNull();
  });

  it("should toggle the NI scanner's disabled flag from the rack context menu", async () => {
    const { rack: r, scanTask } = await createNIRackWithScanner();
    await renderMenu([r]);
    fireEvent.click(await screen.findByText("Toggle NI device scanner"));
    await waitFor(async () => {
      const after = await client.tasks.retrieve({
        key: scanTask.key,
        schemas: NI.Task.SCAN_SCHEMAS,
      });
      expect(after.config.disabled).toBe(!scanTask.config.disabled);
    });
  });
});

describe("permission to write the rack", () => {
  it("should withhold rename, arc creation, and delete from a viewer", async () => {
    const r = await createRack();
    await renderMenu([r], await roles.get("Viewer"));
    expect(await screen.findByText("Copy properties")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    expect(screen.queryByText("Create Arc automation")).toBeNull();
    expect(screen.queryByText("Delete")).toBeNull();
  });

  // Toggling the scanner rewrites the scan task's config, so the item answers to task
  // create and not to the rack write that puts the rest of the menu on screen.
  it("should withhold the scanner toggle from a subject who cannot create tasks", async () => {
    const { rack: r } = await createNIRackWithScanner();
    await renderMenu([r], await createRackWriter(arc.TYPE_ONTOLOGY_ID));
    expect(await screen.findByText("Create Arc automation")).toBeTruthy();
    expect(screen.queryByText("Toggle NI device scanner")).toBeNull();
  });

  it("should offer the scanner toggle to a subject who may create tasks", async () => {
    const { rack: r } = await createNIRackWithScanner();
    await renderMenu([r], await createRackWriter(task.TYPE_ONTOLOGY_ID));
    expect(await screen.findByText("Toggle NI device scanner")).toBeTruthy();
  });
});

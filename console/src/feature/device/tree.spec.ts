// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createTestClient,
  device as deviceClient,
  group,
  NotFoundError,
  ontology,
  rack,
} from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Device } from "@/feature/device";
import { createTestDevice } from "@/platform/device/testutil";
import { findButton } from "@/platform/modals/testutil";
import {
  openTreeRowContextMenu,
  renderOntologyTree,
} from "@/platform/tree/treeTestutil";
import { awaitTextEditingElement, commitTextEdit, uniqueName } from "@/testutil";

const client = createTestClient();

const createDeviceGroup = async (
  ...devices: { key: string; rack: number }[]
): Promise<ontology.ID> => {
  const grp = await client.groups.create({
    parent: ontology.ROOT_ID,
    name: uniqueName("devgrp"),
  });
  for (const dev of devices)
    await client.ontology.moveChildren(
      rack.ontologyID(dev.rack),
      group.ontologyID(grp.key),
      deviceClient.ontologyID(dev.key),
    );
  return group.ontologyID(grp.key);
};

const renderDeviceTree = async (root: ontology.ID) =>
  await renderOntologyTree({
    client,
    root,
    items: { device: Device.TREE_ITEM },
  });

describe("device/ontology", () => {
  it("renders the device with its location", async () => {
    const dev = await createTestDevice(client);
    const root = await createDeviceGroup(dev);
    await renderDeviceTree(root);
    expect(await screen.findByText(dev.name)).toBeTruthy();
    expect(await screen.findByText("dev1")).toBeTruthy();
  });

  it("renames a device from the context menu", async () => {
    const dev = await createTestDevice(client);
    const root = await createDeviceGroup(dev);
    await renderDeviceTree(root);
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Rename"));
    const editor = await awaitTextEditingElement();
    const renamed = uniqueName("renamed");
    commitTextEdit(editor, renamed);
    await waitFor(async () =>
      expect((await client.devices.retrieve({ key: dev.key })).name).toBe(renamed),
    );
  });

  it("deletes a device after confirmation", async () => {
    const dev = await createTestDevice(client);
    const root = await createDeviceGroup(dev);
    await renderDeviceTree(root);
    await openTreeRowContextMenu(dev.name);
    fireEvent.click(await screen.findByText("Delete"));
    await screen.findByText(`Are you sure you want to delete ${dev.name}?`);
    fireEvent.click(findButton("Delete"));
    await waitFor(async () => {
      await expect(client.devices.retrieve({ key: dev.key })).rejects.toSatisfy((e) =>
        NotFoundError.matches(e),
      );
    });
  });
});

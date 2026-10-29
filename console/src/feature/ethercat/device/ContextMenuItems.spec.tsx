// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Menu as PMenu } from "@synnaxlabs/lyra/menu";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { type Device } from "@/platform/device";
import { createDeviceResource } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

interface SlaveLoaderProps {
  deviceKey: device.Key;
}

const SlaveLoader = ({ deviceKey }: SlaveLoaderProps): ReactElement | null => {
  const { data } = EtherCAT.Device.useResultSlave({ key: deviceKey });
  return data == null ? null : <span>{`loaded:${deviceKey}`}</span>;
};

/**
 * Mounts the context menu in two phases: first loads the slaves into the flux device
 * cache (the tree does this in production before a menu can open), then mounts the
 * menu items so their enabled/disabled computation sees the cached devices.
 */
const renderContextMenu = async (devices: EtherCAT.Device.SlaveDevice[]) => {
  const { wrapper, store } = await createConsoleWrapper({ client });
  const proj = await client.projects.create({
    name: uniqueName("proj"),
  });
  store.dispatch(Session.Project.select(proj.key));
  const keys = devices.map((d) => d.key);
  const loaders = keys.map((k) => <SlaveLoader key={k} deviceKey={k} />);
  const result = render(<div>{loaders}</div>, { wrapper });
  for (const k of keys) await screen.findByText(`loaded:${k}`);
  const resources = devices.map((d) =>
    createDeviceResource({ key: d.key, name: d.name, configured: d.configured }),
  );
  const props: Device.ContextMenuItemsProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: resources.map((r) => r.id) }),
    state: createState(resources),
    onConfigure: vi.fn(),
  };
  // The menu suspends on the slaves fetch; a tree that suspends inside a sync
  // act never commits, so the mount needs an async act (see renderHookSuspended).
  await act(async () => {
    result.rerender(
      <>
        {loaders}
        <PMenu.Menu>
          <EtherCAT.Device.ContextMenuItems {...props} />
        </PMenu.Menu>
        <Modals.Stack />
      </>,
    );
  });
  return { store };
};

const FILTER: EtherCAT.Device.StartupParameter = {
  index: 0x8000,
  subIndex: 0x15,
  dataType: "uint8",
  value: 3,
  name: "Filter",
};

const openStartupParameters = async () => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  const slave = await createSlaveDevice(client, rack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    startupParameters: [FILTER],
  });
  await renderContextMenu([slave]);
  fireEvent.click(await screen.findByText("Edit startup parameters"));
  await screen.findByLabelText("Index 1");
  return slave;
};

const setCell = (label: string, value: string) => {
  const input = screen.getByLabelText(label);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const retrieveStartupParameters = async (key: string) =>
  (await client.devices.retrieve({ key, schemas: EtherCAT.Device.SLAVE_SCHEMAS }))
    .properties.startupParameters;

const createSlave = async (enabled: boolean) => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  return await createSlaveDevice(client, rack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    enabled,
  });
};

describe("EtherCAT device ContextMenuItems", () => {
  it("should offer only Disable when every selected slave is enabled", async () => {
    const slave = await createSlave(true);
    await renderContextMenu([slave]);
    await waitFor(() => expect(screen.getByText("Disable")).toBeTruthy());
    expect(screen.queryByText("Enable")).toBeNull();
  });

  it("should offer only Enable when every selected slave is disabled", async () => {
    const slave = await createSlave(false);
    await renderContextMenu([slave]);
    await waitFor(() => expect(screen.getByText("Enable")).toBeTruthy());
    expect(screen.queryByText("Disable")).toBeNull();
  });

  it("should offer both actions for a mixed selection", async () => {
    const enabled = await createSlave(true);
    const disabled = await createSlave(false);
    await renderContextMenu([enabled, disabled]);
    await waitFor(() => expect(screen.getByText("Enable")).toBeTruthy());
    expect(screen.getByText("Disable")).toBeTruthy();
    expect(screen.queryByText("Edit startup parameters")).toBeNull();
  });

  it("should create the vendor task drafts and open their resource tabs", async () => {
    const slave = await createSlave(true);
    const { store } = await renderContextMenu([slave]);
    fireEvent.click(await screen.findByText("Create read task"));
    const readTab = await resolveFocusedTab(store, client);
    if (readTab.variant !== "resource") throw new Error("expected a resource tab");
    expect(readTab.resource.type).toBe(task.TYPE_ONTOLOGY_ID.type);
    const read = await client.tasks.retrieve({ key: readTab.resource.key });
    expect(read.type).toBe(EtherCAT.Task.READ_TYPE);
    fireEvent.click(screen.getByText("Create write task"));
    const writeTab = await resolveFocusedTab(
      store,
      client,
      (t) => t.variant === "resource" && t.resource.key !== readTab.resource.key,
    );
    if (writeTab.variant !== "resource") throw new Error("expected a resource tab");
    const write = await client.tasks.retrieve({ key: writeTab.resource.key });
    expect(write.type).toBe(EtherCAT.Task.WRITE_TYPE);
  });

  it("should disable every selected slave on the Core when Disable is clicked", async () => {
    const slaveA = await createSlave(true);
    const slaveB = await createSlave(true);
    await renderContextMenu([slaveA, slaveB]);
    fireEvent.click(await screen.findByText("Disable"));
    await waitFor(async () => {
      const updatedA = await client.devices.retrieve({
        key: slaveA.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      const updatedB = await client.devices.retrieve({
        key: slaveB.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updatedA.properties.enabled).toBe(false);
      expect(updatedB.properties.enabled).toBe(false);
    });
  });

  it("should enable a disabled slave on the Core when Enable is clicked", async () => {
    const slave = await createSlave(false);
    await renderContextMenu([slave]);
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(async () => {
      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.enabled).toBe(true);
    });
  });

  it("should save edited and added startup parameters to the slave", async () => {
    const slave = await openStartupParameters();
    expect(screen.getByLabelText<HTMLInputElement>("Index 1").value).toBe("0x8000");
    setCell("Value 1", "5");
    fireEvent.click(screen.getByRole("button", { name: "Add row" }));
    setCell("Index 2", "1C12");
    setCell("Subindex 2", "0");
    setCell("Value 2", "2");
    fireEvent.click(screen.getByRole("button", { name: /^Data type 2 / }));
    fireEvent.click(await screen.findByRole("option", { name: /uint16/ }));
    fireEvent.click(findButton("Save"));
    await waitFor(async () =>
      expect(await retrieveStartupParameters(slave.key)).toEqual([
        { ...FILTER, value: 5 },
        { index: 0x1c12, subIndex: 0, dataType: "uint16", value: 2, name: "" },
      ]),
    );
  });

  it("should reject a startup parameter index that is not hexadecimal", async () => {
    const slave = await openStartupParameters();
    setCell("Index 1", "zz");
    setCell("Value 1", "5");
    fireEvent.click(findButton("Save"));
    expect(
      await screen.findByText(
        "Row 1: the index must be a hexadecimal number such as 0x8000.",
      ),
    ).toBeTruthy();
    await act(async () => {});
    expect(await retrieveStartupParameters(slave.key)).toEqual([FILTER]);
  });
});

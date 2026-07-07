// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type device } from "@synnaxlabs/client";
import { Menu as PMenu } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import { createIdentifier, createSlaveDevice } from "@/feature/ethercat/testutil";
import { createDeviceResource } from "@/platform/device/testutil";
import { type Ontology } from "@/platform/ontology";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/ontology/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, resolveFocusedTab, uniqueName } from "@/testutil";

const client = createTestClient();

interface SlaveLoaderProps {
  deviceKey: device.Key;
}

const SlaveLoader = ({ deviceKey }: SlaveLoaderProps): ReactElement | null => {
  const { data } = EtherCAT.Device.useRetrieveSlave({ key: deviceKey });
  if (data == null) return null;
  return <span>{`loaded:${deviceKey}`}</span>;
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
    layout: {},
  });
  store.dispatch(Session.Project.select(proj.key));
  const keys = devices.map((d) => d.key);
  const loaders = keys.map((k) => <SlaveLoader key={k} deviceKey={k} />);
  const result = render(<div>{loaders}</div>, { wrapper });
  for (const k of keys) await screen.findByText(`loaded:${k}`);
  const resources = devices.map((d) =>
    createDeviceResource({ key: d.key, name: d.name, configured: d.configured }),
  );
  const props: Ontology.TreeContextMenuProps = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: resources.map((r) => r.id) }),
    state: createState(resources),
  };
  result.rerender(
    <>
      {loaders}
      <PMenu.Menu>
        <EtherCAT.Device.ContextMenuItems {...props} />
      </PMenu.Menu>
    </>,
  );
  return { store };
};

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
  });

  it("should open the vendor task views carrying the device key", async () => {
    const slave = await createSlave(true);
    const { store } = await renderContextMenu([slave]);
    fireEvent.click(await screen.findByText("Create read task"));
    expect(await resolveFocusedTab(store, client)).toMatchObject({
      variant: "view",
      type: EtherCAT.Task.READ_TYPE,
      args: { deviceKey: slave.key },
    });
    fireEvent.click(screen.getByText("Create write task"));
    expect(
      await resolveFocusedTab(
        store,
        client,
        (t) => t.variant === "view" && t.type === EtherCAT.Task.WRITE_TYPE,
      ),
    ).toMatchObject({
      variant: "view",
      type: EtherCAT.Task.WRITE_TYPE,
      args: { deviceKey: slave.key },
    });
  });

  it("should disable every selected slave on the cluster when Disable is clicked", async () => {
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

  it("should enable a disabled slave on the cluster when Enable is clicked", async () => {
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
});

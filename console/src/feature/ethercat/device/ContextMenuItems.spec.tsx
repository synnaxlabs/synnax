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
import { Triggers } from "@synnaxlabs/lyra/triggers";
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
      <Triggers.Provider>
        {loaders}
        <PMenu.Menu>
          <EtherCAT.Device.ContextMenuItems {...props} />
        </PMenu.Menu>
        <Modals.Stack />
      </Triggers.Provider>,
    );
  });
  return { store };
};

const createSlave = async (
  enabled: boolean,
  properties: Partial<EtherCAT.Device.SlaveProperties> = {},
) => {
  const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
  return await createSlaveDevice(client, rack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    enabled,
    ...properties,
  });
};

const createMode = (
  name: string,
  assignActivate: number,
  sync0Cycle = 0,
): EtherCAT.Device.OpMode => ({
  name,
  assignActivate,
  sync0Cycle,
  sync0Factor: 1,
  sync0Shift: 0,
  sync1Factor: 0,
  sync1Shift: 0,
});

const SM_MODE = createMode("SM", 0);
const DC_MODE = createMode("DC", 0x0300);
const PDI_MODE = createMode("DC (PDI)", 0x0301);
const NO_SYNC0_MODE = createMode("DC (no SYNC0)", 0x0100);
const FIXED_MODE = createMode("DC (300 us)", 0x0300, 300_000);
const STALE_CHOICE = { index: 1, assignActivate: 0x0700, sync0Cycle: 0, sync1Cycle: 0 };
const STALE_ERROR =
  "The device no longer lists this mode, so the network fails to start. Pick a mode.";

/** Whether a save is in flight, which disables the button until the Core answers. */
const saving = () => findButton("Save").getAttribute("aria-disabled") != null;

const retrieveOpMode = async (key: device.Key) =>
  (await client.devices.retrieve({ key, schemas: EtherCAT.Device.SLAVE_SCHEMAS }))
    .properties.opMode;

/** Opens the sync mode form from the menu of a slave with the given properties. */
const openSyncMode = async (properties: Partial<EtherCAT.Device.SlaveProperties>) => {
  const slave = await createSlave(true, properties);
  await renderContextMenu([slave]);
  fireEvent.click(await screen.findByText("Set sync mode"));
  await screen.findByText("Sync mode");
  return slave;
};

const SM_DESCRIPTION = "Each frame from the Driver times the device.";
const DC_DESCRIPTION = "Clock ticks: SYNC0 each bus cycle.";

/** Opens the Mode select and picks the option with the description. */
const pickMode = async (description: string) => {
  fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
  fireEvent.click(await screen.findByText(description));
};

const typeCycle = (label: string, value: string) => {
  const field = screen.getByRole("textbox", { name: label });
  fireEvent.focus(field);
  fireEvent.change(field, { target: { value } });
  fireEvent.blur(field);
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

  describe("sync mode", () => {
    it("should offer Set sync mode for one slave but not for several", async () => {
      const slaveA = await createSlave(true);
      const slaveB = await createSlave(true);
      await renderContextMenu([slaveA, slaveB]);
      await screen.findByText("Disable");
      expect(screen.queryByText("Set sync mode")).toBeNull();
    });

    it("should save the chosen DC mode to the device", async () => {
      const slave = await openSyncMode({ opModes: [SM_MODE, DC_MODE] });
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
      await pickMode(DC_DESCRIPTION);
      expect(screen.getByRole("textbox", { name: "SYNC0 cycle" })).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 1,
          assignActivate: 0x0300,
          sync0Cycle: 0,
          sync1Cycle: 0,
        }),
      );
    });

    it("should hide the cycle fields when SM sync is picked", async () => {
      await openSyncMode({ opModes: [SM_MODE, DC_MODE] });
      await pickMode(DC_DESCRIPTION);
      expect(screen.getByRole("textbox", { name: "SYNC1 cycle" })).toBeTruthy();
      await pickMode(SM_DESCRIPTION);
      expect(screen.queryByRole("textbox", { name: "SYNC1 cycle" })).toBeNull();
    });

    it("should hide the cycle fields for a mode that takes no cycles", async () => {
      await openSyncMode({ opModes: [DC_MODE, PDI_MODE, NO_SYNC0_MODE] });
      expect(screen.getByRole("textbox", { name: "SYNC0 cycle" })).toBeTruthy();
      await pickMode("The device sets its own clock ticks.");
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
      await pickMode("The DC clock runs with no SYNC0 ticks.");
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
    });

    it("should put the slave back in auto when the auto mode is picked", async () => {
      const slave = await openSyncMode({
        opModes: [SM_MODE, DC_MODE],
        opMode: { index: 1, assignActivate: 0x0300, sync0Cycle: 0, sync1Cycle: 0 },
      });
      fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
      fireEvent.click(
        await screen.findByText("The Driver uses SM sync", { exact: false }),
      );
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toBeUndefined(),
      );
    });

    it("should say when a SYNC1 cycle turns on SYNC1", async () => {
      const slave = await openSyncMode({ opModes: [DC_MODE] });
      typeCycle("SYNC1 cycle", "1000000");
      expect(
        screen.getByText("The cycle turns on SYNC1, which the mode leaves off.", {
          exact: false,
        }),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 0,
          assignActivate: 0x0300,
          sync0Cycle: 0,
          sync1Cycle: 1_000_000_000,
        }),
      );
    });

    it("should not save a SYNC1 cycle that is not a multiple of SYNC0", async () => {
      const slave = await openSyncMode({ opModes: [DC_MODE] });
      typeCycle("SYNC0 cycle", "300");
      typeCycle("SYNC1 cycle", "1000");
      expect(
        screen.getByText("SYNC1 cycle must be a multiple of the SYNC0 cycle"),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toBeUndefined();
    });

    it("should check SYNC1 against the fixed SYNC0 cycle of the mode", async () => {
      const slave = await openSyncMode({ opModes: [FIXED_MODE] });
      typeCycle("SYNC1 cycle", "1000");
      expect(
        screen.getByText("SYNC1 cycle must be a multiple of the SYNC0 cycle"),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toBeUndefined();
    });

    it("should show a mode the device no longer lists as an error", async () => {
      await openSyncMode({ opModes: [DC_MODE], opMode: STALE_CHOICE });
      expect(screen.getByText(STALE_ERROR)).toBeTruthy();
      expect(screen.getByText("Mode 2 (0x0700)")).toBeTruthy();
    });

    it("should not save a mode the device no longer lists", async () => {
      const slave = await openSyncMode({
        opModes: [SM_MODE, DC_MODE],
        opMode: STALE_CHOICE,
      });
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(screen.getByText(STALE_ERROR)).toBeTruthy();
      expect(await retrieveOpMode(slave.key)).toEqual(STALE_CHOICE);
    });

    it("should accept the mode now listed at a stale index", async () => {
      const slave = await openSyncMode({
        opModes: [SM_MODE, DC_MODE],
        opMode: STALE_CHOICE,
      });
      await pickMode(DC_DESCRIPTION);
      expect(screen.queryByText(STALE_ERROR)).toBeNull();
      expect(screen.getByRole("textbox", { name: "SYNC0 cycle" })).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 1,
          assignActivate: 0x0300,
          sync0Cycle: 0,
          sync1Cycle: 0,
        }),
      );
    });
  });
});

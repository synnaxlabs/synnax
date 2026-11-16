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
import { describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  createIdentifier,
  createPDOs,
  createSlaveDevice,
  STATUS_ENTRY,
} from "@/feature/ethercat/testutil";
import { Device } from "@/platform/device";
import { createDeviceResource } from "@/platform/device/testutil";
import { Modals } from "@/platform/modals";
import { findButton } from "@/platform/modals/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  getIconButton,
  resolveFocusedTab,
  uniqueName,
} from "@/testutil";

const client = createTestClient();

interface SlaveLoaderProps {
  deviceKey: device.Key;
}

const SlaveLoader = ({ deviceKey }: SlaveLoaderProps): ReactElement | null => {
  const { data } = EtherCAT.Device.useResultSlave({ key: deviceKey });
  return data == null ? null : <span>{`loaded:${deviceKey}`}</span>;
};

/** The menu items, with Configure opening the dialog as the slave select opens it. */
const Items = (props: Omit<Device.ContextMenuItemsProps, "onConfigure">) => {
  const configure = Device.useConfigureModal();
  return (
    <EtherCAT.Device.ContextMenuItems
      {...props}
      onConfigure={(deviceKey) =>
        configure({ deviceKey, hardware: EtherCAT.Device.HARDWARE })
      }
    />
  );
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
  const resources = devices.map((d) => createDeviceResource(d));
  const props = {
    ...createBaseProps({ client, store }),
    selection: createSelection({ ids: resources.map((r) => r.id) }),
    state: createState(resources),
  };
  // The menu suspends on the slaves fetch; a tree that suspends inside a sync
  // act never commits, so the mount needs an async act (see renderHookSuspended).
  await act(async () => {
    result.rerender(
      <Triggers.Provider>
        {loaders}
        <PMenu.Menu>
          <Items {...props} />
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
const NO_CYCLE_MODE = { ...createMode("DC (no cycle)", 0x0300), sync0Factor: 0 };
const MODE_TEXTS = [
  {
    name: "a fixed SYNC0 cycle",
    mode: FIXED_MODE,
    option: "Clock ticks: SYNC0 every 300μs.",
    help: "SYNC0 ticks every 300μs. SYNC1 is off.",
  },
  {
    name: "a negative SYNC0 factor",
    mode: { ...DC_MODE, sync0Factor: -2 },
    option: "Clock ticks: SYNC0 2 times each bus cycle.",
    help: "SYNC0 ticks 2 times each bus cycle. SYNC1 is off.",
  },
  {
    name: "a SYNC0 factor above 1",
    mode: { ...DC_MODE, sync0Factor: 4 },
    option: "Clock ticks: SYNC0 every 4 bus cycles.",
    help: "SYNC0 ticks every 4 bus cycles. SYNC1 is off.",
  },
  {
    name: "a SYNC1 factor above 1",
    mode: { ...createMode("DC", 0x0700), sync1Factor: 3 },
    option: "Clock ticks: SYNC0 each bus cycle, SYNC1 every 3 SYNC0 ticks.",
    help: "SYNC0 ticks each bus cycle. SYNC1 ticks every 3 SYNC0 ticks.",
  },
  {
    name: "SYNC1 on each SYNC0 tick",
    mode: createMode("DC", 0x0700),
    option: "Clock ticks: SYNC0 each bus cycle, SYNC1 with each SYNC0 tick.",
    help: "SYNC0 ticks each bus cycle. SYNC1 ticks with each SYNC0 tick.",
  },
  {
    name: "a negative SYNC1 factor",
    mode: { ...createMode("DC", 0x0700), sync1Factor: -2 },
    option: "Clock ticks: SYNC0 each bus cycle, SYNC1 every 2 bus cycles.",
    help: "SYNC0 ticks each bus cycle. SYNC1 ticks every 2 bus cycles.",
  },
];
const STALE_CHOICE = { index: 1, assignActivate: 0x0700, sync0Cycle: 0, sync1Cycle: 0 };
const STALE_ERROR =
  "The device no longer lists this mode, so the network fails to start. Pick a mode.";

/** Whether a save is in flight, which disables the button until the Core answers. */
const saving = () => findButton("Save").getAttribute("aria-disabled") != null;

const retrieveOpMode = async (key: device.Key) =>
  (await client.devices.retrieve({ key, schemas: EtherCAT.Device.SLAVE_SCHEMAS }))
    .properties.opMode;

/** Opens the Configure dialog from the menu of a slave with the given properties. */
const openConfigure = async (properties: Partial<EtherCAT.Device.SlaveProperties>) => {
  const slave = await createSlave(true, properties);
  await renderContextMenu([slave]);
  fireEvent.click(await screen.findByText("Configure"));
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
    const slave = await createSlave(true, { pdos: createPDOs() });
    const { store } = await renderContextMenu([slave]);
    fireEvent.click(await screen.findByText("Create read task"));
    const readTab = await resolveFocusedTab(store, client);
    if (readTab.variant !== "resource") throw new Error("expected a resource tab");
    expect(readTab.resource.type).toBe(task.TYPE_RESOURCE_ID.type);
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

  it("should offer no write task for a slave with no outputs", async () => {
    const slave = await createSlave(true, {
      pdos: { inputs: [STATUS_ENTRY], outputs: [] },
    });
    await renderContextMenu([slave]);
    expect(await screen.findByText("Create read task")).toBeTruthy();
    expect(screen.queryByText("Create write task")).toBeNull();
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

  it("should offer one Configure item for a new slave", async () => {
    const rack = await client.racks.create({ name: uniqueName("ecat_rack") });
    const slave = await createSlaveDevice(client, rack.key, { network: "eth0" }, false);
    await renderContextMenu([slave]);
    expect(await screen.findAllByText("Configure")).toHaveLength(1);
  });

  it("should offer Configure and not Change identifier for a configured slave", async () => {
    await renderContextMenu([await createSlave(true)]);
    await screen.findByText("Configure");
    expect(screen.queryByText("Change identifier")).toBeNull();
  });

  describe("sync mode", () => {
    it("should offer Configure for one slave but not for several", async () => {
      const slaveA = await createSlave(true);
      const slaveB = await createSlave(true);
      await renderContextMenu([slaveA, slaveB]);
      await screen.findByText("Disable");
      expect(screen.queryByText("Configure")).toBeNull();
    });

    it("should save the chosen DC mode to the device", async () => {
      const slave = await openConfigure({ opModes: [SM_MODE, DC_MODE] });
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
      await openConfigure({ opModes: [SM_MODE, DC_MODE] });
      await pickMode(DC_DESCRIPTION);
      expect(screen.getByRole("textbox", { name: "SYNC1 cycle" })).toBeTruthy();
      await pickMode(SM_DESCRIPTION);
      expect(screen.queryByRole("textbox", { name: "SYNC1 cycle" })).toBeNull();
    });

    it("should hide the cycle fields for a mode that takes no cycles", async () => {
      await openConfigure({ opModes: [DC_MODE, PDI_MODE, NO_SYNC0_MODE] });
      await pickMode(DC_DESCRIPTION);
      expect(screen.getByRole("textbox", { name: "SYNC0 cycle" })).toBeTruthy();
      await pickMode("The device sets its own clock ticks.");
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
      await pickMode("The DC clock runs with no SYNC0 ticks.");
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
    });

    it("should put the slave back in auto when the auto mode is picked", async () => {
      const slave = await openConfigure({
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

    it("should run SM sync in auto on a slave that lists no modes", async () => {
      await openConfigure({ opModes: [] });
      fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
      expect(
        await screen.findByText(
          "The Driver uses SM sync, because the device lists no modes.",
        ),
      ).toBeTruthy();
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
    });

    it("should show only the first mode in auto when there is no SM mode", async () => {
      await openConfigure({ opModes: [FIXED_MODE, DC_MODE] });
      fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
      expect(
        await screen.findByText(
          "The Driver uses DC (300 us), the first mode, because the device lists no " +
            "SM mode. Best for most needs.",
        ),
      ).toBeTruthy();
      expect(screen.queryByRole("textbox", { name: "SYNC0 cycle" })).toBeNull();
      expect(screen.queryByText("SYNC0 ticks every 300μs. SYNC1 is off.")).toBeNull();
    });

    it("should show the cycles in auto when its mode needs one typed", async () => {
      const slave = await openConfigure({ opModes: [NO_CYCLE_MODE] });
      typeCycle("SYNC0 cycle", "300");
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 0,
          assignActivate: 0x0300,
          sync0Cycle: 300_000,
          sync1Cycle: 0,
        }),
      );
    });

    it("should run the mode that the catalog gives in auto", async () => {
      await openConfigure({
        opModes: [SM_MODE, DC_MODE],
        settings: {
          device: [],
          channels: [],
          mode: {
            index: 0x8020,
            subIndex: 9,
            value: 0,
            shared: false,
            opMode: {
              index: 1,
              assignActivate: 0x0300,
              sync0Cycle: 1_000,
              sync1Cycle: 1_000_000_000,
            },
            switch: [],
          },
        },
      });
      fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
      expect(
        await screen.findByText(
          "The Driver uses DC, because the setup of the device needs it. Best for " +
            "most needs.",
        ),
      ).toBeTruthy();
    });

    it("should not run the mode that the catalog gives on an IgH network", async () => {
      await openConfigure({
        network: "igh:0",
        opModes: [SM_MODE, DC_MODE],
        settings: {
          device: [],
          channels: [],
          mode: {
            index: 0x8020,
            subIndex: 9,
            value: 0,
            shared: false,
            opMode: { index: 1, assignActivate: 0x0300, sync0Cycle: 0, sync1Cycle: 0 },
            switch: [],
          },
        },
      });
      fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
      expect(
        await screen.findByText(
          "The Driver uses SM sync, because the device lists it. Best for most needs.",
        ),
      ).toBeTruthy();
    });

    it.each(MODE_TEXTS)(
      "should describe the clock ticks of a mode with $name",
      async ({ mode, option, help }) => {
        await openConfigure({ opModes: [mode] });
        await pickMode(option);
        expect(screen.getByText(help)).toBeTruthy();
      },
    );

    it("should keep a cycle to the nanosecond", async () => {
      const slave = await openConfigure({ opModes: [DC_MODE] });
      await pickMode(DC_DESCRIPTION);
      typeCycle("SYNC0 cycle", "62.5");
      const sync0 = screen.getByRole("textbox", { name: "SYNC0 cycle" });
      expect(sync0).toHaveProperty("value", "62μs 500ns");
      typeCycle("SYNC0 cycle", "62.6");
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 0,
          assignActivate: 0x0300,
          sync0Cycle: 62_600,
          sync1Cycle: 0,
        }),
      );
    });

    it("should put a cycle back to the mode's own when Auto is clicked", async () => {
      const slave = await openConfigure({ opModes: [DC_MODE] });
      await pickMode(DC_DESCRIPTION);
      typeCycle("SYNC0 cycle", "300");
      expect(screen.getByText("SYNC0 ticks every 300μs. SYNC1 is off.")).toBeTruthy();
      fireEvent.click(getIconButton(document.body, "auto"));
      expect(screen.getByRole("textbox", { name: "SYNC0 cycle" })).toHaveProperty(
        "value",
        "",
      );
      expect(
        screen.getByText("SYNC0 ticks each bus cycle. SYNC1 is off."),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 0,
          assignActivate: 0x0300,
          sync0Cycle: 0,
          sync1Cycle: 0,
        }),
      );
    });

    it("should say when a SYNC1 cycle turns on SYNC1", async () => {
      const slave = await openConfigure({ opModes: [DC_MODE] });
      await pickMode(DC_DESCRIPTION);
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
      const slave = await openConfigure({ opModes: [DC_MODE] });
      await pickMode(DC_DESCRIPTION);
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
      const slave = await openConfigure({ opModes: [FIXED_MODE] });
      await pickMode("Clock ticks: SYNC0 every 300μs.");
      typeCycle("SYNC1 cycle", "1000");
      expect(
        screen.getByText("SYNC1 cycle must be a multiple of the SYNC0 cycle"),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toBeUndefined();
    });

    it("should not save a SYNC0 cycle over the limit of its register", async () => {
      const slave = await openConfigure({ opModes: [DC_MODE] });
      await pickMode(DC_DESCRIPTION);
      typeCycle("SYNC0 cycle", "5000000");
      expect(
        screen.getByText("SYNC0 cycle of 5s is over the limit of 4s 294ms 967μs 295ns"),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toBeUndefined();
    });

    it("should not save a SYNC1 cycle over the limit of its register", async () => {
      const slave = await openConfigure({ opModes: [DC_MODE] });
      await pickMode(DC_DESCRIPTION);
      typeCycle("SYNC0 cycle", "1000000");
      typeCycle("SYNC1 cycle", "6000000");
      expect(
        screen.getByText("SYNC1 cycle of 6s is over the limit of 5s 294ms 967μs 295ns"),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toBeUndefined();
    });

    it("should not save a SYNC1 shift that puts SYNC1 before SYNC0", async () => {
      const mode = { ...createMode("DC (shift)", 0x0700), sync1Shift: -1000 };
      const slave = await openConfigure({ opModes: [mode] });
      await pickMode("Clock ticks: SYNC0 each bus cycle, SYNC1 with each SYNC0 tick.");
      typeCycle("SYNC0 cycle", "300");
      expect(
        screen.getByText("SYNC1 shift of -1μs is under the limit of 0s"),
      ).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toBeUndefined();
    });

    it("should save auto with a warning on a mode with no SYNC0 cycle", async () => {
      const slave = await openConfigure({
        opModes: [NO_CYCLE_MODE],
        opMode: {
          index: 0,
          assignActivate: 0x0300,
          sync0Cycle: 300_000,
          sync1Cycle: 0,
        },
      });
      fireEvent.click(screen.getByRole("button", { name: /^Mode/ }));
      fireEvent.click(
        await screen.findByText("The Driver uses DC (no cycle)", { exact: false }),
      );
      const help = screen.getByText("This mode gives no SYNC0 cycle. Type one.");
      expect(help.classList.contains("lyra--status-warning")).toBe(true);
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toBeUndefined(),
      );
    });

    it("should not save a chosen mode that gives no SYNC0 cycle", async () => {
      const smChoice = { index: 0, assignActivate: 0, sync0Cycle: 0, sync1Cycle: 0 };
      const slave = await openConfigure({
        opModes: [SM_MODE, NO_CYCLE_MODE],
        opMode: smChoice,
      });
      await pickMode("You must type a SYNC0 cycle for this mode.");
      const help = screen.getByText("This mode gives no SYNC0 cycle. Type one.");
      expect(help.classList.contains("lyra--status-error")).toBe(true);
      expect(screen.queryByText("each bus cycle", { exact: false })).toBeNull();
      fireEvent.click(findButton("Save"));
      await act(async () => {});
      expect(saving()).toBe(false);
      expect(await retrieveOpMode(slave.key)).toEqual(smChoice);
      typeCycle("SYNC0 cycle", "300");
      expect(
        screen.queryByText("This mode gives no SYNC0 cycle. Type one."),
      ).toBeNull();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 1,
          assignActivate: 0x0300,
          sync0Cycle: 300_000,
          sync1Cycle: 0,
        }),
      );
    });

    it("should drop the cycles of a mode that takes none on save", async () => {
      const slave = await openConfigure({
        opModes: [SM_MODE, PDI_MODE],
        opMode: {
          index: 1,
          assignActivate: 0x0301,
          sync0Cycle: 300_000,
          sync1Cycle: 1_000_000,
        },
      });
      expect(screen.getByText("The device sets its own clock ticks.")).toBeTruthy();
      fireEvent.click(findButton("Save"));
      await waitFor(async () =>
        expect(await retrieveOpMode(slave.key)).toEqual({
          index: 1,
          assignActivate: 0x0301,
          sync0Cycle: 0,
          sync1Cycle: 0,
        }),
      );
    });

    it("should show a mode the device no longer lists as an error", async () => {
      await openConfigure({ opModes: [DC_MODE], opMode: STALE_CHOICE });
      expect(screen.getByText(STALE_ERROR)).toBeTruthy();
      expect(screen.getByText("Mode 2 (0x0700)")).toBeTruthy();
    });

    it("should not save a mode the device no longer lists", async () => {
      const slave = await openConfigure({
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
      const slave = await openConfigure({
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

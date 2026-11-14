// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  channel,
  device,
  type ethercat,
  type rack,
  type Synnax,
  type task,
} from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { TimeStamp } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";
import { type z } from "zod";

import { EtherCAT } from "@/feature/ethercat";
import {
  createCalibratedDescription,
  createDewesoftXSettings,
  createIdentifier,
  createPDOs,
  createRepeatedNamePDOs,
  createRepeatedNameSlave,
  createSettingsDescription,
  createSlaveDevice,
  createUnresolvedReadChannel,
  STATUS_ENTRY,
  STRAIN_ENTRY,
  STRAIN_SAMPLE_ENTRY,
} from "@/feature/ethercat/testutil";
import { Device } from "@/platform/device";
import {
  awaitEditableForm,
  clickDeploy,
  createWithHook,
  deployAndAwaitTask,
  findChannelListItem,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

let testRack: rack.Rack;

beforeAll(async () => {
  testRack = await client.racks.create({ name: uniqueName("ecat_rack") });
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<EtherCAT.Task.ReadSchemas> = {
  name: "EtherCAT read task",
  type: EtherCAT.Task.READ_TYPE,
  config: EtherCAT.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<EtherCAT.Task.ReadSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, EtherCAT.Task.READ_SCHEMAS);

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (config: z.infer<EtherCAT.Task.ReadSchemas["config"]>) => {
  const draft = await createDraft(client, config);
  const statuses: Status.NotificationSpec[] = [];
  const rendered = await renderTaskFormTab(EtherCAT.Task.Read, {
    client,
    taskKey: draft.key,
    onStatuses: (next) => {
      statuses.length = 0;
      statuses.push(...next);
    },
  });
  await awaitEditableForm();
  return { ...rendered, draft, statuses };
};

/** Deploys a new read task with the config and unmounts its form. */
const deployRead = async (config: z.infer<EtherCAT.Task.ReadSchemas["config"]>) => {
  const rendered = await renderRead(config);
  const deployed = await deployAndAwaitTask(
    client,
    rendered.container,
    rendered.draft.key,
    EtherCAT.Task.READ_SCHEMAS,
  );
  rendered.unmount();
  return deployed;
};

const awaitStatus = async (
  statuses: Status.NotificationSpec[],
  pattern: RegExp,
): Promise<void> => {
  await waitFor(() => {
    const matched = statuses.some(
      (s) => pattern.test(s.description ?? "") || pattern.test(s.message),
    );
    expect(matched).toBe(true);
  });
};

const THREE_INPUTS: ethercat.PDOEntry[] = [
  { name: "Status", index: 0x6000, subIndex: 1, bitLength: 16, dataType: "uint16" },
  { name: "Value", index: 0x6000, subIndex: 2, bitLength: 16, dataType: "int16" },
  { name: "Overrange", index: 0x6000, subIndex: 3, bitLength: 8, dataType: "uint8" },
];

const LINEAR = { type: "linear", slope: 2, offset: 0 } as const;

const createStrainSlave = async () =>
  await createSlaveDevice(client, testRack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    pdos: { inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY], outputs: [] },
    settings: createSettingsDescription(),
  });

/** Creates a slave whose catalog calibrates the samples of {@link STRAIN_ENTRY}. */
const createCalibratedSlave = async () =>
  await createSlaveDevice(client, testRack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    pdos: { inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY], outputs: [] },
    settings: createCalibratedDescription(),
  });

/** Creates a slave that DewesoftX sets up and that sends no data only it reads. */
const createSharedSlave = async () =>
  await createSlaveDevice(client, testRack.key, {
    identifier: createIdentifier(),
    network: "eth0",
    pdos: { inputs: [STRAIN_ENTRY], outputs: [] },
    settings: createDewesoftXSettings(true),
  });

const MEASUREMENT = { index: 0x8040, subIndex: 1, dataType: "uint16", name: "" };
const RANGE = { index: 0x8040, subIndex: 3, dataType: "uint16", name: "" };
const EXCITATION = { index: 0x8040, subIndex: 0x1c, dataType: "float32", name: "" };

describe("Read", () => {
  it("should render channels from the task row's config with their port labels", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
        createUnresolvedReadChannel(slave.key, "Input"),
      ],
    });
    expect(await screen.findByRole("option", { name: /0x6000:01/ })).toBeTruthy();
    expect(screen.getByRole("option", { name: /^Input/ })).toBeTruthy();
  });

  it("should show the entry picker and data type when a channel is selected", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, inputs[1])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6010:01/ }));
    await waitFor(() => expect(screen.getByText("Device")).toBeTruthy());
    expect(await screen.findByText("Channel 2 › Input · 0x6010:01")).toBeTruthy();
    expect(screen.getByRole("button", { name: /^Entry/ })).toBeTruthy();
    expect(screen.getByText("Data type")).toBeTruthy();
    expect(screen.queryByText("Mode")).toBeNull();
  });

  it("should list each of eight repeated entry names apart in the picker", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, inputs[0])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    for (let i = 0; i < 8; i++)
      expect(
        await screen.findByRole("option", {
          name: new RegExp(`Channel ${i + 1} › Input`),
        }),
      ).toBeTruthy();
    expect(screen.getAllByText("0x6070:01")).toHaveLength(1);
  });

  it("should list an address that two PDOs map only once", async () => {
    const value = { ...STATUS_ENTRY, name: "Value", pdoName: "Channel 1" };
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [value, { ...value, pdoName: "Compact" }], outputs: [] },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, value)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    await waitFor(() =>
      expect(
        within(screen.getByRole("dialog")).getAllByRole("option", { name: /Value/ }),
      ).toHaveLength(1),
    );
    expect(screen.queryByText("Compact › Value")).toBeNull();
  });

  it("should set the address and data type of the entry picked", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, inputs[0]),
          dataType: "uint16",
        },
      ],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Channel 3 › Input/ }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0]).toMatchObject({
        type: "entry",
        index: 0x6020,
        subIndex: 1,
        dataType: "uint8",
      });
    });
  });

  it("should save the scale picked for a selected channel", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    expect(await screen.findByText("Scale", { selector: "label" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: /None/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Linear/ }));
    expect(await screen.findByText("Slope")).toBeTruthy();
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0]).toMatchObject({
        scale: { type: "linear", slope: 1, offset: 0 },
      });
    });
  });

  it("should calibrate a picked entry that the Driver calibrates", async () => {
    const slave = await createCalibratedSlave();
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_SAMPLE_ENTRY)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6040:01/ }));
    expect(await screen.findByText("Data type")).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Channel 1 › Value/ }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0]).toMatchObject({
        index: 0x6030,
        subIndex: 1,
        calibrated: true,
      });
    });
    expect(screen.queryByText("Data type")).toBeNull();
  });

  it("should stop calibrating a channel moved to a slave that does not calibrate it", async () => {
    const calibrating = await createCalibratedSlave();
    const plain = await createStrainSlave();
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(calibrating.key, STRAIN_ENTRY),
          calibrated: true,
        },
      ],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    fireEvent.click(await findDialogTriggerByText(calibrating.name));
    fireEvent.change(await screen.findByPlaceholderText("Search devices..."), {
      target: { value: plain.name },
    });
    fireEvent.click(await screen.findByText(plain.name));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0]).toMatchObject({
        device: plain.key,
        calibrated: false,
      });
    });
    expect(await screen.findByText("Data type")).toBeTruthy();
  });

  it("should save the setting picked for an entry that the catalog describes", async () => {
    const slave = await createStrainSlave();
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    expect(screen.queryByText("Excitation voltage")).toBeNull();
    fireEvent.click(await screen.findByRole("button", { name: /^Measurement/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Bridge/ }));
    expect(await screen.findByText("Excitation voltage")).toBeTruthy();
    expect(
      await screen.findByText("A bridge needs an excitation voltage above 0 V."),
    ).toBeTruthy();
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.startupParameters).toEqual([
        {
          device: slave.key,
          parameters: [
            { ...MEASUREMENT, value: 6 },
            { ...RANGE, value: 0 },
            { ...EXCITATION, value: 0 },
          ],
        },
      ]);
    });
  });

  it("should name a channel in the list by its entry, then its address", async () => {
    const slave = await createSharedSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    expect(
      await screen.findByRole("option", { name: /^Value 0x6030:01/ }),
    ).toBeTruthy();
  });

  it("should say that DewesoftX sets the settings of a channel of a slave it sets up", async () => {
    const slave = await createSharedSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    expect(
      await screen.findByText("DewesoftX sets the settings of this device."),
    ).toBeTruthy();
    expect(screen.queryByRole("button", { name: /^Measurement/ })).toBeNull();
  });

  it("should name a slave in DewesoftX mode before Start and offer the switch", async () => {
    const slave = await createSharedSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    expect(
      await screen.findByText(
        `Slave ${slave.name} is set up for DewesoftX. Switch it to Synnax before you ` +
          "start the task.",
      ),
    ).toBeTruthy();
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    expect(await screen.findByText("DewesoftX sets up this device")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Switch to Synnax" })).toBeTruthy();
  });

  it("should ask to set up in DewesoftX a slave that Synnax cannot switch", async () => {
    const settings = createDewesoftXSettings(true);
    if (settings.mode == null) throw new Error("DewesoftX settings have no mode");
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [STRAIN_ENTRY], outputs: [] },
      settings: { ...settings, mode: { ...settings.mode, switch: [] } },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    expect(
      await screen.findByText(
        `Slave ${slave.name} is set up for DewesoftX. Set it up for Synnax in ` +
          "DewesoftX before you start the task.",
      ),
    ).toBeTruthy();
  });

  it("should add the DewesoftX step to a task-wide error", async () => {
    const shared = await createSharedSlave();
    const other = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth1",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(shared.key, STRAIN_ENTRY),
        EtherCAT.Task.createEntryReadChannel(other.key, STATUS_ENTRY),
      ],
    });
    expect(
      await screen.findByText(
        "All slaves must be on the same network. Found: eth0, eth1. Pick slaves on " +
          `one network. Slave ${shared.name} is set up for DewesoftX. Switch it to ` +
          "Synnax before you start the task.",
      ),
    ).toBeTruthy();
  });

  it("should not warn of a slave in DewesoftX mode that only disabled channels use", async () => {
    const shared = await createSharedSlave();
    const position = Math.floor(Math.random() * 2 ** 31);
    const other = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      position,
      pdos: createPDOs(),
    });
    const stale = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      position,
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(shared.key, STRAIN_ENTRY),
          disabled: true,
        },
        EtherCAT.Task.createEntryReadChannel(other.key, STATUS_ENTRY),
      ],
    });
    // The whole text matches only when the error carries no DewesoftX step.
    expect(
      await screen.findByText(
        `Slaves ${other.name} and ${stale.name} are both at position ${position} on ` +
          "eth0. Delete the one that is not on the network.",
      ),
    ).toBeTruthy();
  });

  it("should deploy a task on a slave in DewesoftX mode, as the form only warns", async () => {
    const slave = await createSharedSlave();
    const deployed = await deployRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    expect(deployed.config.channels[0].channel).not.toBe(0);
  });

  it("should share the settings of an input between the entries that read it", async () => {
    const slave = await createStrainSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY),
        EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_SAMPLE_ENTRY),
      ],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    fireEvent.click(await screen.findByRole("button", { name: /^Measurement/ }));
    fireEvent.click(await screen.findByRole("option", { name: /Bridge/ }));
    fireEvent.click(await screen.findByRole("option", { name: /0x6040:01/ }));
    expect(
      await screen.findByRole("button", { name: /^Measurement.*Bridge/ }),
    ).toBeTruthy();
  });

  it("should show why an unresolved channel cannot run", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createUnresolvedReadChannel(slave.key, "Input")],
    });
    fireEvent.click(await screen.findByRole("option", { name: /Input/ }));
    const addresses = Array.from(
      { length: 8 },
      (_, i) => `0x60${i}0:01 (Channel ${i + 1})`,
    ).join(", ");
    expect(
      await screen.findByText(
        `'Input' matches more than one input entry on slave ${slave.name}: ` +
          `${addresses}. Pick an entry.`,
      ),
    ).toBeTruthy();
    expect(screen.queryByText("Data type")).toBeNull();
  });

  it("should turn an unresolved channel into an entry when one is picked", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createUnresolvedReadChannel(slave.key, "Input")],
    });
    fireEvent.click(await screen.findByRole("option", { name: /Input/ }));
    fireEvent.click(await screen.findByText("Select entry"));
    fireEvent.click(await screen.findByRole("option", { name: /Channel 3 › Input/ }));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      const [ch] = saved.config.channels;
      expect(ch).toMatchObject({ type: "entry", index: 0x6020, subIndex: 1 });
      expect("pdo" in ch).toBe(false);
    });
    expect(await screen.findByText("Data type")).toBeTruthy();
  });

  it("should title a channel by its address when its slave lacks the entry", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, {
          ...STATUS_ENTRY,
          subIndex: 2,
        }),
      ],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:02/ }));
    await waitFor(() => expect(screen.getAllByText("0x6000:02")).toHaveLength(2));
    expect(screen.queryByText(/· 0x6000:02/)).toBeNull();
  });

  it("should ask for a slave when the picker opens with none selected", async () => {
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createReadChannel([])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /No entry/ }));
    fireEvent.click(await screen.findByText("Select entry"));
    expect(await screen.findByText("Select a device first.")).toBeTruthy();
  });

  it("should ask for a scan when the slave has no input entries", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [], outputs: createPDOs().outputs },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [{ ...EtherCAT.Task.createReadChannel([]), device: slave.key }],
    });
    fireEvent.click(await screen.findByRole("option", { name: /No entry/ }));
    fireEvent.click(await screen.findByText("Select entry"));
    expect(
      await screen.findByText(
        "This device has no input entries. Scan the network again.",
      ),
    ).toBeTruthy();
  });

  describe("a slave whose inputs only DewesoftX reads", () => {
    const renderChannel = async (shared: boolean) => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: { inputs: [STRAIN_SAMPLE_ENTRY], outputs: [] },
        settings: createDewesoftXSettings(shared),
      });
      await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [{ ...EtherCAT.Task.createReadChannel([]), device: slave.key }],
      });
      fireEvent.click(await screen.findByRole("option", { name: /No entry/ }));
    };

    it("should show the setup note and the switch to Synnax in the task", async () => {
      await renderChannel(true);
      expect(
        await screen.findByText("Only DewesoftX can use this device"),
      ).toBeTruthy();
      expect(screen.getByRole("button", { name: "Switch to Synnax" })).toBeTruthy();
    });

    it("should say who sets up the slave in an empty picker", async () => {
      await renderChannel(false);
      fireEvent.click(await screen.findByText("Select entry"));
      expect(
        await screen.findByText("Only DewesoftX can use this device."),
      ).toBeTruthy();
      expect(screen.queryByRole("button", { name: "Switch to Synnax" })).toBeNull();
    });
  });

  it("should add a channel for every unused input of a slave from the context menu", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: THREE_INPUTS, outputs: [] },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, THREE_INPUTS[0])],
    });
    fireEvent.contextMenu(await screen.findByRole("option", { name: /0x6000:01/ }));
    fireEvent.click(await screen.findByText(`Add every input of ${slave.name}`));
    await waitFor(() =>
      expect(screen.getByRole("option", { name: /0x6000:02/ })).toBeTruthy(),
    );
    expect(screen.getByRole("option", { name: /0x6000:03/ })).toBeTruthy();
    expect(screen.getAllByRole("option", { name: /0x6000:01/ })).toHaveLength(1);
    fireEvent.contextMenu(screen.getByRole("option", { name: /0x6000:01/ }));
    await screen.findByText("Duplicate");
    expect(screen.queryByText(/Add every input/)).toBeNull();
  });

  it("should not offer every input of a slave when more than one channel is selected", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: THREE_INPUTS, outputs: [] },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: THREE_INPUTS.slice(0, 2).map((e) =>
        EtherCAT.Task.createEntryReadChannel(slave.key, e),
      ),
    });
    const first = await screen.findByRole("option", { name: /0x6000:01/ });
    const second = screen.getByRole("option", { name: /0x6000:02/ });
    fireEvent.contextMenu(first);
    expect(await screen.findByText(`Add every input of ${slave.name}`)).toBeTruthy();
    fireEvent.keyDown(window, { key: "Escape", code: "Escape" });
    fireEvent.keyUp(window, { key: "Escape", code: "Escape" });
    await waitFor(() => expect(screen.queryByText("Duplicate")).toBeNull());
    fireEvent.click(first);
    fireEvent.keyDown(window, { key: "Control", code: "ControlLeft" });
    fireEvent.click(second);
    fireEvent.keyUp(window, { key: "Control", code: "ControlLeft" });
    await waitFor(() => {
      expect(first.getAttribute("aria-selected")).toBe("true");
      expect(second.getAttribute("aria-selected")).toBe("true");
    });
    fireEvent.contextMenu(second);
    await screen.findByText("Duplicate");
    expect(screen.queryByText(/Add every input/)).toBeNull();
  });

  it("should add every input of an unresolved channel's slave from the context menu", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [createUnresolvedReadChannel(slave.key, "Input")],
    });
    fireEvent.contextMenu(await screen.findByRole("option", { name: /Input/ }));
    fireEvent.click(await screen.findByText(`Add every input of ${slave.name}`));
    await waitFor(() =>
      expect(screen.getByRole("option", { name: /0x6000:01/ })).toBeTruthy(),
    );
    expect(screen.getByRole("option", { name: /0x6070:01/ })).toBeTruthy();
  });

  it("should create the index and data channels, update the slave, and save the task", async () => {
    const identifier = createIdentifier();
    const namedChannel = uniqueName("ecat_named");
    const second = { ...STATUS_ENTRY, name: "Counter", subIndex: 2 };
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY, second], outputs: [] },
    });
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, second),
          name: namedChannel,
        },
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(created.type).toBe(EtherCAT.Task.READ_TYPE);
    expect(created.rack).toBe(testRack.key);
    const [status, named] = created.config.channels;
    expect(status.channel).not.toBe(0);
    expect(named.channel).not.toBe(0);

    const statusChannel = await client.channels.retrieve(status.channel);
    expect(statusChannel.name).toBe(`${identifier}_Status`);
    expect(statusChannel.dataType.toString()).toBe("uint16");
    const namedRetrieved = await client.channels.retrieve(named.channel);
    expect(namedRetrieved.name).toBe(namedChannel);

    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.readIndex).not.toBe(0);
    expect(updated.properties.read.channels).toEqual([
      {
        index: 0x6000,
        subIndex: 1,
        scaled: false,
        calibrated: false,
        channel: status.channel,
      },
      {
        index: 0x6000,
        subIndex: 2,
        scaled: false,
        calibrated: false,
        channel: named.channel,
      },
    ]);
    const index = await client.channels.retrieve(updated.properties.readIndex);
    expect(index.name).toBe(`${identifier}_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should create one channel for each of eight entries that share a name", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { identifier } = slave.properties;
    const { inputs } = createRepeatedNamePDOs();
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: inputs.map((e) => EtherCAT.Task.createEntryReadChannel(slave.key, e)),
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const keys = created.config.channels.map(({ channel }) => channel);
    expect(new Set(keys).size).toBe(8);
    const channels = await client.channels.retrieve(keys);
    expect(channels.map(({ name }) => name).sort()).toEqual(
      inputs.map((_, i) => `${identifier}_Channel_${i + 1}_Input`).sort(),
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    inputs.forEach((e, i) =>
      expect(updated.properties.read.channels).toContainEqual({
        index: e.index,
        subIndex: 1,
        scaled: false,
        calibrated: false,
        channel: keys[i],
      }),
    );
  });

  it("should show why a channel that shares an entry cannot run, and mark the entry in use", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, inputs[3]),
        EtherCAT.Task.createEntryReadChannel(slave.key, inputs[3]),
      ],
    });
    fireEvent.click((await screen.findAllByRole("option", { name: /0x6030:01/ }))[0]);
    expect(
      await screen.findByText(
        `More than one channel uses entry 0x6030:01 on slave ${slave.name}. Pick a ` +
          "different entry for one of them.",
      ),
    ).toBeTruthy();
    fireEvent.click(await screen.findByRole("button", { name: /^Entry/ }));
    expect(
      await screen.findByRole("option", {
        name: /Channel 4 › Input.*0x6030:01 \(in use\)/,
      }),
    ).toBeTruthy();
    expect(
      screen.getByRole("option", { name: /Channel 1 › Input.*0x6000:01$/ }),
    ).toBeTruthy();
  });

  it("should name the slave that is not connected at the position of another device", async () => {
    const position = Math.floor(Math.random() * 2 ** 31);
    const props = {
      identifier: createIdentifier(),
      network: "eth0",
      position,
      pdos: createPDOs(),
    };
    const stale = await createSlaveDevice(client, testRack.key, props);
    const connected = await createSlaveDevice(client, testRack.key, {
      ...props,
      identifier: createIdentifier(),
    });
    await client.statuses.set({
      owner: device.resourceID(connected.key),
      name: connected.name,
      variant: "success",
      message: "Device connected",
      time: TimeStamp.now(),
      details: { rack: testRack.key, connected: true },
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(stale.key, STATUS_ENTRY)],
    });
    expect(
      await screen.findByText(
        `Slaves ${stale.name} and ${connected.name} are both at position ` +
          `${position} on eth0. Delete ${stale.name}, which is not connected.`,
      ),
    ).toBeTruthy();
  });

  it("should show why a channel on an entry the slave does not have cannot run", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    // The address of an entry picked on another slave before the slave was switched.
    const { inputs } = createRepeatedNamePDOs();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, inputs[3])],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    expect(
      await screen.findByText(
        `Slave ${slave.name} has no input entry at 0x6030:01. Pick an entry.`,
      ),
    ).toBeTruthy();
  });

  it("should save one value for each setting of each input that a channel reads", async () => {
    const slave = await createStrainSlave();
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY),
        EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_SAMPLE_ENTRY),
      ],
      startupParameters: [
        {
          device: slave.key,
          parameters: [
            { ...EXCITATION, value: 5 },
            { ...MEASUREMENT, value: 6 },
          ],
        },
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(created.config.startupParameters).toEqual([
      {
        device: slave.key,
        parameters: [
          { ...MEASUREMENT, value: 6 },
          { ...RANGE, value: 0 },
          { ...EXCITATION, value: 5 },
        ],
      },
    ]);
  });

  it("should keep the setup of the card for a task that sets no values", async () => {
    const card: Record<number, number> = { 1: 6, 3: 2, 0x1c: 5 };
    const description = createSettingsDescription();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY], outputs: [] },
      settings: {
        ...description,
        channels: description.channels.map((ch) => ({
          ...ch,
          settings: ch.settings.map((f) => ({ ...f, value: card[f.subIndex] })),
        })),
      },
    });
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(created.config.startupParameters).toEqual([
      {
        device: slave.key,
        parameters: [
          { ...MEASUREMENT, value: 6 },
          { ...RANGE, value: 2 },
          { ...EXCITATION, value: 5 },
        ],
      },
    ]);
  });

  it("should save no settings for a slave that DewesoftX sets up", async () => {
    const slave = await createSharedSlave();
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
      startupParameters: [
        { device: slave.key, parameters: [{ ...MEASUREMENT, value: 0 }] },
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(created.config.startupParameters).toEqual([]);
  });

  it("should leave out the settings of an input that only a disabled channel reads", async () => {
    const slave = await createStrainSlave();
    const other = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(other.key, createPDOs().inputs[0]),
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY),
          disabled: true,
        },
      ],
      startupParameters: [
        { device: slave.key, parameters: [{ ...MEASUREMENT, value: 6 }] },
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(created.config.startupParameters).toEqual([]);
  });

  it("should mark in the list a channel whose setting breaks a rule of the catalog", async () => {
    const slave = await createStrainSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
      startupParameters: [
        { device: slave.key, parameters: [{ ...MEASUREMENT, value: 6 }] },
      ],
    });
    const option = await screen.findByRole("option", { name: /0x6030:01/ });
    await waitFor(() =>
      expect(option.classList.contains("lyra--status-error")).toBe(true),
    );
  });

  it("should show why a channel whose setting breaks a rule of the catalog cannot run", async () => {
    const slave = await createStrainSlave();
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
      startupParameters: [
        { device: slave.key, parameters: [{ ...MEASUREMENT, value: 6 }] },
      ],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6030:01/ }));
    expect(
      await screen.findByText(
        `Entry 0x6030:01 on slave ${slave.name}: A bridge needs an excitation ` +
          "voltage above 0 V.",
      ),
    ).toBeTruthy();
  });

  it("should configure a task whose unresolved channel is disabled", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    const { inputs } = createRepeatedNamePDOs();
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        EtherCAT.Task.createEntryReadChannel(slave.key, inputs[0]),
        createUnresolvedReadChannel(slave.key, "Input", { disabled: true }),
      ],
    });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const [bound, disabled] = created.config.channels;
    expect(bound.channel).not.toBe(0);
    expect(disabled).toMatchObject({ type: "unresolved", pdo: "Input", channel: 0 });
  });

  it("should show why a channel with no entry cannot run", async () => {
    const slave = await createRepeatedNameSlave(client, testRack.key);
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [{ ...EtherCAT.Task.createReadChannel([]), device: slave.key }],
    });
    expect(
      await screen.findByText(
        `A channel on slave ${slave.name} has no entry. Pick an entry.`,
      ),
    ).toBeTruthy();
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    };
    const first = await renderRead(config);
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const second = await renderRead(config);
    await screen.findByText(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: EtherCAT.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should reject a custom name that the slave maps to another entry", async () => {
    const identifier = createIdentifier();
    const second = { ...STATUS_ENTRY, name: "Counter", subIndex: 2 };
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY, second], outputs: [] },
    });
    const first = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, second),
          name: `${identifier}_Status`,
        },
      ],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `A channel named ${identifier}_Status already exists\\. Give the channel ` +
          "another name\\.",
      ),
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.read.channels).not.toContainEqual(
      expect.objectContaining({ index: 0x6000, subIndex: 2 }),
    );
  });

  it("should reject an index name that another slave's index holds", async () => {
    const base = createIdentifier();
    const index = await client.channels.create({
      name: `${base}_state_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    const owner = await createSlaveDevice(client, testRack.key, {
      identifier: base,
      network: "eth0",
      writeStateIndex: index.key,
    });
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: `${base}_state`,
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY], outputs: [] },
    });
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `Slave ${owner.name} uses the index channel ${base}_state_time\\. Change ` +
          "the identifier",
      ),
    );
  });

  it("should reject a custom name that the slave's index holds", async () => {
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY], outputs: [] },
    });
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
          name: `${identifier}_time`,
        },
      ],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `A channel named ${identifier}_time already exists\\. Give the channel ` +
          "another name\\.",
      ),
    );
  });

  it("should reject an index name that another slave's command index holds", async () => {
    const base = createIdentifier();
    const index = await client.channels.create({
      name: `${base}_cmd_time`,
      dataType: "timestamp",
      isIndex: true,
    });
    const cmd = await client.channels.create({
      name: `${base}_cmd`,
      dataType: "uint8",
      index: index.key,
    });
    const deleted = await client.channels.create({
      name: `${base}_deleted_cmd`,
      dataType: "uint8",
      index: index.key,
    });
    await client.channels.delete(deleted.key);
    const owner = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      write: {
        channels: [
          { index: 0x7000, subIndex: 1, command: deleted.key, state: 0 },
          { index: 0x7000, subIndex: 2, command: cmd.key, state: 0 },
        ],
      },
    });
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: `${base}_cmd`,
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY], outputs: [] },
    });
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `Slave ${owner.name} uses the index channel ${base}_cmd_time\\. Change the ` +
          "identifier",
      ),
    );
  });

  it("should reject the index that a renamed slave keeps", async () => {
    const identifier = createIdentifier();
    const pdos = { inputs: [STATUS_ENTRY], outputs: [] };
    const renamed = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos,
    });
    const first = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(renamed.key, STATUS_ENTRY)],
    });
    await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();
    const deployed = await client.devices.retrieve({
      key: renamed.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    await Device.setProperties(
      client,
      {
        key: deployed.key,
        properties: { ...deployed.properties, identifier: createIdentifier() },
      },
      deployed.properties,
      EtherCAT.Device.SLAVE_SCHEMAS,
    );
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos,
    });
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `Slave ${renamed.name} uses the index channel ${identifier}_time\\. Change ` +
          "the identifier",
      ),
    );
  });

  it("should reuse the existing index and channels when redeployed", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    };
    const first = await renderRead(config);
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();

    const second = await renderRead(config);
    const secondTask = await deployAndAwaitTask(
      client,
      second.container,
      second.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(secondTask.config.channels[0].channel).toBe(
      firstTask.config.channels[0].channel,
    );
  });

  it("should give a calibrated entry a float64 channel", async () => {
    const slave = await createCalibratedSlave();
    const deployed = await deployRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY),
          calibrated: true,
        },
      ],
    });
    const created = await client.channels.retrieve(deployed.config.channels[0].channel);
    expect(created.dataType.toString()).toBe("float64");
  });

  it("should keep the raw data type of an uncalibrated channel on a calibrated entry", async () => {
    const slave = await createCalibratedSlave();
    const deployed = await deployRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_ENTRY)],
    });
    const created = await client.channels.retrieve(deployed.config.channels[0].channel);
    expect(created.dataType.toString()).toBe("int32");
  });

  it("should give a scaled entry a float64 channel apart from its raw channel", async () => {
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: createPDOs(),
    });
    const raw = EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY);
    const first = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [raw],
    });
    const rawTask = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    first.unmount();

    const second = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [{ ...raw, scale: { type: "linear", slope: 2, offset: 0 } }],
    });
    const scaledTask = await deployAndAwaitTask(
      client,
      second.container,
      second.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const rawKey = rawTask.config.channels[0].channel;
    const scaledKey = scaledTask.config.channels[0].channel;
    expect(scaledKey).not.toBe(rawKey);
    const scaled = await client.channels.retrieve(scaledKey);
    expect(scaled.name).toBe(`${identifier}_Status_scaled`);
    expect(scaled.dataType.toString()).toBe("float64");
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.read.channels).toEqual([
      { index: 0x6000, subIndex: 1, scaled: false, calibrated: false, channel: rawKey },
      {
        index: 0x6000,
        subIndex: 1,
        scaled: true,
        calibrated: false,
        channel: scaledKey,
      },
    ]);
  });

  it("should give a calibrated scaled channel its own channel and name", async () => {
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: createPDOs(),
    });
    const scaled = (calibrated: boolean): ethercat.EntryReadChannel => ({
      ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
      scale: { type: "linear", slope: 2, offset: 0 },
      calibrated,
    });
    const deploy = async (ch: ethercat.EntryReadChannel) => {
      const { container, draft, unmount } = await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [ch],
      });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        EtherCAT.Task.READ_SCHEMAS,
      );
      unmount();
      return deployed.config.channels[0].channel;
    };
    const rawKey = await deploy(scaled(false));
    const calibratedKey = await deploy(scaled(true));
    expect(calibratedKey).not.toBe(rawKey);
    const [raw, calibrated] = await client.channels.retrieve([rawKey, calibratedKey]);
    expect(raw.name).toBe(`${identifier}_Status_scaled`);
    expect(calibrated.name).toBe(`${identifier}_Status_scaled_calibrated`);
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.read.channels).toEqual([
      { index: 0x6000, subIndex: 1, scaled: true, calibrated: false, channel: rawKey },
      {
        index: 0x6000,
        subIndex: 1,
        scaled: true,
        calibrated: true,
        channel: calibratedKey,
      },
    ]);
  });

  describe("a typed name that the other channel of the entry holds", () => {
    type Overrides = Partial<ethercat.EntryReadChannel>;

    const expectSecondDeployRejected = async (first: Overrides, second: Overrides) => {
      const name = uniqueName("ecat_typed");
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: createPDOs(),
      });
      const render = async (overrides: Overrides) =>
        await renderRead({
          ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
          channels: [
            {
              ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
              name,
              ...overrides,
            },
          ],
        });
      const initial = await render(first);
      await deployAndAwaitTask(
        client,
        initial.container,
        initial.draft.key,
        EtherCAT.Task.READ_SCHEMAS,
      );
      initial.unmount();
      const { container, statuses } = await render(second);
      await clickDeploy(container);
      await awaitStatus(
        statuses,
        new RegExp(
          `A channel named ${name} already exists\\. Give the channel another name\\.`,
        ),
      );
    };

    it("should reject it for a scaled channel when the raw channel holds it", async () => {
      await expectSecondDeployRejected({}, { scale: LINEAR });
    });

    it("should reject it for a raw channel when the scaled channel holds it", async () => {
      await expectSecondDeployRejected({ scale: LINEAR }, {});
    });
  });

  it("should reject two new channels that share a typed name", async () => {
    const name = uniqueName("ecat_typed");
    const pdos = createRepeatedNamePDOs();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos,
    });
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: pdos.inputs.slice(0, 2).map((entry) => ({
        ...EtherCAT.Task.createEntryReadChannel(slave.key, entry),
        name,
      })),
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(
        `Two channels of the task use the name ${name}\\. Choose another name for ` +
          "one of them\\.",
      ),
    );
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.read.channels).toEqual([]);
  });

  it("should reject a typed name that new channels of two slaves share", async () => {
    const name = uniqueName("ecat_typed");
    const slaves = await Promise.all(
      [0, 1].map(
        async () =>
          await createSlaveDevice(client, testRack.key, {
            identifier: createIdentifier(),
            network: "eth0",
            pdos: createPDOs(),
          }),
      ),
    );
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: slaves.map((slave) => ({
        ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
        name,
      })),
    });
    await clickDeploy(container);
    await awaitStatus(
      statuses,
      new RegExp(`Two channels of the task use the name ${name}\\.`),
    );
    for (const slave of slaves) {
      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.readIndex).toBe(0);
      expect(updated.properties.read.channels).toEqual([]);
    }
    expect(await client.channels.retrieve({ name: [name] })).toEqual([]);
  });

  it("should deploy once the user fixes a rejected name", async () => {
    const name = uniqueName("ecat_typed");
    const pdos = createRepeatedNamePDOs();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos,
    });
    const render = async (second: string) =>
      await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [
          { ...EtherCAT.Task.createEntryReadChannel(slave.key, pdos.inputs[0]), name },
          {
            ...EtherCAT.Task.createEntryReadChannel(slave.key, pdos.inputs[1]),
            name: second,
          },
        ],
      });
    const rejected = await render(name);
    await clickDeploy(rejected.container);
    await awaitStatus(rejected.statuses, /Two channels of the task use the name/);
    rejected.unmount();
    const fixed = await render(`${name}_2`);
    const deployed = await deployAndAwaitTask(
      client,
      fixed.container,
      fixed.draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const created = await client.channels.retrieve(
      deployed.config.channels.map(({ channel }) => channel),
    );
    expect(created.map((c) => c.name).toSorted()).toEqual([name, `${name}_2`]);
  });

  it("should keep a typed name for a scaled channel when no raw channel holds it", async () => {
    const name = uniqueName("ecat_typed");
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const { container, draft } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
          name,
          scale: LINEAR,
        },
      ],
    });
    const deployed = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      EtherCAT.Task.READ_SCHEMAS,
    );
    const scaled = await client.channels.retrieve(deployed.config.channels[0].channel);
    expect(scaled.name).toBe(name);
  });

  it("should reuse a scaled entry's float64 channel when redeployed", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
          scale: LINEAR,
        },
      ],
    };
    const deploy = async () => (await deployRead(config)).config.channels[0].channel;
    const first = await deploy();
    expect(await deploy()).toBe(first);
  });

  it("should keep the channels that a slave stores as a map keyed by address", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    const config = {
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    };
    const deploy = async () => (await deployRead(config)).config.channels[0].channel;
    const first = await deploy();
    await client.devices.dispatch(
      slave.key,
      device.set({
        path: ["properties", "read", "channels"],
        value: { [`${0x6000}_1`]: first },
      }),
    );
    expect(await deploy()).toBe(first);
    const updated = await client.devices.retrieve({
      key: slave.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.read.channels).toEqual([
      { index: 0x6000, subIndex: 1, scaled: false, calibrated: false, channel: first },
    ]);
  });

  describe("a data type that differs from the type of the entry's channel", () => {
    const createStatusSlave = async () => {
      const identifier = createIdentifier();
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const statusChannel = (dataType: string, name = "") => ({
        ...EtherCAT.Task.createEntryReadChannel(slave.key, {
          ...STATUS_ENTRY,
          dataType,
        }),
        name,
      });
      const tasks: task.Key[] = [];
      const deploy = async (dataTypes: string[], name = "") => {
        const deployed = await deployRead({
          ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
          channels: dataTypes.map((dataType) => statusChannel(dataType, name)),
        });
        tasks.push(deployed.key);
        return deployed.config.channels.map(({ channel }) => channel);
      };
      return { identifier, slave, statusChannel, deploy, tasks };
    };

    it("should get its own channel, and the first type should keep its channel", async () => {
      const { identifier, slave, deploy } = await createStatusSlave();
      const [first] = await deploy(["uint16"]);
      const [typed] = await deploy(["int16"]);
      expect(typed).not.toBe(first);
      const created = await client.channels.retrieve(typed);
      expect(created.name).toBe(`${identifier}_Status_int16`);
      expect(created.dataType.toString()).toBe("int16");
      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.read.channels).toEqual([
        {
          index: 0x6000,
          subIndex: 1,
          scaled: false,
          calibrated: false,
          channel: first,
        },
        {
          index: 0x6000,
          subIndex: 1,
          scaled: false,
          calibrated: false,
          channel: typed,
        },
      ]);
      expect(await deploy(["uint16"])).toEqual([first]);
    });

    it("should keep its own channel after the entry's first channel is deleted", async () => {
      const { deploy, tasks } = await createStatusSlave();
      const [first] = await deploy(["uint16"]);
      const [typed] = await deploy(["int16"]);
      // The Core refuses to delete a channel that a task uses.
      await client.tasks.delete(tasks);
      await client.channels.delete(first);
      expect(await deploy(["int16"])).toEqual([typed]);
    });

    it("should replace the binding of a deleted channel", async () => {
      const { slave, deploy, tasks } = await createStatusSlave();
      const [first] = await deploy(["uint16"]);
      await client.tasks.delete(tasks);
      await client.channels.delete(first);
      const [second] = await deploy(["uint16"]);
      expect(second).not.toBe(first);
      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.read.channels).toEqual([
        {
          index: 0x6000,
          subIndex: 1,
          scaled: false,
          calibrated: false,
          channel: second,
        },
      ]);
    });

    it("should end the name the user gave it in its type", async () => {
      const { deploy } = await createStatusSlave();
      const name = uniqueName("ecat_pressure");
      const [first] = await deploy(["uint16"], name);
      const [typed] = await deploy(["int16"], name);
      expect(typed).not.toBe(first);
      const created = await client.channels.retrieve(typed);
      expect(created.name).toBe(`${name}_int16`);
      expect(created.dataType.toString()).toBe("int16");
      expect(await deploy(["uint16"], name)).toEqual([first]);
    });

    it("should show the name of its own channel before deploying", async () => {
      const { identifier, statusChannel, deploy } = await createStatusSlave();
      await deploy(["uint16"]);
      await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [statusChannel("int16")],
      });
      const item = await findChannelListItem(`${identifier}_Status_int16`);
      expect(within(item).getByRole("img", { name: "New channel" })).toBeTruthy();
    });

    it("should show the name the user gave it in its type before deploying", async () => {
      const { statusChannel, deploy } = await createStatusSlave();
      const name = uniqueName("ecat_pressure");
      await deploy(["uint16"], name);
      await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [statusChannel("int16", name)],
      });
      const item = await findChannelListItem(`${name}_int16`);
      expect(within(item).getByRole("img", { name: "New channel" })).toBeTruthy();
    });

    it("should keep one scaled channel for every raw type", async () => {
      const { identifier, statusChannel } = await createStatusSlave();
      const deployScaled = async (dataType: string) =>
        (
          await deployRead({
            ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
            channels: [{ ...statusChannel(dataType), scale: LINEAR }],
          })
        ).config.channels[0].channel;
      const first = await deployScaled("uint16");
      expect(await deployScaled("int16")).toBe(first);
      expect((await client.channels.retrieve(first)).name).toBe(
        `${identifier}_Status_scaled`,
      );
    });

    it("should not end the name of a scaled channel in its raw type", async () => {
      const { statusChannel, deploy } = await createStatusSlave();
      await deploy(["uint16"]);
      const name = uniqueName("ecat_pressure");
      const deployed = await deployRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [{ ...statusChannel("int16", name), scale: LINEAR }],
      });
      const key = deployed.config.channels[0].channel;
      expect((await client.channels.retrieve(key)).name).toBe(name);
    });

    it("should not add its type to a name that ends in it", async () => {
      const { deploy } = await createStatusSlave();
      const name = uniqueName("ecat_pressure");
      await deploy(["uint16"], name);
      const [typed] = await deploy(["int16"], `${name}_int16`);
      expect((await client.channels.retrieve(typed)).name).toBe(`${name}_int16`);
    });
  });

  it.each([
    ["in DewesoftX mode", true, "In Configure, switch the slave to Synnax"],
    ["switched but not restarted", false, "Restart the power of the slave"],
  ])(
    "should show why a channel of an input that only DewesoftX reads cannot run on a slave %s",
    async (_, shared, step) => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier: createIdentifier(),
        network: "eth0",
        pdos: { inputs: [STRAIN_ENTRY, STRAIN_SAMPLE_ENTRY], outputs: [] },
        settings: createDewesoftXSettings(shared),
      });
      await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [
          EtherCAT.Task.createEntryReadChannel(slave.key, STRAIN_SAMPLE_ENTRY),
        ],
      });
      fireEvent.click(await screen.findByRole("option", { name: /0x6040:01/ }));
      expect(
        await screen.findByText(
          `Entry 0x6040:01 on slave ${slave.name} sends data that only DewesoftX ` +
            `reads. ${step}.`,
        ),
      ).toBeTruthy();
    },
  );

  describe("a slave with the identifier of a deleted slave", () => {
    const deploySlave = async (
      identifier: string,
      dataType = STATUS_ENTRY.dataType,
    ) => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const { container, draft, unmount } = await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [
          {
            ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
            dataType,
          },
        ],
      });
      const deployed = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        EtherCAT.Task.READ_SCHEMAS,
      );
      unmount();
      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      return {
        slave: updated,
        channel: deployed.config.channels[0].channel,
        task: deployed.key,
      };
    };

    // The Core refuses to delete a slave that a task uses.
    const deleteSlave = async (slave: string, task: string) => {
      await client.tasks.delete(task);
      await client.devices.delete(slave);
    };

    const renderSlave = async (
      identifier: string,
      dataType = STATUS_ENTRY.dataType,
    ) => {
      const slave = await createSlaveDevice(client, testRack.key, {
        identifier,
        network: "eth0",
        pdos: createPDOs(),
      });
      const rendered = await renderRead({
        ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
        channels: [
          {
            ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
            dataType,
          },
        ],
      });
      return { slave, ...rendered };
    };

    it("should reuse the index and data channels of the deleted slave", async () => {
      const identifier = createIdentifier();
      const first = await deploySlave(identifier);
      await deleteSlave(first.slave.key, first.task);
      const second = await deploySlave(identifier);
      expect(second.channel).toBe(first.channel);
      expect(second.slave.properties.readIndex).toBe(first.slave.properties.readIndex);
      expect(second.slave.properties.read.channels).toEqual([
        {
          index: 0x6000,
          subIndex: 1,
          scaled: false,
          calibrated: false,
          channel: first.channel,
        },
      ]);
    });

    it("should reject two entries that share the name of a deleted slave's channel", async () => {
      const identifier = createIdentifier();
      const name = uniqueName("ecat_shared");
      const second = { ...STATUS_ENTRY, name: "Counter", subIndex: 2 };
      const pdos = { inputs: [STATUS_ENTRY, second], outputs: [] };
      const render = async (entries: ethercat.PDOEntry[]) => {
        const slave = await createSlaveDevice(client, testRack.key, {
          identifier,
          network: "eth0",
          pdos,
        });
        const rendered = await renderRead({
          ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
          channels: entries.map((e) => ({
            ...EtherCAT.Task.createEntryReadChannel(slave.key, e),
            name,
          })),
        });
        return { slave, ...rendered };
      };
      const first = await render([STATUS_ENTRY]);
      const deployed = await deployAndAwaitTask(
        client,
        first.container,
        first.draft.key,
        EtherCAT.Task.READ_SCHEMAS,
      );
      first.unmount();
      await deleteSlave(first.slave.key, deployed.key);
      const { container, statuses } = await render([STATUS_ENTRY, second]);
      await clickDeploy(container);
      await awaitStatus(
        statuses,
        new RegExp(`Two channels of the task use the name ${name}\\.`),
      );
    });

    it("should reject an identifier that gives the channel names of a live slave", async () => {
      const base = createIdentifier();
      const owner = await deploySlave(`${base}-a`);
      const { slave, container, statuses } = await renderSlave(`${base}_a`);
      await clickDeploy(container);
      await awaitStatus(
        statuses,
        new RegExp(
          `Device ${owner.slave.name} has an identifier that gives the same channel ` +
            `names as ${base}_a\\. Change the identifier of one of them\\.`,
        ),
      );
      const updated = await client.devices.retrieve({
        key: slave.key,
        schemas: EtherCAT.Device.SLAVE_SCHEMAS,
      });
      expect(updated.properties.readIndex).toBe(0);
    });

    it("should reject an index name that a channel that is not an index holds", async () => {
      const identifier = createIdentifier();
      const taken = await client.channels.create({
        name: `${identifier}_time`,
        dataType: "float32",
        virtual: true,
      });
      const { slave, container, statuses } = await renderSlave(identifier);
      await clickDeploy(container);
      await awaitStatus(
        statuses,
        new RegExp(
          `A channel named ${taken.name} already exists and is not an index\\. ` +
            `Change the identifier of slave ${slave.name}\\.`,
        ),
      );
    });

    it("should reject a data type that differs from the deleted slave's channel", async () => {
      const identifier = createIdentifier();
      const first = await deploySlave(identifier, "uint16");
      await deleteSlave(first.slave.key, first.task);
      const { container, statuses } = await renderSlave(identifier, "int16");
      await clickDeploy(container);
      await awaitStatus(
        statuses,
        new RegExp(
          `^Channel ${identifier}_Status holds uint16 data, not int16\\. Set the ` +
            `data type to uint16, or delete channel ${identifier}_Status so the ` +
            "task can create it again\\.$",
        ),
      );
    });
  });

  it("should show why a channel whose data type does not fit its entry cannot run", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
          dataType: "float64",
        },
      ],
    });
    fireEvent.click(await screen.findByRole("option", { name: /0x6000:01/ }));
    expect(
      await screen.findByText(
        `Data type float64 does not fit entry 0x6000:01 on slave ${slave.name}, ` +
          "which holds 16 bits. Pick a 2 byte number type.",
      ),
    ).toBeTruthy();
  });

  it("should mark the channel list when the task has no channels", async () => {
    const { container } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [],
    });
    await clickDeploy(container);
    expect(
      await screen.findByText("At least one enabled channel is required"),
    ).toBeTruthy();
  });

  it("should surface an error when a slave is not configured", async () => {
    const slave = await createSlaveDevice(
      client,
      testRack.key,
      { identifier: createIdentifier(), network: "eth0", pdos: createPDOs() },
      false,
    );
    const { container, statuses } = await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY)],
    });
    await clickDeploy(container);
    await awaitStatus(statuses, /Failed to/);
    await awaitStatus(statuses, /is not configured/);
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should create the task from a slave with a channel for each input", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: createRepeatedNamePDOs(),
    });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
      { deviceKey: slave.key },
    );
    const addresses = tsk.config.channels.map((ch) =>
      ch.type === "entry"
        ? { device: ch.device, index: ch.index, subIndex: ch.subIndex }
        : ch.type,
    );
    expect(addresses).toEqual(
      slave.properties.pdos.inputs.map(({ index, subIndex }) => ({
        device: slave.key,
        index,
        subIndex,
      })),
    );
  });

  it("should create the task from a slave with a channel for each input of its catalog", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY, STRAIN_ENTRY], outputs: [] },
      settings: createSettingsDescription(),
    });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
      { deviceKey: slave.key },
    );
    expect(
      tsk.config.channels.map((ch) => ch.type === "entry" && [ch.index, ch.subIndex]),
    ).toEqual([[STRAIN_ENTRY.index, STRAIN_ENTRY.subIndex]]);
  });

  it("should calibrate the inputs of a slave that the task starts from", async () => {
    const slave = await createCalibratedSlave();
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
      { deviceKey: slave.key },
    );
    expect(
      tsk.config.channels.map((ch) => ch.type === "entry" && ch.calibrated),
    ).toEqual([true, false]);
  });

  it("should create the task from a slave in DewesoftX mode with no entry", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [STATUS_ENTRY, STRAIN_SAMPLE_ENTRY], outputs: [] },
      settings: createDewesoftXSettings(true),
    });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
      { deviceKey: slave.key },
    );
    expect(tsk.config.channels.map(EtherCAT.Task.hasEntry)).toEqual([false]);
    expect(tsk.config.channels[0].device).toEqual(slave.key);
  });

  it("should create the task from a slave without inputs with one channel on it", async () => {
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier: createIdentifier(),
      network: "eth0",
      pdos: { inputs: [], outputs: createPDOs().outputs },
    });
    const tsk = await createWithHook(
      EtherCAT.Task.useCreateRead,
      EtherCAT.Task.READ_SCHEMAS,
      { deviceKey: slave.key },
    );
    expect(tsk.config.channels).toHaveLength(1);
    expect(tsk.config.channels[0].device).toEqual(slave.key);
  });
});

describe("Read channel names", () => {
  const renderStatus = async (overrides: { disabled?: boolean } = {}) => {
    const identifier = createIdentifier();
    const slave = await createSlaveDevice(client, testRack.key, {
      identifier,
      network: "eth0",
      pdos: createPDOs(),
    });
    await renderRead({
      ...EtherCAT.Task.READ_SCHEMAS.config.parse({}),
      channels: [
        {
          ...EtherCAT.Task.createEntryReadChannel(slave.key, STATUS_ENTRY),
          ...overrides,
        },
      ],
    });
    return await findChannelListItem(channel.escapeInvalidName(`${identifier}_Status`));
  };

  it("should show the name deploying gives a channel in the list", async () => {
    const item = await renderStatus();
    expect(within(item).getByRole("img", { name: "New channel" })).toBeTruthy();
  });

  it("should not mark a disabled channel as new", async () => {
    const item = await renderStatus({ disabled: true });
    expect(within(item).queryByRole("img", { name: "New channel" })).toBeNull();
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { LabJack } from "@/feature/labjack";
import {
  createAnalogReadChannel,
  createDigitalReadChannel,
  createLabJackDevice,
  createThermocoupleReadChannel,
} from "@/feature/labjack/testutil";
import {
  awaitEditableForm,
  createWithHook,
  deployAndAwaitTask,
  findChannelListItem,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { findDialogTriggerByText, uniqueName } from "@/testutil";

const client = createTestClient();

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(LabJack.Task.Read, options);
  await awaitEditableForm();
  return rendered;
};

const createConfig = (
  device: string,
  channels: LabJack.Task.ReadChannel[],
): z.infer<LabJack.Task.ReadSchemas["config"]> => ({
  ...LabJack.Task.READ_SCHEMAS.config.parse({}),
  device,
  channels,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<LabJack.Task.ReadSchemas> = {
  name: "LabJack read task",
  type: LabJack.Task.READ_TYPE,
  config: LabJack.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<LabJack.Task.ReadSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, LabJack.Task.READ_SCHEMAS);

describe("Read", () => {
  it("should prompt for a selection when the form carries no device", async () => {
    const draft = await createDraft(client, createConfig("", []));
    await renderRead({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText("No device selected")).toBeTruthy());
  });

  it("should prompt to configure an unconfigured device", async () => {
    const dev = await createLabJackDevice(client, { configured: false });
    const draft = await createDraft(client, createConfig(dev.key, []));
    await renderRead({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText(`Configure ${dev.name}`)).toBeTruthy());
  });

  it("should address each channel of a config built without explicit keys", async () => {
    const dev = await createLabJackDevice(client);
    // A client that leaves the key to the schema, the way every generated client and
    // every documented example does.
    const channels = ["alpha", "bravo", "charlie"].map((name, i) => ({
      ...LabJack.Task.createReadChannel("analog"),
      port: `AIN${i}`,
      name,
    }));
    const draft = await createDraft(client, createConfig(dev.key, channels));
    await renderRead({ client, taskKey: draft.key });
    for (const name of ["alpha", "bravo", "charlie"])
      await waitFor(() => expect(screen.getAllByText(name)).toHaveLength(1));
  });

  it("should render channel ports using their model aliases when available", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogReadChannel("AIN0"),
        createAnalogReadChannel("AIN4"),
        createDigitalReadChannel("DIO8"),
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    await findChannelListItem("AIN0");
    await findChannelListItem("FIO4");
    await findChannelListItem("EIO0");
  });

  it("should fall back to the raw port when the model does not list it", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN999")]),
    );
    await renderRead({ client, taskKey: draft.key });
    await findChannelListItem("AIN999");
  });

  it("should show the AI detail form with its scale editor", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Max voltage")).toBeTruthy());
    expect(screen.getByText("Scale", { selector: "label" })).toBeTruthy();
    expect(screen.queryByText("Slope")).toBeNull();
  });

  it("should show slope and offset for a linear-scaled AI channel", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogReadChannel("AIN0", {
          scale: { type: "linear", slope: 2, offset: 1 },
        }),
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Slope")).toBeTruthy());
    expect(screen.getByText("Offset")).toBeTruthy();
  });

  it("should show the thermocouple detail form for a TC channel", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createThermocoupleReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Thermocouple type")).toBeTruthy());
    expect(screen.getByText("Temperature units")).toBeTruthy();
    expect(screen.getByText("Negative channel")).toBeTruthy();
    // The port selector is the only control for the positive lead.
    expect(screen.queryByText("Positive channel")).toBeNull();
    expect(screen.getByText("CJC source")).toBeTruthy();
    expect(screen.getByText("CJC slope")).toBeTruthy();
    expect(screen.getByText("CJC offset")).toBeTruthy();
  });

  it("should show no extra detail fields for a DI channel", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalReadChannel("DIO8")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("EIO0"));
    await waitFor(() => expect(screen.getByText("Channel type")).toBeTruthy());
    expect(screen.queryByText("Max voltage")).toBeNull();
    expect(screen.queryByText("Thermocouple type")).toBeNull();
  });

  it("should swap the channel type and remap its port to the new port space", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Max voltage")).toBeTruthy());
    fireEvent.click(await findDialogTriggerByText("Analog input"));
    fireEvent.click(await screen.findByText("Digital input"));
    await waitFor(() => expect(screen.queryByText("Max voltage")).toBeNull());
    await waitFor(() => expect(screen.getAllByText("FIO4").length).toBeGreaterThan(0));
  });

  it("should drop the scale section when the channel swaps to an unscaled type", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() =>
      expect(screen.getByText("Scale", { selector: "label" })).toBeTruthy(),
    );
    fireEvent.click(await findDialogTriggerByText("Analog input"));
    fireEvent.click(await screen.findByText("Digital input"));
    await waitFor(() =>
      expect(screen.queryByText("Scale", { selector: "label" })).toBeNull(),
    );
    fireEvent.click(await findDialogTriggerByText("Digital input"));
    fireEvent.click(await screen.findByText("Thermocouple"));
    await waitFor(() =>
      expect(screen.getByText("Scale", { selector: "label" })).toBeTruthy(),
    );
  });

  it("should create the index and data channels, update the device, and save the task", async () => {
    const dev = await createLabJackDevice(client);
    const namedChannel = uniqueName("lj_named");
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogReadChannel("AIN0"),
        createDigitalReadChannel("DIO8", { name: namedChannel }),
      ]),
    );
    const { container } = await renderRead({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      LabJack.Task.READ_SCHEMAS,
    );
    expect(created.type).toBe(LabJack.Task.READ_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [ai, di] = created.config.channels;
    expect(ai.channel).not.toBe(0);
    expect(di.channel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const aiChannel = await client.channels.retrieve(ai.channel);
    expect(aiChannel.name).toBe(`${identifier}_AIN0`);
    expect(aiChannel.dataType.toString()).toBe("float32");
    const diChannel = await client.channels.retrieve(di.channel);
    expect(diChannel.name).toBe(namedChannel);
    expect(diChannel.dataType.toString()).toBe("uint8");

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: LabJack.Device.SCHEMAS,
    });
    expect(updated.properties.readIndex).not.toBe(0);
    expect(updated.properties.AI.channels.AIN0).toBe(ai.channel);
    expect(updated.properties.DI.channels.DIO8).toBe(di.channel);
    const index = await client.channels.retrieve(updated.properties.readIndex);
    expect(index.name).toBe(`${identifier}_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createLabJackDevice(client);
    const config = createConfig(dev.key, [createAnalogReadChannel("AIN0")]);
    const firstDraft = await createDraft(client, config);
    const first = await renderRead({ client, taskKey: firstDraft.key });
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      LabJack.Task.READ_SCHEMAS,
    );
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const secondDraft = await createDraft(client, config);
    await renderRead({ client, taskKey: secondDraft.key });
    await screen.findByText(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: secondDraft.key,
        schemas: LabJack.Task.READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should reuse existing channels when redeployed", async () => {
    const dev = await createLabJackDevice(client);
    const config = createConfig(dev.key, [createAnalogReadChannel("AIN0")]);
    const firstDraft = await createDraft(client, config);
    const first = await renderRead({ client, taskKey: firstDraft.key });
    const firstTask = await deployAndAwaitTask(
      client,
      first.container,
      firstDraft.key,
      LabJack.Task.READ_SCHEMAS,
    );
    first.unmount();

    const secondDraft = await createDraft(client, config);
    const second = await renderRead({ client, taskKey: secondDraft.key });
    const secondTask = await deployAndAwaitTask(
      client,
      second.container,
      secondDraft.key,
      LabJack.Task.READ_SCHEMAS,
    );
    expect(secondTask.config.channels[0].channel).toBe(
      firstTask.config.channels[0].channel,
    );
  });

  it("should recreate the index when the stored one no longer exists", async () => {
    const dev = await createLabJackDevice(client, {
      properties: { readIndex: 999999999 },
    });
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    const { container } = await renderRead({ client, taskKey: draft.key });
    await deployAndAwaitTask(client, container, draft.key, LabJack.Task.READ_SCHEMAS);
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: LabJack.Device.SCHEMAS,
    });
    expect(updated.properties.readIndex).not.toBe(0);
    expect(updated.properties.readIndex).not.toBe(999999999);
    const index = await client.channels.retrieve(updated.properties.readIndex);
    expect(index.isIndex).toBe(true);
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      LabJack.Task.useCreateRead,
      LabJack.Task.READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });
});

describe("Read ports", () => {
  const renderWith = async (
    channels: LabJack.Task.ReadChannel[],
    model?: LabJack.Device.Model,
  ) => {
    const dev = await createLabJackDevice(client, { model });
    const draft = await createDraft(client, createConfig(dev.key, channels));
    const rendered = await renderRead({ client, taskKey: draft.key });
    return { dev, draft, ...rendered };
  };

  const savedPorts = async (key: string) => {
    const saved = await client.tasks.retrieve({
      key,
      schemas: LabJack.Task.READ_SCHEMAS,
    });
    return saved.config.channels.map(({ type, port }) => [type, port]);
  };

  const cell = (name: string, group: string) =>
    within(screen.getByRole("group", { name: group })).getByRole("button", { name });

  const openPorts = async () => {
    fireEvent.click(await screen.findByText("Ports"));
    await screen.findByRole("group", { name: "Analog inputs" });
  };

  it("should set the channel's port from a click on the grid", async () => {
    const { draft } = await renderWith([createAnalogReadChannel("AIN0")]);
    fireEvent.click(await findChannelListItem("AIN0"));
    await screen.findByRole("group", { name: "AIN0 to AIN3" });
    expect(cell("AIN0", "AIN0 to AIN3").getAttribute("aria-pressed")).toBe("true");
    fireEvent.click(cell("AIN2", "AIN0 to AIN3"));
    await waitFor(() =>
      expect(cell("AIN2", "AIN0 to AIN3").getAttribute("aria-pressed")).toBe("true"),
    );
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([["analog", "AIN2"]]),
    );
  });

  it("should add a channel on the lowest free port", async () => {
    const { draft } = await renderWith([
      createAnalogReadChannel("AIN0"),
      createAnalogReadChannel("AIN1"),
    ]);
    fireEvent.click(await screen.findByText("Add channel"));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["analog", "AIN0"],
        ["analog", "AIN1"],
        ["analog", "AIN2"],
      ]),
    );
  });

  it("should move a channel to the lowest free port of its new type", async () => {
    const { draft } = await renderWith([
      createAnalogReadChannel("AIN0"),
      createDigitalReadChannel("DIO4"),
    ]);
    fireEvent.click(await findChannelListItem("AIN0"));
    fireEvent.click(await findDialogTriggerByText("Analog input"));
    fireEvent.click(await screen.findByText("Digital input"));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["digital", "DIO5"],
        ["digital", "DIO4"],
      ]),
    );
  });

  it("should add a channel on a free port, of the type the port takes", async () => {
    const { draft } = await renderWith([createThermocoupleReadChannel("AIN0")]);
    await openPorts();
    fireEvent.click(cell("AIN1", "Analog inputs"));
    await waitFor(async () => expect(await savedPorts(draft.key)).toHaveLength(2));
    fireEvent.click(
      within(screen.getByRole("group", { name: "Digital inputs" })).getByRole(
        "button",
        { name: "FIO5" },
      ),
    );
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["thermocouple", "AIN0"],
        ["thermocouple", "AIN1"],
        ["digital", "DIO5"],
      ]),
    );
    expect(cell("FIO5", "Digital inputs").getAttribute("aria-pressed")).toBe("true");
  });

  it("should select the channel on a used port", async () => {
    await renderWith([
      createAnalogReadChannel("AIN0", { name: "chan_first" }),
      createAnalogReadChannel("AIN1", { name: "chan_second" }),
    ]);
    await openPorts();
    fireEvent.click(cell("AIN1", "Analog inputs"));
    await waitFor(() =>
      expect(cell("AIN1", "Analog inputs").getAttribute("aria-pressed")).toBe("true"),
    );
    expect(cell("AIN0", "Analog inputs").getAttribute("aria-pressed")).toBe("false");
    expect(screen.getByText("Port AIN1")).toBeTruthy();
  });

  it("should mark a port that two channels share", async () => {
    await renderWith([
      createAnalogReadChannel("AIN0"),
      createAnalogReadChannel("AIN0"),
    ]);
    await openPorts();
    expect(cell("AIN0", "Analog inputs").className).toContain(
      "console-port-grid__cell--conflict",
    );
    expect(screen.getByText("1 of 12 ports used")).toBeTruthy();
  });

  it("should open the channel's menu on a used port", async () => {
    const { draft } = await renderWith([createAnalogReadChannel("AIN0")]);
    await openPorts();
    fireEvent.contextMenu(cell("AIN0", "Analog inputs"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(async () => expect(await savedPorts(draft.key)).toEqual([]));
    expect(cell("AIN0", "Analog inputs").className).toContain(
      "console-port-grid__cell--free",
    );
  });

  it("should add a channel from the menu of a free port or of the title", async () => {
    const { draft } = await renderWith([createAnalogReadChannel("AIN0")]);
    await openPorts();
    fireEvent.contextMenu(cell("AIN3", "Analog inputs"));
    fireEvent.click(await screen.findByText("Add channel"));
    await waitFor(async () => expect(await savedPorts(draft.key)).toHaveLength(2));
    fireEvent.contextMenu(screen.getByText("Analog inputs"));
    fireEvent.click(await screen.findByText("Add channel"));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["analog", "AIN0"],
        ["analog", "AIN3"],
        ["analog", "AIN1"],
      ]),
    );
  });

  it.each([
    ["an analog", createAnalogReadChannel("AIN4"), "Digital inputs", 19],
    ["a digital", createDigitalReadChannel("DIO4"), "Analog inputs", 12],
  ])(
    "should mark the terminal %s channel holds as used in the other type",
    async (_, ch, other, total) => {
      await renderWith([ch]);
      await openPorts();
      expect(cell("FIO4", other).className).toContain("console-port-grid__cell--used");
      const section = screen.getByRole("group", { name: other });
      expect(within(section).getByText(`1 of ${total} ports used`)).toBeTruthy();
    },
  );

  it.each([
    ["analog", [createAnalogReadChannel("AIN4"), createDigitalReadChannel("DIO4")]],
    ["digital", [createDigitalReadChannel("DIO4"), createAnalogReadChannel("AIN4")]],
  ])(
    "should mark a terminal two types share as a conflict, with the %s channel first",
    async (_, channels) => {
      await renderWith(channels);
      await openPorts();
      for (const section of ["Analog inputs", "Digital inputs"])
        expect(cell("FIO4", section).className).toContain(
          "console-port-grid__cell--conflict",
        );
      fireEvent.click(cell("FIO4", "Analog inputs"));
      expect(await screen.findByText("Another channel uses FIO4")).toBeTruthy();
    },
  );

  it("should skip a terminal another type holds when it adds a channel", async () => {
    const { draft } = await renderWith([
      createAnalogReadChannel("AIN4"),
      createDigitalReadChannel("DIO6"),
    ]);
    fireEvent.click(await findChannelListItem("FIO6"));
    fireEvent.click(screen.getByText("Add channel"));
    await waitFor(async () =>
      expect(await savedPorts(draft.key)).toEqual([
        ["analog", "AIN4"],
        ["digital", "DIO6"],
        ["digital", "DIO5"],
      ]),
    );
  });

  it("should keep the ports of each type on their own terminals on a T7", async () => {
    await renderWith(
      [createAnalogReadChannel("AIN4"), createDigitalReadChannel("DIO4")],
      LabJack.Device.T7_MODEL,
    );
    await openPorts();
    expect(cell("AIN4", "Analog inputs").className).toContain(
      "console-port-grid__cell--used",
    );
    expect(cell("FIO4", "Digital inputs").className).toContain(
      "console-port-grid__cell--used",
    );
  });

  it("should split the ports of each type into a bank for each terminal prefix", async () => {
    await renderWith([createAnalogReadChannel("AIN0")]);
    await openPorts();
    const banks = (section: string) =>
      within(screen.getByRole("group", { name: section }))
        .getAllByRole("group")
        .map((g) => g.getAttribute("aria-label"));
    expect(banks("Analog inputs")).toEqual([
      "AIN0 to AIN3",
      "FIO4 to FIO7",
      "EIO0 to EIO3",
    ]);
    expect(banks("Digital inputs")).toEqual([
      "FIO4 to FIO7",
      "EIO0 to EIO7",
      "CIO0 to CIO3",
      "MIO0 to MIO2",
    ]);
  });

  it("should label each port by the number of its terminal", async () => {
    await renderWith([createAnalogReadChannel("AIN0")]);
    await openPorts();
    const bank = within(screen.getByRole("group", { name: "Analog inputs" })).getByRole(
      "group",
      { name: "FIO4 to FIO7" },
    );
    expect(
      within(bank)
        .getAllByRole("button")
        .map((b) => [b.getAttribute("aria-label"), b.textContent]),
    ).toEqual([
      ["FIO4", "4"],
      ["FIO5", "5"],
      ["FIO6", "6"],
      ["FIO7", "7"],
    ]);
  });

  it("should name the channels on a port in its tooltip", async () => {
    await renderWith([
      createAnalogReadChannel("AIN0", { name: "chan_a" }),
      createAnalogReadChannel("AIN1", { name: "chan_b" }),
      createAnalogReadChannel("AIN1", { name: "chan_c" }),
    ]);
    await openPorts();
    for (const [port, tooltip] of [
      ["AIN0", "AIN0: chan_a"],
      ["AIN1", "AIN1 is used by chan_b and chan_c"],
      ["AIN2", "AIN2 is free"],
    ]) {
      const target = cell(port, "Analog inputs");
      fireEvent.pointerOver(target, { pointerType: "mouse" });
      expect(await screen.findByText(tooltip)).toBeTruthy();
      fireEvent.pointerOut(target, { pointerType: "mouse" });
    }
  });

  it("should title the ports the MUX80 needs", async () => {
    await renderWith([createAnalogReadChannel("AIN0")], LabJack.Device.T7_MODEL);
    await openPorts();
    const section = screen.getByRole("group", { name: "Analog inputs" });
    expect(within(section).getByText("AIN (MUX80)")).toBeTruthy();
    expect(
      within(section).getByRole("group", { name: "AIN48 to AIN127" }),
    ).toBeTruthy();
  });

  it("should lay a bank out in groups of sixteen ports, eight to a row", async () => {
    await renderWith([createAnalogReadChannel("AIN0")], LabJack.Device.T7_MODEL);
    await openPorts();
    const bank = (section: string, name: string) =>
      within(screen.getByRole("group", { name: section })).getByRole("group", { name });
    const layout = (grid: HTMLElement) =>
      [...grid.children].map((g) => [
        (g as HTMLElement).style.gridTemplateColumns,
        within(g as HTMLElement).getAllByRole("button").length,
      ]);
    const mux = bank("Analog inputs", "AIN48 to AIN127");
    expect(layout(mux)).toEqual(
      Array(5).fill(["repeat(8, var(--console-port-cell))", 16]),
    );
    expect(layout(bank("Digital inputs", "MIO0 to MIO2"))).toEqual([
      ["repeat(3, var(--console-port-cell))", 3],
    ]);
    const last = within(mux).getByRole("button", { name: "AIN127" });
    expect([last.style.gridRow, last.style.gridColumn]).toEqual(["2", "8"]);
  });

  describe("disabled channels", () => {
    it("should show a port a disabled channel holds as used and disabled", async () => {
      await renderWith([
        createAnalogReadChannel("AIN0"),
        createAnalogReadChannel("AIN2", { disabled: true, name: "chan_off" }),
      ]);
      await openPorts();
      const port = cell("AIN2", "Analog inputs");
      expect(port.className).toContain("console-port-grid__cell--used");
      expect(port.className).toContain("console-port-grid__cell--disabled");
      fireEvent.pointerOver(port, { pointerType: "mouse" });
      expect(await screen.findByText("AIN2: chan_off (disabled)")).toBeTruthy();
      fireEvent.pointerOut(port, { pointerType: "mouse" });
      fireEvent.contextMenu(port);
      expect(await screen.findByText("Enable")).toBeTruthy();
    });

    it("should show no conflict for a terminal a disabled channel shares until it is enabled", async () => {
      await renderWith([
        createAnalogReadChannel("AIN4", { disabled: true, name: "chan_off" }),
        createDigitalReadChannel("DIO4", { name: "chan_on" }),
      ]);
      await openPorts();
      for (const section of ["Analog inputs", "Digital inputs"]) {
        const port = cell("FIO4", section);
        expect(port.className).toContain("console-port-grid__cell--used");
        expect(port.className).not.toContain("console-port-grid__cell--conflict");
      }
      fireEvent.click(screen.getByText("List"));
      expect(screen.queryByText("Another channel uses FIO4")).toBeNull();
      fireEvent.click(
        within(await findChannelListItem("chan_off")).getByLabelText("Enabled"),
      );
      expect(await screen.findByText("Another channel uses FIO4")).toBeTruthy();
    });

    it("should deploy a disabled channel on a terminal another type uses", async () => {
      const { container, draft } = await renderWith([
        createAnalogReadChannel("AIN4", { disabled: true }),
        createDigitalReadChannel("DIO4"),
      ]);
      await deployAndAwaitTask(client, container, draft.key);
    });

    it("should deploy and redeploy a disabled channel on a port another channel uses", async () => {
      const dev = await createLabJackDevice(client);
      const firstDraft = await createDraft(
        client,
        createConfig(dev.key, [
          createAnalogReadChannel("AIN0", { disabled: true }),
          createAnalogReadChannel("AIN0"),
        ]),
      );
      const first = await renderRead({ client, taskKey: firstDraft.key });
      const { config } = await deployAndAwaitTask(
        client,
        first.container,
        firstDraft.key,
        LabJack.Task.READ_SCHEMAS,
      );
      expect(config.channels[1].channel).toBe(config.channels[0].channel);
      first.unmount();
      const secondDraft = await createDraft(client, config);
      const second = await renderRead({ client, taskKey: secondDraft.key });
      await deployAndAwaitTask(client, second.container, secondDraft.key);
    });

    it("should name a new shared data channel after its enabled channel", async () => {
      const enabled = uniqueName("chan_on");
      const { container, draft } = await renderWith([
        createAnalogReadChannel("AIN0", {
          disabled: true,
          name: uniqueName("chan_off"),
        }),
        createAnalogReadChannel("AIN0", { name: enabled }),
      ]);
      const { config } = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        LabJack.Task.READ_SCHEMAS,
      );
      const created = await client.channels.retrieve(config.channels[0].channel);
      expect(created.name).toBe(enabled);
    });

    it("should add a channel on the next port past one a disabled channel holds", async () => {
      const { draft } = await renderWith([
        createAnalogReadChannel("AIN0", { disabled: true }),
      ]);
      fireEvent.click(await screen.findByText("Add channel"));
      await waitFor(async () =>
        expect(await savedPorts(draft.key)).toEqual([
          ["analog", "AIN0"],
          ["analog", "AIN1"],
        ]),
      );
    });
  });
});

describe("Read channel names", () => {
  it("should show the name configuring gives each channel in the list", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    const item = await findChannelListItem(`${dev.properties.identifier}_AIN0`);
    expect(within(item).queryByText("No channel")).toBeNull();
  });
});

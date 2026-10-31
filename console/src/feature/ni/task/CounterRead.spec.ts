// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import {
  awaitStatusDescription,
  createHardware,
  createNIDevice,
  renderNITaskForm,
} from "@/feature/ni/task/testutil";
import {
  clickDeploy,
  createWithHook,
  deployAndAwaitTask,
  findChannelListItem,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { findDialogTriggerByText, getIconButton, uniqueName } from "@/testutil";

const client = createTestClient();

type CreateChannelOverrides = Partial<NI.Task.CIChannel>;

const createChannel = (
  type: NI.Task.CIChannelType,
  port: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.CIChannel =>
  ({
    ...NI.Task.createCIChannel(type),
    key: id.create(),
    port,
    device: "placeholder_device",
    ...overrides,
  }) as NI.Task.CIChannel;

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<NI.Task.CounterReadSchemas> = {
  name: "NI counter read task",
  type: NI.Task.COUNTER_READ_TYPE,
  config: NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  config: task.Payload<NI.Task.CounterReadSchemas>["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, NI.Task.COUNTER_READ_SCHEMAS);

const renderCounterRead = async (
  config: task.Payload<NI.Task.CounterReadSchemas>["config"],
) => {
  const draft = await createDraft(config);
  const rendered = await renderNITaskForm(NI.Task.CounterRead, {
    client,
    taskKey: draft.key,
  });
  return { ...rendered, draft };
};

const createConfig = (channels: NI.Task.CIChannel[]) => ({
  ...NI.Task.COUNTER_READ_SCHEMAS.config.parse({}),
  channels,
});

describe("CounterRead", () => {
  it("should render the detail form for every channel type as it is selected", async () => {
    const cases: [NI.Task.CIChannelType, string][] = [
      ["ci_frequency", "Measurement method"],
      ["ci_edge_count", "Count direction"],
      ["ci_period", "Measurement method"],
      ["ci_pulse_width", "Starting edge"],
      ["ci_semi_period", "Scaled units"],
      ["ci_two_edge_sep", "Edge 1"],
      ["ci_velocity_linear", "Distance / pulse"],
      ["ci_velocity_angular", "Pulses / rev"],
      ["ci_position_linear", "Z index enable"],
      ["ci_position_angular", "Initial angle"],
      ["ci_duty_cycle", "Active edge"],
    ];
    await renderCounterRead(
      createConfig(
        cases.map(([type], i) => createChannel(type, i, { name: `chan_${type}` })),
      ),
    );
    for (const [type, distinguishingLabel] of cases) {
      fireEvent.click(await findChannelListItem(`chan_${type}`));
      await waitFor(
        () =>
          expect(
            screen.getAllByText(distinguishingLabel).length,
            `detail form for ${type} did not render "${distinguishingLabel}"`,
          ).toBeGreaterThan(0),
        { onTimeout: (e) => new Error(`${type}: ${e.message}`) },
      );
    }
  });

  it("should hide the scale section for unscaled channel types", async () => {
    await renderCounterRead(
      createConfig([
        createChannel("ci_edge_count", 0, { name: "chan_unscaled" }),
        createChannel("ci_frequency", 1, { name: "chan_scaled" }),
      ]),
    );
    fireEvent.click(await screen.findByText("chan_scaled"));
    await waitFor(() => expect(screen.getByText("Custom scaling")).toBeTruthy());
    fireEvent.click(screen.getByText("chan_unscaled"));
    await waitFor(() => expect(screen.getByText("Count direction")).toBeTruthy());
    expect(screen.queryByText("Custom scaling")).toBeNull();
  });

  it("should reveal the measurement time field for two-counter high-frequency measurement", async () => {
    await renderCounterRead(
      createConfig([createChannel("ci_frequency", 0, { measMethod: "HighFreq2Ctr" })]),
    );
    await waitFor(() => expect(screen.getByText("Measurement time (s)")).toBeTruthy());
    expect(screen.queryByText("Divisor")).toBeNull();
  });

  it("should reveal the divisor field for two-counter large-range measurement", async () => {
    await renderCounterRead(
      createConfig([createChannel("ci_frequency", 0, { measMethod: "LargeRng2Ctr" })]),
    );
    await waitFor(() => expect(screen.getByText("Divisor")).toBeTruthy());
    expect(screen.queryByText("Measurement time (s)")).toBeNull();
  });

  it("should swap the channel to the newly selected type", async () => {
    await renderCounterRead(createConfig([createChannel("ci_frequency", 0)]));
    await screen.findByText("Measurement method");
    await selectFromDropdown("Frequency", "Edge count");
    await waitFor(() => expect(screen.getByText("Count direction")).toBeTruthy());
    expect(screen.queryByText("Measurement method")).toBeNull();
  });

  it("should drop the scale section on a swap to an unscaled type and restore it", async () => {
    await renderCounterRead(createConfig([createChannel("ci_frequency", 0)]));
    await screen.findByText("Custom scaling");
    await selectFromDropdown("Frequency", "Edge count");
    await waitFor(() => expect(screen.queryByText("Custom scaling")).toBeNull());
    await selectFromDropdown("Edge count", "Frequency");
    await screen.findByText("Custom scaling");
  });

  it("should create counter channels and update the device", async () => {
    const dev = await createNIDevice(client);
    const namedChannel = uniqueName("ctr_named");
    const rendered = await renderCounterRead(
      createConfig([
        createChannel("ci_frequency", 0, { device: dev.key }),
        createChannel("ci_edge_count", 1, { device: dev.key, name: namedChannel }),
      ]),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
    const created = await client.tasks.retrieve({
      key: rendered.draft.key,
      schemas: NI.Task.COUNTER_READ_SCHEMAS,
    });
    expect(created.type).toBe(NI.Task.COUNTER_READ_TYPE);
    expect(created.rack).toBe(dev.rack);
    const [c0, c1] = created.config.channels;
    expect(c0.channel).not.toBe(0);
    expect(c1.channel).not.toBe(0);

    const identifier = dev.properties.identifier;
    const defaultNamed = await client.channels.retrieve(c0.channel);
    expect(defaultNamed.name).toBe(`${identifier}_ctr_0`);
    const named = await client.channels.retrieve(c1.channel);
    expect(named.name).toBe(namedChannel);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: NI.Device.SCHEMAS,
    });
    expect(updated.properties.counterInput.channels["0"]).toBe(c0.channel);
    expect(updated.properties.counterInput.channels["1"]).toBe(c1.channel);
    const index = await client.channels.retrieve(updated.properties.counterInput.index);
    expect(index.name).toBe(`${identifier}_ctr_time`);
    expect(index.isIndex).toBe(true);
  });

  it("should bind a new entry to the channel the device already maps", async () => {
    const dev = await createNIDevice(client);
    const config = createConfig([
      createChannel("ci_frequency", 0, { device: dev.key }),
    ]);
    const first = await renderCounterRead(config);
    await deployAndAwaitTask(client, first.container, first.draft.key);
    const firstTask = await client.tasks.retrieve({
      key: first.draft.key,
      schemas: NI.Task.COUNTER_READ_SCHEMAS,
    });
    first.unmount();
    const existing = await client.channels.retrieve(
      firstTask.config.channels[0].channel,
    );
    const second = await renderCounterRead(config);
    await findChannelListItem(existing.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: second.draft.key,
        schemas: NI.Task.COUNTER_READ_SCHEMAS,
      });
      expect(saved.config.channels[0].channel).toBe(existing.key);
    });
  });

  it("should mark the channel list when the task has no channels", async () => {
    const { container } = await renderCounterRead(createConfig([]));
    await clickDeploy(container);
    expect(
      await screen.findByText("At least one enabled channel is required"),
    ).toBeTruthy();
  });

  it("should surface an error when channels span devices on different racks", async () => {
    const devA = await createNIDevice(client);
    const devB = await createNIDevice(client);
    const { statuses, container } = await renderCounterRead(
      createConfig([
        createChannel("ci_frequency", 0, { device: devA.key }),
        createChannel("ci_frequency", 1, { device: devB.key }),
      ]),
    );
    await clickDeploy(container);
    await awaitStatusDescription(
      statuses,
      /Cannot create task with channels from multiple racks/,
    );
  });

  it("should create the task with one channel", async () => {
    const tsk = await createWithHook(
      NI.Task.useCreateCounterRead,
      NI.Task.COUNTER_READ_SCHEMAS,
    );
    expect(tsk.config.channels).toHaveLength(1);
  });

  it("should create the task with one channel on the given device", async () => {
    const dev = await createNIDevice(client);
    const tsk = await createWithHook(
      NI.Task.useCreateCounterRead,
      NI.Task.COUNTER_READ_SCHEMAS,
      { deviceKey: dev.key },
    );
    expect(tsk.config.channels.map(({ device }) => device)).toEqual([dev.key]);
  });
});

describe("CounterRead ports", () => {
  it("should list the PFI terminals of the device", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    await renderCounterRead(
      createConfig([createChannel("ci_frequency", 0, { device: dev.key })]),
    );
    fireEvent.click(await findDialogTriggerByText("Select input terminal"));
    await screen.findByText("PFI3");
    expect(screen.queryByText("PFI4")).toBeNull();
  });

  it("should deploy a channel on a counter that does not report its types", async () => {
    const hardware = createHardware();
    hardware.counterInputs[0] = { ...hardware.counterInputs[0], types: [] };
    const dev = await createNIDevice(client, { properties: { hardware } });
    const rendered = await renderCounterRead(
      createConfig([createChannel("ci_frequency", 0, { device: dev.key })]),
    );
    await deployAndAwaitTask(client, rendered.container, rendered.draft.key);
  });

  it("should mark a counter the device does not have", async () => {
    const dev = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    await renderCounterRead(
      createConfig([createChannel("ci_frequency", 4, { device: dev.key })]),
    );
    expect(await screen.findByText("The device has no counter ctr4")).toBeTruthy();
  });

  it("should add a channel on the lowest free counter of the selected channel's device", async () => {
    const devA = await createNIDevice(client, {
      properties: { hardware: createHardware() },
    });
    const devB = await createNIDevice(client);
    const { container, draft } = await renderCounterRead(
      createConfig([
        createChannel("ci_frequency", 0, { device: devA.key }),
        createChannel("ci_frequency", 0, { device: devB.key }),
        createChannel("ci_frequency", 1, { device: devB.key }),
      ]),
    );
    await screen.findByText("Ports");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: NI.Task.COUNTER_READ_SCHEMAS,
      });
      expect(saved.config.channels.map(({ device, port }) => [device, port])).toEqual([
        [devA.key, 0],
        [devB.key, 0],
        [devB.key, 1],
        [devA.key, 1],
      ]);
    });
  });

  it("should list PFI0 to PFI15 for a device with no port list", async () => {
    const dev = await createNIDevice(client);
    await renderCounterRead(
      createConfig([createChannel("ci_frequency", 0, { device: dev.key })]),
    );
    fireEvent.click(await findDialogTriggerByText("Select input terminal"));
    expect(await screen.findByText("PFI15")).toBeTruthy();
  });
});

describe("CounterRead channel names", () => {
  it("should show the name configuring gives each channel in the list", async () => {
    const dev = await createNIDevice(client);
    await renderCounterRead(
      createConfig([createChannel("ci_frequency", 1, { device: dev.key })]),
    );
    const item = await findChannelListItem(`${dev.properties.identifier}_ctr_1`);
    expect(within(item).queryByText("No channel")).toBeNull();
  });
});

describe("CounterRead disabled channels", () => {
  it("should deploy and redeploy a disabled channel on a counter another channel uses", async () => {
    const dev = await createNIDevice(client);
    const first = await renderCounterRead(
      createConfig([
        createChannel("ci_frequency", 0, { device: dev.key, disabled: true }),
        createChannel("ci_frequency", 0, { device: dev.key }),
      ]),
    );
    const { config } = await deployAndAwaitTask(
      client,
      first.container,
      first.draft.key,
      NI.Task.COUNTER_READ_SCHEMAS,
    );
    expect(config.channels[1].channel).toBe(config.channels[0].channel);
    first.unmount();
    const second = await renderCounterRead(config);
    await deployAndAwaitTask(client, second.container, second.draft.key);
  });
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rack as clientRack, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { id, TimeStamp } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { beforeEach, describe, expect, it } from "vitest";

import { EtherCAT } from "@/feature/ethercat";
import {
  CONTROL_ENTRY,
  createSlaveDevice as createSlave,
  STATUS_ENTRY,
} from "@/feature/ethercat/testutil";
import { Device } from "@/platform/device";
import {
  answerCommand,
  type CommandAnswer,
  reportTaskRunning,
} from "@/platform/task/testutil";
import { createAsyncSynnaxWrapper, uniqueName } from "@/testutil";

const client = createTestClient();

const createSlaveDevice = async (
  rackKey: number,
  properties: Partial<EtherCAT.Device.SlaveProperties> = {},
) => await createSlave(client, rackKey, properties);

let wrapper: React.FC<PropsWithChildren>;
let rack: { key: number };

beforeEach(async () => {
  wrapper = await createAsyncSynnaxWrapper({ client });
  rack = await client.racks.create({ name: `test-ethercat-rack-${id.create()}` });
});

describe("useResultSlave", () => {
  it("should retrieve a slave device by key", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Test Slave",
      network: "eth0",
      position: 1,
    });

    const { result } = renderHook(
      () => EtherCAT.Device.useResultSlave({ key: dev.key }).data,
      { wrapper },
    );

    await waitFor(() => expect(result.current?.key).toEqual(dev.key));
    expect(result.current?.name).toEqual("Test Slave");
  });

  it("should update when device properties change", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Original Name",
      network: "eth0",
    });

    const { result } = renderHook(
      () => EtherCAT.Device.useResultSlave({ key: dev.key }).data,
      { wrapper },
    );

    await waitFor(() => expect(result.current?.name).toEqual("Original Name"));

    await act(async () => {
      await client.devices.rename(dev.key, "Updated Name");
    });

    await waitFor(() => {
      expect(result.current?.name).toEqual("Updated Name");
    });
  });

  it("should return proper SlaveProperties type with PDOs", async () => {
    const pdos = {
      inputs: [
        {
          name: "Status",
          index: 0x6000,
          subIndex: 1,
          bitLength: 16,
          dataType: "uint16",
        },
      ],
      outputs: [
        {
          name: "Control",
          index: 0x7000,
          subIndex: 1,
          bitLength: 16,
          dataType: "uint16",
        },
      ],
    };

    const dev = await createSlaveDevice(rack.key, {
      name: "PDO Device",
      network: "eth0",
      position: 2,
      pdos,
      enabled: true,
    });

    const { result } = renderHook(
      () => EtherCAT.Device.useResultSlave({ key: dev.key }).data,
      { wrapper },
    );

    await waitFor(() =>
      expect(result.current?.properties?.pdos?.inputs).toHaveLength(1),
    );
    expect(result.current?.properties?.pdos?.outputs).toHaveLength(1);
    expect(result.current?.properties?.pdos?.inputs[0].name).toEqual("Status");
    expect(result.current?.properties?.enabled).toBe(true);
  });

  it("should serve the slave from the cache", async () => {
    const dev = await createSlaveDevice(rack.key, { name: "Cached Test" });

    const { result } = renderHook(
      () => EtherCAT.Device.useResultSlave({ key: dev.key }).data,
      { wrapper },
    );

    await waitFor(() => expect(result.current?.key).toEqual(dev.key));
    expect(result.current?.name).toEqual("Cached Test");
  });

  it("should skip the read for a null query", () => {
    const { result } = renderHook(() => EtherCAT.Device.useResultSlave(null).data, {
      wrapper,
    });
    expect(result.current).toBeUndefined();
  });
});

describe("useSlavesByKeys", () => {
  it("should retrieve the slaves the keys name once there are keys", async () => {
    const first = await createSlaveDevice(rack.key, { network: "eth0" });
    const second = await createSlaveDevice(rack.key, { network: "eth1" });
    const none: string[] = [];
    const { result, rerender } = renderHook(
      ({ keys }: { keys: string[] }) => EtherCAT.Device.useSlavesByKeys(keys),
      { wrapper, initialProps: { keys: none } },
    );
    await act(async () => {});
    expect(result.current).toBeUndefined();
    rerender({ keys: [first.key, second.key] });
    await waitFor(() =>
      expect(result.current?.map(({ key }) => key).sort()).toEqual(
        [first.key, second.key].sort(),
      ),
    );
  });
});

describe("useCommonNetwork", () => {
  it("should return network from first device in channels", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Network Test Device",
      network: "test-network",
    });

    const channels: EtherCAT.Task.Channel[] = [
      {
        ...EtherCAT.Task.createEntryReadChannel(dev.key, STATUS_ENTRY),
        channel: 1,
        name: "Test Channel",
      },
    ];

    const { result } = renderHook(() => EtherCAT.Device.useCommonNetwork(channels), {
      wrapper,
    });

    await waitFor(() => expect(result.current).toEqual("test-network"));
  });

  it("should return empty string when channels array is empty", async () => {
    const channels: EtherCAT.Task.Channel[] = [];

    const { result } = renderHook(() => EtherCAT.Device.useCommonNetwork(channels), {
      wrapper,
    });

    expect(result.current).toEqual("");
  });

  it("should return empty string when no channels have devices", async () => {
    const channels: EtherCAT.Task.Channel[] = [
      {
        ...EtherCAT.Task.createEntryReadChannel("", STATUS_ENTRY),
        channel: 1,
        name: "No Device Channel",
      },
    ];

    const { result } = renderHook(() => EtherCAT.Device.useCommonNetwork(channels), {
      wrapper,
    });

    expect(result.current).toEqual("");
  });

  it("should update when device's network changes", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Network Change Device",
      network: "original-network",
    });

    const channels: EtherCAT.Task.Channel[] = [
      {
        ...EtherCAT.Task.createEntryReadChannel(dev.key, STATUS_ENTRY),
        channel: 1,
        name: "Test Channel",
      },
    ];

    const { result } = renderHook(() => EtherCAT.Device.useCommonNetwork(channels), {
      wrapper,
    });

    await waitFor(() => expect(result.current).toEqual("original-network"));

    await act(async () => {
      await Device.setProperties(
        client,
        { key: dev.key, properties: { ...dev.properties, network: "updated-network" } },
        dev.properties,
        EtherCAT.Device.SLAVE_SCHEMAS,
      );
    });

    await waitFor(() => expect(result.current).toEqual("updated-network"));
  });
});

describe("useToggleEnabled", () => {
  it("should toggle enabled from true to false", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Toggle Test Device",
      network: "eth0",
      enabled: true,
    });

    const { result } = renderHook(() => EtherCAT.Device.useToggleEnabled(), {
      wrapper,
    });

    await act(async () => {
      await result.current.updateAsync({ keys: dev.key });
    });

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.enabled).toBe(false);
  });

  it("should toggle enabled from false to true", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Toggle False Device",
      network: "eth0",
      enabled: false,
    });

    const { result } = renderHook(() => EtherCAT.Device.useToggleEnabled(), {
      wrapper,
    });

    await act(async () => {
      await result.current.updateAsync({ keys: dev.key });
    });

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties.enabled).toBe(true);
  });

  it("should toggle multiple devices at once", async () => {
    const dev1 = await createSlaveDevice(rack.key, {
      name: "Multi Device 1",
      network: "eth0",
      enabled: true,
    });
    const dev2 = await createSlaveDevice(rack.key, {
      name: "Multi Device 2",
      network: "eth0",
      enabled: true,
    });

    const { result } = renderHook(() => EtherCAT.Device.useToggleEnabled(), {
      wrapper,
    });

    await act(async () => {
      await result.current.updateAsync({ keys: [dev1.key, dev2.key] });
    });

    const updated1 = await client.devices.retrieve({
      key: dev1.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    const updated2 = await client.devices.retrieve({
      key: dev2.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated1.properties.enabled).toBe(false);
    expect(updated2.properties.enabled).toBe(false);
  });

  it("should apply same enabled value to all devices", async () => {
    const dev1 = await createSlaveDevice(rack.key, {
      name: "Same Value Device 1",
      network: "eth0",
      enabled: true,
    });
    const dev2 = await createSlaveDevice(rack.key, {
      name: "Same Value Device 2",
      network: "eth0",
      enabled: false,
    });

    const { result } = renderHook(() => EtherCAT.Device.useToggleEnabled(), {
      wrapper,
    });

    await act(async () => {
      await result.current.updateAsync({
        keys: [dev1.key, dev2.key],
        enabled: true,
      });
    });

    const updated1 = await client.devices.retrieve({
      key: dev1.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    const updated2 = await client.devices.retrieve({
      key: dev2.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated1.properties.enabled).toBe(true);
    expect(updated2.properties.enabled).toBe(true);
  });

  it("should keep the other properties of the slave", async () => {
    const dev = await createSlaveDevice(rack.key, {
      name: "Kept Properties Device",
      network: "eth1",
      position: 3,
      enabled: true,
    });

    const { result } = renderHook(() => EtherCAT.Device.useToggleEnabled(), {
      wrapper,
    });

    await act(async () => {
      await result.current.updateAsync({ keys: dev.key });
    });

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: EtherCAT.Device.SLAVE_SCHEMAS,
    });
    expect(updated.properties).toMatchObject({
      enabled: false,
      network: "eth1",
      position: 3,
    });
  });
});

describe("useSetStandalone", () => {
  const createReadTask = async (r: clientRack.Rack, device: string) =>
    await r.createTask({
      name: uniqueName("ecat_read"),
      type: EtherCAT.Task.READ_TYPE,
      config: EtherCAT.Task.READ_SCHEMAS.config.parse({
        channels: [EtherCAT.Task.createEntryReadChannel(device, STATUS_ENTRY)],
      }),
    });

  /**
   * Answers each command sent to the tasks as a Driver does, with the answer that
   * answers holds for its type. Resolves with the commands once it answers one of the
   * type last.
   */
  const answerCommandsUntil = async (
    tasks: task.Task[],
    last: string,
    answers: Record<string, Omit<CommandAnswer, "type">> = {},
  ): Promise<{ answered: Promise<task.Command[]> }> => {
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    const answer = async (): Promise<task.Command[]> => {
      const commands: task.Command[] = [];
      try {
        for (;;) {
          const frame = await streamer.read();
          for (const sample of frame.get(task.COMMAND_CHANNEL_NAME)) {
            const cmd = task.commandZ.parse(sample);
            const tsk = tasks.find((t) => t.key === cmd.task);
            if (tsk == null) continue;
            commands.push(cmd);
            await answerCommand(client, tsk, cmd, answers[cmd.type]);
            if (cmd.type === last) return commands;
          }
        }
      } finally {
        streamer.close();
      }
    };
    return { answered: answer() };
  };

  it("should stop the running tasks on the network of the slave, then switch it", async () => {
    const r = await client.racks.create({ name: uniqueName("ecat_rack") });
    const slave = await createSlaveDevice(r.key, { network: "eth0" });
    const neighbor = await createSlaveDevice(r.key, { network: "eth0" });
    const elsewhere = await createSlaveDevice(r.key, { network: "eth1" });
    const readHere = await createReadTask(r, neighbor.key);
    const writeHere = await r.createTask({
      name: uniqueName("ecat_write"),
      type: EtherCAT.Task.WRITE_TYPE,
      config: EtherCAT.Task.WRITE_SCHEMAS.config.parse({
        channels: [EtherCAT.Task.createEntryWriteChannel(slave.key, CONTROL_ENTRY)],
      }),
    });
    const readElsewhere = await createReadTask(r, elsewhere.key);
    const stoppedHere = await createReadTask(r, neighbor.key);
    const scan = await r.createTask({
      name: uniqueName("ecat_scan"),
      type: EtherCAT.Task.SCAN_TYPE,
      config: {},
    });
    for (const tsk of [readHere, writeHere, readElsewhere])
      await reportTaskRunning(client, tsk);
    const driver = await answerCommandsUntil(
      [readHere, writeHere, readElsewhere, stoppedHere, scan],
      "set_standalone",
    );
    const { result } = renderHook(() => EtherCAT.Device.useSetStandalone(), {
      wrapper,
    });
    await act(async () => {
      await result.current.updateAsync({ slave });
    });
    const commands = await driver.answered;
    const stopped = commands.filter((c) => c.type === "stop").map((c) => c.task);
    expect(stopped.sort()).toEqual([readHere.key, writeHere.key].sort());
    expect(commands).toHaveLength(3);
    expect(commands[2]).toMatchObject({
      task: scan.key,
      type: "set_standalone",
      args: { device: slave.key },
    });
  });

  /**
   * Runs a read task on the network of a slave, then switches the slave while the
   * Driver gives answers. Resolves with the commands in the order they arrive.
   */
  const switchWithAnswers = async (
    answers: Record<string, Omit<CommandAnswer, "type">>,
  ): Promise<{ commands: string[]; switched: boolean }> => {
    const r = await client.racks.create({ name: uniqueName("ecat_rack") });
    const slave = await createSlaveDevice(r.key, { network: "eth0" });
    const read = await createReadTask(r, slave.key);
    const scan = await r.createTask({
      name: uniqueName("ecat_scan"),
      type: EtherCAT.Task.SCAN_TYPE,
      config: {},
    });
    await reportTaskRunning(client, read);
    const driver = await answerCommandsUntil([read, scan], "start", answers);
    const { result } = renderHook(() => EtherCAT.Device.useSetStandalone(), {
      wrapper,
    });
    let switched = true;
    await act(async () => {
      switched = await result.current.updateAsync({ slave });
    });
    const names = new Map([
      [read.key, "read"],
      [scan.key, "scan"],
    ]);
    const commands = (await driver.answered).map(
      (c) => `${names.get(c.task)} ${c.type}`,
    );
    return { commands, switched };
  };

  it("should start the stopped tasks again when the switch fails", async () => {
    const result = await switchWithAnswers({
      set_standalone: { variant: "error", message: "Slave 3 refused the switch" },
    });
    expect(result).toEqual({
      commands: ["read stop", "scan set_standalone", "read start"],
      switched: false,
    });
  });

  it("should start the tasks again when a stop fails", async () => {
    const result = await switchWithAnswers({
      stop: { variant: "error", message: "Failed to stop" },
    });
    expect(result).toEqual({ commands: ["read stop", "read start"], switched: false });
  });

  it("should keep the tasks stopped when the Driver gives no answer to the switch", async () => {
    const r = await client.racks.create({ name: uniqueName("ecat_rack") });
    const slave = await createSlaveDevice(r.key, { network: "eth0" });
    const read = await createReadTask(r, slave.key);
    const scan = await r.createTask({
      name: uniqueName("ecat_scan"),
      type: EtherCAT.Task.SCAN_TYPE,
      config: {},
    });
    await reportTaskRunning(client, read);
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    const types: string[] = [];
    const driver = (async () => {
      try {
        for (;;) {
          const frame = await streamer.read();
          for (const sample of frame.get(task.COMMAND_CHANNEL_NAME)) {
            const cmd = task.commandZ.parse(sample);
            if (cmd.task !== read.key && cmd.task !== scan.key) continue;
            types.push(cmd.type);
            if (cmd.type === "stop") await answerCommand(client, read, cmd);
            // The Driver goes down during the switch, so the switch gets no answer.
            if (cmd.type === "set_standalone")
              await client.statuses.set({
                owner: clientRack.resourceID(r.key),
                name: "Rack Status",
                variant: "error",
                message: "Driver not running",
                time: TimeStamp.now(),
                details: { running: false },
              });
            if (cmd.type === "end") return;
          }
        }
      } finally {
        streamer.close();
      }
    })();
    const { result } = renderHook(() => EtherCAT.Device.useSetStandalone(), {
      wrapper,
    });
    let switched = true;
    await act(async () => {
      switched = await result.current.updateAsync({ slave });
    });
    await client.tasks.executeCommand({ task: read.key, type: "end" });
    await driver;
    expect(switched).toBe(false);
    expect(types).toEqual(["stop", "set_standalone", "end"]);
  });
});

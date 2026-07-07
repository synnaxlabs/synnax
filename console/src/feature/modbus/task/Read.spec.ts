// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, task } from "@synnaxlabs/client";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { awaitTaskKey, renderTaskFormView } from "@/platform/task/testutil";
import { getIconButton, stubGeometry } from "@/testutil";

const client = createTestClient();

stubGeometry();

describe("Modbus.Read", () => {
  it("should build channels in the form and create them on the cluster on configure", async () => {
    const dev = await createModbusDevice(client);
    const rendered = await renderTaskFormView(Modbus.Task.Read, Modbus.Task.READ_TYPE, {
      client,
      args: { deviceKey: dev.key },
    });
    const { container } = rendered;
    await screen.findByRole("button", { name: /Configure/ });

    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("Coil");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(screen.getAllByText("Coil")).toHaveLength(2));

    fireEvent.click(screen.getAllByText("Coil")[1]);
    fireEvent.click(await screen.findByText("Register"));
    await screen.findByText("Register");

    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(rendered);

    const tsk = await client.tasks.retrieve({ key: taskKey });
    expect(task.rackKey(tsk.key)).toBe(dev.rack);
    const config = Modbus.Task.READ_SCHEMAS.config.parse(tsk.config);
    expect(config.device).toBe(dev.key);
    expect(config.channels).toHaveLength(2);
    const [coil, register] = config.channels;
    expect(coil.type).toBe("coil_input");
    expect(coil.address).toBe(0);
    expect(coil.channel).not.toBe(0);
    expect(register.type).toBe("register_input");
    expect(register.address).toBe(1);
    expect(register.channel).not.toBe(0);

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(updated.properties.read.index).not.toBe(0);
    expect(updated.properties.read.channels["coil-input-0"]).toBe(coil.channel);
    const index = await client.channels.retrieve(updated.properties.read.index);
    expect(index.name).toBe(`${dev.name}_time`);
    expect(index.isIndex).toBe(true);

    const coilCh = await client.channels.retrieve(coil.channel);
    expect(coilCh.name).toBe(`${dev.name}_coil_input_0`);
    expect(coilCh.index).toBe(index.key);
    const registerCh = await client.channels.retrieve(register.channel);
    expect(registerCh.name.startsWith(`${dev.name}_register_input_1`)).toBe(true);
    expect(registerCh.dataType.toString()).toBe("uint8");
  });

  it("should reuse the existing index and channels when reconfiguring", async () => {
    const dev = await createModbusDevice(client);
    const first = await renderTaskFormView(Modbus.Task.Read, Modbus.Task.READ_TYPE, {
      client,
      args: { deviceKey: dev.key },
    });
    await screen.findByRole("button", { name: /Configure/ });
    fireEvent.click(getIconButton(first.container, "add"));
    await screen.findByText("Coil");
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(first);
    const afterFirst = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    first.unmount();

    await renderTaskFormView(Modbus.Task.Read, Modbus.Task.READ_TYPE, {
      client,
      args: { deviceKey: dev.key, taskKey },
    });
    await screen.findByText("Coil");
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    await waitFor(async () => {
      const afterSecond = await client.devices.retrieve({
        key: dev.key,
        schemas: Modbus.Device.SCHEMAS,
      });
      expect(afterSecond.properties.read.index).toBe(afterFirst.properties.read.index);
      expect(afterSecond.properties.read.channels).toEqual(
        afterFirst.properties.read.channels,
      );
      const matches = await client.channels.retrieve([`${dev.name}_coil_input_0`]);
      expect(matches).toHaveLength(1);
    });
  });
});

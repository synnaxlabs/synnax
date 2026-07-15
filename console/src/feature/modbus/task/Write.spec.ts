// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Modbus } from "@/feature/modbus";
import { createModbusDevice } from "@/feature/modbus/testutil";
import { awaitTaskKey, renderTaskFormTab } from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  getIconButton,
  stubGeometry,
} from "@/testutil";

const client = createTestClient();

stubGeometry();

describe("Modbus.Write", () => {
  it("should create command channels and indexes for the built channels on configure", async () => {
    const dev = await createModbusDevice(client);
    const rendered = await renderTaskFormTab(
      Modbus.Task.Write,
      Modbus.Task.WRITE_TYPE,
      { client, params: { deviceKey: dev.key } },
    );
    const { container } = rendered;
    await screen.findByRole("button", { name: /Configure/ });

    fireEvent.click(getIconButton(container, "add"));
    await screen.findByText("Coil");
    fireEvent.click(getIconButton(container, "add"));
    await waitFor(() => expect(screen.getAllByText("Coil")).toHaveLength(2));

    fireEvent.click(screen.getAllByText("Coil")[1]);
    fireEvent.click(await screen.findByText("Holding Register"));
    await screen.findByText("Holding Register");

    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    const taskKey = await awaitTaskKey(rendered);

    const tsk = await client.tasks.retrieve({ key: taskKey });
    expect(tsk.rack).toBe(dev.rack);
    const config = Modbus.Task.WRITE_SCHEMAS.config.parse(tsk.config);
    expect(config.channels).toHaveLength(2);
    const [coil, holding] = config.channels;
    expect(coil.type).toBe("coil_output");
    expect(coil.address).toBe(0);
    expect(coil.channel).not.toBe(0);
    expect(holding.type).toBe("holding_register_output");
    expect(holding.address).toBe(1);
    expect(holding.channel).not.toBe(0);

    const coilCmd = await client.channels.retrieve(coil.channel);
    expect(coilCmd.name).toBe(`${dev.name}_coil_output_0_cmd`);
    expect(coilCmd.dataType.toString()).toBe("uint8");
    const coilIndex = await client.channels.retrieve(coilCmd.index);
    expect(coilIndex.name).toBe(`${dev.name}_coil_output_0_cmd_time`);
    expect(coilIndex.isIndex).toBe(true);

    const holdingCmd = await client.channels.retrieve(holding.channel);
    expect(holdingCmd.name).toBe(`${dev.name}_holding_register_output_1_cmd`);
    expect(holdingCmd.dataType.toString()).toBe("uint8");

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: Modbus.Device.SCHEMAS,
    });
    expect(updated.properties.write.channels["coil-output-0"]).toBe(coil.channel);
  });

  it("should reuse existing command channels when reconfiguring", async () => {
    const dev = await createModbusDevice(client);
    const first = await renderTaskFormTab(Modbus.Task.Write, Modbus.Task.WRITE_TYPE, {
      client,
      params: { deviceKey: dev.key },
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

    await renderTaskFormTab(Modbus.Task.Write, Modbus.Task.WRITE_TYPE, {
      client,
      params: { deviceKey: dev.key, taskKey },
    });
    await screen.findByText("Coil");
    fireEvent.click(screen.getByRole("button", { name: /Configure/ }));
    await waitFor(async () => {
      const afterSecond = await client.devices.retrieve({
        key: dev.key,
        schemas: Modbus.Device.SCHEMAS,
      });
      expect(afterSecond.properties.write.channels).toEqual(
        afterFirst.properties.write.channels,
      );
      const matches = await client.channels.retrieve([`${dev.name}_coil_output_0_cmd`]);
      expect(matches).toHaveLength(1);
    });
  });

  it("should rename and remove a channel through the context menu", async () => {
    const dev = await createModbusDevice(client);
    const { container } = await renderTaskFormTab(
      Modbus.Task.Write,
      Modbus.Task.WRITE_TYPE,
      { client, params: { deviceKey: dev.key } },
    );
    await screen.findByRole("button", { name: /Configure/ });
    fireEvent.click(getIconButton(container, "add"));
    fireEvent.contextMenu(await screen.findByText("No channel"));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
    fireEvent.contextMenu(screen.getByText("my_cmd_channel"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() => expect(screen.queryByText("my_cmd_channel")).toBeNull());
  });
});

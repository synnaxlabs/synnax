// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { scheduler } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice } from "@/feature/opcua/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import {
  answerCommand,
  answerNextCommand,
  awaitCommand,
} from "@/platform/task/testutil";
import {
  CaptureStatuses,
  createConsoleWrapper,
  getIconButton,
  getIconButtons,
  uniqueName,
} from "@/testutil";

const NODE: OPCUA.Task.ScannedNode = {
  key: "ns=2;s=Demo.Static.Scalar.Float",
  nodeId: "ns=2;s=Demo.Static.Scalar.Float",
  name: "Demo Float",
  nodeClass: "Variable",
  dataType: "float32",
  isArray: false,
  field: "",
};

const ITEM: Haul.Item = { type: OPCUA.Device.HAUL_TYPE, key: NODE.nodeId, data: NODE };

const OTHER: Haul.Item = { type: "other_type", key: "other" };

describe("opc device haul utilities", () => {
  describe("canDropHaulItem", () => {
    it("returns true when at least one item is an opc item", () => {
      expect(
        OPCUA.Device.canDropHaulItem({
          source: OTHER,
          items: [ITEM, OTHER],
        }),
      ).toBe(true);
    });

    it("returns false when no item is an opc item", () => {
      expect(OPCUA.Device.canDropHaulItem({ source: OTHER, items: [OTHER] })).toBe(
        false,
      );
    });
  });
});

describe("Browser", () => {
  it("should surface a scan task retrieval failure as an error status", async () => {
    const client = createTestClient();
    const dev = await createOPCDevice(client);
    const { wrapper } = await createConsoleWrapper({ client });
    let statuses: Status.NotificationSpec[] = [];
    const { container } = render(
      <>
        <OPCUA.Device.Browser device={dev} />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
      { wrapper },
    );
    await screen.findByText("Browser");
    await screen.findByText(/not found/i);
    expect(statuses).toHaveLength(0);
    const refresh = getIconButton(container, "refresh");
    expect(refresh.disabled).toBe(false);
    fireEvent.click(refresh);
    await screen.findByText(/not found/i);
    expect(statuses).toHaveLength(0);
  });
});

const MOTOR: OPCUA.Task.ScannedNode = {
  key: "ns=1;s=TestMotor",
  nodeId: "ns=1;s=TestMotor",
  name: "TestMotor",
  nodeClass: "Variable",
  dataType: "",
  isArray: false,
  field: "",
};

const member = (field: string, name: string, dataType: string) => ({
  ...MOTOR,
  key: `${MOTOR.nodeId}/${field}`,
  name,
  dataType,
  field,
});

describe("Browser structures", () => {
  it("should browse members by field and name them after their structure", async () => {
    const client = createTestClient();
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const dev = await createOPCDevice(client, { rack: rack.key });
    const browse = async (channels: OPCUA.Task.ScannedNode[]) =>
      await answerNextCommand(client, scan, {
        type: OPCUA.Task.BROWSE_COMMAND_TYPE,
        data: { channels },
      });
    const { wrapper } = await createConsoleWrapper({ client });
    const root = await browse([MOTOR]);
    render(<OPCUA.Device.Browser device={dev} />, { wrapper });
    // The command stream decodes the args, so node_id reads as nodeId here.
    expect((await root.answered).args).toMatchObject({ nodeId: "", field: "" });
    const members = await browse([
      member("Speed", "Speed", "float64"),
      member("Position", "Position", ""),
    ]);
    fireEvent.click(await screen.findByText("TestMotor"));
    expect((await members.answered).args).toMatchObject({
      nodeId: MOTOR.nodeId,
      field: "",
    });
    await screen.findByText("TestMotor.Speed");
    const nested = await browse([member("Position.X", "X", "float32")]);
    fireEvent.click(await screen.findByText("TestMotor.Position"));
    expect((await nested.answered).args).toMatchObject({
      nodeId: MOTOR.nodeId,
      field: "Position",
    });
    await screen.findByText("TestMotor.Position.X");
  });
});

const variable = (name: string): OPCUA.Task.ScannedNode => ({
  key: `ns=2;s=${name}`,
  nodeId: `ns=2;s=${name}`,
  name,
  nodeClass: "Variable",
  dataType: "float64",
  isArray: false,
  field: "",
});

const TANK = variable("TankPressure");

const PUMP = variable("PumpSpeed");

const DETAILS = {
  node: TANK,
  nodeClass: "Variable",
  browseName: "TankPressure",
  typeName: "Double",
  valueRank: -1,
  readable: true,
  writable: false,
  historyReadable: true,
  status: "Good",
  value: "2.5",
};

const renderBrowser = async () => {
  const client = createTestClient();
  const rack = await createRunningRack(client, "opc");
  const scan = await rack.createTask({
    name: uniqueName("opcua_scan"),
    type: OPCUA.Task.SCAN_TYPE,
    config: {},
  });
  const dev = await createOPCDevice(client, { rack: rack.key });
  const root = await answerNextCommand(client, scan, {
    type: OPCUA.Task.BROWSE_COMMAND_TYPE,
    data: { channels: [TANK, PUMP] },
  });
  const { wrapper } = await createConsoleWrapper({ client });
  const { container } = render(<OPCUA.Device.Browser device={dev} />, { wrapper });
  await root.answered;
  await screen.findByText("TankPressure");
  return { client, scan, container };
};

describe("Browser search", () => {
  it("should show only the loaded nodes whose name or node ID matches", async () => {
    await renderBrowser();
    fireEvent.change(screen.getByPlaceholderText("Search, or enter a node ID"), {
      target: { value: "pump" },
    });
    await screen.findByText("PumpSpeed");
    expect(screen.queryByText("TankPressure")).toBeNull();
  });

  it("should read the node of a node ID that the user enters", async () => {
    const { client, scan } = await renderBrowser();
    const read = await answerNextCommand(client, scan, {
      type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
      data: DETAILS,
    });
    const input = screen.getByPlaceholderText("Search, or enter a node ID");
    fireEvent.change(input, { target: { value: "ns=2;s=TankPressure" } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect((await read.answered).args).toMatchObject({
      nodeId: "ns=2;s=TankPressure",
      field: "",
    });
    await screen.findByText("Read, History");
  });

  it("should keep the spaces at the end of an entered node ID", async () => {
    const { client, scan } = await renderBrowser();
    const read = await answerNextCommand(client, scan, {
      type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
      data: DETAILS,
    });
    const input = screen.getByPlaceholderText("Search, or enter a node ID");
    fireEvent.change(input, { target: { value: " ns=2;s=Tank " } });
    fireEvent.keyDown(input, { key: "Enter" });
    expect((await read.answered).args).toMatchObject({ nodeId: "ns=2;s=Tank " });
  });

  it("should show the node that the user selects after entering a node ID", async () => {
    const { client, scan } = await renderBrowser();
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    const answerRead = async (): Promise<task.Command> => {
      const cmd = await awaitCommand(
        streamer,
        scan.key,
        OPCUA.Task.READ_NODE_COMMAND_TYPE,
      );
      await answerCommand(client, scan, cmd, { data: DETAILS });
      return cmd;
    };
    try {
      const input = screen.getByPlaceholderText("Search, or enter a node ID");
      fireEvent.click(screen.getByText("TankPressure"));
      await answerRead();
      fireEvent.change(input, { target: { value: "ns=2;s=Other" } });
      fireEvent.keyDown(input, { key: "Enter" });
      expect((await answerRead()).args).toMatchObject({ nodeId: "ns=2;s=Other" });
      fireEvent.change(input, { target: { value: "" } });
      fireEvent.click(screen.getByText("PumpSpeed"));
      await answerRead();
      fireEvent.click(screen.getByText("TankPressure"));
      expect((await answerRead()).args).toMatchObject({ nodeId: TANK.nodeId });
    } finally {
      streamer.close();
    }
  });
});

describe("Browser details", () => {
  it("should show the attributes and value of a selected node", async () => {
    const { client, scan } = await renderBrowser();
    const read = await answerNextCommand(client, scan, {
      type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
      data: DETAILS,
    });
    fireEvent.click(screen.getByText("TankPressure"));
    expect((await read.answered).args).toMatchObject({ nodeId: TANK.nodeId });
    await screen.findByText("Double");
    expect(screen.getByText("Scalar")).toBeTruthy();
    expect(screen.getByText("2.5")).toBeTruthy();
    expect(screen.getByText("Good")).toBeTruthy();
  });

  it("should read the node again when Read again is clicked", async () => {
    const { client, scan, container } = await renderBrowser();
    const first = await answerNextCommand(client, scan, {
      type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
      data: DETAILS,
    });
    fireEvent.click(screen.getByText("TankPressure"));
    await first.answered;
    await screen.findByText("2.5");
    const second = await answerNextCommand(client, scan, {
      type: OPCUA.Task.READ_NODE_COMMAND_TYPE,
      data: { ...DETAILS, value: "3.5" },
    });
    const [, readAgain] = getIconButtons(container, "refresh");
    fireEvent.click(readAgain);
    await second.answered;
    await screen.findByText("3.5");
    expect(screen.queryByText("2.5")).toBeNull();
  });

  it("should read only the latest node selected while a read is open", async () => {
    const { client, scan } = await renderBrowser();
    const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
    try {
      fireEvent.click(screen.getByText("TankPressure"));
      const open = await awaitCommand(
        streamer,
        scan.key,
        OPCUA.Task.READ_NODE_COMMAND_TYPE,
      );
      fireEvent.click(screen.getByText("PumpSpeed"));
      // Lets the effect for PumpSpeed start before the next selection aborts it.
      await act(async () => await scheduler.flushTaskQueue());
      fireEvent.click(screen.getByText("TankPressure"));
      await answerCommand(client, scan, open, { data: DETAILS });
      const next = await awaitCommand(
        streamer,
        scan.key,
        OPCUA.Task.READ_NODE_COMMAND_TYPE,
      );
      expect(next.args).toMatchObject({ nodeId: TANK.nodeId });
      await answerCommand(client, scan, next, { data: { ...DETAILS, value: "3.5" } });
      await screen.findByText("3.5");
    } finally {
      streamer.close();
    }
  });

  it("should mark a node whose browse fails and keep the rest of the tree", async () => {
    const { client, scan } = await renderBrowser();
    const browse = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      variant: "error",
      message: "Failed to browse nodes",
      description: "The server returned BadUserAccessDenied.",
    });
    fireEvent.click(screen.getByText("TankPressure"));
    await browse.answered;
    await screen.findByLabelText(/BadUserAccessDenied/);
    expect(screen.getByText("PumpSpeed")).toBeTruthy();
  });

  it("should show the children of a node whose members do not list, and mark it", async () => {
    const { client, scan } = await renderBrowser();
    const browse = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      variant: "warning",
      message: "Failed to list structure members",
      description:
        "cannot decode Broken: the server did not define the type of a member",
      data: { channels: [variable("TankLevel")] },
    });
    fireEvent.click(screen.getByText("TankPressure"));
    await browse.answered;
    await screen.findByText("TankLevel");
    expect(screen.getByLabelText(/cannot decode Broken/)).toBeTruthy();
  });
});

describe("Browser limits", () => {
  it("should warn only when the server lists more nodes than the Driver loads", async () => {
    const client = createTestClient();
    const rack = await createRunningRack(client, "opc");
    const scan = await rack.createTask({
      name: uniqueName("opcua_scan"),
      type: OPCUA.Task.SCAN_TYPE,
      config: {},
    });
    const dev = await createOPCDevice(client, { rack: rack.key });
    const root = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [TANK, PUMP], truncated: true },
    });
    const { wrapper } = await createConsoleWrapper({ client });
    let statuses: Status.NotificationSpec[] = [];
    render(
      <>
        <OPCUA.Device.Browser device={dev} />
        <CaptureStatuses onStatuses={(s) => (statuses = s)} />
      </>,
      { wrapper },
    );
    await root.answered;
    await waitFor(() =>
      expect(statuses).toEqual([
        expect.objectContaining({
          variant: "warning",
          name: dev.name,
          message: "Showing the first 2 nodes",
          description:
            "The Driver stopped at a limit before it loaded every node. Enter a node ID in the search to open a node that is not shown.",
        }),
      ]),
    );
    const child = await answerNextCommand(client, scan, {
      type: OPCUA.Task.BROWSE_COMMAND_TYPE,
      data: { channels: [variable("TankLevel")], truncated: false },
    });
    fireEvent.click(screen.getByText("TankPressure"));
    await child.answered;
    await screen.findByText("TankLevel");
    expect(statuses).toHaveLength(1);
  });
});

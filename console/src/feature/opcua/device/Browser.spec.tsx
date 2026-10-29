// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Haul } from "@synnaxlabs/lyra/haul";
import { type Status } from "@synnaxlabs/lyra/status";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { createOPCDevice } from "@/feature/opcua/testutil";
import { createRunningRack } from "@/platform/device/testutil";
import { answerNextCommand } from "@/platform/task/testutil";
import {
  CaptureStatuses,
  createConsoleWrapper,
  getIconButton,
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

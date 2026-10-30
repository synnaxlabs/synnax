// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { OPCUA } from "@/feature/opcua";
import { CSV } from "@/feature/opcua/task/csv";

const convertRead = (node: OPCUA.Task.ScannedNode): OPCUA.Task.ReadChannel => ({
  key: OPCUA.Task.nodeIdentity(node),
  nodeName: node.name,
  nodeId: node.nodeId,
  field: node.field,
  dataType: node.dataType,
  channel: 0,
  disabled: false,
  isIndex: false,
  name: "",
});

const convertWrite = (node: OPCUA.Task.ScannedNode): OPCUA.Task.WriteChannel | null =>
  node.field !== ""
    ? null
    : {
        key: node.nodeId,
        nodeName: node.name,
        nodeId: node.nodeId,
        dataType: node.dataType,
        cmdChannel: 0,
        disabled: false,
        name: "",
      };

const decodeRead = (text: string, existing: OPCUA.Task.ReadChannel[] = []) =>
  CSV.decode(text, { existing, convert: convertRead, indexed: true });

const decodeWrite = (text: string, existing: OPCUA.Task.WriteChannel[] = []) =>
  CSV.decode(text, { existing, convert: convertWrite });

interface CreateReadChannelOverrides
  extends
    Partial<Omit<OPCUA.Task.ReadChannel, "nodeId">>,
    Pick<OPCUA.Task.ReadChannel, "nodeId"> {}

const createReadChannel = (
  overrides: CreateReadChannelOverrides,
): OPCUA.Task.ReadChannel => {
  const ch: OPCUA.Task.ReadChannel = {
    key: "",
    nodeName: "",
    field: "",
    dataType: "float32",
    channel: 0,
    disabled: false,
    isIndex: false,
    name: "",
    ...overrides,
  };
  return { ...ch, key: OPCUA.Task.nodeIdentity(ch) };
};

interface CreateWriteChannelOverrides
  extends
    Partial<Omit<OPCUA.Task.WriteChannel, "nodeId">>,
    Pick<OPCUA.Task.WriteChannel, "nodeId"> {}

const createWriteChannel = (
  overrides: CreateWriteChannelOverrides,
): OPCUA.Task.WriteChannel => {
  const ch: OPCUA.Task.WriteChannel = {
    key: "",
    nodeName: "",
    dataType: "float32",
    cmdChannel: 0,
    disabled: false,
    name: "",
    ...overrides,
  };
  return { ...ch, key: ch.nodeId };
};

const READ_CHANNELS = [
  createReadChannel({ nodeId: "ns=2;s=Pump", nodeName: "Pump", name: "pump, inlet" }),
  createReadChannel({
    nodeId: "ns=2;s=Motor",
    nodeName: "Motor.Speed",
    field: "Speed",
    disabled: true,
  }),
  createReadChannel({
    nodeId: "nsu=urn:plc;i=1001",
    nodeName: "1001",
    dataType: "timestamp",
    isIndex: true,
  }),
];

const WRITE_CHANNELS = [
  createWriteChannel({ nodeId: "ns=2;s=Valve", nodeName: "Valve", name: "valve_cmd" }),
  createWriteChannel({
    nodeId: 'ns=2;s="Setpoint"',
    nodeName: '"Setpoint"',
    dataType: "uint8",
    disabled: true,
  }),
];

describe("CSV", () => {
  describe("encode", () => {
    it("should write an index column for a read task", () => {
      expect(CSV.encode(READ_CHANNELS, { indexed: true })).toBe(
        [
          "name,node_name,node_id,field,data_type,enabled,index",
          '"pump, inlet",Pump,ns=2;s=Pump,,float32,true,false',
          ",Motor.Speed,ns=2;s=Motor,Speed,float32,false,false",
          ",1001,nsu=urn:plc;i=1001,,timestamp,true,true",
          "",
        ].join("\n"),
      );
    });

    it("should write no index column for a write task", () => {
      expect(CSV.encode(WRITE_CHANNELS)).toBe(
        [
          "name,node_name,node_id,field,data_type,enabled",
          "valve_cmd,Valve,ns=2;s=Valve,,float32,true",
          ',"""Setpoint""","ns=2;s=""Setpoint""",,uint8,false',
          "",
        ].join("\n"),
      );
    });

    it("should write only the header for an empty channel list", () => {
      expect(CSV.encode([], { indexed: true })).toBe(
        "name,node_name,node_id,field,data_type,enabled,index\n",
      );
    });
  });

  describe("decode", () => {
    it("should decode the channels a read task encodes", () => {
      expect(decodeRead(CSV.encode(READ_CHANNELS, { indexed: true }))).toEqual(
        READ_CHANNELS,
      );
    });

    it("should decode the channels a write task encodes", () => {
      expect(decodeWrite(CSV.encode(WRITE_CHANNELS))).toEqual(WRITE_CHANNELS);
    });

    it("should keep the node names that a folder gives its channels", () => {
      const channels = [
        createReadChannel({
          nodeId: "ns=2;s=Line1.Pressure",
          nodeName: "Tank_Pressure",
        }),
      ];
      expect(decodeRead(CSV.encode(channels, { indexed: true }))).toEqual(channels);
    });

    it("should keep the spaces at the ends of a node ID", () => {
      const channels = [
        createReadChannel({ nodeId: "ns=2;s=Pump ", nodeName: "Pump " }),
      ];
      expect(decodeRead(CSV.encode(channels, { indexed: true }))).toEqual(channels);
    });

    it("should keep commas and quotes inside quoted fields", () => {
      const [ch] = decodeRead(
        'name,node_id\n"flow, total","ns=2;s=Line ""A"", flow"\n',
      );
      expect(ch).toMatchObject({
        name: "flow, total",
        nodeId: 'ns=2;s=Line "A", flow',
        nodeName: 'Line "A", flow',
      });
    });

    it("should decode tab delimited text", () => {
      const [ch] = decodeRead("name\tnode_id\nflow, total\tns=2;s=Flow\n");
      expect(ch).toMatchObject({ name: "flow, total", nodeId: "ns=2;s=Flow" });
    });

    it("should match headers without regard to case and ignore unknown columns", () => {
      const [ch] = decodeRead(
        "Comment,NODE_ID,Data_Type,Name,Enabled\nspare,ns=2;s=Tank,Float64,tank,FALSE\n",
      );
      expect(ch).toMatchObject({
        nodeId: "ns=2;s=Tank",
        dataType: "float64",
        name: "tank",
        disabled: true,
      });
    });

    it("should default a missing column to an enabled float32 channel with no name", () => {
      const [ch] = decodeRead("node_id\nns=2;s=Tank\n");
      expect(ch).toEqual(
        createReadChannel({ nodeId: "ns=2;s=Tank", nodeName: "Tank" }),
      );
    });

    it("should name a node after its node ID when the ID has no identifier type", () => {
      const [ch] = decodeRead("node_id\nTank\n");
      expect(ch.nodeName).toBe("Tank");
    });

    it("should accept 1 and 0 as booleans", () => {
      const channels = decodeRead(
        "node_id,enabled,index\nns=2;s=A,0,1\nns=2;s=B,1,0\n",
      );
      expect(channels.map(({ disabled, isIndex }) => [disabled, isIndex])).toEqual([
        [true, true],
        [false, false],
      ]);
    });

    it("should skip rows for a node and field that the task or an earlier row holds", () => {
      const existing = [createReadChannel({ nodeId: "ns=2;s=Motor" })];
      const channels = decodeRead(
        [
          "node_id,field,name",
          "ns=2;s=Motor,,held",
          "ns=2;s=Motor,Speed,speed",
          "ns=2;s=Pump,,pump",
          "ns=2;s=Pump,,again",
          "ns=2;s=Motor,Speed,again",
        ].join("\n"),
        existing,
      );
      expect(channels.map(({ name }) => name)).toEqual(["speed", "pump"]);
    });

    it("should skip structure field rows for a write task", () => {
      const channels = decodeWrite(
        "node_id,field\nns=2;s=Motor,Speed\nns=2;s=Valve,\n",
      );
      expect(channels).toEqual([
        createWriteChannel({ nodeId: "ns=2;s=Valve", nodeName: "Valve" }),
      ]);
    });

    it("should ignore the index column for a write task", () => {
      const [ch] = decodeWrite("node_id,index\nns=2;s=Valve,maybe\n");
      expect(ch).toEqual(
        createWriteChannel({ nodeId: "ns=2;s=Valve", nodeName: "Valve" }),
      );
    });

    it.each([
      ["no text", "", "The CSV file has no node_id column."],
      [
        "no node_id column",
        "name,field\npump,\n",
        "The CSV file has no node_id column.",
      ],
      ["an empty node_id", "name,node_id\na,ns=2;s=A\nb,\n", "Row 3 has no node_id."],
      [
        "an unknown data type",
        "node_id,data_type\nns=2;s=A,float\n",
        'Row 2 has data_type "float", which is not a Synnax data type.',
      ],
      [
        "the unknown data type",
        "node_id,data_type\nns=2;s=A,unknown\n",
        'Row 2 has data_type "unknown", which is not a Synnax data type.',
      ],
      [
        "an enabled value that is not a boolean",
        "node_id,enabled\nns=2;s=A,yes\n",
        'Row 2 has enabled "yes", which is not true or false.',
      ],
      [
        "an index value that is not a boolean",
        "node_id,index\nns=2;s=A,maybe\n",
        'Row 2 has index "maybe", which is not true or false.',
      ],
    ])("should reject text with %s", (_, text, message) => {
      expect(() => decodeRead(text)).toThrow(new Error(message));
    });
  });
});

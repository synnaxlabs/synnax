// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, type Synnax, type task } from "@synnaxlabs/client";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, screen } from "@testing-library/react";

import {
  type Device,
  MAKE,
  type Properties,
  SCHEMAS,
  ZERO_PROPERTIES,
} from "@/feature/opcua/device/types";
import {
  type CertificateInfo,
  SCAN_TYPE,
  type ScannedNode,
} from "@/feature/opcua/task/types";
import { createRunningRack } from "@/platform/device/testutil";
import { uniqueName } from "@/testutil";

/** A real self-signed certificate, with the details a Driver reports for it. */
export const SERVER_CERTIFICATE: CertificateInfo = {
  pem: `-----BEGIN CERTIFICATE-----
MIIBbjCCARSgAwIBAgIUbAC1w3Ys7iTmtmRuMq9Yc9mWGIYwCgYIKoZIzj0EAwIw
HTEbMBkGA1UEAwwSVGVzdCBPUEMgVUEgU2VydmVyMB4XDTI2MTAyODAzMjYwNFoX
DTI5MTAyNzAzMjYwNFowHTEbMBkGA1UEAwwSVGVzdCBPUEMgVUEgU2VydmVyMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEdYv3TM93q9WM57JxVZXJrPZo/ZbJhbTL
G1+3vlX0+Pegd7JQU3IUR1xGqU/Ru61WW2wvZ4+udIzz18RUlh3VI6MyMDAwHQYD
VR0OBBYEFNp4E77F1OoaPcOBxy4GgjjSHCTfMA8GA1UdEwEB/wQFMAMBAf8wCgYI
KoZIzj0EAwIDSAAwRQIgLX31Bclin170gZ3T2k2Qs7vbnbzDTBnjS4LJ7Cu84TAC
IQDyvwdbi7qXLS7bkjYtMlxZ94PTGKApVOsBgqdDntq6EA==
-----END CERTIFICATE-----
`,
  thumbprint: "FF6516FCE022F97F23B6F679179DB7FE68CB2D9C",
  subject: "CN=Test OPC UA Server",
  notAfter: "2029-10-27T03:26:04Z",
};

export interface CreateOPCDeviceOptions {
  configured?: boolean;
  properties?: Partial<Properties>;
  /** The rack of the device. A new rack when absent. */
  rack?: rack.Key;
}

/** Creates a configured OPC UA server device on the live Core. */
export const createOPCDevice = async (
  client: Synnax,
  { configured = true, properties, rack }: CreateOPCDeviceOptions = {},
): Promise<Device> => {
  const rackKey = rack ?? (await client.racks.create({ name: uniqueName("rack") })).key;
  return await client.devices.create(
    {
      key: id.create(),
      name: uniqueName("opc_server"),
      rack: rackKey,
      location: "opc.tcp://localhost:4840",
      make: MAKE,
      model: "OPC UA server",
      configured,
      properties: { ...deep.copy(ZERO_PROPERTIES), ...properties },
    },
    SCHEMAS,
  );
};

/**
 * Creates a variable node with a unique name, as the browser lists it. A structure node
 * has no data type.
 */
export const createVariableNode = (dataType: string = ""): ScannedNode => {
  const name = uniqueName("Node").replace(/_/g, "");
  return {
    key: `ns=1;s=${name}`,
    nodeId: `ns=1;s=${name}`,
    name,
    nodeClass: "Variable",
    dataType,
    isArray: false,
    field: "",
  };
};

/** Creates a running rack with an OPC UA scan task, and a device on that rack. */
export const createScannedDevice = async (
  client: Synnax,
): Promise<{ scan: task.Task; device: Device }> => {
  const rack = await createRunningRack(client, "opc");
  const scan = await rack.createTask({
    name: uniqueName("opcua_scan"),
    type: SCAN_TYPE,
    config: {},
  });
  return { scan, device: await createOPCDevice(client, { rack: rack.key }) };
};

/** Creates the member at field of a structure node, as the browser lists it. */
export const createMember = (
  node: ScannedNode,
  field: string,
  dataType: string,
): ScannedNode => ({
  ...node,
  key: `${node.nodeId}/${field}`,
  name: field,
  dataType,
  field,
});

/** Drags the browser node with the name onto the element that shows the target. */
export const dragNode = async (name: string, target: string): Promise<void> => {
  fireEvent.dragStart(await screen.findByText(name));
  fireEvent.drop(await screen.findByText(new RegExp(target)));
};

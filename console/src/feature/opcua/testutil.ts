// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { deep, id } from "@synnaxlabs/x";

import { Device } from "@/feature/opcua/device";
import { uniqueName } from "@/testutil";

/** A real self-signed certificate, with the details a Driver reports for it. */
export const SERVER_CERTIFICATE: Device.CertificateInfo = {
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
  properties?: Partial<Device.Properties>;
}

/** Creates a rack and a configured OPC UA server device on the live Core. */
export const createOPCDevice = async (
  client: Synnax,
  { configured = true, properties }: CreateOPCDeviceOptions = {},
): Promise<Device.Device> => {
  const rack = await client.racks.create({ name: uniqueName("rack") });
  return await client.devices.create(
    {
      key: id.create(),
      name: uniqueName("opc_server"),
      rack: rack.key,
      location: "opc.tcp://localhost:4840",
      make: Device.MAKE,
      model: "OPC UA server",
      configured,
      properties: { ...deep.copy(Device.ZERO_PROPERTIES), ...properties },
    },
    Device.SCHEMAS,
  );
};

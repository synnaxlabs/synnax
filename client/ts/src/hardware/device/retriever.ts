// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient, sendRequired } from "@synnaxlabs/freighter";
import { z } from "zod";

import { type Device, deviceKeyZ, deviceZ } from "@/hardware/device/payload";

const RETRIEVE_ENDPOINT = "/hardware/device/retrieve";

const retrieveDeviceReqZ = z.object({
  keys: deviceKeyZ.array(),
});

const retrieveDeviceResZ = z.object({
  devices: deviceZ.array(),
});

export class Retriever {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async retrieve(keys: string[]): Promise<Device[]> {
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(this.client, RETRIEVE_ENDPOINT, { keys }, retrieveDeviceResZ);
    return res.devices;
  }
}

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
import { nullableArrayZ } from "@/util/zod";

const RETRIEVE_ENDPOINT = "/hardware/device/retrieve";

const retrieveDeviceReqZ = z.object({
  search: z.string().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  keys: deviceKeyZ.array().optional(),
});

const retrieveDeviceResZ = z.object({
  devices: nullableArrayZ(deviceZ),
});

export class Retriever {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async search(term: string): Promise<Device[]> {
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(this.client, RETRIEVE_ENDPOINT, { keys: [term] }, retrieveDeviceResZ);
    return res.devices;
  }

  async page(offset: number, limit: number): Promise<Device[]> {
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(this.client, RETRIEVE_ENDPOINT, { offset, limit }, retrieveDeviceResZ);
    return res.devices;
  }
  

  async retrieve(keys: string[]): Promise<Device[]> {
    const res = await sendRequired<
      typeof retrieveDeviceReqZ,
      typeof retrieveDeviceResZ
    >(this.client, RETRIEVE_ENDPOINT, { keys }, retrieveDeviceResZ);
    return res.devices;
  }
}

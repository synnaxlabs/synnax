// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AsyncTermSearcher } from "@synnaxlabs/x/search";
import { toArray } from "@synnaxlabs/x/toArray";

import { type framer } from "@/framer";
import { type Device, deviceZ, type DeviceKey } from "@/hardware/device/payload";
import { type Retriever } from "@/hardware/device/retriever";
import { type Writer } from "@/hardware/device/writer";
import { signals } from "@/signals";

const DEVICE_SET_NAME = "sy_device_set";
const DEVICE_DELETE_NAME = "sy_device_delete";

export class Client implements AsyncTermSearcher<string, DeviceKey, Device> {
  private readonly retriever: Retriever;
  private readonly writer: Writer;
  private readonly frameClient: framer.Client;

  constructor(retriever: Retriever, writer: Writer, frameClient: framer.Client) {
    this.retriever = retriever;
    this.writer = writer;
    this.frameClient = frameClient;
  }

  async create(device: Device): Promise<Device> {
    const res = await this.writer.create([device]);
    return res[0];
  }

  async retrieve(key: string): Promise<Device>;
  async retrieve(keys: string[]): Promise<Device[]>;

  async retrieve(keys: string | string[]): Promise<Device | Device[]> {
    const res = await this.retriever.retrieve(toArray(keys));
    return Array.isArray(keys) ? res : res[0];
  }

  async search(term: string): Promise<Device[]> {
    const res = await this.retriever.search(term);
    return res;
  }

  async page(offset: number, limit: number): Promise<Device[]> {
    const res = await this.retriever.page(offset, limit);
    return res;
  }

  async delete(keys: string[]): Promise<void> {
    await this.writer.delete(keys);
  }

  async openDeviceTracker(): Promise<signals.Observable<string, Device>> {
    return await signals.Observable.open<string, Device>(
      this.frameClient,
      DEVICE_SET_NAME,
      DEVICE_DELETE_NAME,
      decodeDeviceChanges,
    );
  }
}

const decodeDeviceChanges: signals.Decoder<string, Device> = (variant, data) => {
  if (variant === "delete")
    return data.toStrings().map((k) => ({ variant, key: k, value: undefined }));
  return data.parseJSON(deviceZ).map((d) => ({ variant, key: d.key, value: d }));
};

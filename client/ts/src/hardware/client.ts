// Copyright 202 4Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Rack } from "@/hardware/rack";
import { type Retriever } from "@/hardware/retriever";
import { type NewRackPayload, type RackPayload, type Writer, TaskPayload, DevicePayload, deviceZ, NewTaskPayload } from "@/hardware/writer";
import { signals } from "@/signals";
import {framer} from "@/framer";

const DEVICE_SET_NAME = "sy_device_set";
const DEVICE_DELETE_NAME = "sy_device_delete";

export class Client {
  private readonly retriever: Retriever;
  private readonly writer: Writer;
  private readonly frameClient: framer.Client;

  constructor(retriever: Retriever, writer: Writer, frameClient: framer.Client) {
    this.retriever = retriever;
    this.writer = writer;
    this.frameClient = frameClient;
  }

  async createRack(rack: NewRackPayload): Promise<Rack> {
    const res = await this.writer.createRack([rack]);
    return this.sugarRacks(res)[0];
  }

  async retrieveRack(key: number): Promise<Rack> {
    const res = await this.retriever.retrieveRacks([key]);
    return this.sugarRacks(res)[0];
  }

  async createTask(task: NewTaskPayload): Promise<TaskPayload> {
    const res = await this.writer.createTask([task]);
    return res[0];
  }

  async retrieveTask(key: bigint): Promise<TaskPayload> {
    const res = await this.retriever.retrieveTasks(0, [key]);
    return res[0];
  }

  async createDevice(device: DevicePayload): Promise<DevicePayload> {
    const res = await this.writer.createDevice([device]);
    return res[0];
  }

  async retrieveDevice(key: string): Promise<DevicePayload> {
    const res = await this.retriever.retrieveDevices([key]);
    return res[0];
  }

  private sugarRacks(payloads: RackPayload[]): Rack[] {
    return payloads.map((payload) => {
      return new Rack(payload.key, payload.name, this.writer, this.retriever);
    });
  }

  async openDeviceTracker(): Promise<signals.Observable<string, DevicePayload>> {
    return await signals.Observable.open<string, DevicePayload>(
      this.frameClient,
      DEVICE_SET_NAME,
      DEVICE_DELETE_NAME,
      decodeDeviceChanges
    );
  }
}

const decodeDeviceChanges: signals.Decoder<string, DevicePayload> = (variant, data) => {
  if (variant === "delete") return data.toStrings().map((k) => ({ variant, key: k, value: undefined}))
  return data.parseJSON(deviceZ).map((d) => ({ variant, key: d.key, value: d }));
}
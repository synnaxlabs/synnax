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
import { type NewRackPayload, type RackPayload, type Writer, ModulePayload, DevicePayload } from "@/hardware/writer";

export class Client {
  private readonly retriever: Retriever;
  private readonly writer: Writer;

  constructor(retriever: Retriever, writer: Writer) {
    this.retriever = retriever;
    this.writer = writer;
  }

  async createRack(rack: NewRackPayload): Promise<Rack> {
    const res = await this.writer.createRack([rack]);
    return this.sugarRacks(res)[0];
  }

  async retrieveRack(key: number): Promise<Rack> {
    const res = await this.retriever.retrieveRacks([key]);
    return this.sugarRacks(res)[0];
  }

  async retrieveModule(key: bigint): Promise<ModulePayload> {
    const res = await this.retriever.retrieveModules(0, [key]);
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

}

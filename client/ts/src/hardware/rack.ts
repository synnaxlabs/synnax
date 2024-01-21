// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type NewModulePayload,
  type ModulePayload,
  type Writer,
} from "@/hardware/writer";

import { type Retriever } from "@/hardware/retriever";

export class Rack {
  key: number;
  name: string;
  private readonly writer: Writer;
  private readonly retriever: Retriever;

  constructor(key: number, name: string, _writer: Writer, _retriever: Retriever) {
    this.key = key;
    this.name = name;
    this.writer = _writer;
    this.retriever = _retriever;
  }

  async listModules(): Promise<ModulePayload[]> {
    return await this.retriever.retrieveModules(this.key);
  }

  async retrieveModules(): Promise<ModulePayload[]> {
    return [];
  }

  async createModule(module: NewModulePayload): Promise<ModulePayload> {
    module.key = (BigInt(this.key) << 32n) + (BigInt(module.key ?? 0) & 0xffffffffn);
    const res = await this.writer.createModule([module]);
    return res[0];
  }

  async deleteModule(module: bigint): Promise<void> {
    await this.writer.deleteModule([module]);
  }
}

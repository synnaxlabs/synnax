// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { type UnknownRecord } from "@synnaxlabs/x/record";

import { type Key, type Params, type PID } from "@/workspace/pid/payload";
import { Retriever } from "@/workspace/pid/retriever";
import { Writer, type NewPID } from "@/workspace/pid/writer";

export class Client {
  private readonly writer: Writer;
  private readonly retriever: Retriever;

  constructor(client: UnaryClient) {
    this.writer = new Writer(client);
    this.retriever = new Retriever(client);
  }

  async create(workspace: string, pid: NewPID): Promise<PID> {
    return await this.writer.create(workspace, pid);
  }

  async rename(key: Key, name: string): Promise<void> {
    await this.writer.rename(key, name);
  }

  async setData(key: Key, data: UnknownRecord): Promise<void> {
    await this.writer.setData(key, data);
  }

  async retrieve(key: Key): Promise<PID>;

  async retrieve(keys: Key[]): Promise<PID[]>;

  async retrieve(keys: Params): Promise<PID | PID[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }

  async delete(keys: Params): Promise<void> {
    await this.writer.delete(keys);
  }

  async copy(key: Key, name: string, snapshot: boolean): Promise<PID> {
    return await this.writer.copy(key, name, snapshot);
  }
}

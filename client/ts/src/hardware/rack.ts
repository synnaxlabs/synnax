// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type NewTaskPayload,
  type TaskPayload,
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

  async listTasks(): Promise<TaskPayload[]> {
    return await this.retriever.retrieveTasks(this.key);
  }

  async retrieveTasks(): Promise<TaskPayload[]> {
    return [];
  }

  async createTask(task: NewTaskPayload): Promise<TaskPayload> {
    // @ts-expect-error
    task.key = ((BigInt(this.key) << 32n) + (BigInt(task.key ?? 0) & 0xffffffffn)).toString();
    const res = await this.writer.createTask([task]);
    return res[0];
  }

  async deleteTask(task: bigint): Promise<void> {
    await this.writer.deleteTask([task]);
  }
}

// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { toArray } from "@synnaxlabs/x";

import { type NewTask, type Task } from "@/hardware/task/payload";
import { type RetrieveRequest, type Retriever } from "@/hardware/task/retriever";
import { type Writer } from "@/hardware/task/writer";

export class Client {
  private readonly retriever: Retriever;
  private readonly writer: Writer;

  constructor(retriever: Retriever, writer: Writer) {
    this.retriever = retriever;
    this.writer = writer;
  }

  async create(task: NewTask): Promise<Task> {
    const res = await this.writer.create([task]);
    return res[0];
  }

  async retrieve(rack: number): Promise<Task[]>;

  async retrieve(keys: string[]): Promise<Task[]>;

  async retrieve(key: string): Promise<Task>;

  async retrieve(rack: number | string | string[]): Promise<Task | Task[]> {
    const params: RetrieveRequest = {};
    let multi: boolean = true;
    if (typeof rack === "number") params.rack = rack;
    else if (typeof rack === "string") {
      multi = false;
      params.keys = [rack];
    } else params.keys = toArray(rack);
    const res = await this.retriever.retrieve(params);
    return multi ? res : res[0];
  }
}

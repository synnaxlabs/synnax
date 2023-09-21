// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";

import { Creator, type CrudePID } from "@/workspace/pid/creator";
import { Deleter } from "@/workspace/pid/deleter";
import { type Key, type Params, type PID } from "@/workspace/pid/payload";
import { Retriever } from "@/workspace/pid/retriever";

export class Client {
  private readonly writer: Creator;
  private readonly retriever: Retriever;
  private readonly deleter: Deleter;

  constructor(client: UnaryClient) {
    this.writer = new Creator(client);
    this.retriever = new Retriever(client);
    this.deleter = new Deleter(client);
  }

  async set(pid: CrudePID): Promise<PID> {
    return await this.writer.create(pid);
  }

  async retrieve(key: Key): Promise<PID>;

  async retrieve(keys: Key[]): Promise<PID[]>;

  async retrieve(keys: Params): Promise<PID | PID[]> {
    const isMany = Array.isArray(keys);
    const res = await this.retriever.retrieve(keys);
    return isMany ? res : res[0];
  }
}

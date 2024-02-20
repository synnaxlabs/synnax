// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";
import { NewTask, Task, newTaskZ, taskKeyZ, taskZ } from "@/hardware/task/payload";
import { UnaryClient, sendRequired } from "@synnaxlabs/freighter";


const CREATE_ENDPOINT = "/hardware/task/create";
const DELETE_ENDPOINT = "/hardware/task/delete";

const createReqZ = z.object({
  tasks: newTaskZ.array(),
});

const createResZ = z.object({
  tasks: taskZ.array(),
});

const deleteReqZ = z.object({
  keys: taskKeyZ.array(),
});

const deleteResZ = z.object({});

export class Writer {
    private readonly client: UnaryClient;
    
    constructor(client: UnaryClient) {
        this.client = client;
    }
    
    async create(tasks: NewTask[]): Promise<Task[]> {
        const res = await sendRequired<typeof createReqZ, typeof createResZ>(
            this.client,
            CREATE_ENDPOINT,
            { tasks: tasks.map((t) => ({...t, config: JSON.stringify(t.config)})) },
            createResZ,
        );
        return res.tasks;
    }
    
    async delete(keys: bigint[]): Promise<void> {
        await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
            this.client,
            DELETE_ENDPOINT,
            { keys },
            deleteResZ,
        );
    }
}
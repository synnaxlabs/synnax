// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";
import { NewRack, RackPayload, newRackZ, rackKeyZ, rackZ } from "@/hardware/rack/payload";
import { UnaryClient, sendRequired } from "@synnaxlabs/freighter";

const CREATE_RACK_ENDPOINT = "/hardware/rack/create";
const DELETE_RACK_ENDPOINT = "/hardware/rack/delete";

const createReqZ = z.object({
    racks: newRackZ.array(),
});

const createResZ = z.object({
    racks: rackZ.array(),
});

const deleteReqZ = z.object({
    keys: rackKeyZ.array(),
});

const deleteResZ = z.object({});

export class Writer {
    private readonly client: UnaryClient;

    constructor(client: UnaryClient) {
        this.client = client;
    }

    async create(racks: NewRack[]): Promise<RackPayload[]> {
        const res = await sendRequired<typeof createReqZ, typeof createResZ>(
            this.client,
            CREATE_RACK_ENDPOINT,
            { racks },
            createReqZ,
            createResZ,
        );
        return res.racks;
    }

    async delete(keys: number[]): Promise<void> {
        await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
            this.client,
            DELETE_RACK_ENDPOINT,
            { keys },
            deleteReqZ,
            deleteResZ,
        );
    }

}
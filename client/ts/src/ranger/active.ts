// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { QueryError } from "@/errors";
import { type Payload, payloadZ, type Key } from "@/ranger/payload";

const setActiveResZ = z.object({});

const retrieveActiveResZ = z.object({
  range: payloadZ,
});

const clearActiveResZ = z.object({});

const SET_ENDPOINT = "/range/set-active";
const RETRIEVE_ENDPOINT = "/range/retrieve-active";
const CLEAR_ENDPOINT = "/range/clear-active";

export class Active {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async setActive(range: Key): Promise<void> {
    const [, err] = await this.client.send(SET_ENDPOINT, { range }, setActiveResZ);
    if (err != null) throw err;
  }

  async retrieveActive(): Promise<Payload | null> {
    const [res, err] = await this.client.send(
      RETRIEVE_ENDPOINT,
      {},
      retrieveActiveResZ,
    );
    if (err instanceof QueryError) return null;
    if (err != null) throw err;
    return res.range;
  }

  async clearActive(range: Key): Promise<void> {
    const [, err] = await this.client.send(CLEAR_ENDPOINT, { range }, clearActiveResZ);
    if (err != null) throw err;
  }
}

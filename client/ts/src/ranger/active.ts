// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import { QueryError } from "@/errors";
import { type Key, keyZ, type Payload, payloadZ } from "@/ranger/payload";

const setActiveResZ = z.object({});

const retrieveActiveResZ = z.object({
  range: payloadZ,
});

const setActiveReqZ = z.object({
  range: keyZ,
});

const clearActiveReqZ = z.object({
  range: keyZ,
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
    await sendRequired<typeof setActiveReqZ, typeof setActiveResZ>(
      this.client,
      SET_ENDPOINT,
      { range },
      setActiveReqZ,
      setActiveResZ,
    );
  }

  async retrieveActive(): Promise<Payload | null> {
    const [res, err] = await this.client.send(
      RETRIEVE_ENDPOINT,
      {},
      z.object({}),
      retrieveActiveResZ,
    );
    if (QueryError.matches(err)) return null;
    if (err != null) throw err;
    return res.range;
  }

  async clearActive(range: Key): Promise<void> {
    await sendRequired<typeof clearActiveReqZ, typeof clearActiveResZ>(
      this.client,
      CLEAR_ENDPOINT,
      { range },
      clearActiveReqZ,
      clearActiveResZ,
    );
  }
}

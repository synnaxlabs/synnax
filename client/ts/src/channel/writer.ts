// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { z } from "zod";

import {
  type Key,
  keyZ,
  type NewPayload,
  newPayload,
  type Payload,
  payload,
} from "@/channel/payload";
import { type CacheRetriever } from "@/channel/retriever";

const createReqZ = z.object({ channels: newPayload.array() });
const createResZ = z.object({ channels: payload.array() });

const deleteReqZ = z.object({
  keys: keyZ.array().optional(),
  names: z.string().array().optional(),
});
const deleteResZ = z.object({});

const renameReqZ = z.object({
  keys: keyZ.array(),
  names: z.string().array(),
});
const renameResZ = z.object({});

const CREATE_ENDPOINT = "/channel/create";
const DELETE_ENDPOINT = "/channel/delete";
const RENAME_ENDPOINT = "/channel/rename";

export type DeleteProps = z.input<typeof deleteReqZ>;
export type RenameProps = z.input<typeof renameReqZ>;

export class Writer {
  private readonly client: UnaryClient;
  private readonly cache: CacheRetriever;

  constructor(client: UnaryClient, cache: CacheRetriever) {
    this.client = client;
    this.cache = cache;
  }

  async create(channels: NewPayload[]): Promise<Payload[]> {
    const { channels: created } = await sendRequired<
      typeof createReqZ,
      typeof createResZ
    >(this.client, CREATE_ENDPOINT, { channels }, createReqZ, createResZ);
    this.cache.set(created);
    return created;
  }

  async delete(props: DeleteProps): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof deleteResZ>(
      this.client,
      DELETE_ENDPOINT,
      props,
      deleteReqZ,
      deleteResZ,
    );
    if (props.keys != null) this.cache.delete(props.keys);
    if (props.names != null) this.cache.delete(props.names);
  }

  async rename(keys: Key[], names: string[]): Promise<void> {
    await sendRequired<typeof renameReqZ, typeof renameResZ>(
      this.client,
      RENAME_ENDPOINT,
      { keys, names },
      renameReqZ,
      renameResZ,
    );
    this.cache.rename(keys, names);
  }
}

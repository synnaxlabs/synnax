// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { sendRequired, type UnaryClient } from "@synnaxlabs/freighter";
import { array } from "@synnaxlabs/x/array";
import { z } from "zod/v4";

import { type Key, keyZ, type Label, labelZ } from "@/label/payload";
import { ontology } from "@/ontology";

export const newZ = labelZ.extend({ key: keyZ.optional() });
export interface New extends z.infer<typeof newZ> {}

const createReqZ = z.object({ labels: newZ.array() });

const createResZ = z.object({ labels: labelZ.array() });

const deleteReqZ = z.object({ keys: keyZ.array() });

const setReqZ = z.object({
  id: ontology.idZ,
  labels: keyZ.array(),
  replace: z.boolean().optional(),
});

interface SetReq extends z.infer<typeof setReqZ> {}
export interface SetOptions extends Pick<SetReq, "replace"> {}

const removeReqZ = setReqZ.omit({ replace: true });

const emptyResZ = z.object({});

const CREATE_ENDPOINT = "/label/create";
const DELETE_ENDPOINT = "/label/delete";
const SET_ENDPOINT = "/label/set";
const REMOVE_ENDPOINT = "/label/remove";

export class Writer {
  private readonly client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(labels: New | New[]): Promise<Label[]> {
    const res = await sendRequired<typeof createReqZ, typeof createResZ>(
      this.client,
      CREATE_ENDPOINT,
      { labels: array.toArray(labels) },
      createReqZ,
      createResZ,
    );
    return res.labels;
  }

  async delete(keys: Key | Key[]): Promise<void> {
    await sendRequired<typeof deleteReqZ, typeof emptyResZ>(
      this.client,
      DELETE_ENDPOINT,
      { keys: array.toArray(keys) },
      deleteReqZ,
      emptyResZ,
    );
  }

  async set(
    id: ontology.ID,
    labels: Key[],
    { replace }: SetOptions = {},
  ): Promise<void> {
    await sendRequired<typeof setReqZ, typeof emptyResZ>(
      this.client,
      SET_ENDPOINT,
      { id, labels, replace },
      setReqZ,
      emptyResZ,
    );
  }

  async remove(id: ontology.ID, labels: Key[]): Promise<void> {
    await sendRequired<typeof removeReqZ, typeof emptyResZ>(
      this.client,
      REMOVE_ENDPOINT,
      { id, labels },
      removeReqZ,
      emptyResZ,
    );
  }
}

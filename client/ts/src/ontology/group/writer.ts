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

import {
  groupZ,
  type Key,
  keyZ,
  type Name,
  nameZ,
  type Payload,
} from "@/ontology/group/payload";
import { type ID as OntologyID, idZ as ontologyIDZ } from "@/ontology/payload";

const resZ = z.object({ group: groupZ });

const createReqZ = z.object({ parent: ontologyIDZ, key: keyZ.optional(), name: nameZ });

const renameReqZ = z.object({ key: keyZ, name: nameZ });

const deleteReqZ = z.object({ keys: z.array(keyZ) });

export class Writer {
  private static readonly CREATE_ENDPOINT = "/ontology/create-group";
  private static readonly RENAME_ENDPOINT = "/ontology/rename-group";
  private static readonly DELETE_ENDPOINT = "/ontology/delete-group";
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async create(parent: OntologyID, name: Name, key?: Key): Promise<Payload> {
    const res = await sendRequired(
      this.client,
      Writer.CREATE_ENDPOINT,
      { parent, name, key },
      createReqZ,
      resZ,
    );
    return res.group;
  }

  async rename(key: Key, name: Name): Promise<void> {
    await sendRequired(
      this.client,
      Writer.RENAME_ENDPOINT,
      { key, name },
      renameReqZ,
      z.object({}),
    );
  }

  async delete(keys: Key[]): Promise<void> {
    await sendRequired(
      this.client,
      Writer.DELETE_ENDPOINT,
      { keys },
      deleteReqZ,
      z.object({}),
    );
  }
}

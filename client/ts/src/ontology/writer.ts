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

import { type ID, idZ } from "@/ontology/payload";

const ADD_CHILDREN_ENDPOINT = "/ontology/add-children";
const REMOVE_CHILDREN_ENDPOINT = "/ontology/remove-children";
const MOVE_CHILDREN_ENDPOINT = "/ontology/move-children";

const addRemoveChildrenReqZ = z.object({ id: idZ, children: idZ.array() });

const moveChildrenReqZ = z.object({ from: idZ, to: idZ, children: idZ.array() });

const emptyResZ = z.object({});

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    await sendRequired<typeof addRemoveChildrenReqZ, typeof emptyResZ>(
      this.client,
      ADD_CHILDREN_ENDPOINT,
      { id, children },
      addRemoveChildrenReqZ,
      emptyResZ,
    );
  }

  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    await sendRequired<typeof addRemoveChildrenReqZ, typeof emptyResZ>(
      this.client,
      REMOVE_CHILDREN_ENDPOINT,
      { id, children },
      addRemoveChildrenReqZ,
      emptyResZ,
    );
  }

  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    await sendRequired<typeof moveChildrenReqZ, typeof emptyResZ>(
      this.client,
      MOVE_CHILDREN_ENDPOINT,
      { from, to, children },
      moveChildrenReqZ,
      emptyResZ,
    );
  }
}

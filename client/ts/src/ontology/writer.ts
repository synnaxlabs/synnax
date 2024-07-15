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

import { type ID,idZ } from "@/ontology/payload";

const ENDPOINTS = {
  ADD_CHILDREN: "/ontology/add-children",
  REMOVE_CHILDREN: "/ontology/remove-children",
  MOVE_CHILDREN: "/ontology/move-children",
};

const addRemoveChildrenReqZ = z.object({
  id: idZ,
  children: idZ.array(),
});

const moveChildrenReqZ = z.object({
  from: idZ,
  to: idZ,
  children: idZ.array(),
});

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    await sendRequired<typeof addRemoveChildrenReqZ, z.ZodTypeAny>(
      this.client, 
      ENDPOINTS.ADD_CHILDREN, 
      { id, children },
      addRemoveChildrenReqZ, 
      z.object({})
    );
  }

  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    await sendRequired<typeof addRemoveChildrenReqZ, z.ZodTypeAny>(
      this.client,
      ENDPOINTS.REMOVE_CHILDREN,
      { id, children },
      addRemoveChildrenReqZ,
      z.object({}),
    );
  }

  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    const req = { from, to, children };
    await sendRequired(this.client, ENDPOINTS.MOVE_CHILDREN, req, moveChildrenReqZ, z.object({}));
  }
}

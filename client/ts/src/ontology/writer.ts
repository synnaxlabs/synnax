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

import { type ID } from "@/ontology/payload";

const ENDPOINTS = {
  ADD_CHILDREN: "/ontology/add-children",
  REMOVE_CHILDREN: "/ontology/remove-children",
  MOVE_CHILDREN: "/ontology/move-children",
};

export class Writer {
  client: UnaryClient;

  constructor(client: UnaryClient) {
    this.client = client;
  }

  async addChildren(id: ID, ...children: ID[]): Promise<void> {
    const req = { id, children };
    const [, err] = await this.client.send(ENDPOINTS.ADD_CHILDREN, req, z.object({}));
    if (err != null) throw err;
  }

  async removeChildren(id: ID, ...children: ID[]): Promise<void> {
    const req = { id, children };
    const [, err] = await this.client.send(
      ENDPOINTS.REMOVE_CHILDREN,
      req,
      z.object({}),
    );
    if (err != null) throw err;
  }

  async moveChildren(from: ID, to: ID, ...children: ID[]): Promise<void> {
    const req = { from, to, children };
    const [, err] = await this.client.send(ENDPOINTS.MOVE_CHILDREN, req, z.object({}));
    if (err != null) throw err;
  }
}

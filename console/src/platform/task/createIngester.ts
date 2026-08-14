// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, task } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/platform/import";
import { create, type CreateUseCreateParams } from "@/platform/task/useCreate";

export const createIngester =
  <S extends task.Schemas = task.Schemas>(
    params: CreateUseCreateParams<S>,
  ): Import.FileIngester =>
  async (data, { openTab, client }) => {
    if (!Access.createGranted({ id: task.TYPE_ONTOLOGY_ID, client }))
      throw new Error("You do not have permission to import tasks");
    if (client == null) throw new DisconnectedError();
    const created = await create({ client, config: data, ...params });
    const id = task.ontologyID(created.key);
    openTab({ variant: "resource", resource: id });
    return id;
  };

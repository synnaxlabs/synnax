// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/table/layout";

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, workspaceKey },
) => {
  if (!Access.updateGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import tables");
  if (client == null) throw new DisconnectedError();
  const parsed = table.tableZ.parse(data);
  const { key: _ignoredKey, name, ...rest } = parsed;
  const created = await client.tables.create(workspaceKey ?? uuid.ZERO, {
    ...rest,
    name: layout?.name ?? name,
  });
  store.tables.set(created.key, created);
  placeLayout(
    create({ ...layout, key: created.key, name: created.name, type: LAYOUT_TYPE }),
  );
};

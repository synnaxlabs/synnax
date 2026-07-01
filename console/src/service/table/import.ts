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

import { create, LAYOUT_TYPE } from "@/component/table/layout";
import { type Import } from "@/service/import";
import { Session } from "@/session";

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import tables");
  if (client == null) throw new DisconnectedError();
  const newPayload = Session.Table.parseImport(data, layout?.name);
  const created = await client.tables.create(projectKey, newPayload);
  store.tables.set(created.key, created);
  placeLayout(
    create({ ...layout, key: created.key, name: created.name, type: LAYOUT_TYPE }),
  );
  return table.ontologyID(created.key);
};

export const FILE_INGESTERS: Import.FileIngesters = { [LAYOUT_TYPE]: ingest };

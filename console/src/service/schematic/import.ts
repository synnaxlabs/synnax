// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, schematic } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Schematic as CSchematic } from "@/component/schematic";
import { type Import } from "@/service/import";
import { Session } from "@/session";

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: schematic.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import schematics");
  if (client == null) throw new DisconnectedError();
  const newPayload = Session.Schematic.parseImport(data, layout?.name);
  const created = await client.schematics.create(projectKey, newPayload);
  const { key, name } = created;
  store.schematics.set(key, created);
  placeLayout(
    CSchematic.create({ ...layout, key, name, type: CSchematic.LAYOUT_TYPE }),
  );
  return schematic.ontologyID(key);
};

export const FILE_INGESTERS: Import.FileIngesters = {
  [CSchematic.LAYOUT_TYPE]: ingest,
};

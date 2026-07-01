// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, DisconnectedError } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/feature/import";
import { create } from "@/primitive/arc/layout";
import { Session } from "@/session";

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client },
) => {
  if (!Access.updateGranted({ id: arc.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import Arc automations");
  if (client == null) throw new DisconnectedError();
  const newPayload = Session.Arc.parseImport(data, layout?.name);
  const created = await client.arcs.create(newPayload);
  store.arcs.set(created.key, created);
  placeLayout(create({ ...layout, key: created.key, name: created.name }));
  return arc.ontologyID(created.key);
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, lineplot } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/platform/import";
import { create, LAYOUT_TYPE } from "@/platform/lineplot/layout";
import { Session } from "@/session";

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: lineplot.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import line plots");
  if (client == null) throw new DisconnectedError();
  const newPayload = Session.LinePlot.parseImport(data, layout?.name);
  const created = await client.lineplots.create(projectKey, newPayload);
  store.lineplots.set(created.key, created);
  placeLayout(create({ ...layout, key: created.key, name: created.name }));
};

export const FILE_INGESTERS: Import.FileIngesters = { [LAYOUT_TYPE]: ingest };

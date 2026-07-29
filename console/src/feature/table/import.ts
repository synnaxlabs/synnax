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

import { type Import } from "@/platform/import";

// Typeless files are legacy Console states: v0 persists the structural model
// inline (layout/cells), v1 carries selectedCells/hideIndicators alongside an
// optional pendingUpload. The markers are frozen — they describe historical file
// shapes.
const match = (data: Record<string, unknown>): boolean =>
  ("layout" in data && "cells" in data) ||
  "selectedCells" in data ||
  "hideIndicators" in data;

// The Core owns table envelope decoding, legacy-version migration, file-name
// naming, and project parenting, so the file's bytes are streamed up nearly
// untouched — the type field is injected when absent, since legacy Console states
// never carried one — and the table is created under the project in one call.
export const ingest: Import.FileIngester = async (
  data,
  { openTab, store, client, projectKey, fileName },
) => {
  if (!Access.createGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import tables");
  if (client == null) throw new DisconnectedError();
  const body =
    typeof data === "object" && data != null
      ? { type: table.TYPE_ONTOLOGY_ID.type, ...data }
      : data;
  const id = await client.imex.import(JSON.stringify(body), {
    encoding: "JSON",
    fileName,
    project: projectKey,
  });
  openTab({ variant: "resource", resource: id });
  return id;
};
ingest.match = match;

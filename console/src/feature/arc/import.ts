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

import { type Import } from "@/platform/import";

// Typeless files are legacy Console states: v0-v2 persist the graph inline
// alongside text and mode, v3 parks the document under pendingUpload. The markers
// are frozen — they describe historical file shapes.
const match = (data: Record<string, unknown>): boolean =>
  "graph" in data && ("mode" in data || "text" in data || "pendingUpload" in data);

// The Core owns arc envelope decoding, legacy-version migration, and file-name
// naming, so the file's bytes are streamed up nearly untouched — the type field
// is injected when absent, since legacy Console states never carried one. Arcs
// are not project children; the project only scopes the import's access check.
export const ingest: Import.FileIngester = async (
  data,
  { openTab, store, client, projectKey, fileName },
) => {
  if (!Access.createGranted({ id: arc.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import Arc automations");
  if (client == null) throw new DisconnectedError();
  const body =
    typeof data === "object" && data != null
      ? { type: arc.TYPE_ONTOLOGY_ID.type, ...data }
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

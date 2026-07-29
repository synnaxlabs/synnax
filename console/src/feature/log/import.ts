// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, log } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/platform/import";

// Typeless files are legacy Console states, which persist channels as an array
// (bare keys at v0, config objects at v1); no other resource's state does. The
// marker is frozen — it describes historical file shapes.
const match = (data: Record<string, unknown>): boolean => Array.isArray(data.channels);

// The Core owns log envelope decoding, legacy-version migration, file-name naming, and
// project parenting, so the file's bytes are streamed up nearly untouched — the type
// field is injected when absent, since legacy Console states never carried one — and
// the log is created under the project in a single network call.
export const ingest: Import.FileIngester = async (
  data,
  { openTab, store, client, projectKey, fileName },
) => {
  if (!Access.createGranted({ id: log.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import logs");
  if (client == null) throw new DisconnectedError();
  const body =
    typeof data === "object" && data != null
      ? { type: log.TYPE_ONTOLOGY_ID.type, ...data }
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

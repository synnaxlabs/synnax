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

// The Core owns log envelope decoding, legacy-version migration, file-name naming, and
// project parenting, so the file's bytes are streamed up untouched and the log is
// created under the project in a single network call.
export const ingest: Import.FileIngester = async (
  data,
  { openTab, client, projectKey, fileName },
) => {
  if (!Access.createGranted({ id: log.TYPE_ONTOLOGY_ID, client }))
    throw new Error("You do not have permission to import logs");
  if (client == null) throw new DisconnectedError();
  const id = await client.imex.import(JSON.stringify(data), {
    encoding: "JSON",
    fileName,
    project: projectKey,
  });
  openTab({ variant: "resource", resource: id });
  return id;
};

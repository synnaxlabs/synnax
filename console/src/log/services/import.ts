// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Import } from "@/import";
import { create } from "@/log/Log";
import { stateZ } from "@/log/slice";

export const ingest: Import.FileIngestor = (
  data,
  { layout, placeLayout, store, client },
) => {
  const state = stateZ.parse(data);
  if (!Access.updateGranted({ id: log.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import logs");
  // create with an undefined key so we do not have to worry about the key that was from
  // the imported data overwriting existing logs in the cluster
  placeLayout(create({ ...state, key: layout?.key, ...layout }));
};

export const import_ = Import.createImporter(ingest, "log");

export const useImport = (workspaceKey?: string) => Import.use(import_, workspaceKey);

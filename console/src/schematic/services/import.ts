// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/schematic/Schematic";
import { selectHasPermission } from "@/schematic/selectors";
import { anyStateZ } from "@/schematic/slice";

export const ingest: Import.FileIngestor = (data, { layout, placeLayout, store }) => {
  const state = anyStateZ.parse(data);
  const canCreate = selectHasPermission(store.getState());
  if (!canCreate)
    throw new Error(
      "You do not have permission to create a schematic. Please contact an admin to change your permissions.",
    );
  // create with an undefined key so we do not have to worry about the key that was from
  // the imported data overwriting existing schematics in the cluster
  placeLayout(create({ ...state, key: layout?.key, ...layout, type: LAYOUT_TYPE }));
};

export const import_ = Import.createImporter(ingest, "schematic");

export const useImport = (workspaceKey?: string) => Import.use(import_, workspaceKey);

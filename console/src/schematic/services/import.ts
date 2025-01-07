// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

import { Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/schematic/Schematic";
import { selectHasPermission } from "@/schematic/selectors";
import { anyStateZ, migrateState } from "@/schematic/slice";

export const ingest: Import.Ingestor = ({ data, name, store, key, layout }) => {
  const state = migrateState(anyStateZ.parse(JSON.parse(data)));
  const canCreate = selectHasPermission(store.getState());
  if (!canCreate)
    throw new Error(
      "You do not have permission to create a schematic. Please contact an admin to change your permissions.",
    );
  // create with an undefined key so we do not have to worry about existing schematics
  return create({ ...state, name, key, ...layout, type: LAYOUT_TYPE });
};

export const import_ = Import.createImporter(ingest, "schematic");

export const useImport = (workspaceKey?: string) =>
  Import.useImport(import_, workspaceKey);

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/import";
import { stateZ } from "@/table/slice";
import { create } from "@/table/Table";

export const ingest: Import.FileIngester = (
  data,
  { layout, placeLayout, store, client },
) => {
  const state = stateZ.parse(data);
  if (!Access.updateGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import tables");
  // create with an undefined key so we do not have to worry about the key that was from
  // the imported data overwriting existing tables in the cluster
  placeLayout(create({ ...state, key: layout?.key, ...layout }));
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";

import { type Import } from "@/import";
import { create } from "@/lineplot/layout";
import { anyStateZ } from "@/lineplot/slice";

export const ingest: Import.FileIngester = (
  data,
  { layout, placeLayout, store, client },
) => {
  if (!Access.updateGranted({ id: lineplot.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import line plots");
  const state = anyStateZ.parse({ ...(data as record.Unknown), remoteCreated: false });
  placeLayout(create({ ...state, key: layout?.key, ...layout }));
};

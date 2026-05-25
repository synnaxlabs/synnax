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
import { z } from "zod";

import { type Import } from "@/import";
import { create } from "@/lineplot/layout";
import { anyStateZ, type PendingUpload } from "@/lineplot/slice";

// importedBodyZ parses the body fields from a v5-or-earlier export. The v6
// state migration drops these fields entirely because users upgrading their
// own persisted state already have the body on the server; importers, by
// contrast, are explicitly bringing new data into the cluster and need it
// uploaded, so we pluck the body off the raw input before routing through
// anyStateZ and stage it as pendingUpload.
const importedBodyZ = z
  .object({
    title: lineplot.titleZ,
    legend: lineplot.legendZ,
    channels: lineplot.channelsZ,
    ranges: lineplot.rangesZ,
    axes: z.object({ axes: lineplot.axesZ }).transform(({ axes }) => axes),
    lines: z.array(lineplot.lineZ),
    rules: z.array(lineplot.ruleZ),
  })
  .partial();

export const ingest: Import.FileIngester = (
  data,
  { layout, placeLayout, store, client },
) => {
  const state = anyStateZ.parse(data);
  const pendingUpload: PendingUpload = importedBodyZ.parse(data);
  if (!Access.updateGranted({ id: lineplot.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import line plots");
  placeLayout(create({ ...state, pendingUpload, key: layout?.key, ...layout }));
};

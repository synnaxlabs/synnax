// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Import } from "@/import";
import { LinePlot } from "@/lineplot";
import { LinePlotServices } from "@/lineplot/services";
import { Log } from "@/log";
import { LogServices } from "@/log/services";
import { Schematic } from "@/schematic";
import { SchematicServices } from "@/schematic/services";
import { Table } from "@/table";
import { TableServices } from "@/table/services";

export const INGESTORS: Record<string, Import.FileIngestor> = {
  [LinePlot.LAYOUT_TYPE]: LinePlotServices.ingest,
  [Log.LAYOUT_TYPE]: LogServices.ingest,
  [Schematic.LAYOUT_TYPE]: SchematicServices.ingest,
  [Table.LAYOUT_TYPE]: TableServices.ingest,
};

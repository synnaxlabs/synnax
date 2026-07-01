// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@/feature/arc";
import { LinePlot } from "@/feature/lineplot";
import { Log } from "@/feature/log";
import { Schematic } from "@/feature/schematic";
import { Table } from "@/feature/table";
import { Task } from "@/feature/task";
import { type Import } from "@/platform/import";

export const FILE_INGESTERS: Import.FileIngesters = {
  ...Arc.FILE_INGESTERS,
  ...Log.ImEx.FILE_INGESTERS,
  ...LinePlot.ImEx.FILE_INGESTERS,
  ...Schematic.ImEx.FILE_INGESTERS,
  ...Table.ImEx.FILE_INGESTERS,
  ...Task.FILE_INGESTERS,
};

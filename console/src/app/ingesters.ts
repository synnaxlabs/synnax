// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Import } from "@/service/import";
import { Arc } from "@/service/arc";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";
import { Task } from "@/service/task";

export const FILE_INGESTERS: Import.FileIngesters = {
  ...Arc.FILE_INGESTERS,
  ...Log.ImEx.FILE_INGESTERS,
  ...LinePlot.ImEx.FILE_INGESTERS,
  ...Schematic.ImEx.FILE_INGESTERS,
  ...Table.ImEx.FILE_INGESTERS,
  ...Task.FILE_INGESTERS,
};

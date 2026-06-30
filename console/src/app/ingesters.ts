// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Hardware } from "@/hardware";
import { type Import } from "@/import";
import { Arc } from "@/service/arc";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";

export const FILE_INGESTERS: Import.FileIngesters = {
  ...Arc.FILE_INGESTERS,
  ...Hardware.FILE_INGESTERS,
  ...Log.ImEx.FILE_INGESTERS,
  ...LinePlot.ImEx.FILE_INGESTERS,
  ...Schematic.ImEx.FILE_INGESTERS,
  ...Table.ImEx.FILE_INGESTERS,
};

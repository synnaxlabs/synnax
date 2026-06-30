// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Export } from "@/export";
import { Hardware } from "@/hardware";
import { Arc } from "@/service/arc";
import { LinePlot } from "@/service/lineplot";
import { Log } from "@/service/log";
import { Schematic } from "@/service/schematic";
import { Table } from "@/service/table";

export const EXTRACTORS: Export.Extractors = {
  ...Arc.EXTRACTORS,
  ...Hardware.EXTRACTORS,
  ...LinePlot.EXTRACTORS,
  ...Log.EXTRACTORS,
  ...Schematic.EXTRACTORS,
  ...Table.EXTRACTORS,
};
